// Reference-output generator for fake_db_init host-inode integration.
//
// It links untouched fs/fake-db.c, fs/fake-migrate.c, and fs/fake-rebuild.c to
// the platform SQLite runtime. The included rebuild harness supplies a
// deterministic fstatat/unlinkat/linkat host. This is a local C oracle only;
// production Rust uses GraphiteSQL.
#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

#include <assert.h>
#include <fcntl.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>

// Reuse the exact deterministic host, metadata seed, and FNV hash functions
// from the standalone fake-rebuild C oracle. Rename its entry point so this
// harness can invoke upstream fake_db_init instead.
#define main fake_rebuild_dump_unused_main
#include "fake-rebuild-dump.c"
#undef main

static int64_t scalar_i64(sqlite3 *db, const char *sql) {
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(db, sql, -1, &stmt, NULL), db);
    assert(sqlite3_step(stmt) == SQLITE_ROW);
    int64_t value = sqlite3_column_int64(stmt, 0);
    check(sqlite3_finalize(stmt), db);
    return value;
}

static void remove_database_files(const char *path) {
    char auxiliary_path[512];
    assert(strlen(path) + strlen("-journal") < sizeof(auxiliary_path));
    unlink(path);
    for (const char *suffix = "-journal"; suffix != NULL;
         suffix = strcmp(suffix, "-journal") == 0 ? "-wal" : NULL) {
        snprintf(auxiliary_path, sizeof(auxiliary_path), "%s%s", path, suffix);
        unlink(auxiliary_path);
    }
}

int main(void) {
    char database_path[] = "/tmp/ish-fake-db-init-XXXXXX";
    int descriptor = mkstemp(database_path);
    assert(descriptor >= 0);
    assert(close(descriptor) == 0);

    sqlite3 *seed;
    check(sqlite3_open_v2(database_path, &seed,
                          SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE, NULL),
          seed);
    exec_sql(seed, "pragma foreign_keys=off");
    exec_sql(seed,
        "create table meta (id integer unique default 0, db_inode integer);"
        "insert into meta (db_inode) values (0);"
        "create table stats (inode integer primary key, stat blob);"
        "create table paths (path blob primary key, inode integer references stats(inode));"
        "create index inode_to_path on paths (inode, path);"
        "insert into stats (inode, stat) values (9, x'09000000010000000200000003000000');"
        "insert into stats (inode, stat) values (5, x'05000000060000000700000008000000');"
        "insert into stats (inode, stat) values (7, x'0700000008000000090000000a000000');"
        "insert into paths (path, inode) values (x'', 9);"
        "insert into paths (path, inode) values (x'2f61', 5);"
        "insert into paths (path, inode) values (x'2f62', 5);"
        "insert into paths (path, inode) values (x'2f62726f6b656e', 5);"
        "insert into paths (path, inode) values (x'2f63', 6);"
        "insert into paths (path, inode) values (x'2f64', 6);"
        "insert into paths (path, inode) values (x'2f6d697373696e67', 7);"
        "pragma user_version=3;");
    check(sqlite3_close(seed), seed);

    struct fakefs_db fs = {0};
    assert(fake_db_init(&fs, database_path, 17) == 0);
    unsigned first_stat_calls = stat_calls;
    unsigned first_unlink_calls = unlink_calls;
    unsigned first_link_calls = link_calls;

    // A fresh fake_db_init on the same on-disk database must take C's
    // meta-inode match path and must not ask the host adapter to rebuild.
    assert(fake_db_deinit(&fs) == SQLITE_OK);
    memset(&fs, 0, sizeof(fs));
    assert(fake_db_init(&fs, database_path, 17) == 0);
    unsigned unchanged_stat_calls = stat_calls - first_stat_calls;
    unsigned unchanged_unlink_calls = unlink_calls - first_unlink_calls;
    unsigned unchanged_link_calls = link_calls - first_link_calls;

    struct stat statbuf;
    assert(stat(database_path, &statbuf) == 0);
    uint64_t stored_inode = (uint64_t) scalar_i64(fs.db, "select db_inode from meta");

    puts("# fake-db-init reference, generated from unmodified iSH fake-db/migrate/rebuild C");
    puts("# D initialized logical stats-and-paths hash");
    puts("# H initialized mock-host hash");
    puts("# C mismatched-inode fstatat unlinkat linkat call counts");
    puts("# N matching-inode fstatat unlinkat linkat call counts");
    puts("# M meta.db_inode equals the actual database file inode");
    printf("D %016llx\n", (unsigned long long) database_hash(fs.db));
    printf("H %016llx\n", (unsigned long long) host_hash());
    printf("C %u %u %u\n", first_stat_calls, first_unlink_calls, first_link_calls);
    printf("N %u %u %u\n", unchanged_stat_calls, unchanged_unlink_calls, unchanged_link_calls);
    printf("M %u\n", stored_inode == (uint64_t) statbuf.st_ino);

    assert(fake_db_deinit(&fs) == SQLITE_OK);
    remove_database_files(database_path);
    return 0;
}
