#!/bin/sh
# Regenerate tests/fixtures/futex_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_futex_reference.sh
#
# The oracle *includes* kernel/futex.c, so it can drive the queue algebra
# directly (wake counts, requeue moves, reference transfers, entry lifetimes)
# without threads. kernel/memory.c is linked in for real, so `futex_load` reads
# a genuine page table; kernel/user.c is linked for the timeout and robust-list
# copies; util/sync.c is linked for the waits, with pthread_cond_* and
# clock_gettime wrapped so every wait's outcome is scripted.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/kernel/futex.c" "$ISH_SRC/kernel/memory.c" \
    "$ISH_SRC/kernel/user.c" "$ISH_SRC/util/sync.c"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include -o /tmp/futex-dump \
    tools/futex-dump.c "$ISH_SRC/kernel/memory.c" "$ISH_SRC/kernel/errno.c" \
    "$ISH_SRC/kernel/user.c" "$ISH_SRC/util/sync.c" -pthread \
    -Wl,--wrap=pthread_cond_wait -Wl,--wrap=pthread_cond_timedwait \
    -Wl,--wrap=pthread_cond_broadcast -Wl,--wrap=pthread_cond_signal \
    -Wl,--wrap=clock_gettime

tmp=/tmp/futex_reference.txt.new
/tmp/futex-dump >"$tmp"

count() {
    grep -c "^$1" "$tmp"
}

waits=$(count 'W ')
parks=$(count 'P ')
wakes=$(count 'K ')
tables=$(count 'H ')
queues=$(count 'Q ')
notifies=$(count 'N ')
robust=$(count 'L ')

if [ "$waits" -ne 18 ] || [ "$parks" -ne 5 ] || [ "$wakes" -ne 10 ]; then
    echo "unexpected corpus shape: $waits wait records, $parks parks, $wakes wakes" >&2
    exit 1
fi
if [ "$tables" -lt 12 ] || [ "$queues" -ne 10 ] || [ "$notifies" -ne 9 ] || [ "$robust" -ne 5 ]; then
    echo "unexpected record counts: $tables table, $queues queue, $notifies notify, $robust robust" >&2
    exit 1
fi

# futex_load's two failures and the successful wait.
grep -q '^W unmapped rc=-14' "$tmp" || { echo "missing the unmapped EFAULT" >&2; exit 1; }
grep -q '^W mismatch rc=-11' "$tmp" || { echo "missing the value mismatch EAGAIN" >&2; exit 1; }
grep -q '^W woken rc=0' "$tmp" || { echo "missing the successful wait" >&2; exit 1; }
# The wait contract the futex inherits from util/sync.c.
grep -q '^W timeout rc=-110' "$tmp" || { echo "missing the timeout" >&2; exit 1; }
grep -q '^W timeout-error rc=0' "$tmp" || { echo "missing the non-timeout failure" >&2; exit 1; }
grep -q '^W pending-before rc=-4' "$tmp" || { echo "missing the pre-wait EINTR" >&2; exit 1; }
grep -q '^W pending-masked rc=0' "$tmp" || { echo "missing the masked-signal wait" >&2; exit 1; }
grep -q '^W pending-during rc=-4' "$tmp" || { echo "missing the post-wait EINTR" >&2; exit 1; }
grep -q '^W timeout-fault rc=-14' "$tmp" || { echo "missing the faulting timeout EFAULT" >&2; exit 1; }
# A wait leaves nothing behind.
grep -q '^H after-waits entries=0' "$tmp" || { echo "a wait leaked its futex" >&2; exit 1; }
# Wake counting: zero wakes nobody, the count is honoured, and a wake does not
# take the waiters' references.
grep -q '^K wake0 rc=0' "$tmp" || { echo "missing the zero-length wake" >&2; exit 1; }
grep -q '^K wake2 rc=2' "$tmp" || { echo "missing the counted wake" >&2; exit 1; }
grep -q '^H after-wake2-returns addr=0x100000 refs=1 queue=1' "$tmp" || {
    echo "a woken waiter must still hold its own reference until it returns" >&2
    exit 1
}
# Requeue: one woken, two moved, and the two references moved with them.
grep -q '^K requeue rc=3' "$tmp" || { echo "missing the requeue count" >&2; exit 1; }
grep -q '^H requeue-before addr=0x140000 refs=1 queue=1' "$tmp" || {
    echo "missing the requeue target's initial state" >&2
    exit 1
}
grep -q '^H requeue-after addr=0x100000 refs=1 queue=0' "$tmp" || {
    echo "the requeued waiters must leave the source queue" >&2
    exit 1
}
grep -q '^H requeue-after addr=0x140000 refs=3 queue=3' "$tmp" || {
    echo "the moved waiters must take their references to the target" >&2
    exit 1
}
grep -q '^Q requeue-after id=6 futex=0x140000 queued=1' "$tmp" || {
    echo "a moved waiter must be re-pointed at the target" >&2
    exit 1
}
# Releasing every reference takes the entries out of the table.
grep -q '^H drained entries=0' "$tmp" || { echo "the table did not drain" >&2; exit 1; }
# The syscall-level dispatch.
grep -q '^K private-wake rc=0' "$tmp" || { echo "the private flag must be masked off" >&2; exit 1; }
grep -q '^K unsupported rc=-38' "$tmp" || { echo "an unknown operation must be ENOSYS" >&2; exit 1; }
grep -q '^K sys-requeue rc=2' "$tmp" || { echo "missing requeue through sys_futex" >&2; exit 1; }
# sys_set_robust_list checks only the length; sys_get_robust_list only answers
# for the current task.
grep -q '^L set-len8 rc=-22' "$tmp" || { echo "missing the robust-list EINVAL" >&2; exit 1; }
grep -q '^L set-len12 rc=0 robust=0x1000' "$tmp" || { echo "missing the robust-list set" >&2; exit 1; }
grep -q '^L get-self rc=0 list=0x1000 len=12' "$tmp" || { echo "missing the robust-list get" >&2; exit 1; }
grep -q '^L get-other rc=-1' "$tmp" || { echo "another pid must be EPERM" >&2; exit 1; }
# Every notify must name a waiter the corpus registered: an `unknown` here
# would mean the wake went to a condvar the transcript cannot identify.
if grep -q '^N broadcast=unknown' "$tmp" || grep -q '^N signal=unknown' "$tmp"; then
    echo "a notify reached an unregistered condvar" >&2
    exit 1
fi

# The stubs must not have been load-bearing.
grep -q '^# asbestos_invalidations [0-9]* fd_closes 0 signals_sent 0' "$tmp" || {
    echo "a stub was reached; the port's omissions are no longer inert:" >&2
    grep '^# asbestos' "$tmp" >&2
    exit 1
}

out=tests/fixtures/futex_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $waits wait records, $wakes wakes, $tables table records"
