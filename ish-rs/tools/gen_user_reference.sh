#!/bin/sh
# Regenerate tests/fixtures/user_reference.txt from an unmodified iSH checkout.
#
#   ISH_SRC=/path/to/ish tools/gen_user_reference.sh
#
# This links the real kernel/user.c, the kernel/memory.c it sits on, and the real
# kernel/errno.c, so it needs the same stub sqlite3.h as gen_memory_reference.sh
# (see tools/stub-include).
set -e

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/kernel/user.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include -o /tmp/user-dump \
    tools/user-dump.c "$ISH_SRC/kernel/user.c" "$ISH_SRC/kernel/memory.c" \
    "$ISH_SRC/kernel/errno.c"

tmp=/tmp/user_reference.txt.new
/tmp/user-dump >"$tmp"

# the stubs must not have been load-bearing: no fd was ever closed and no signal
# was ever sent, so nothing the port leaves out was on the path
if ! grep -q '^# asbestos_invalidations [0-9]* fd_closes 0 signals_sent 0$' "$tmp"; then
    echo "a stub was reached; the port's omissions are no longer inert:" >&2
    grep '^# asbestos_invalidations' "$tmp" >&2
    exit 1
fi

out=tests/fixtures/user_reference.txt
mv "$tmp" "$out"
lines=$(wc -l <"$out")
ops=$(grep -c '^O ' "$out")
bytes=$(du -h "$out" | cut -f1)
echo "wrote $out: $lines lines, $ops operations, $bytes"
grep '^# asbestos_invalidations' "$out"
