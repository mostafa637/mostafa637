#!/bin/sh
# Regenerate tests/fixtures/ipc_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_ipc_reference.sh
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/kernel/ipc.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include \
    -o /tmp/ipc-dump tools/ipc-dump.c "$ISH_SRC/kernel/ipc.c"

tmp=/tmp/ipc_reference.txt.new
/tmp/ipc-dump >"$tmp"

ops=$(grep -c '^O IPC ' "$tmp")
returns=$(grep -c '^R ' "$tmp")
if [ "$ops" -ne 5 ] || [ "$returns" -ne "$ops" ]; then
    echo "malformed ipc corpus: $ops operations and $returns returns" >&2
    exit 1
fi

out=tests/fixtures/ipc_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $ops C operations"
