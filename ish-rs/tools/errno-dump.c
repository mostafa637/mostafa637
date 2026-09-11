// Reference-output generator for the errno differential test.
//
// Links the *unmodified* kernel/errno.c and prints err_map() for every input in
// a range that covers both the mapped errnos and the unknown-value fallback,
// then drives errno_map() by setting the real `errno` and reporting whether the
// C decided to raise SIGPIPE.
//
//   cc -O2 -I<ish-src> -o errno-dump errno-dump.c <ish-src>/kernel/errno.c
//
// The host errno numbers in this output are whatever this platform's <errno.h>
// says, which is the point: tools/gen_errno_table.py asks the host for the same
// numbers, and this fixture checks that the table it built behaves like the C.

#include <errno.h>
#include <stdio.h>
#include <string.h>

#include "kernel/errno.h"
#include "kernel/signal.h"
#include "kernel/task.h"

// ---- stubs ----

// kernel/log.c. The fallback path in err_map() printk's the unknown value; the
// count of those calls is printed at the end so the port can be checked against
// it rather than against a log nobody reads.
static long unknown_errors;
void ish_printk(const char *msg, ...) {
    if (strstr(msg, "unknown error") != NULL)
        unknown_errors++;
}

// kernel/signal.c. errno_map() sends SIGPIPE on EPIPE; a counter records that it
// would have, and the corpus reports it per call.
static long sigpipes_sent;
void send_signal(struct task *task, int sig, struct siginfo_ info) {
    (void) task;
    (void) info;
    if (sig == SIGPIPE_)
        sigpipes_sent++;
}

__thread struct task *current;

int main(void) {
    printf("# errno reference output, generated from unmodified iSH kernel/errno.c\n");
    printf("# host EPERM %d ENOENT %d EAGAIN %d EPIPE %d EOPNOTSUPP %d EDQUOT %d\n",
           EPERM, ENOENT, EAGAIN, EPIPE, EOPNOTSUPP, EDQUOT);
    printf("# host EAGAIN==EWOULDBLOCK %d EOPNOTSUPP==ENOTSUP %d\n",
           EAGAIN == EWOULDBLOCK, EOPNOTSUPP == ENOTSUP);

    // every input the corpus cares about: the negatives err_map() can be handed
    // by a mistake, the whole mapped range, and enough past it to exercise the
    // -(err | 0x1000) fallback
    for (int e = -16; e < 4200; e++)
        printf("M %d %d\n", e, err_map(e));

    // errno_map() reads the real errno and may raise SIGPIPE on the way out
    static const int probes[] = {
        0, EPERM, ENOENT, EAGAIN, EPIPE, EOPNOTSUPP, EDQUOT, EINTR, ECHILD, 4095,
    };
    for (unsigned i = 0; i < sizeof probes / sizeof probes[0]; i++) {
        errno = probes[i];
        long before = sigpipes_sent;
        int mapped = errno_map();
        printf("N %d %d %d\n", probes[i], mapped, sigpipes_sent > before);
    }

    printf("# unknown_errors %ld sigpipes_sent %ld\n", unknown_errors, sigpipes_sent);
    return 0;
}
