#!/bin/sh
# Regenerate tests/fixtures/sync_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_sync_reference.sh
#
# The harness links the untouched util/sync.c. Its three host dependencies are
# replaced at link time — pthread_cond_wait / pthread_cond_timedwait decide the
# wait's fate, clock_gettime(CLOCK_MONOTONIC) supplies the timeout origin, and
# pthread_cond_broadcast / pthread_cond_signal reveal the notify target — so
# every recorded value is independent of the machine running the generator.
#
# util/sync.c is compiled at -O2, as in a normal build. The *harness* is
# compiled at -O0 on purpose: `sigunwind_start` is a `static inline` wrapper
# around `sigsetjmp`, and the corpus deliberately excludes the one path that
# longjmps into it (see tools/sync-dump.c), which a portable oracle cannot
# script without depending on how the compiler lays out that frame.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/util/sync.c" "$ISH_SRC/util/sync.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include \
    -c -o /tmp/sync-under-test.o "$ISH_SRC/util/sync.c"
cc -O0 -g -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include \
    -o /tmp/sync-dump tools/sync-dump.c /tmp/sync-under-test.o -pthread \
    -Wl,--wrap=pthread_cond_wait -Wl,--wrap=pthread_cond_timedwait \
    -Wl,--wrap=clock_gettime \
    -Wl,--wrap=pthread_cond_broadcast -Wl,--wrap=pthread_cond_signal

tmp=/tmp/sync_reference.txt.new
/tmp/sync-dump >"$tmp"

waits=$(grep -c '^W ' "$tmp")
ignores=$(grep -c '^I ' "$tmp")
parks=$(grep -c '^P ' "$tmp")
notifies=$(grep -c '^C ' "$tmp")
unwinds=$(grep -c '^U ' "$tmp")
locks=$(grep -c '^L ' "$tmp")
wrlocks=$(grep -c '^R ' "$tmp")
if [ "$waits" -lt 12 ] || [ "$ignores" -ne 3 ] || [ "$parks" -lt 14 ]; then
    echo "sync corpus is too small: $waits waits, $ignores ignore-signal waits, $parks parks" >&2
    exit 1
fi
if [ "$notifies" -ne 2 ] || [ "$unwinds" -ne 6 ] || [ "$locks" -ne 4 ] || [ "$wrlocks" -ne 6 ]; then
    echo "unexpected record counts: $notifies notify, $unwinds unwind, $locks lock, $wrlocks wrlock" >&2
    exit 1
fi
# The corpus must cover both EINTR checks, the timeout mapping, the carry, and
# the unnormalized `> 1000000000` deadline C leaves behind.
grep -q '^W pending=1 blocked=0 rc=-4' "$tmp" || { echo "missing the pre-wait EINTR" >&2; exit 1; }
grep -q '^W pending=0 blocked=0 rc=-4' "$tmp" || { echo "missing the post-wait EINTR" >&2; exit 1; }
grep -q ' rc=-110' "$tmp" || { echo "missing the timeout mapping" >&2; exit 1; }
grep -q '^P timedwait 101.200000000' "$tmp" || { echo "missing the nanosecond carry" >&2; exit 1; }
grep -q '^P timedwait 100.1000000000' "$tmp" || { echo "missing the unnormalized deadline" >&2; exit 1; }
grep -q ' during_cond=0 during_lock=0' "$tmp" || { echo "missing the no-current-task wait" >&2; exit 1; }
grep -q '^L trylock_free rc=0 owner_self=0' "$tmp" || {
    echo "trylock must not set the lock owner" >&2
    exit 1
}
grep -q '^R write val=-1 file=set' "$tmp" || { echo "missing the write-lock counter" >&2; exit 1; }

out=tests/fixtures/sync_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $waits waits, $ignores ignore-signal waits, $parks parks"
