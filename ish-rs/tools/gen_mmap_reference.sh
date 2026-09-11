#!/bin/sh
# Regenerate tests/fixtures/mmap_reference.txt from an unmodified iSH checkout.
#
#   ISH_SRC=/path/to/ish tools/gen_mmap_reference.sh
set -e

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/kernel/mmap.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include -o /tmp/mmap-dump \
    tools/mmap-dump.c "$ISH_SRC/kernel/mmap.c" "$ISH_SRC/kernel/memory.c" \
    "$ISH_SRC/kernel/user.c" "$ISH_SRC/kernel/errno.c"

tmp=/tmp/mmap_reference.txt.new
/tmp/mmap-dump >"$tmp"

# no fd may ever be closed and no signal sent, and f_get must have been reached
# exactly once - by the one non-anonymous mmap, which is the only reason the port
# can answer EBADF for file-backed mappings
if ! grep -q '^# asbestos_invalidations [0-9]* fd_closes 0 fd_lookups 1 signals_sent 0$' "$tmp"; then
    echo "the fd stubs were not used the way the port assumes:" >&2
    grep '^# asbestos_invalidations' "$tmp" >&2
    exit 1
fi

out=tests/fixtures/mmap_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $(grep -c '^O ' "$out") operations"
grep '^# asbestos_invalidations' "$out"
