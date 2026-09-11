#!/bin/sh
# Regenerate tests/fixtures/fake_db_init_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_fake_db_init_reference.sh
#
# The harness runs upstream fake_db_init against a temporary SQLite database
# with deterministic host operations. It records the post-initialization
# metadata/host state, host syscall counts, and meta.db_inode update. SQLite is
# used only in this local C oracle; production Rust uses GraphiteSQL.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in \
    "$ISH_SRC/fs/fake-db.c" \
    "$ISH_SRC/fs/fake-migrate.c" \
    "$ISH_SRC/fs/fake-rebuild.c" \
    "$ISH_SRC/fs/fake-db.h" \
    "$ISH_SRC/fs/sqlutil.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

# A fixture generated from modified C would not establish behavioral fidelity.
# Reject dirty source when ISH_SRC is a Git checkout; exported trees with no
# HEAD remain usable for local users.
if git -C "$ISH_SRC" rev-parse --verify HEAD >/dev/null 2>&1 \
    && ! git -C "$ISH_SRC" diff --quiet HEAD -- \
        fs/fake-db.c fs/fake-migrate.c fs/fake-rebuild.c fs/fake-db.h fs/sqlutil.h; then
    echo "fakefs C sources and headers must be unmodified for this C oracle" >&2
    exit 1
fi

build() {
    cc -O2 -Wall -Wextra -Werror -D_GNU_SOURCE -I"$ISH_SRC" -Itools/stub-include \
        -o /tmp/fake-db-init-dump tools/fake-db-init-dump.c \
        "$ISH_SRC/fs/fake-db.c" "$ISH_SRC/fs/fake-migrate.c" \
        "$ISH_SRC/fs/fake-rebuild.c" "$@"
}

# Arena's Linux image has the SQLite runtime but not necessarily its development
# symlink. It is linked only into the untouched-C local oracle.
if ! build -Wl,-l:libsqlite3.so.0 2>/dev/null; then
    build -lsqlite3
fi

tmp=/tmp/fake_db_init_reference.txt.new
/tmp/fake-db-init-dump >"$tmp"

if [ "$(grep -c '^D ' "$tmp")" -ne 1 ] \
    || [ "$(grep -c '^H ' "$tmp")" -ne 1 ] \
    || [ "$(grep -c '^C ' "$tmp")" -ne 1 ] \
    || [ "$(grep -c '^N ' "$tmp")" -ne 1 ] \
    || [ "$(grep -c '^M ' "$tmp")" -ne 1 ]; then
    echo "malformed fake-db-init corpus" >&2
    exit 1
fi

out=tests/fixtures/fake_db_init_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines from unmodified C fake_db_init"
