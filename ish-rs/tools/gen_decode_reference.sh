#!/bin/sh
# Regenerate tests/fixtures/decode_reference.txt from an unmodified iSH checkout.
#
#   ISH_SRC=/path/to/ish tools/gen_decode_reference.sh
#
# emu/decode.h is a template, not a library: asbestos/gen.c includes it twice
# and supplies ~150 macros. tools/decode-dump.c supplies the same macros as
# recorders, so the fixture is the unmodified header's own behaviour.
set -e

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/emu/decode.h" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" \
    -o /tmp/decode-dump tools/decode-dump.c "$ISH_SRC/emu/tlb.c"

tmp=/tmp/decode_reference.txt.new
/tmp/decode-dump >"$tmp"

# the generator checks its own arithmetic and its own central assumption
declared=$(grep '^# cases ' "$tmp" | cut -d' ' -f3)
records=$(grep -c '^C ' "$tmp")
if [ "$records" -ne "$declared" ]; then
    echo "generator produced $records cases but declared $declared" >&2
    exit 1
fi
if ! grep -q '^# class_invariant ' "$tmp"; then
    echo "the ModRM class invariant did not run" >&2
    exit 1
fi

out=tests/fixtures/decode_reference.txt
mv "$tmp" "$out"
lines=$(wc -l <"$out")
echo "wrote $out: $lines lines, $records cases"
grep '^# class_invariant ' "$out" | cut -c1-120
