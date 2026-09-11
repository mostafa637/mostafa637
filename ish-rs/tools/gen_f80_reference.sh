#!/bin/sh
# Regenerate tests/fixtures/f80_reference.txt from the *unmodified* iSH C
# sources, so the Rust port can be checked bit-for-bit against the original.
#
#   ./tools/gen_f80_reference.sh /path/to/ish-checkout
#
# Needs only a C compiler and an iSH checkout (https://github.com/ish-app/ish).
set -eu

ISH_SRC="${1:-/tmp/ish-src}"
here=$(cd "$(dirname "$0")" && pwd)

if [ ! -f "$ISH_SRC/emu/float80.c" ]; then
    echo "error: $ISH_SRC does not look like an iSH checkout (no emu/float80.c)" >&2
    exit 1
fi

out="$here/../tests/fixtures/f80_reference.txt"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# -DNDEBUG: two corpus entries trip an assertion inside iSH's own f80_add
# (f80_sub of the smallest denormal from the smallest normal yields an encoding
# whose exponent field says "denormal" while the integer bit is still set).
# That assertion is a debug-build check on an upstream edge case; the
# arithmetic itself is unchanged, so the reference is built the way iSH ships:
# asserts compiled out. tests/differential.rs documents both entries.
cc -O2 -Wall -DNDEBUG -I"$ISH_SRC" -I"$ISH_SRC/emu" -o "$tmp/f80-dump" \
    "$here/f80-dump.c" "$ISH_SRC/emu/float80.c" -lm
mkdir -p "$(dirname "$out")"
timeout 300 "$tmp/f80-dump" > "$out"

echo "wrote $out ($(wc -l < "$out") lines)"
