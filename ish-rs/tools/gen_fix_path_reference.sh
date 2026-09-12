#!/bin/sh
# Regenerate tests/fixtures/fix_path_reference.txt from unmodified iSH C.
set -eu
cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish}"

if [ ! -f "$ISH_SRC/fs/fix_path.h" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" -o /tmp/fix-path-dump tools/fix-path-dump.c

tmp=/tmp/fix_path_reference.txt.new
/tmp/fix-path-dump >"$tmp"

out=tests/fixtures/fix_path_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines"
cat "$out"
