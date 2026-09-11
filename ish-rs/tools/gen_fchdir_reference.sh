#!/bin/sh
# Regenerate tests/fixtures/fchdir_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_fchdir_reference.sh
#
# The oracle *includes* util/fchdir.c — the file's own `static lock_t` is what
# it probes — and wraps `fchdir` at link time, so every host call is visible
# with the dirfd the caller passed, while the caller still holds the lock. Two
# worker threads contend for that lock under scripted flags, so the transcript
# does not depend on how the scheduler interleaves them. No other part of iSH is
# linked: util/fchdir.c needs nothing but util/sync.h, whose lock/unlock/trylock
# are header-only.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/util/fchdir.c" "$ISH_SRC/util/sync.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

cc -O2 -Wall -Wextra -I"$ISH_SRC" -o /tmp/fchdir-dump tools/fchdir-dump.c \
    -pthread -Wl,--wrap=fchdir

tmp=/tmp/fchdir_reference.txt.new
/tmp/fchdir-dump >"$tmp"

count() {
    grep -c "^$1" "$tmp"
}

dirfds=$(count 'G ')
records=$(count 'T ')

if [ "$dirfds" -ne 1 ] || [ "$records" -ne 12 ]; then
    echo "unexpected corpus shape: $dirfds dirfd records, $records transcript records" >&2
    exit 1
fi

# The static lock starts unlocked, and one thread is enough to show the whole
# dance: taken around the host call, held while it runs, free afterwards.
grep -q '^T probe-before held=0' "$tmp" || { echo "the static lock did not start free" >&2; exit 1; }
grep -q '^T self fd=3 rc=0 owner=1' "$tmp" || { echo "missing the locked host call" >&2; exit 1; }
grep -q '^T self released held=0' "$tmp" || { echo "unlock_fchdir did not release" >&2; exit 1; }
# A host call that fails is still made under the lock, and its result goes
# nowhere: `lock_fchdir` returns void in C.
grep -q '^T self fd=-1 rc=-1 owner=1' "$tmp" || {
    echo "a failing fchdir must still be made, still under the lock" >&2
    exit 1
}
# The lock is held for the whole host call, and it is what keeps the second
# worker out: it is already knocking, and only one call has been made.
grep -q '^T w1 in fd=3 held=1' "$tmp" || { echo "the host call did not run under the lock" >&2; exit 1; }
grep -q '^T w2 blocked calls=3' "$tmp" || {
    echo "the lock did not exclude the second worker" >&2
    exit 1
}
grep -q '^T w2 in fd=4 held=1' "$tmp" || { echo "the second worker never got the lock" >&2; exit 1; }
grep -q '^T w1 done rc=0' "$tmp" || { echo "missing the first worker's host result" >&2; exit 1; }
grep -q '^T w2 done rc=0' "$tmp" || { echo "missing the second worker's host result" >&2; exit 1; }
# Everything unwinds: one thread at the end holds nothing, and exactly four host
# calls were made.
grep -q '^T end held=0' "$tmp" || { echo "the lock was not released at the end" >&2; exit 1; }
grep -q '^T calls 4' "$tmp" || { echo "the host call count changed" >&2; exit 1; }

out=tests/fixtures/fchdir_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $records transcript records, 2 contending workers"
