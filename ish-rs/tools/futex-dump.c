// Reference-output generator for the kernel/futex differential test.
//
// This *includes* the unmodified kernel/futex.c, which is what makes the hard
// part testable: the queue algebra (wake counts, requeue moves, reference
// transfers, entry lifetimes) can be driven directly on `struct futex_wait`
// objects the harness builds itself, with no threads and no races.
//
// Scripted at link time:
//   * `pthread_cond_wait` / `pthread_cond_timedwait` decide what the wait
//     returns (`0`, `ETIMEDOUT`, or another error) and whether a signal lands
//     while the task sleeps;
//   * `pthread_cond_broadcast` / `pthread_cond_signal` record which condvar
//     `notify` / `notify_once` reached, so a wake's *target* is visible;
//   * `clock_gettime(CLOCK_MONOTONIC)` fixes the timeout origin.
//
// `current` is a real task over a real `struct mem` (kernel/memory.c is linked
// in), so `futex_load`'s fault handling is the real thing: page 0x100 is mapped
// and holds a dword the corpus rewrites, and page 0x200 is left unmapped.
//
// Build (normally via tools/gen_futex_reference.sh):
//   cc -O2 -Wall -Wextra -I<ish-src> -Itools/stub-include -o futex-dump
//      tools/futex-dump.c <ish-src>/kernel/memory.c <ish-src>/kernel/errno.c
//      <ish-src>/kernel/user.c -pthread
//      -Wl,--wrap=pthread_cond_wait -Wl,--wrap=pthread_cond_timedwait
//      -Wl,--wrap=pthread_cond_broadcast -Wl,--wrap=pthread_cond_signal
//      -Wl,--wrap=clock_gettime

#define _GNU_SOURCE

#include <errno.h>
#include <pthread.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <time.h>

#include "kernel/calls.h"
#include "kernel/memory.h"
#include "kernel/signal.h"

// ---- stubs for the parts of iSH this oracle does not include --------------

// kernel/task.h's thread-local task pointer.
__thread struct task *current;

// asbestos: futex.c never touches it, memory.c only counts invalidations.
static long asbestos_invalidations;
void *asbestos_new(struct mmu *mmu) {
    (void) mmu;
    return (void *) 1;
}
void asbestos_free(void *asbestos) { (void) asbestos; }
void asbestos_invalidate_page(void *asbestos, page_t page) {
    (void) asbestos;
    (void) page;
    asbestos_invalidations++;
}

// kernel/vdso.c: a NULL vdso makes every mapping look mmap'd, which is what the
// corpus wants.
const char *vdso_data;

// kernel/log.c
void ish_printk(const char *msg, ...) { (void) msg; }
void warn(const char *msg, ...) { (void) msg; }
void die(const char *msg, ...) {
    fprintf(stderr, "die: %s\n", msg);
    exit(1);
}
int current_pid(void) { return current == NULL ? 0 : current->pid; }

// kernel/errno.c watches for SIGPIPE; nothing here writes to a pipe.
static long signals_sent;
void send_signal(struct task *task, int sig, struct siginfo_ info) {
    (void) task;
    (void) sig;
    (void) info;
    signals_sent++;
}

// fs/fd.c: no mapping in this corpus carries an fd.
static long fd_closes;
int fd_close(struct fd *fd) {
    (void) fd;
    fd_closes++;
    return 0;
}

// kernel/task.c: the robust-list syscalls are the only callers here.
struct task *pid_get_task(dword_t pid) {
    return (pid_t_) pid == current->pid ? current : NULL;
}

// util/sync.c is linked for real, so only the task lock it expects is missing.
lock_t pids_lock = LOCK_INITIALIZER;

#include "kernel/futex.c"

// ---- the scripted platform -------------------------------------------------

static cond_t *known_conds[16];
static const char *known_names[16];
static int known_count;

struct table_entry {
    addr_t addr;
    unsigned refs;
    int queue;
};
static long notified[16];
static long broadcast_count;
static long signal_count;

// The corpus reuses `struct probe_waiter` storage, so the same cond address can
// be registered again under a new waiter id. Its address *is* its identity
// (C's `cond_t *`), so re-registering updates the existing slot's name.
static void register_cond(cond_t *cond, long id) {
    char *name = malloc(16);
    snprintf(name, 16, "w%ld", id);
    for (int i = 0; i < known_count; i++) {
        if (known_conds[i] == cond) {
            known_names[i] = name;
            return;
        }
    }
    known_conds[known_count] = cond;
    known_names[known_count] = name;
    known_count++;
}

static const char *cond_name(cond_t *cond) {
    for (int i = 0; i < known_count; i++)
        if (known_conds[i] == cond)
            return known_names[i];
    return "unknown";
}

int __wrap_pthread_cond_broadcast(pthread_cond_t *cond) {
    broadcast_count++;
    notified[known_count > 0 ? 0 : 0] = notified[0]; // keep -Wunused quiet
    printf("N broadcast=%s\n", cond_name((cond_t *) cond));
    return 0;
}

int __wrap_pthread_cond_signal(pthread_cond_t *cond) {
    signal_count++;
    printf("N signal=%s\n", cond_name((cond_t *) cond));
    return 0;
}

static struct timespec script_now = {100, 0};
static int script_park_rc;
static bool script_pending_after;

int __wrap_pthread_cond_wait(pthread_cond_t *cond, pthread_mutex_t *lock) {
    (void) cond;
    (void) lock;
    printf("P wait - rc=%d pending_after=%d\n", script_park_rc, script_pending_after ? 1 : 0);
    if (script_pending_after)
        current->pending = 1;
    return script_park_rc;
}

int __wrap_pthread_cond_timedwait(pthread_cond_t *cond, pthread_mutex_t *lock,
                                  const struct timespec *deadline) {
    (void) cond;
    (void) lock;
    printf("P timedwait %ld.%09ld rc=%d pending_after=%d\n", (long) deadline->tv_sec,
           (long) deadline->tv_nsec, script_park_rc, script_pending_after ? 1 : 0);
    if (script_pending_after)
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

// ---- the corpus's guest state ---------------------------------------------

static struct task task;
static struct mem task_mem;
static struct sighand sighand;
static char *page_a; // 0x100, holds the dword every futex address points at
static long waiter_ids;

static addr_t dword_at(page_t page) {
    return page << 12;
}

static void set_guest_dword(addr_t addr, dword_t value) {
    struct pt_entry *pt = mem_pt(&task_mem, addr >> 12);
    if (pt == NULL) {
        fprintf(stderr, "the corpus expected %#x to be mapped\n", addr);
        exit(1);
    }
    memcpy((char *) pt->data->data + pt->offset + (addr & 0xfff), &value, sizeof(value));
    (void) page_a;
}

static dword_t get_guest_dword(addr_t addr) {
    dword_t value = 0;
    struct pt_entry *pt = mem_pt(&task_mem, addr >> 12);
    if (pt != NULL)
        memcpy(&value, (char *) pt->data->data + pt->offset + (addr & 0xfff), sizeof(value));
    return value;
}

// A waiter the harness owns, built exactly the way futex_wait builds one.
struct probe_waiter {
    struct futex_wait wait;
    long id;
};

// `futex_get` hands back the table lock still held; the corpus instead builds
// its waiters the way the inside of a locked section does, with the unlocked
// pair, so a whole queue can be assembled without deadlocking on itself.
static void waiter_init(struct probe_waiter *probe, addr_t addr) {
    probe->id = ++waiter_ids;
    cond_init(&probe->wait.cond);
    probe->wait.futex = futex_get_unlocked(addr);
    list_init(&probe->wait.queue);
    register_cond(&probe->wait.cond, probe->id);
}

static void waiter_enqueue(struct probe_waiter *probe) {
    list_add_tail(&probe->wait.futex->queue, &probe->wait.queue);
}

static void waiter_dequeue(struct probe_waiter *probe) {
    list_remove_safe(&probe->wait.queue);
}

static void corpus_build(struct probe_waiter *probe, addr_t addr) {
    lock(&futex_lock);
    waiter_init(probe, addr);
    waiter_enqueue(probe);
    unlock(&futex_lock);
}

static void corpus_release(struct probe_waiter *probe) {
    lock(&futex_lock);
    waiter_dequeue(probe);
    futex_put_unlocked(probe->wait.futex);
    unlock(&futex_lock);
}

// ---- dumps -----------------------------------------------------------------

static void dump_waiter(struct probe_waiter *probe, const char *tag) {
    bool queued = probe->wait.queue.next != NULL && !list_empty(&probe->wait.queue);
    printf("Q %s id=%ld futex=%#x queued=%d\n", tag, probe->id, probe->wait.futex->addr,
           queued ? 1 : 0);
}

static int entry_cmp(const void *a, const void *b) {
    const struct table_entry *x = a, *y = b;
    return x->addr < y->addr ? -1 : x->addr > y->addr;
}

// The dump is ordered by address, not by bucket: which bucket an address lands
// in depends on the low bits of the host address of `struct mem`, so the bucket
// index is a build artefact and is deliberately not part of the reference.
static void dump_table(const char *tag) {
    struct table_entry entries[FUTEX_HASH_SIZE];
    int count = 0;
    for (int i = 0; i < FUTEX_HASH_SIZE; i++) {
        struct futex *futex;
        list_for_each_entry(&futex_hash[i], futex, chain) {
            int queued = 0;
            struct futex_wait *wait;
            list_for_each_entry(&futex->queue, wait, queue) queued++;
            entries[count++] = (struct table_entry) {futex->addr, futex->refcount, queued};
        }
    }
    qsort(entries, count, sizeof(entries[0]), entry_cmp);
    for (int i = 0; i < count; i++)
        printf("H %s addr=%#x refs=%u queue=%d\n", tag, entries[i].addr, entries[i].refs,
               entries[i].queue);
    printf("H %s entries=%d\n", tag, count);
}

// ---- corpus steps ----------------------------------------------------------

/// The locked wrappers: building a waiter, releasing one, and dumping the table
/// all need the table lock, because `futex_get_unlocked`/`futex_put_unlocked`
/// are the "caller already holds it" pair.
static void corpus_dump_table(const char *tag) {
    lock(&futex_lock);
    dump_table(tag);
    unlock(&futex_lock);
}

static void corpus_dump_waiter(struct probe_waiter *probe, const char *tag) {
    lock(&futex_lock);
    dump_waiter(probe, tag);
    unlock(&futex_lock);
}

static void reset_futexes(void) {
    for (int i = 0; i < FUTEX_HASH_SIZE; i++)
        list_init(&futex_hash[i]);
    waiter_ids = 0;
    known_count = 0;
    broadcast_count = 0;
    signal_count = 0;
    memset(notified, 0, sizeof(notified));
}

// A full `sys_futex(FUTEX_WAIT)` with the harness scripting the park.
static void step_wait(const char *label, addr_t addr, dword_t val, int timeout_mode, int park_rc,
                      bool pending_after, dword_t initial_pending, dword_t initial_blocked) {
    // timeout_mode: 0 = no timeout pointer, 1 = a mapped guest timespec,
    // 2 = a pointer into an unmapped page.

    task.pending = initial_pending;
    task.blocked = initial_blocked;
    script_park_rc = park_rc;
    script_pending_after = pending_after;
    struct timespec_ timeout_ = {1, 500000000};
    addr_t timeout_addr = 0;
    if (timeout_mode == 1) {
        // place the guest timespec on the mapped page, after the futex dword
        timeout_addr = dword_at(0x100) + 64;
        memcpy((char *) mem_pt(&task_mem, 0x100)->data->data + 64, &timeout_, sizeof(timeout_));
    } else if (timeout_mode == 2) {
        timeout_addr = dword_at(0x200) + 8; // unmapped
    }
    printf("W %s addr=%#x val=%d timeout=%d park=%d pending_before=%u blocked_before=%u\n", label,
           addr, val, timeout_mode, park_rc, initial_pending, initial_blocked);
    int rc = sys_futex(addr, FUTEX_WAIT_, val, timeout_addr, 0, 0);
    printf("W %s rc=%d pending_after=%ld\n", label, rc, (long) task.pending);
    task.pending = 0;
    task.blocked = 0;
}

int main(void) {
    setvbuf(stdout, NULL, _IONBF, 0);

    memset(&task, 0, sizeof(task));
    memset(&sighand, 0, sizeof(sighand));
    task.pid = 47;
    lock_init(&sighand.lock);
    task.sighand = &sighand;
    current = &task;
    mem_init(&task_mem);
    task.mem = &task_mem;
    pt_map_nothing(&task_mem, 0x100, 1, P_RWX);
    page_a = NULL;

    init_futex_hash();
    printf("# futex reference, generated from unmodified iSH kernel/futex.c\n");

    // The dword every futex address points at, with page 0x200 left unmapped.
    set_guest_dword(dword_at(0x100), 7);
    printf("G dword=%u mapped_pages=%d\n", get_guest_dword(dword_at(0x100)), task_mem.pgdir_used);

    // 1. futex_load's two failure modes and the successful wait.
    step_wait("unmapped", dword_at(0x200), 7, 0, 0, false, 0, 0);
    step_wait("mismatch", dword_at(0x100), 8, 0, 0, false, 0, 0);
    step_wait("woken", dword_at(0x100), 7, 0, 0, false, 0, 0);
    step_wait("timeout", dword_at(0x100), 7, 1, ETIMEDOUT, false, 0, 0);
    // A non-timeout pthread failure is *not* a timeout: `wait_for` reports 0.
    step_wait("timeout-error", dword_at(0x100), 7, 1, EINVAL, false, 0, 0);
    step_wait("pending-before", dword_at(0x100), 7, 0, 0, false, 0x10, 0);
    // The same signal, masked: nothing is pending, so the wait runs.
    step_wait("pending-masked", dword_at(0x100), 7, 0, 0, false, 0x10, 0x10);
    step_wait("pending-during", dword_at(0x100), 7, 1, 0, true, 0, 0);
    // A timeout pointer into an unmapped page is EFAULT, before any waiting.
    step_wait("timeout-fault", dword_at(0x100), 7, 2, 0, false, 0, 0);
    corpus_dump_table("after-waits");

    // 2. The wait publishes its waiter in the queue and removes it afterwards:
    //    the table must be empty again.
    reset_futexes();

    // 3. Wake counting over a hand-built queue.
    struct probe_waiter waiters[4];

    // 3. Wake counting over a hand-built queue: three waiters on A, one on B.
    for (int i = 0; i < 3; i++) corpus_build(&waiters[i], dword_at(0x100));
    corpus_build(&waiters[3], dword_at(0x140));
    corpus_dump_table("three-one");
    // A zero-length wake wakes nobody.
    printf("K wake0 rc=%d\n", futex_wake(dword_at(0x100), 0));
    printf("K wake2 rc=%d\n", futex_wake(dword_at(0x100), 2));
    corpus_dump_table("after-wake2");
    for (int i = 0; i < 4; i++) corpus_dump_waiter(&waiters[i], "after-wake2");
    // The two woken waiters are no longer queued; each releases its own
    // reference as it returns, which is the `futex_put` at the end of
    // `futex_wait`.
    for (int i = 0; i < 2; i++) corpus_release(&waiters[i]);
    corpus_dump_table("after-wake2-returns");
    printf("K wake5 rc=%d\n", futex_wake(dword_at(0x100), 5));
    corpus_release(&waiters[2]);
    corpus_dump_table("after-wake5");

    // 4. Requeue: one woken, two moved. Each moved waiter takes its reference
    //    with it, and the source keeps the one `futex_get` took.
    for (int i = 0; i < 3; i++) corpus_build(&waiters[i], dword_at(0x100));
    corpus_dump_table("requeue-before");
    printf("K requeue rc=%d\n", futex_wakelike(FUTEX_REQUEUE_, dword_at(0x100), 1, 2, dword_at(0x140)));
    corpus_dump_table("requeue-after");
    for (int i = 0; i < 4; i++) corpus_dump_waiter(&waiters[i], "requeue-after");

    // 5. Draining: A's queue is empty, B's holds three, and releasing every
    //    reference takes both entries out of the table.
    printf("K wake-rest rc=%d\n", futex_wake(dword_at(0x100), 10));
    printf("K wake-target rc=%d\n", futex_wake(dword_at(0x140), 10));
    for (int i = 0; i < 4; i++) corpus_release(&waiters[i]);
    corpus_dump_table("drained");

    // 6. The syscall-level dispatch: the private flag, an unsupported operation,
    //    and requeue through sys_futex.
    reset_futexes();
    printf("K private-wake rc=%d\n", sys_futex(dword_at(0x100), FUTEX_WAKE_ | FUTEX_PRIVATE_FLAG_, 1, 0, 0, 0));
    printf("K unsupported rc=%d\n", sys_futex(dword_at(0x100), 9, 1, 0, 0, 0));
    corpus_build(&waiters[0], dword_at(0x100));
    corpus_build(&waiters[1], dword_at(0x100));
    printf("K sys-requeue rc=%d\n", sys_futex(dword_at(0x100), FUTEX_REQUEUE_, 0, 2, dword_at(0x140), 0));
    corpus_dump_table("after-sys-requeue");
    for (int i = 0; i < 2; i++) corpus_dump_waiter(&waiters[i], "after-sys-requeue");
    printf("K wake-target2 rc=%d\n", futex_wake(dword_at(0x140), 10));
    for (int i = 0; i < 2; i++) corpus_release(&waiters[i]);
    corpus_dump_table("end");

    // 7. The robust-list calls.
    int rc = sys_set_robust_list(0x1000, 8);
    printf("L set-len8 rc=%d robust=%#x\n", rc, task.robust_list);
    rc = sys_set_robust_list(0x1000, 12);
    printf("L set-len12 rc=%d robust=%#x\n", rc, task.robust_list);
    addr_t list_out = dword_at(0x100) + 128;
    addr_t len_out = dword_at(0x100) + 132;
    rc = sys_get_robust_list(47, list_out, len_out);
    printf("L get-self rc=%d list=%#x len=%u\n", rc, get_guest_dword(list_out),
           get_guest_dword(len_out));
    printf("L get-other rc=%d\n", sys_get_robust_list(48, list_out, len_out));
    // `sys_set_robust_list` never touches its address argument, so a NULL one
    // with the right length succeeds.
    rc = sys_set_robust_list(0, 12);
    printf("L set-null rc=%d robust=%#x\n", rc, task.robust_list);

    printf("# asbestos_invalidations %ld fd_closes %ld signals_sent %ld broadcasts %ld signals %ld\n",
           asbestos_invalidations, fd_closes, signals_sent, broadcast_count, signal_count);
    return 0;
}
