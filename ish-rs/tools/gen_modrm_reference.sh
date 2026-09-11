#!/bin/sh
# Regenerate tests/fixtures/modrm_reference.txt from an unmodified iSH checkout.
#
#   ISH_SRC=/path/to/ish tools/gen_modrm_reference.sh
set -e

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/emu/modrm.h" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" \
    -o /tmp/modrm-dump tools/modrm-dump.c "$ISH_SRC/emu/tlb.c"

out=tests/fixtures/modrm_reference.txt
tmp=/tmp/modrm_reference.txt.new
/tmp/modrm-dump >"$tmp"

records=$(grep -c '^M ' "$tmp")
declared=$(grep '^# cases ' "$tmp" | cut -d' ' -f3)
if [ "$records" -ne "$declared" ]; then
    echo "generator produced $records records but declared $declared" >&2
    exit 1
fi

mv "$tmp" "$out"
lines=$(wc -l <"$out")
echo "wrote $out: $lines lines, $records cases"
