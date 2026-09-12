#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish}"

if [ ! -f "$ISH_SRC/util/bits.h" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" -o /tmp/bits-dump tools/bits-dump.c

tmp=/tmp/bits_reference.txt.new
/tmp/bits-dump >"$tmp"

out=tests/fixtures/bits_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines"
cat "$out"
