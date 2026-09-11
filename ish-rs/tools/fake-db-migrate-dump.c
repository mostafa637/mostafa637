// Reference-output generator for the fakefs migration differential test.
//
// This *includes* the unmodified fs/fake-migrate.c so that the migration SQL it
// prints is byte-for-byte the text the C compiler builds, not a copy pasted
// into a harness. It then replays the ladder over corpus databases that start
// at each historical schema version and records what the migration produced:
// the user_version, a hash of the ordered logical tables, the sqlite_master
// object list, the `paths` foreign key, and a delete probe that shows whether
// the version 2 trigger survived.
//
// Build (normally via tools/gen_fake_db_migrate_reference.sh):
//   cc -O2 -Wall -Wextra -I<ish-src> -Itools/stub-include
//      -o fake-db-migrate-dump tools/fake-db-migrate-dump.c
//      -Wl,-l:libsqlite3.so.0
//
// The harness defines the same `die`/`ish_printk` hooks the other fake-db
// tools define; the migration code itself is untouched and unstubbed.

#define _GNU_SOURCE

#include <assert.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "fs/fake-db.h"
#include "fs/fake-migrate.c"

_Noreturn void die(const char *message, ...) {
    va_list args;
    va_start(args, message);
    vfprintf(stderr, message, args);
    va_end(args);
    fputc('\n', stderr);
    abort();
}

void ish_printk(const char *UNUSED(message), ...) {}

static void check(int status, sqlite3 *db) {
    if (status != SQLITE_OK && status != SQLITE_ROW && status != SQLITE_DONE) {
        fprintf(stderr, "sqlite failure %d: %s\n", status, sqlite3_errmsg(db));
        abort();
    }
}

static void exec_sql(sqlite3 *db, const char *sql) {
    check(sqlite3_exec(db, sql, NULL, NULL, NULL), db);
}

static uint64_t hash_byte(uint64_t hash, uint8_t byte) {
    hash ^= byte;
    return hash * UINT64_C(0x100000001b3);
}

static uint64_t hash_bytes(uint64_t hash, const void *data, size_t count) {
    const uint8_t *bytes = data;
    for (size_t i = 0; i < count; i++)
        hash = hash_byte(hash, bytes[i]);
    return hash;
}

static uint64_t hash_u32(uint64_t hash, uint32_t value) {
    for (unsigned i = 0; i < 4; i++)
        hash = hash_byte(hash, (uint8_t) (value >> (i * 8)));
    return hash;
}

static uint64_t hash_u64(uint64_t hash, uint64_t value) {
    for (unsigned i = 0; i < 8; i++)
        hash = hash_byte(hash, (uint8_t) (value >> (i * 8)));
    return hash;
}

// Decode the corpus's hex text into the bytes a stat blob holds.
static void unhex(const char *hex_text, uint8_t *out, size_t count) {
    for (size_t i = 0; i < count; i++) {
        unsigned value = 0;
        if (sscanf(hex_text + i * 2, "%2x", &value) != 1) {
            fprintf(stderr, "bad hex in corpus: %s\n", hex_text);
            abort();
        }
        out[i] = (uint8_t) value;
    }
}

static void emit_hex(const void *data, size_t count) {
    const uint8_t *bytes = data;
    for (size_t i = 0; i < count; i++)
        printf("%02x", bytes[i]);
}

// Hash the logical tables, not SQLite pages or pointer identities.
static uint64_t database_hash(sqlite3 *db) {
    uint64_t hash = UINT64_C(0xcbf29ce484222325);
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(db, "select inode, stat from stats order by inode", -1, &stmt, NULL), db);
    hash = hash_bytes(hash, "stats", 5);
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        hash = hash_u64(hash, (uint64_t) sqlite3_column_int64(stmt, 0));
        const void *stat = sqlite3_column_blob(stmt, 1);
        int count = sqlite3_column_bytes(stmt, 1);
        hash = hash_u32(hash, (uint32_t) count);
        hash = hash_bytes(hash, stat, (size_t) count);
    }
    check(sqlite3_finalize(stmt), db);

    check(sqlite3_prepare_v2(db, "select path, inode from paths order by path", -1, &stmt, NULL), db);
    hash = hash_bytes(hash, "paths", 5);
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        const void *path = sqlite3_column_blob(stmt, 0);
        int count = sqlite3_column_bytes(stmt, 0);
        hash = hash_u32(hash, (uint32_t) count);
        hash = hash_bytes(hash, path, (size_t) count);
        hash = hash_u64(hash, (uint64_t) sqlite3_column_int64(stmt, 1));
    }
    check(sqlite3_finalize(stmt), db);
    return hash;
}

// `<type>:<name>` for every sqlite_master row, sorted by type then name.
static void print_objects(sqlite3 *db, const char *label) {
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(db,
                             "select type, name from sqlite_master order by type, name",
                             -1, &stmt, NULL), db);
    printf("%s", label);
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        printf(" %s:%s", sqlite3_column_text(stmt, 0), sqlite3_column_text(stmt, 1));
    }
    putchar('\n');
    check(sqlite3_finalize(stmt), db);
}

// `paths` foreign keys as `table.from->to`, in the pragma's row order.
static void print_foreign_keys(sqlite3 *db) {
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(db, "pragma foreign_key_list(paths)", -1, &stmt, NULL), db);
    printf("K");
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        printf(" %s.%s->%s", sqlite3_column_text(stmt, 2), sqlite3_column_text(stmt, 3),
               sqlite3_column_text(stmt, 4));
    }
    putchar('\n');
    check(sqlite3_finalize(stmt), db);
}

static int user_version(sqlite3 *db) {
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(db, "pragma user_version", -1, &stmt, NULL), db);
    check(sqlite3_step(stmt), db);
    int version = sqlite3_column_int(stmt, 0);
    check(sqlite3_finalize(stmt), db);
    return version;
}

// Historical schema generations. `version 0` is the oldest database layout in
// the ladder's history: no `inode_to_path` index and no foreign key on paths.
static void create_schema(sqlite3 *db, int version) {
    exec_sql(db, "create table meta (id integer unique default 0, db_inode integer);"
                 "insert into meta (db_inode) values (0);"
                 "create table stats (inode integer primary key, stat blob);");
    if (version < 2) {
        exec_sql(db, "create table paths (path blob primary key, inode integer);");
        if (version >= 1)
            exec_sql(db, "create index inode_to_path on paths (inode, path);");
    } else {
        exec_sql(db, "create table paths (path blob primary key, inode integer references stats(inode));"
                     "create index inode_to_path on paths (inode, path);");
        if (version == 2) {
            exec_sql(db, "create trigger delete_path after delete on paths "
                         "when not exists (select 1 from paths where inode = old.inode) "
                         "begin delete from stats where not exists "
                         "(select 1 from paths where inode = old.inode) and inode = old.inode; end;");
        }
    }
    char pragma[64];
    snprintf(pragma, sizeof(pragma), "pragma user_version = %d", version);
    exec_sql(db, pragma);
}

// The corpus: two hardlinked paths, a nested path, a stat with no path, and a
// path with no stat (both are filtered by migration 2).
struct row {
    const char *path;
    long long inode;
    const char *stat; // NULL means "no stats row"
};

static const struct row CORPUS[] = {
    {"/a", 1, "a4810000e80300006400000000000000"},
    {"/b", 1, "a4810000e80300006400000000000000"},
    {"/c/d", 2, "41ed0000e80300006400000000000000"},
    {"/dangling", 9, NULL},
};
static const long long ORPHAN_STAT_INODE = 3;
static const char *ORPHAN_STAT = "07000000e80300006400000000000000";

// True when an earlier corpus row already supplied this stat.
static bool stat_seen_before(size_t index) {
    for (size_t j = 0; j < index; j++)
        if (CORPUS[j].stat != NULL && CORPUS[j].inode == CORPUS[index].inode)
            return true;
    return false;
}

static struct fakefs_db *open_case(int version) {
    sqlite3 *raw;
    check(sqlite3_open(":memory:", &raw), raw);
    create_schema(raw, version);
    // Rows go in with prepared statements so byte paths stay byte-exact.
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(raw, "insert into paths (path, inode) values (?, ?)", -1, &stmt, NULL), raw);
    for (size_t i = 0; i < sizeof(CORPUS) / sizeof(CORPUS[0]); i++) {
        sqlite3_bind_blob(stmt, 1, CORPUS[i].path, (int) strlen(CORPUS[i].path), SQLITE_TRANSIENT);
        sqlite3_bind_int64(stmt, 2, CORPUS[i].inode);
        check(sqlite3_step(stmt), raw);
        check(sqlite3_reset(stmt), raw);
    }
    check(sqlite3_finalize(stmt), raw);

    check(sqlite3_prepare_v2(raw, "insert into stats (inode, stat) values (?, ?)", -1, &stmt, NULL), raw);
    uint8_t stat_bytes[16];
    for (size_t i = 0; i < sizeof(CORPUS) / sizeof(CORPUS[0]); i++) {
        if (CORPUS[i].stat == NULL || stat_seen_before(i))
            continue;
        unhex(CORPUS[i].stat, stat_bytes, sizeof(stat_bytes));
        sqlite3_bind_int64(stmt, 1, CORPUS[i].inode);
        sqlite3_bind_blob(stmt, 2, stat_bytes, sizeof(stat_bytes), SQLITE_TRANSIENT);
        check(sqlite3_step(stmt), raw);
        check(sqlite3_reset(stmt), raw);
    }
    unhex(ORPHAN_STAT, stat_bytes, sizeof(stat_bytes));
    sqlite3_bind_int64(stmt, 1, ORPHAN_STAT_INODE);
    sqlite3_bind_blob(stmt, 2, stat_bytes, sizeof(stat_bytes), SQLITE_TRANSIENT);
    check(sqlite3_step(stmt), raw);
    check(sqlite3_reset(stmt), raw);
    check(sqlite3_finalize(stmt), raw);

    struct fakefs_db *fs = calloc(1, sizeof(struct fakefs_db));
    fs->db = raw;
    return fs;
}

// Emit the corpus itself so the Rust side can build the same starting
// databases: `S inode stat` for each stat row, then `P path inode` for each
// path mapping in insertion order.
static void print_corpus(void) {
    printf("S %llx %s\n", ORPHAN_STAT_INODE, ORPHAN_STAT);
    for (size_t i = 0; i < sizeof(CORPUS) / sizeof(CORPUS[0]); i++) {
        if (CORPUS[i].stat != NULL && !stat_seen_before(i))
            printf("S %llx %s\n", CORPUS[i].inode, CORPUS[i].stat);
    }
    for (size_t i = 0; i < sizeof(CORPUS) / sizeof(CORPUS[0]); i++) {
        printf("P ");
        emit_hex(CORPUS[i].path, strlen(CORPUS[i].path));
        printf(" %llx\n", CORPUS[i].inode);
    }
}

// Delete `/c/d`, the only path of its inode. With the version 2 trigger
// present the stats row goes with it; without the trigger the stat survives.
#define PROBE_PATH "X'2f632f64'"

static uint64_t delete_probe(struct fakefs_db *fs) {
    exec_sql(fs->db, "delete from paths where path = " PROBE_PATH);
    return database_hash(fs->db);
}

int main(void) {
    // The migration SQL itself, straight from the compiled C table.
    for (size_t i = 0; i < sizeof(migrations) / sizeof(migrations[0]); i++) {
        printf("M %zu ", i);
        if (migrations[i].sql == NULL)
            printf("-");
        else
            emit_hex(migrations[i].sql, strlen(migrations[i].sql));
        printf(" %s\n", migrations[i].migrate == NULL ? "none" : "hook");
    }
    print_corpus();

    const int starts[] = {0, 1, 2, 3, 5};
    for (size_t i = 0; i < sizeof(starts) / sizeof(starts[0]); i++) {
        int start = starts[i];
        printf("C %d\n", start);

        // First copy: record the schema it starts from and whether the trigger
        // (if this generation has one) cascades a delete into stats.
        struct fakefs_db *before = open_case(start);
        printf("B %d\n", user_version(before->db));
        print_objects(before->db, "BO");
        printf("T0 %016llx\n", (unsigned long long) delete_probe(before));
        sqlite3 *before_raw = before->db;
        free(before);
        check(sqlite3_close(before_raw), before_raw);

        // Second copy: run the real ladder.
        struct fakefs_db *fs = open_case(start);
        int err = fakefs_migrate(fs, -1);
        assert(err == 0);
        printf("V %d\n", user_version(fs->db));
        printf("H %016llx\n", (unsigned long long) database_hash(fs->db));
        print_objects(fs->db, "O");
        print_foreign_keys(fs->db);
        // The migrated database must behave like the trigger-free version 3:
        // deleting the same path no longer cascades into stats.
        printf("T1 %016llx\n", (unsigned long long) delete_probe(fs));
        sqlite3 *raw = fs->db;
        free(fs);
        check(sqlite3_close(raw), raw);
    }
    return 0;
}
