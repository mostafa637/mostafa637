#!/bin/sh
# Regenerate tests/fixtures/uname_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_uname_reference.sh
#
# The C source obtains a host node name, uptime/load and Linux sysinfo values.
# uname-dump.c controls those three host boundaries. SOURCE_DATE_EPOCH fixes
# __DATE__/__TIME__, which uname.c includes in its guest version field.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/kernel/uname.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

SOURCE_DATE_EPOCH=0 cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include \
    -o /tmp/uname-dump tools/uname-dump.c "$ISH_SRC/kernel/uname.c" \
    -Wl,--wrap=uname -Wl,--wrap=sysinfo

tmp=/tmp/uname_reference.txt.new
/tmp/uname-dump >"$tmp"

ops=$(grep -c '^O ' "$tmp")
returns=$(grep -c '^R ' "$tmp")
states=$(grep -c '^S ' "$tmp")
values=$(grep -c '^V ' "$tmp")
if [ "$returns" -ne "$ops" ] || [ "$states" -ne $((ops + 1)) ]; then
    echo "malformed uname corpus: $ops operations, $returns returns, $states states" >&2
    exit 1
fi
if [ "$values" -ne 3 ]; then
    echo "expected three direct do_uname records, got $values" >&2
    exit 1
fi
for op in DU UN SH SI; do
    if ! grep -q "^O $op" "$tmp"; then
        echo "uname corpus did not exercise $op" >&2
        exit 1
    fi
done

out=tests/fixtures/uname_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $ops C operations, $states snapshots"
