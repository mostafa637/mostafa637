#!/bin/sh
# Regenerate tests/fixtures/fake_rebuild_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_fake_rebuild_reference.sh
#
# The harness links untouched fs/fake-rebuild.c to the platform SQLite runtime,
# seeds old fakefs metadata, and substitutes deterministic fstatat/unlinkat/
# linkat calls. It records rebuilt database and host-state hashes plus syscall
# counts for comparison with the pure-Rust implementation.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/fs/fake-rebuild.c" "$ISH_SRC/fs/fake-db.h" "$ISH_SRC/fs/sqlutil.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

# When the input is a Git checkout, reject locally edited oracle source rather
# than silently blessing behavior from a modified C implementation. Exported
# source trees remain usable because they have no Git HEAD to compare against.
if git -C "$ISH_SRC" rev-parse --verify HEAD >/dev/null 2>&1 \
    && ! git -C "$ISH_SRC" diff --quiet HEAD -- fs/fake-rebuild.c; then
    echo "fs/fake-rebuild.c must be unmodified to regenerate this C oracle" >&2
    exit 1
fi

build() {
    cc -O2 -Wall -Wextra -Werror -D_GNU_SOURCE -I"$ISH_SRC" -Itools/stub-include \
        -o /tmp/fake-rebuild-dump tools/fake-rebuild-dump.c \
        "$ISH_SRC/fs/fake-rebuild.c" "$@"
}

# Arena's Linux image intentionally has the SQLite runtime but no libsqlite3.so
# development symlink. It is only for this local untouched-C oracle; production
# Rust uses the vendored pure-Rust GraphiteSQL SQLite engine.
if ! build -Wl,-l:libsqlite3.so.0 2>/dev/null; then
    build -lsqlite3
fi

tmp=/tmp/fake_rebuild_reference.txt.new
/tmp/fake-rebuild-dump >"$tmp"

if [ "$(grep -c '^D ' "$tmp")" -ne 1 ] \
    || [ "$(grep -c '^H ' "$tmp")" -ne 1 ] \
    || [ "$(grep -c '^C ' "$tmp")" -ne 1 ] \
    || [ "$(grep -c '^T ' "$tmp")" -ne 1 ]; then
    echo "malformed fake-rebuild corpus" >&2
    exit 1
fi

out=tests/fixtures/fake_rebuild_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines from unmodified C rebuild"
