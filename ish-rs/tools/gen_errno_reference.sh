#!/bin/sh
# Regenerate tests/fixtures/errno_reference.txt from an unmodified iSH checkout.
#
#   ISH_SRC=/path/to/ish tools/gen_errno_reference.sh
#
# Run this on the host you build for: the fixture records that host's errno
# numbers, and so does src/errno_table.rs (via tools/gen_errno_table.py). The two
# have to come from the same place.
set -e

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/kernel/errno.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include -o /tmp/errno-dump \
    tools/errno-dump.c "$ISH_SRC/kernel/errno.c"

out=tests/fixtures/errno_reference.txt
/tmp/errno-dump >"$out"
lines=$(wc -l <"$out")
echo "wrote $out: $lines lines, $(grep -c '^M ' "$out") err_map inputs"
grep '^# host EPERM' "$out"
grep '^# unknown_errors' "$out"
