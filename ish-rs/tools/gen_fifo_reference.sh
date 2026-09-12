#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish}"

if [ ! -f "$ISH_SRC/util/fifo.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" -o /tmp/fifo-dump tools/fifo-dump.c "$ISH_SRC/util/fifo.c"

tmp=/tmp/fifo_reference.txt.new
/tmp/fifo-dump >"$tmp"

out=tests/fixtures/fifo_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines"
cat "$out"
