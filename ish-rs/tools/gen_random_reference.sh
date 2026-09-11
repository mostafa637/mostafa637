#!/bin/sh
# Regenerate tests/fixtures/random_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_random_reference.sh
#
# random.c's Linux entropy function calls syscall(SYS_getrandom). random-dump.c
# wraps that one host boundary at link time, providing deterministic bytes and
# failures while leaving kernel/random.c itself unmodified.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/kernel/random.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include \
    -o /tmp/random-dump tools/random-dump.c "$ISH_SRC/kernel/random.c" \
    -Wl,--wrap=syscall

tmp=/tmp/random_reference.txt.new
/tmp/random-dump >"$tmp"

ops=$(grep -c '^O ' "$tmp")
returns=$(grep -c '^R ' "$tmp")
states=$(grep -c '^S ' "$tmp")
if [ "$returns" -ne "$ops" ] || [ "$states" -ne $((ops + 1)) ]; then
    echo "malformed random corpus: $ops operations, $returns returns, $states states" >&2
    exit 1
fi
if [ "$(grep '^S ' "$tmp" | tail -1 | cut -d' ' -f2)" -ne 8 ]; then
    echo "the expected eight deterministic host-source calls did not occur" >&2
    exit 1
fi

out=tests/fixtures/random_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $ops C operations"
