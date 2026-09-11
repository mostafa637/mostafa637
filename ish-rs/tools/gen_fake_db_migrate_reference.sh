#!/bin/sh
# Regenerate tests/fixtures/fake_db_migrate_reference.txt from unmodified iSH C.
#
#   ISH_SRC=/path/to/ish tools/gen_fake_db_migrate_reference.sh
#
# The harness *includes* fs/fake-migrate.c, so the migration SQL printed into
# the fixture is the text the C compiler built for the C table, and the ladder
# under test is the untouched upstream one. It drives that ladder over corpus
# databases starting at every historical schema generation and records what the
# rebuild produced: user_version, a hash of the ordered logical tables, the
# sqlite_master object list, the `paths` foreign key, and a delete probe taken
# before and after the migration that shows the version 2 trigger in action.
set -eu

cd "$(dirname "$0")/.."
ISH_SRC="${ISH_SRC:-/tmp/ish-src}"

for source in "$ISH_SRC/fs/fake-migrate.c" "$ISH_SRC/fs/fake-db.h"; do
    if [ ! -f "$source" ]; then
        echo "ISH_SRC=$ISH_SRC does not look like an iSH checkout" >&2
        exit 1
    fi
done

build() {
    cc -O2 -Wall -Wextra -I"$ISH_SRC" -Itools/stub-include \
        -o /tmp/fake-db-migrate-dump tools/fake-db-migrate-dump.c "$@"
}

# The image has SQLite's runtime but no libsqlite3.so development symlink.
if ! build -Wl,-l:libsqlite3.so.0 2>/dev/null; then
    build -lsqlite3
fi

tmp=/tmp/fake_db_migrate_reference.txt.new
/tmp/fake-db-migrate-dump >"$tmp"

migrations=$(grep -c '^M ' "$tmp")
cases=$(grep -c '^C ' "$tmp")
if [ "$migrations" -ne 3 ]; then
    echo "generator produced $migrations migration records instead of 3" >&2
    exit 1
fi
if [ "$cases" -lt 4 ]; then
    echo "generator produced only $cases corpus cases" >&2
    exit 1
fi
for start in 0 1 2 3 5; do
    if ! grep -q "^C $start\$" "$tmp"; then
        echo "corpus is missing the database generation $start case" >&2
        exit 1
    fi
done
for field in B BO T0 V H O K T1; do
    records=$(grep -c "^$field " "$tmp")
    if [ "$records" -ne "$cases" ]; then
        echo "expected $cases $field records, found $records" >&2
        exit 1
    fi
done
# The version 2 case must show the trigger cascading before the migration and
# its removal after it; the already-current cases must not change at all.
t0_2=$(awk '/^C 2$/{seen=1} seen && /^T0 /{print $2; exit}' "$tmp")
t1_2=$(awk '/^C 2$/{seen=1} seen && /^T1 /{print $2; exit}' "$tmp")
t0_3=$(awk '/^C 3$/{seen=1} seen && /^T0 /{print $2; exit}' "$tmp")
t1_3=$(awk '/^C 3$/{seen=1} seen && /^T1 /{print $2; exit}' "$tmp")
if [ "$t0_2" = "$t1_2" ] || [ "$t0_3" != "$t1_3" ]; then
    echo "delete probe did not observe the version 2 trigger" >&2
    exit 1
fi
if ! grep -q '^BO .*trigger:delete_path$' "$tmp" || grep -q '^O .*trigger:delete_path$' "$tmp"; then
    echo "the version 2 trigger must exist before migration and not after" >&2
    exit 1
fi

out=tests/fixtures/fake_db_migrate_reference.txt
mv "$tmp" "$out"
echo "wrote $out: $(wc -l <"$out") lines, $cases C generation cases, $migrations migrations"
