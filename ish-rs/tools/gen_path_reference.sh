#!/bin/sh
set -eu
cd "$(dirname "$0")/.."

cc -O2 -Wall -Wextra -o /tmp/path-dump tools/path-dump.c

tmp=/tmp/path_reference.txt.new
/tmp/path-dump >"$tmp"

out=tests/fixtures/path_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines"
cat "$out"
