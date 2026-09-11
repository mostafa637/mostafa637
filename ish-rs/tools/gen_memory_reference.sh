#!/bin/sh
# Regenerate tests/fixtures/memory_reference.txt from an unmodified iSH checkout.
#
#   ISH_SRC=/path/to/ish tools/gen_memory_reference.sh
#
# This one links the real kernel/memory.c rather than including a header, so it
# also needs kernel/errno.c and a stub sqlite3.h (see tools/stub-include):
# memory.c includes fs/fd.h, which includes fs/fake-db.h, which only needs three
# sqlite types as opaque pointers.
set -e

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/kernel/memory.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include -o /tmp/memory-dump \
    tools/memory-dump.c "$ISH_SRC/kernel/memory.c" "$ISH_SRC/kernel/errno.c"

tmp=/tmp/memory_reference.txt.new
/tmp/memory-dump >"$tmp"

# the stubs must not have been load-bearing: no fd was ever closed and no signal
# was ever sent, so nothing the port leaves out was on the path
if ! grep -q '^# asbestos_invalidations [0-9]* fd_closes 0 signals_sent 0$' "$tmp"; then
    echo "a stub was reached; the port's omissions are no longer inert:" >&2
    grep '^# asbestos_invalidations' "$tmp" >&2
    exit 1
fi

out=tests/fixtures/memory_reference.txt
mv "$tmp" "$out"
lines=$(wc -l <"$out")
ops=$(grep -c '^O ' "$out")
echo "wrote $out: $lines lines, $ops operations"
grep '^# asbestos_invalidations' "$out"
