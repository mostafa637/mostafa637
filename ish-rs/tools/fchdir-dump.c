// Reference-output generator for the util/fchdir differential test.
//
// This *includes* the unmodified util/fchdir.c, so the file's own `static
// lock_t fchdir_lock` is in scope and the harness can probe it exactly the way
// `lock_fchdir` itself does. `fchdir` is wrapped at link time, so the harness
// sees every host call with the dirfd the caller passed — and sees it *while
// the caller still holds the lock*, which is the property the file exists for.
//
// The corpus needs two threads, and its transcript must not depend on how the
// scheduler interleaves them: the workers never print. They only set flags and
// record what the wrapped `fchdir` saw, and the driver prints a record only
// after the flag that makes its value deterministic. The one deliberately
// time-dependent record is the mutual-exclusion probe below, where a bounded
// sleep gives a *broken* lock every chance to let the second worker in.
//
// Build (normally via tools/gen_fchdir_reference.sh):
//   cc -O2 -Wall -Wextra -I<ish-src> -o fchdir-dump tools/fchdir-dump.c
//      -pthread -Wl,--wrap=fchdir

#define _GNU_SOURCE

#include <fcntl.h>
#include <pthread.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <time.h>
#include <unistd.h>

#include "util/fchdir.c"

// ---- the scripted platform -------------------------------------------------

static pthread_mutex_t script_lock = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t script_cond = PTHREAD_COND_INITIALIZER;

// Which worker is inside the wrapped `fchdir`, so it knows which flag to set
// and which release to wait for. C's `pthread_self()` would do as well; the
// label keeps the transcript names and the flags in one place.
static __thread const char *who = "main";

struct worker {
    bool started;    // about to call lock_fchdir
    bool in_fchdir;  // inside the host call, holding the lock
    bool released;   // the driver let the host call finish
    bool unlocked;   // unlock_fchdir returned
    int fd;          // the dirfd this worker passed
    int rc;          // what the host's fchdir returned for it
};
static struct worker w1;
static struct worker w2;

// Every wrapped call, in order, plus what the last one was handed.
static int calls;
static int last_fd;
static int last_rc;

static void script_set(bool *flag) {
    pthread_mutex_lock(&script_lock);
    *flag = true;
    pthread_cond_broadcast(&script_cond);
    pthread_mutex_unlock(&script_lock);
}

static void script_wait(const bool *flag) {
    pthread_mutex_lock(&script_lock);
    while (!*flag)
        pthread_cond_wait(&script_cond, &script_lock);
    pthread_mutex_unlock(&script_lock);
}

static int script_calls(void) {
    pthread_mutex_lock(&script_lock);
    int count = calls;
    pthread_mutex_unlock(&script_lock);
    return count;
}

int __real_fchdir(int dirfd);

int __wrap_fchdir(int dirfd) {
    pthread_mutex_lock(&script_lock);
    calls++;
    last_fd = dirfd;
    pthread_mutex_unlock(&script_lock);

    struct worker *me = NULL;
    if (strcmp(who, "w1") == 0)
        me = &w1;
    else if (strcmp(who, "w2") == 0)
        me = &w2;

    if (me != NULL) {
        pthread_mutex_lock(&script_lock);
        me->fd = dirfd;
        pthread_mutex_unlock(&script_lock);
        // The work a C call site does with relative paths while holding the
        // lock (`mkfifo`): here, a pause the driver ends.
        script_set(&me->in_fchdir);
        script_wait(&me->released);
    }

    int rc = __real_fchdir(dirfd);

    pthread_mutex_lock(&script_lock);
    last_rc = rc;
    if (me != NULL)
        me->rc = rc;
    pthread_mutex_unlock(&script_lock);
    return rc;
}

// `trylock(&fchdir_lock)` from the driver: 1 when the lock is held. A
// successful probe takes the lock, so it is released again; like C's `trylock`,
// the probe never records an owner.
static int held(void) {
    if (trylock(&fchdir_lock) == 0) {
        unlock(&fchdir_lock);
        return 0;
    }
    return 1;
}

// `pthread_equal(fchdir_lock.owner, pthread_self())`: the field `lock` sets and
// `unlock` clears.
static int owner_is_self(void) {
    return pthread_equal(fchdir_lock.owner, pthread_self()) != 0;
}

// ---- the two contending workers --------------------------------------------

static int fd_a, fd_b;

static void *worker1(void *unused) {
    (void) unused;
    who = "w1";
    script_set(&w1.started);
    lock_fchdir(fd_a);
    unlock_fchdir();
    script_set(&w1.unlocked);
    return NULL;
}

static void *worker2(void *unused) {
    (void) unused;
    who = "w2";
    script_set(&w2.started);
    lock_fchdir(fd_b);
    unlock_fchdir();
    script_set(&w2.unlocked);
    return NULL;
}

// ---- the corpus ------------------------------------------------------------

// Make `*fd` refer to the same open directory as descriptor `number`.
static int pin_fd(int *fd, int number) {
    if (*fd == number)
        return 0;
    if (dup2(*fd, number) < 0)
        return -1;
    close(*fd);
    *fd = number;
    return 0;
}

static void settle(void) {
    // A tenth of a second: long enough that a lock that is not really held
    // cannot hide, short enough to be irrelevant to a correct run.
    const struct timespec pause = {0, 100 * 1000 * 1000};
    nanosleep(&pause, NULL);
}

int main(void) {
    setvbuf(stdout, NULL, _IONBF, 0);
    int rc, owner;

    // Two real directories for the two workers to change into. They are opened
    // before anything else, so the harness can pin their numbers in the
    // transcript and the Rust replay can use the same ones.
    char template[] = "/tmp/fchdir-oracle-XXXXXX";
    char *root = mkdtemp(template);
    if (root == NULL) {
        perror("mkdtemp");
        return 1;
    }
    char a[256], b[256];
    snprintf(a, sizeof(a), "%s/a", root);
    snprintf(b, sizeof(b), "%s/b", root);
    if (mkdir(a, 0700) < 0 || mkdir(b, 0700) < 0) {
        perror("mkdir");
        return 1;
    }
    fd_a = open(a, O_RDONLY | O_DIRECTORY);
    fd_b = open(b, O_RDONLY | O_DIRECTORY);
    // The transcript pins the dirfds, and the Rust replay passes those numbers
    // to its scripted host, so the harness forces them: a freshly started
    // process may already have low descriptors open for its own reasons, and
    // which ones they are is not this corpus's business.
    if (pin_fd(&fd_a, 3) < 0 || pin_fd(&fd_b, 4) < 0) {
        fprintf(stderr, "could not pin the directory fds to 3 and 4\n");
        return 1;
    }

    printf("# fchdir reference, generated from unmodified util/fchdir.c\n");
    printf("G dirfds %d %d\n", fd_a, fd_b);

    // 1. One thread at a time: the lock is free, taken around the host call,
    //    and free again after `unlock_fchdir`.
    printf("T probe-before held=%d\n", held());
    lock_fchdir(fd_a);
    rc = last_rc;
    owner = owner_is_self();
    printf("T self fd=%d rc=%d owner=%d\n", last_fd, rc, owner);
    unlock_fchdir();
    printf("T self released held=%d\n", held());
    // A host call that fails is still made, still under the lock, and its result
    // goes nowhere: `lock_fchdir` returns void.
    lock_fchdir(-1);
    rc = last_rc;
    owner = owner_is_self();
    printf("T self fd=%d rc=%d owner=%d\n", last_fd, rc, owner);
    unlock_fchdir();
    printf("T self released held=%d\n", held());

    // 2. Two threads: w1 holds the lock while its host call is in flight, w2 is
    //    already knocking, and it cannot get in until w1 unlocks.
    pthread_t t1, t2;
    pthread_create(&t1, NULL, worker1, NULL);
    script_wait(&w1.started);
    pthread_create(&t2, NULL, worker2, NULL);
    script_wait(&w2.started);
    script_wait(&w1.in_fchdir);
    int w1_fd = w1.fd, w1_probe = held();
    printf("T w1 in fd=%d held=%d\n", w1_fd, w1_probe);
    settle();
    int blocked_calls = script_calls();
    printf("T w2 blocked calls=%d\n", blocked_calls);
    script_set(&w1.released);
    script_wait(&w1.unlocked);
    int w1_rc = w1.rc;
    printf("T w1 done rc=%d\n", w1_rc);
    script_wait(&w2.in_fchdir);
    int w2_fd = w2.fd, w2_probe = held();
    printf("T w2 in fd=%d held=%d\n", w2_fd, w2_probe);
    script_set(&w2.released);
    script_wait(&w2.unlocked);
    int w2_rc = w2.rc;
    printf("T w2 done rc=%d\n", w2_rc);
    pthread_join(t1, NULL);
    pthread_join(t2, NULL);

    printf("T end held=%d\n", held());
    int total = script_calls();
    printf("T calls %d\n", total);

    rmdir(a);
    rmdir(b);
    rmdir(root);
    return 0;
}
