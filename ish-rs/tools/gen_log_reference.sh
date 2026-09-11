#!/bin/sh
# Regenerate tests/fixtures/log_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_log_reference.sh
#
# log-dump.c selects log.c's dprintf handler and wraps only writev(2), turning
# host-visible lines into deterministic state while preserving kernel/log.c and
# util/fifo.c unchanged.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/kernel/log.c" "$ISH_SRC/util/fifo.c"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

cc -O2 -Wall -Wextra -DLOG_HANDLER_DPRINTF=1 -I"$ISH_SRC" -Itools/stub-include \
    -o /tmp/log-dump tools/log-dump.c "$ISH_SRC/kernel/log.c" "$ISH_SRC/util/fifo.c" \
    -pthread -Wl,--wrap=writev

tmp=/tmp/log_reference.txt.new
/tmp/log-dump >"$tmp"

ops=$(grep -c '^O ' "$tmp")
returns=$(grep -c '^R ' "$tmp")
states=$(grep -c '^S ' "$tmp")
values=$(grep -c '^V ' "$tmp")
if [ "$returns" -ne "$ops" ] || [ "$states" -ne $((ops + 1)) ]; then
    echo "malformed log corpus: $ops operations, $returns returns, $states states" >&2
    exit 1
fi
if [ "$values" -ne 9 ]; then
    echo "expected nine defined log-read outputs, got $values" >&2
    exit 1
fi
if ! grep -q '^O G 00002000 000000ff 00000011$' "$tmp"; then
    echo "log corpus did not cover the deterministic circular-buffer wrap" >&2
    exit 1
fi

out=tests/fixtures/log_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $ops C operations, $states snapshots"
