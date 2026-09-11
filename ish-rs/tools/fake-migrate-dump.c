// Reference-output generator for the fakefs schema-migration Rust port.
//
// It seeds historical metadata schemas, links untouched fs/fake-migrate.c to
// the platform SQLite runtime, and emits logical state plus durable schema
// facts. It is a local C oracle only; production Rust code uses GraphiteSQL.
#define _GNU_SOURCE

#include <assert.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#include "fs/fake-db.h"

int fakefs_migrate(struct fakefs_db *fs, int root_fd);

void ish_printk(const char *UNUSED(message), ...) {}
_Noreturn void die(const char *message, ...) {
    va_list args;
    va_start(args, message);
    vfprintf(stderr, message, args);
    va_end(args);
    fputc('\n', stderr);
    abort();
}

static void check(int status, sqlite3 *db) {
    if (status != SQLITE_OK && status != SQLITE_ROW && status != SQLITE_DONE) {
        fprintf(stderr, "sqlite failure %d: %s\n", status, sqlite3_errmsg(db));
        abort();
    }
}

static void exec_sql(sqlite3 *db, const char *sql) {
    check(sqlite3_exec(db, sql, NULL, NULL, NULL), db);
}

static void hash_byte(uint64_t *hash, uint8_t byte) {
    *hash ^= byte;
    *hash *= UINT64_C(0x100000001b3);
}

static void hash_bytes(uint64_t *hash, const void *data, size_t count) {
    const uint8_t *bytes = data;
    for (size_t i = 0; i < count; i++)
        hash_byte(hash, bytes[i]);
}

static void hash_u32(uint64_t *hash, uint32_t value) {
    for (unsigned i = 0; i < 4; i++)
        hash_byte(hash, value >> (i * 8));
}

static void hash_u64(uint64_t *hash, uint64_t value) {
    for (unsigned i = 0; i < 8; i++)
        hash_byte(hash, value >> (i * 8));
}

static uint64_t database_hash(sqlite3 *db) {
    uint64_t hash = UINT64_C(0xcbf29ce484222325);
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(db, "select inode, stat from stats order by inode", -1, &stmt, NULL), db);
    hash_bytes(&hash, "stats", 5);
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        hash_u64(&hash, (uint64_t) sqlite3_column_int64(stmt, 0));
        const void *stat = sqlite3_column_blob(stmt, 1);
        int count = sqlite3_column_bytes(stmt, 1);
        assert(count >= 0);
        hash_u32(&hash, (uint32_t) count);
        hash_bytes(&hash, stat, (size_t) count);
    }
    check(sqlite3_finalize(stmt), db);

    check(sqlite3_prepare_v2(db, "select path, inode from paths order by path", -1, &stmt, NULL), db);
    hash_bytes(&hash, "paths", 5);
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        const void *path = sqlite3_column_blob(stmt, 0);
        int count = sqlite3_column_bytes(stmt, 0);
        assert(count >= 0);
        hash_u32(&hash, (uint32_t) count);
        hash_bytes(&hash, path, (size_t) count);
        hash_u64(&hash, (uint64_t) sqlite3_column_int64(stmt, 1));
    }
    check(sqlite3_finalize(stmt), db);
    return hash;
}

static int scalar(sqlite3 *db, const char *sql) {
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(db, sql, -1, &stmt, NULL), db);
    assert(sqlite3_step(stmt) == SQLITE_ROW);
    int value = (int) sqlite3_column_int64(stmt, 0);
    check(sqlite3_finalize(stmt), db);
    return value;
}

static void setup_legacy(sqlite3 *db, int version) {
    exec_sql(db,
        "create table meta (id integer unique default 0, db_inode integer);"
        "insert into meta (db_inode) values (0);"
        "create table stats (inode integer primary key, stat blob);");
    if (version < 2) {
        exec_sql(db,
            "create table paths (path blob primary key, inode integer);"
            "insert into stats (inode, stat) values (5, x'a4810000e80300006400000000000000');"
            "insert into stats (inode, stat) values (6, x'ed410000d0070000c800000009000000');"
            "insert into paths (path, inode) values (x'2f6b657074', 5);"
            "insert into paths (path, inode) values (x'2f64616e676c696e67', 99);");
    } else {
        exec_sql(db,
            "create table paths (path blob primary key, inode integer references stats(inode));"
            "create index inode_to_path on paths (inode, path);"
            "insert into stats (inode, stat) values (12, x'ed410000d0070000c800000009000000');"
            "insert into paths (path, inode) values (x'2f7632', 12);"
            "create trigger delete_path after delete on paths "
            "when not exists (select 1 from paths where inode = old.inode) "
            "begin delete from stats where not exists (select 1 from paths where inode = old.inode) and inode = old.inode; end;");
    }
    char pragma[64];
    snprintf(pragma, sizeof(pragma), "pragma user_version=%d", version);
    exec_sql(db, pragma);
}

int main(void) {
    puts("# fake-migrate reference, generated from unmodified iSH fs/fake-migrate.c");
    puts("# M legacy-version resulting-user-version logical-table-hash inode-index-count delete-trigger-count");
    puts("# D version-2 logical-table-hash-after-unlink");
    for (int version = 0; version <= 2; version++) {
        sqlite3 *db;
        check(sqlite3_open_v2(":memory:", &db, SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE, NULL), db);
        exec_sql(db, "pragma foreign_keys=true");
        setup_legacy(db, version);
        struct fakefs_db fs = {.db = db};
        assert(fakefs_migrate(&fs, -1) == 0);
        int current = scalar(db, "pragma user_version");
        int index_count = scalar(db,
            "select count(*) from sqlite_master where type = 'index' and name = 'inode_to_path'");
        int trigger_count = scalar(db,
            "select count(*) from sqlite_master where type = 'trigger' and name = 'delete_path'");
        printf("M %d %d %016llx %d %d\n", version, current,
               (unsigned long long) database_hash(db), index_count, trigger_count);
        if (version == 2) {
            exec_sql(db, "delete from paths where path = x'2f7632'");
            printf("D 2 %016llx\n", (unsigned long long) database_hash(db));
        }
        check(sqlite3_close(db), db);
    }
    return 0;
}
