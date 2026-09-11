// Reference-output generator for the util/timer differential test.
//
// This links the *unmodified* util/timer.c and runs its real worker thread
// against a scripted platform:
//
//   * `clock_gettime` returns a clock the harness owns; it only advances when a
//     scripted `nanosleep` completes, so every value the timer computes is
//     machine-independent;
//   * `nanosleep` parks the worker until the driver releases the sleep (it
//     completes and the clock advances) or pokes it (`SIGUSR1` arrives, so the
//     sleep returns `EINTR` and the timer recomputes its deadline). After a
//     poke the worker stays parked until the driver acknowledges it, so the
//     log order never depends on which thread wins a race;
//   * `free` is wrapped, so the harness can tell whether `timer_free` freed the
//     timer inline or the worker freed it on its way out.
//
// The driver only ever pokes a worker it has observed `in_sleep` for, and only
// observes state while the worker is parked or finished, which is what makes
// the transcript deterministic.
//
// Build (normally via tools/gen_timer_reference.sh):
//   cc -O2 -Wall -Wextra -I<ish-src> -Itools/stub-include -o timer-dump
//      tools/timer-dump.c <ish-src>/util/timer.c -pthread
//      -Wl,--wrap=clock_gettime -Wl,--wrap=nanosleep -Wl,--wrap=free

#define _GNU_SOURCE

#include <errno.h>
#include <pthread.h>
#include <signal.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "util/timer.h"

// ---- scripted platform -----------------------------------------------------

static pthread_mutex_t script_lock = PTHREAD_MUTEX_INITIALIZER;
static struct timespec script_now = {100, 0};
static bool in_sleep;
static bool release_sleep;
static bool interrupted_ack;
static volatile sig_atomic_t signalled;
static int sleep_count;
static int callbacks;

static pthread_t main_thread;
static void *tracked_timer;

static void emit(const char *format, ...) {
    va_list args;
    va_start(args, format);
    vprintf(format, args);
    va_end(args);
    putchar('\n');
    fflush(stdout);
}

static bool in_main_thread(void) {
    return pthread_equal(pthread_self(), main_thread) != 0;
}

int __real_clock_gettime(clockid_t clockid, struct timespec *ts);
int __real_nanosleep(const struct timespec *req, struct timespec *rem);
void __real_free(void *ptr);

int __wrap_clock_gettime(clockid_t clockid, struct timespec *ts) {
    if (clockid != CLOCK_MONOTONIC && clockid != CLOCK_REALTIME)
        return __real_clock_gettime(clockid, ts);
    pthread_mutex_lock(&script_lock);
    *ts = script_now;
    emit("C %ld.%09ld", (long) ts->tv_sec, (long) ts->tv_nsec);
    pthread_mutex_unlock(&script_lock);
    return 0;
}

// Sleep for real time without moving the scripted clock, so the driver can wait
// for a flag without perturbing what the code under test observes.
static void wait_tick(void) {
    const struct timespec tick = {0, 100000};
    pthread_mutex_unlock(&script_lock);
    __real_nanosleep(&tick, NULL);
    pthread_mutex_lock(&script_lock);
}

int __wrap_nanosleep(const struct timespec *req, struct timespec *rem) {
    if (rem != NULL)
        memset(rem, 0, sizeof(*rem));
    pthread_mutex_lock(&script_lock);
    sleep_count++;
    emit("S#%d %ld.%09ld", sleep_count, (long) req->tv_sec, (long) req->tv_nsec);
    in_sleep = true;
    signalled = 0;
    while (!release_sleep && !signalled)
        wait_tick();
    int rc;
    if (signalled) {
        // The driver decides when the interrupted worker resumes, so whatever
        // it wants to print about the poke comes first.
        signalled = 0;
        in_sleep = false;
        while (!interrupted_ack)
            wait_tick();
        interrupted_ack = false;
        errno = EINTR;
        rc = -1;
    } else {
        script_now = timespec_add(script_now, *req);
        release_sleep = false;
        in_sleep = false;
        rc = 0;
    }
    pthread_mutex_unlock(&script_lock);
    return rc;
}

static void sigusr1_handler(int sig) {
    (void) sig;
    signalled = 1;
}

void __wrap_free(void *ptr) {
    // The fast path must not take the lock: libc frees from inside printf while
    // this harness already holds it.
    if (ptr != NULL && ptr == tracked_timer) {
        pthread_mutex_lock(&script_lock);
        emit("F %s", in_main_thread() ? "caller" : "thread");
        tracked_timer = NULL;
        pthread_mutex_unlock(&script_lock);
    }
    __real_free(ptr);
}

// ---- driver helpers --------------------------------------------------------

static void wait_for_sleep(int expected) {
    pthread_mutex_lock(&script_lock);
    while (sleep_count < expected || !in_sleep)
        wait_tick();
    pthread_mutex_unlock(&script_lock);
}

static void release_sleep_now(void) {
    pthread_mutex_lock(&script_lock);
    release_sleep = true;
    pthread_mutex_unlock(&script_lock);
}

static void acknowledge_poke(void) {
    pthread_mutex_lock(&script_lock);
    interrupted_ack = true;
    pthread_mutex_unlock(&script_lock);
}

static void wait_for_callbacks(int expected) {
    pthread_mutex_lock(&script_lock);
    while (callbacks < expected)
        wait_tick();
    pthread_mutex_unlock(&script_lock);
}

// Wait until the worker has left its loop for good.
static void wait_for_worker(struct timer *timer) {
    for (;;) {
        lock(&timer->lock);
        bool running = timer->thread_running;
        unlock(&timer->lock);
        pthread_mutex_lock(&script_lock);
        if (!running) {
            emit("R running=0");
            pthread_mutex_unlock(&script_lock);
            return;
        }
        wait_tick();
        pthread_mutex_unlock(&script_lock);
    }
}

// ---- logging points --------------------------------------------------------

static void log_new(struct timer *timer, const char *label) {
    pthread_mutex_lock(&script_lock);
    emit("N %s now=%ld.%09ld active=%d running=%d dead=%d", label, (long) script_now.tv_sec,
         (long) script_now.tv_nsec, timer->active, timer->thread_running, timer->dead);
    pthread_mutex_unlock(&script_lock);
}

static void log_state(struct timer *timer, const char *label) {
    lock(&timer->lock);
    pthread_mutex_lock(&script_lock);
    emit("X %s start=%ld.%09ld end=%ld.%09ld interval=%ld.%09ld active=%d running=%d dead=%d", label,
         (long) timer->start.tv_sec, (long) timer->start.tv_nsec, (long) timer->end.tv_sec,
         (long) timer->end.tv_nsec, (long) timer->interval.tv_sec, (long) timer->interval.tv_nsec,
         timer->active, timer->thread_running, timer->dead);
    pthread_mutex_unlock(&script_lock);
    unlock(&timer->lock);
}

static void on_fire(void *data) {
    struct timer *timer = data;
    pthread_mutex_lock(&script_lock);
    callbacks++;
    emit("K %d now=%ld.%09ld start=%ld.%09ld end=%ld.%09ld interval=%ld.%09ld active=%d running=%d",
         callbacks, (long) script_now.tv_sec, (long) script_now.tv_nsec, (long) timer->start.tv_sec,
         (long) timer->start.tv_nsec, (long) timer->end.tv_sec, (long) timer->end.tv_nsec,
         (long) timer->interval.tv_sec, (long) timer->interval.tv_nsec, timer->active,
         timer->thread_running);
    pthread_mutex_unlock(&script_lock);
}

// `timer_new` leaves `start`, `end` and `interval` uninitialized, so the first
// set on a fresh timer passes a NULL `oldspec`: reading the old spec before any
// set has happened would read uninitialized memory.
static void do_set_fresh(struct timer *timer, long value_sec, long value_nsec, long interval_sec,
                         long interval_nsec, const char *label) {
    struct timer_spec spec = {{value_sec, value_nsec}, {interval_sec, interval_nsec}};
    int rc = timer_set(timer, spec, NULL);
    pthread_mutex_lock(&script_lock);
    emit("T %s rc=%d old=NULL", label, rc);
    pthread_mutex_unlock(&script_lock);
}

static void do_set(struct timer *timer, long value_sec, long value_nsec, long interval_sec,
                   long interval_nsec, const char *label) {
    struct timer_spec spec = {{value_sec, value_nsec}, {interval_sec, interval_nsec}};
    struct timer_spec old = {{-1, -1}, {-1, -1}};
    int rc = timer_set(timer, spec, &old);
    pthread_mutex_lock(&script_lock);
    emit("T %s rc=%d old_value=%ld.%09ld old_interval=%ld.%09ld", label, rc, (long) old.value.tv_sec,
         (long) old.value.tv_nsec, (long) old.interval.tv_sec, (long) old.interval.tv_nsec);
    pthread_mutex_unlock(&script_lock);
}

static void log_summary(void) {
    pthread_mutex_lock(&script_lock);
    emit("Z callbacks=%d sleeps=%d clock=%ld.%09ld", callbacks, sleep_count,
         (long) script_now.tv_sec, (long) script_now.tv_nsec);
    pthread_mutex_unlock(&script_lock);
}

int main(void) {
    setvbuf(stdout, NULL, _IONBF, 0);
    main_thread = pthread_self();

    struct sigaction action = {0};
    action.sa_handler = sigusr1_handler;
    sigemptyset(&action.sa_mask);
    action.sa_flags = 0; // no SA_RESTART: a pending SIGUSR1 interrupts nanosleep
    if (sigaction(SIGUSR1, &action, NULL) != 0) {
        perror("sigaction");
        return 1;
    }

    printf("# timer reference, generated from unmodified iSH util/timer.{h,c}\n");

    // Phase 1: a fresh timer, then a zero-value set that starts no thread.
    struct timer *timer = timer_new(CLOCK_MONOTONIC, on_fire, NULL);
    timer->data = timer; // the callback reads its own timer, as the itimer does
    tracked_timer = timer;
    log_new(timer, "fresh");

    do_set_fresh(timer, 0, 0, 0, 0, "zero");
    log_state(timer, "zero");

    // Phase 2: arm it. The worker parks in the first sleep.
    do_set(timer, 0, 500000000, 0, 250000000, "arm");
    wait_for_sleep(1);
    log_state(timer, "arm");
    release_sleep_now();
    wait_for_callbacks(1);
    wait_for_sleep(2);
    log_state(timer, "fired1");

    // Phase 3: the worker is sleeping the interval out. Rescheduling pokes it,
    // so the sleep returns EINTR and it sleeps for the new value instead.
    do_set(timer, 2, 0, 0, 0, "requeue");
    acknowledge_poke();
    wait_for_sleep(3);
    log_state(timer, "requeue");
    release_sleep_now();
    wait_for_callbacks(2);
    wait_for_worker(timer);
    log_state(timer, "done");

    // The worker is gone, so this free happens inline.
    timer_free(timer);

    // Phase 4: deactivate a running timer without freeing it.
    struct timer *second = timer_new(CLOCK_MONOTONIC, on_fire, NULL);
    second->data = second;
    tracked_timer = second;
    log_new(second, "second");
    do_set_fresh(second, 0, 500000000, 0, 0, "arm2");
    wait_for_sleep(4);
    log_state(second, "arm2");
    do_set(second, 0, 0, 0, 0, "stop");
    acknowledge_poke();
    wait_for_worker(second);
    log_state(second, "stop");
    timer_free(second);

    // Phase 5: free a timer while its worker is mid-sleep. The worker sees the
    // cleared `active` flag, leaves the loop, and frees the timer itself.
    struct timer *third = timer_new(CLOCK_MONOTONIC, on_fire, NULL);
    third->data = third;
    tracked_timer = third;
    log_new(third, "third");
    do_set_fresh(third, 0, 750000000, 0, 0, "arm3");
    wait_for_sleep(5);
    log_state(third, "arm3");
    pthread_mutex_lock(&script_lock);
    emit("D free_while_sleeping in_sleep=%d", in_sleep ? 1 : 0);
    pthread_mutex_unlock(&script_lock);
    timer_free(third);
    acknowledge_poke();

    pthread_mutex_lock(&script_lock);
    while (tracked_timer != NULL)
        wait_tick();
    pthread_mutex_unlock(&script_lock);

    // Phase 6: a nanosecond count above one second. `timespec_add` carries once
    // and never re-checks, so `end` stays unnormalized; the worker's remaining
    // time is then computed against it, and the callback fires when that
    // unnormalized deadline is reached.
    struct timer *fourth = timer_new(CLOCK_MONOTONIC, on_fire, NULL);
    fourth->data = fourth;
    tracked_timer = fourth;
    log_new(fourth, "fourth");
    do_set_fresh(fourth, 0, 1500000000, 0, 0, "carry");
    wait_for_sleep(6);
    log_state(fourth, "carry");
    release_sleep_now();
    wait_for_callbacks(3);
    wait_for_worker(fourth);
    log_state(fourth, "carry_done");
    timer_free(fourth);

    log_summary();
    return 0;
}
