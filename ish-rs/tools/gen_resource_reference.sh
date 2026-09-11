#!/bin/sh
# Regenerate tests/fixtures/resource_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_resource_reference.sh
#
# resource.c normally reads current host-thread CPU time and the host's online
# CPU count. resource-dump.c links the source untouched, but wraps those two
# host functions at link time and feeds them deterministic corpus values. Its
# two-page user-memory harness only supplies the read/write-or-EFAULT boundary;
# kernel/user.c and kernel/memory.c are independently differential-tested.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

if [ ! -f "$ISH_SRC/kernel/resource.c" ]; then
    echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
    exit 1
fi

cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include \
    -o /tmp/resource-dump tools/resource-dump.c "$ISH_SRC/kernel/resource.c" \
    -pthread -Wl,--wrap=getrusage -Wl,--wrap=sysconf

tmp=/tmp/resource_reference.txt.new
/tmp/resource-dump >"$tmp"

ops=$(grep -c '^O ' "$tmp")
returns=$(grep -c '^R ' "$tmp")
states=$(grep -c '^S ' "$tmp")
if [ "$returns" -ne "$ops" ]; then
    echo "generator produced $ops operations but $returns return records" >&2
    exit 1
fi
if [ "$states" -ne $((ops + 1)) ]; then
    echo "generator produced $ops operations but $states state snapshots" >&2
    exit 1
fi
# The corpus is intentionally broad. These checks fail loudly if a future edit
# accidentally drops one of resource.c's distinct syscall families.
for op in RL G32 OG32 S32 PR RC RU ADD AFF SAFF GPRI SPRI GPAR GSCH SSCH PMAX IGET ISET; do
    if ! grep -q "^O $op" "$tmp"; then
        echo "resource corpus did not exercise $op" >&2
        exit 1
    fi
done
if ! grep -q '^V CURRENT ' "$tmp"; then
    echo "resource corpus did not record the deterministic current rusage" >&2
    exit 1
fi

out=tests/fixtures/resource_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $ops C operations, $states snapshots"
printf 'families:'
grep '^O ' "$out" | cut -d' ' -f2 | sort -u | tr '\n' ' '
printf '\n'
