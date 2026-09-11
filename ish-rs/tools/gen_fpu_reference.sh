#!/bin/sh
# Regenerate tests/fixtures/fpu_reference.txt from the *unmodified* iSH C
# sources, so the Rust cpu/fpu port can be checked word-for-word against the
# original.
#
#   ./tools/gen_fpu_reference.sh /path/to/ish-checkout
#
# Needs only a C compiler and an iSH checkout (https://github.com/ish-app/ish).
set -eu

ISH_SRC="${1:-/tmp/ish-src}"
here=$(cd "$(dirname "$0")" && pwd)

if [ ! -f "$ISH_SRC/emu/fpu.c" ]; then
    echo "error: $ISH_SRC does not look like an iSH checkout (no emu/fpu.c)" >&2
    exit 1
fi

out="$here/../tests/fixtures/fpu_reference.txt"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# -DNDEBUG, for the same reason as the float80 reference: some operands trip
# assertions inside iSH's own debug build (see tests/differential.rs). The
# arithmetic is identical either way.
cc -O2 -Wall -DNDEBUG -I"$ISH_SRC" -o "$tmp/fpu-dump" \
    "$here/fpu-dump.c" "$ISH_SRC/emu/fpu.c" "$ISH_SRC/emu/float80.c" -lm

mkdir -p "$(dirname "$out")"
timeout 300 "$tmp/fpu-dump" > "$out"

echo "wrote $out ($(wc -l < "$out") lines)"
