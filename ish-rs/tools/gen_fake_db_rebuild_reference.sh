#!/bin/sh
# Regenerate tests/fixtures/fake_db_rebuild_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_fake_db_rebuild_reference.sh
#
# The harness links the untouched fs/fake-rebuild.c against a fully scripted
# host filesystem: fstatat/unlinkat/linkat are replaced at link time, so the
# recorded inode map and the operation log do not depend on the machine running
# the generator. The corpus covers a hardlink pair, a hardlink triple, a nested
# path, and a path the host no longer has.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/fs/fake-rebuild.c" "$ISH_SRC/fs/fake-db.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

build() {
    cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include \
        -o /tmp/fake-db-rebuild-dump tools/fake-db-rebuild-dump.c \
        "$ISH_SRC/fs/fake-rebuild.c" "$@" \
        -Wl,--wrap=fstatat -Wl,--wrap=unlinkat -Wl,--wrap=linkat
}

# The image has SQLite's runtime but no libsqlite3.so development symlink.
if ! build -Wl,-l:libsqlite3.so.0 2>/dev/null; then
    build -lsqlite3
fi

tmp=/tmp/fake_db_rebuild_reference.txt.new
/tmp/fake-db-rebuild-dump >"$tmp"

stats=$(grep -c '^S ' "$tmp")
paths=$(grep -c '^P ' "$tmp")
steps=$(grep -c '^W ' "$tmp")
files=$(grep -c '^F ' "$tmp")
ops=$(grep -c '^L ' "$tmp")
if [ "$stats" -ne 5 ] || [ "$paths" -ne 8 ] || [ "$steps" -ne 8 ] || [ "$files" -ne 7 ]; then
    echo "unexpected corpus shape: $stats stats, $paths paths, $steps steps, $files host files" >&2
    exit 1
fi
# The rebuild must have restored both hardlink groups, in scan order; the path
# the host lost must not appear at all.
actual=$(sed -n 's/^L //p' "$tmp" | tr '\n' ' ')
if [ "$ops" -ne 6 ] || [ "$actual" != "unlink b link a b unlink f link e f unlink g link e g " ]; then
    echo "host operation log is not the expected hardlink restoration: $actual" >&2
    exit 1
fi
if ! grep -q '^A [0-9a-f]\{16\}$' "$tmp" || ! grep -q '^B [0-9a-f]\{16\}$' "$tmp"; then
    echo "missing database hashes" >&2
    exit 1
fi
if [ "$(grep '^B ' "$tmp" | cut -d' ' -f2)" = "$(grep '^A ' "$tmp" | cut -d' ' -f2)" ]; then
    echo "the rebuild did not change the metadata tables" >&2
    exit 1
fi
if ! grep -q '^O .*index:inode_to_path' "$tmp"; then
    echo "the rebuilt schema lost inode_to_path" >&2
    exit 1
fi
if grep -q 'paths_old\|stats_old' "$tmp"; then
    echo "the rebuild left its scratch tables behind" >&2
    exit 1
fi
out=tests/fixtures/fake_db_rebuild_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $stats stat rows, $ops host operations"
