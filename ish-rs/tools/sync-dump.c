// Reference-output generator for the util/sync differential test.
//
// This links the *unmodified* util/sync.c and drives every entry point it
// exports through the same host seams the Rust port exposes:
//
//   * `pthread_cond_wait` / `pthread_cond_timedwait` are linker-wrapped, so the
//     harness decides whether a wait is woken, times out, or is rejected, and
//     records the absolute deadline C computed for it;
//   * `clock_gettime(CLOCK_MONOTONIC)` is wrapped so the timeout arithmetic is
//     deterministic;
//   * `current` is a fake task whose `pending`, `blocked`, `sighand` and
//     `waiting_cond` / `waiting_lock` fields the corpus scripts.
//
// For every step the harness prints the return value plus the state C's
// contract makes observable: whether the task was published as a waiter while
// the wait was in flight, whether it was cleared afterwards, the deadline, the
// lock owner, and the write-lock counter.
//
// Build (normally via tools/gen_sync_reference.sh):
//   cc -O2 -Wall -Wextra -I<ish-src> -Itools/stub-include
//      -o sync-dump tools/sync-dump.c <ish-src>/util/sync.c -pthread
//      -Wl,--wrap=pthread_cond_wait -Wl,--wrap=pthread_cond_timedwait
//      -Wl,--wrap=clock_gettime

#define _GNU_SOURCE

#include <assert.h>
#include <errno.h>
#include <pthread.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "kernel/task.h"
#include "util/sync.h"

// ---- harness substitutions for the kernel's global state -------------------

__thread struct task *current;
lock_t pids_lock = LOCK_INITIALIZER;

int current_pid(void) {
    return current == NULL ? 0 : current->pid;
}

void ish_printk(const char *UNUSED(message), ...) {}

_Noreturn void die(const char *message, ...) {
    va_list args;
    va_start(args, message);
    vfprintf(stderr, message, args);
    va_end(args);
    fputc('\n', stderr);
    abort();
}

// ---- the scripted pthread host ---------------------------------------------

static struct timespec script_now = {100, 0};
static int script_park_rc;        // returned by the wrapped cond wait
static bool script_pending_after; // whether the wait makes a signal pending

// The wait the harness is currently watching, so it can print whether `wait_for`
// published this task as a waiter while the call was in flight.
static cond_t *watched_cond;
static lock_t *watched_lock;

static void record_park(const char *kind, const struct timespec *deadline) {
    printf("P %s", kind);
    if (deadline != NULL)
        printf(" %ld.%09ld", (long) deadline->tv_sec, (long) deadline->tv_nsec);
    else
        printf(" -");
    printf(" rc=%d pending_after=%d during_cond=%d during_lock=%d\n", script_park_rc,
           script_pending_after, current != NULL && current->waiting_cond != NULL,
           current != NULL && current->waiting_lock != NULL);
}

int __wrap_pthread_cond_wait(pthread_cond_t *cond, pthread_mutex_t *lock) {
    (void) cond;
    (void) lock;
    record_park("wait", NULL);
    if (script_pending_after && current != NULL)
        current->pending = 1;
    return script_park_rc;
}

int __wrap_pthread_cond_timedwait(pthread_cond_t *cond, pthread_mutex_t *lock,
                                  const struct timespec *deadline) {
    (void) cond;
    (void) lock;
    record_park("timedwait", deadline);
    if (script_pending_after && current != NULL)
        current->pending = 1;
    return script_park_rc;
}

int __real_clock_gettime(clockid_t clockid, struct timespec *ts);

int __wrap_clock_gettime(clockid_t clockid, struct timespec *ts) {
    if (clockid == CLOCK_MONOTONIC) {
        *ts = script_now;
        return 0;
    }
    return __real_clock_gettime(clockid, ts);
}

// ---- the fake kernel objects ----------------------------------------------

static struct task task;
static struct sighand sighand;

static void setup(void) {
    memset(&task, 0, sizeof(task));
    memset(&sighand, 0, sizeof(sighand));
    task.pid = 47;
    lock_init(&task.waiting_cond_lock);
    lock_init(&sighand.lock);
    task.sighand = &sighand;
    lock_init(&pids_lock);
}

// ---- the corpus ------------------------------------------------------------

// A wait case: `pending`/`blocked` select the signal state, `timeout` the wait
// flavor, `park_rc` what pthread would return, `pending_after` whether a signal
// lands while the task sleeps, and `hold_sighand_lock` whether the waited-on
// lock *is* the task's sighand lock.
static void wait_case(bool ignore_signals, unsigned long pending, unsigned long blocked,
                      const struct timespec *timeout, int park_rc, bool pending_after,
                      bool hold_sighand_lock) {
    cond_t cond;
    lock_t lock;
    cond_init(&cond);
    lock_init(&lock);
    lock_t *published_lock = hold_sighand_lock ? &sighand.lock : &lock;

    struct task *saved = current;
    current = &task;
    task.pending = pending;
    task.blocked = blocked;
    script_park_rc = park_rc;
    script_pending_after = pending_after;
    watched_cond = &cond;
    watched_lock = published_lock;

    if (hold_sighand_lock)
        lock(&sighand.lock);
    int rc = ignore_signals
                 ? wait_for_ignore_signals(&cond, published_lock, (struct timespec *) timeout)
                 : wait_for(&cond, published_lock, (struct timespec *) timeout);
    if (hold_sighand_lock)
        unlock(&sighand.lock);

    printf("%s pending=%lx blocked=%lx rc=%d cleared_after=%d\n", ignore_signals ? "I" : "W",
           pending, blocked, rc, current->waiting_cond == NULL && current->waiting_lock == NULL);

    watched_cond = NULL;
    watched_lock = NULL;
    current = saved;
    cond_destroy(&cond);
    task.pending = 0;
    task.blocked = 0;
}

// `wait_for` with no current task at all: `is_signal_pending` is false and
// nothing is published.
static void no_task_case(void) {
    cond_t cond;
    lock_t lock;
    cond_init(&cond);
    lock_init(&lock);
    struct task *saved = current;
    current = NULL;
    watched_cond = &cond;
    watched_lock = &lock;
    script_park_rc = 0;
    script_pending_after = false;
    int rc = wait_for(&cond, &lock, NULL);
    printf("W no_current rc=%d\n", rc);
    current = saved;
    watched_cond = NULL;
    watched_lock = NULL;
    cond_destroy(&cond);
}

// The `sigunwind_start` contract.
//
// The "handler fires while an unwind is armed" case is deliberately *not* in
// this corpus: it is the one path where `sigusr1_handler` longjmps into
// `sigunwind_start`'s frame, which a portable oracle cannot script without
// depending on the compiler's handling of `sigsetjmp` inside a `static inline`
// function (a harness at -O2 and a harness at -O0 disagree about where the
// buffer resumes). The remaining contract — arming, disarming, and the
// no-request no-op — is deterministic, and the Rust port models the jump
// itself as an explicit `Unwind::Unwound` transition.
static void unwind_case(void) {
    extern __thread bool should_unwind;
    extern void sigusr1_handler(void);

    int r = sigunwind_start();
    printf("U start rc=%d flag=%d\n", r, should_unwind);
    printf("U armed_again flag=%d\n", should_unwind);
    sigunwind_end();
    printf("U end flag=%d\n", should_unwind);
    // A handler call with no armed unwind is a no-op.
    sigusr1_handler();
    printf("U idle_handler flag=%d\n", should_unwind);
    // Arming after the flag was cleared starts a fresh unwind.
    int again = sigunwind_start();
    printf("U rearm rc=%d flag=%d\n", again, should_unwind);
    sigunwind_end();
    printf("U rearm_end flag=%d\n", should_unwind);
}

static void lock_case(void) {
    lock_t lock;
    lock_init(&lock);
    lock(&lock);
    printf("L locked owner_self=%d\n", pthread_equal(lock.owner, pthread_self()) ? 1 : 0);
    int busy = trylock(&lock);
    printf("L trylock_held rc=%d owner_self=%d\n", busy,
           pthread_equal(lock.owner, pthread_self()) ? 1 : 0);
    unlock(&lock);
    printf("L unlocked owner_cleared=%d\n",
           pthread_equal(lock.owner, zero_init(pthread_t)) ? 1 : 0);
    int again = trylock(&lock);
    printf("L trylock_free rc=%d owner_self=%d\n", again,
           pthread_equal(lock.owner, pthread_self()) ? 1 : 0);
    unlock(&lock);
}

static void wrlock_case(void) {
    wrlock_t rw;
    wrlock_init(&rw);
    printf("R init val=%d\n", rw.val);
    read_wrlock(&rw);
    read_wrlock(&rw);
    printf("R read2 val=%d\n", rw.val);
    read_wrunlock(&rw);
    printf("R read1 val=%d\n", rw.val);
    read_wrunlock(&rw);
    printf("R read0 val=%d\n", rw.val);
    write_wrlock(&rw);
    printf("R write val=%d file=%s line_set=%d pid=%d\n", rw.val, rw.file != NULL ? "set" : "null",
           rw.line != 0, rw.pid);
    write_wrunlock(&rw);
    printf("R write0 val=%d file=%s line_set=%d\n", rw.val, rw.file != NULL ? "set" : "null",
           rw.line != 0);
}

// `notify` must broadcast and `notify_once` must signal, on the condvar the
// caller passed. The wrappers below record which object each call reached.
static cond_t *known_conds[4];
static const char *known_names[4];
static int known_count;

static const char *cond_name(cond_t *cond) {
    for (int i = 0; i < known_count; i++)
        if (known_conds[i] == cond)
            return known_names[i];
    return "unknown";
}

int __wrap_pthread_cond_broadcast(pthread_cond_t *cond) {
    printf("C broadcast=%s\n", cond_name((cond_t *) cond));
    return 0;
}

int __wrap_pthread_cond_signal(pthread_cond_t *cond) {
    printf("C signal=%s\n", cond_name((cond_t *) cond));
    return 0;
}

static void notify_case(void) {
    cond_t a, b;
    cond_init(&a);
    cond_init(&b);
    known_conds[0] = &a;
    known_names[0] = "a";
    known_conds[1] = &b;
    known_names[1] = "b";
    known_count = 2;
    notify(&a);
    notify_once(&b);
    known_count = 0;
    cond_destroy(&a);
    cond_destroy(&b);
}

int main(void) {
    setvbuf(stdout, NULL, _IONBF, 0);
    setup();
    printf("# sync reference, generated from unmodified iSH util/sync.{h,c}\n");

    struct timespec one_second = {1, 0};
    struct timespec normal = {1, 500000000};

    // wait_for: the two EINTR checks, always with the lock held on entry.
    wait_case(false, 1, 0, NULL, 0, false, false);   // pending before the wait
    wait_case(false, 1, 1, NULL, 0, false, false);   // masked, woken normally
    wait_case(false, 0, 0, NULL, 0, true, false);    // signal lands while waiting
    wait_case(false, 1, 1, NULL, 0, false, true);    // the waited-on lock is the sighand lock
    wait_case(false, 0, 0, &normal, 0, false, false);
    wait_case(false, 0, 0, &one_second, ETIMEDOUT, false, false);
    // Any other pthread error is not a timeout, so C reports success.
    wait_case(false, 0, 0, &one_second, EINVAL, false, false);
    wait_case(false, 0, 0, &one_second, EINTR, false, false);

    // The timeout arithmetic: the carry, and the strict `> 1000000000` test.
    script_now = (struct timespec) {100, 700000000};
    struct timespec carry = {0, 500000000};
    wait_case(false, 0, 0, &carry, 0, false, false);
    script_now = (struct timespec) {100, 600000000};
    struct timespec exact = {0, 400000000};
    wait_case(false, 0, 0, &exact, EINVAL, false, false);
    script_now = (struct timespec) {200, 250000000};
    wait_case(false, 0, 0, &normal, 0, false, false);

    // wait_for_ignore_signals: no EINTR checks at all.
    wait_case(true, 1, 0, NULL, 0, false, false);
    wait_case(true, 0, 0, &one_second, ETIMEDOUT, false, false);
    wait_case(true, 0, 0, NULL, 0, false, false);

    no_task_case();
    notify_case();
    unwind_case();
    lock_case();
    wrlock_case();
    return 0;
}
