#!/bin/sh
# Regenerate tests/fixtures/tlb_reference.txt from an unmodified iSH checkout.
#
#   ISH_SRC=/path/to/ish tools/gen_tlb_reference.sh
#
# Only emu/tlb.c is compiled in - the real page table lives in
# kernel/memory.c and is replaced by a fake backend identical to the one in
# tests/tlb_differential.rs.
set -e

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/emu/tlb.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra \
    -I"$ISH_SRC" \
    -o /tmp/tlb-dump tools/tlb-dump.c "$ISH_SRC/emu/tlb.c"

out=tests/fixtures/tlb_reference.txt
tmp=/tmp/tlb_reference.txt.new
/tmp/tlb-dump >"$tmp"

lines=$(wc -l <"$tmp")
# 2 header lines + one per step
steps=$(grep -c '^T ' "$tmp")
expected=$(grep -o 'steps [0-9]*' "$tmp" | head -1 | cut -d' ' -f2)
if [ "$steps" -ne "$expected" ]; then
    echo "generator produced $steps step records but declared $expected" >&2
    exit 1
fi

mv "$tmp" "$out"
echo "wrote $out: $lines lines, $steps steps"
