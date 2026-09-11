#!/bin/sh
# Regenerate tests/fixtures/fake_migrate_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_fake_migrate_reference.sh
#
# The harness seeds v0, v1, and v2 metadata schemas, then links the untouched
# fs/fake-migrate.c against the platform SQLite runtime. It records hashes of
# the ordered logical tables after each migration and after the v2 unlink that
# distinguishes the transient v2 delete trigger from the current v3 schema.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/fs/fake-migrate.c" "$ISH_SRC/fs/fake-db.h" "$ISH_SRC/fs/sqlutil.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

# When the input is a Git checkout, reject locally edited oracle source rather
# than silently blessing behavior from a modified C implementation. Exported
# source trees remain usable because they have no Git HEAD to compare against.
if git -C "$ISH_SRC" rev-parse --verify HEAD >/dev/null 2>&1 \
    && ! git -C "$ISH_SRC" diff --quiet HEAD -- fs/fake-migrate.c; then
    echo "fs/fake-migrate.c must be unmodified to regenerate this C oracle" >&2
    exit 1
fi

build() {
    cc -O2 -Wall -Wextra -Werror -I"$ISH_SRC" -Itools/stub-include \
        -o /tmp/fake-migrate-dump tools/fake-migrate-dump.c \
        "$ISH_SRC/fs/fake-migrate.c" "$@"
}

# Arena's Linux image intentionally has the SQLite runtime but no libsqlite3.so
# development symlink. This native library is used only for the untouched C
# oracle; Rust production code uses the vendored pure-Rust GraphiteSQL engine.
if ! build -Wl,-l:libsqlite3.so.0 2>/dev/null; then
    build -lsqlite3
fi

tmp=/tmp/fake_migrate_reference.txt.new
/tmp/fake-migrate-dump >"$tmp"

migrations=$(grep -c '^M ' "$tmp")
unlinks=$(grep -c '^D ' "$tmp")
if [ "$migrations" -ne 3 ] || [ "$unlinks" -ne 1 ] \
    || ! grep -q '^M 0 3 ' "$tmp" \
    || ! grep -q '^M 1 3 ' "$tmp" \
    || ! grep -q '^M 2 3 ' "$tmp" \
    || ! grep -q '^D 2 ' "$tmp"; then
    echo "malformed fake-migrate corpus: $migrations migrations, $unlinks unlinks" >&2
    exit 1
fi

out=tests/fixtures/fake_migrate_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $migrations C migrations, $unlinks v2 unlink"
