// Reference-output generator for the fakefs host-inode rebuild Rust port.
//
// It links untouched fs/fake-rebuild.c to the platform SQLite runtime and
// substitutes deterministic fstatat/unlinkat/linkat operations. It is a local
// C oracle only; production Rust code uses GraphiteSQL.
#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

#include <assert.h>
#include <errno.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>

#include "fs/fake-db.h"

int fakefs_rebuild(struct fakefs_db *fs, int root_fd);

void ish_printk(const char *UNUSED(message), ...) {}
_Noreturn void die(const char *message, ...) {
    va_list args;
    va_start(args, message);
    vfprintf(stderr, message, args);
    va_end(args);
    fputc('\n', stderr);
    abort();
}

struct host_file {
    const char *path;
    bool exists;
    uint64_t inode;
    bool unlink_fails;
};

static struct host_file files[] = {
    {".", true, 900, false},
    {"a", true, 101, false},
    {"b", true, 202, false},
    {"broken", true, 404, true},
    {"c", true, 303, false},
    {"d", true, 606, false},
    {"missing", false, 0, false},
};
static unsigned stat_calls;
static unsigned unlink_calls;
static unsigned link_calls;

static struct host_file *find_file(const char *path) {
    for (unsigned i = 0; i < sizeof(files) / sizeof(files[0]); i++)
        if (strcmp(files[i].path, path) == 0)
            return &files[i];
    return NULL;
}

int fstatat(int root_fd, const char *path, struct stat *statbuf, int flags) {
    assert(root_fd == 17);
    assert(flags == 0);
    stat_calls++;
    struct host_file *file = find_file(path);
    if (file == NULL || !file->exists) {
        errno = ENOENT;
        return -1;
    }
    memset(statbuf, 0, sizeof(*statbuf));
    statbuf->st_ino = (ino_t) file->inode;
    return 0;
}

int unlinkat(int root_fd, const char *path, int flags) {
    assert(root_fd == 17);
    assert(flags == 0);
    unlink_calls++;
    struct host_file *file = find_file(path);
    if (file == NULL || !file->exists || file->unlink_fails) {
        errno = ENOENT;
        return -1;
    }
    file->exists = false;
    return 0;
}

int linkat(int old_root_fd, const char *old_path, int new_root_fd, const char *new_path, int flags) {
    assert(old_root_fd == 17);
    assert(new_root_fd == 17);
    assert(flags == 0);
    link_calls++;
    struct host_file *old_file = find_file(old_path);
    struct host_file *new_file = find_file(new_path);
    if (old_file == NULL || !old_file->exists || new_file == NULL || new_file->exists) {
        errno = EEXIST;
        return -1;
    }
    new_file->exists = true;
    new_file->inode = old_file->inode;
    return 0;
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

static uint64_t host_hash(void) {
    uint64_t hash = UINT64_C(0xcbf29ce484222325);
    for (unsigned i = 0; i < sizeof(files) / sizeof(files[0]); i++) {
        struct host_file *file = &files[i];
        hash_bytes(&hash, file->path, strlen(file->path));
        hash_byte(&hash, 0);
        hash_byte(&hash, file->exists);
        hash_u64(&hash, file->inode);
    }
    return hash;
}

static int scalar(sqlite3 *db, const char *sql) {
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(db, sql, -1, &stmt, NULL), db);
    assert(sqlite3_step(stmt) == SQLITE_ROW);
    int value = sqlite3_column_int(stmt, 0);
    check(sqlite3_finalize(stmt), db);
    return value;
}

int main(void) {
    sqlite3 *db;
    check(sqlite3_open_v2(":memory:", &db, SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE, NULL), db);
    exec_sql(db, "pragma foreign_keys=off");
    exec_sql(db,
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
        "insert into paths (path, inode) values (x'2f6d697373696e67', 7);");
    exec_sql(db, "pragma foreign_keys=on");

    struct fakefs_db fs = {.db = db};
    assert(fakefs_rebuild(&fs, 17) == 0);
    puts("# fake-rebuild reference, generated from unmodified iSH fs/fake-rebuild.c");
    puts("# D rebuilt logical stats-and-paths hash");
    puts("# H rebuilt mock-host hash");
    puts("# C fstatat unlinkat linkat call counts");
    puts("# T remaining paths_old stats_old table counts");
    printf("D %016llx\n", (unsigned long long) database_hash(db));
    printf("H %016llx\n", (unsigned long long) host_hash());
    printf("C %u %u %u\n", stat_calls, unlink_calls, link_calls);
    printf("T %d %d\n",
           scalar(db, "select count(*) from sqlite_master where type = 'table' and name = 'paths_old'"),
           scalar(db, "select count(*) from sqlite_master where type = 'table' and name = 'stats_old'"));
    check(sqlite3_close(db), db);
    return 0;
}
