#!/bin/sh
# Regenerate tests/fixtures/fake_db_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_fake_db_reference.sh
#
# The harness links fs/fake-db.c to the platform SQLite runtime and creates the
# current fakefs schema itself. Migration/rebuild are intentionally no-op hooks
# because this corpus targets the reusable metadata API in fake-db.c.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/fs/fake-db.c" "$ISH_SRC/fs/fake-db.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

build() {
    cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include \
        -o /tmp/fake-db-dump tools/fake-db-dump.c "$ISH_SRC/fs/fake-db.c" "$@"
}

# Arena's Linux image intentionally has the SQLite runtime but no libsqlite3.so
# development symlink. Prefer the runtime's exact linker name, then fall back
# to a conventional development installation or Apple SDK.
if ! build -Wl,-l:libsqlite3.so.0 2>/dev/null; then
    build -lsqlite3
fi

tmp=/tmp/fake_db_reference.txt.new
/tmp/fake-db-dump >"$tmp"

ops=$(grep -c '^O ' "$tmp")
returns=$(grep -c '^R ' "$tmp")
states=$(grep -c '^S ' "$tmp")
values=$(grep -c '^V ' "$tmp")
if [ "$ops" -ne 40 ] || [ "$returns" -ne "$ops" ] || [ "$states" -ne $((ops + 1)) ]; then
    echo "malformed fake-db corpus: $ops operations, $returns returns, $states states" >&2
    exit 1
fi
if [ "$values" -ne 9 ] || ! grep -q '^O N 2f61 2f78$' "$tmp" || ! grep -q '^O P 2f6170706c65$' "$tmp" || ! grep -q '^O F 0$' "$tmp" || ! grep -q '^O A$' "$tmp"; then
    echo "fake-db corpus missed rename, sibling boundary, rollback, orphan cleanup, or metadata reads" >&2
    exit 1
fi

out=tests/fixtures/fake_db_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $ops C operations, $states snapshots"
