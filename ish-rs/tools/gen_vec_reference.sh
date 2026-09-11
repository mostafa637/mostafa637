#!/bin/sh
# Regenerate tests/fixtures/vec_reference.txt from an unmodified iSH checkout.
#
#   ISH_SRC=/path/to/ish tools/gen_vec_reference.sh
#
# Compiles emu/vec.c + emu/mmx.c untouched; the op table comes from
# tools/gen_vec_ops.py reading emu/vec.h, so a new function in the header is
# picked up automatically rather than silently skipped.
set -e

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/emu/vec.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

python3 tools/gen_vec_ops.py "$ISH_SRC" > tools/vec-ops.inc

cc -O2 -Wall -Wextra -fno-strict-aliasing \
    -I"$ISH_SRC" -Itools \
    -o /tmp/vec-dump tools/vec-dump.c "$ISH_SRC/emu/vec.c" "$ISH_SRC/emu/mmx.c" -lm

out=tests/fixtures/vec_reference.txt
tmp=/tmp/vec_reference.txt.new
/tmp/vec-dump >"$tmp"

# 2 header lines + NCASES case lines + one record per op per case
records=$(grep -c '^V ' "$tmp")
declared_ops=$(grep -o 'ops [0-9]*' "$tmp" | head -1 | cut -d' ' -f2)
cases=$(grep -c '^# C ' "$tmp")
expected=$((declared_ops * cases))
if [ "$records" -ne "$expected" ]; then
    echo "generator produced $records records but declared $expected" >&2
    exit 1
fi
distinct=$(grep '^V ' "$tmp" | cut -d' ' -f2 | sort -u | wc -l)
if [ "$distinct" -ne "$declared_ops" ]; then
    echo "only $distinct distinct operations were exercised, expected $declared_ops" >&2
    exit 1
fi

mv "$tmp" "$out"
lines=$(wc -l <"$out")
echo "wrote $out: $lines lines, $records records, $distinct operations, $cases cases"
