// Reference-output generator for the fakefs rebuild differential test.
//
// This links the unmodified fs/fake-rebuild.c and drives it through a fully
// scripted host filesystem, so the comparison does not depend on the inode
// numbers of the machine running the generator:
//
//   * `fstatat`, `unlinkat` and `linkat` are replaced with linker `--wrap`
//     implementations that answer from a fixed path-to-inode table and record
//     the operations the rebuild performs, in order;
//   * the metadata database is built by the same API calls the Rust side uses,
//     so both sides start from identical `stats`/`paths` tables.
//
// The corpus contains a hardlink pair, a triple, a nested path, a path the
// host no longer has, and an ordinary path. Every record the harness prints is
// replayed by tests/fake_db_rebuild_differential.rs.
//
// Build (normally via tools/gen_fake_db_rebuild_reference.sh):
//   cc -O2 -Wall -Wextra -I<ish-src> -Itools/stub-include
//      -o fake-db-rebuild-dump tools/fake-db-rebuild-dump.c
//      <ish-src>/fs/fake-rebuild.c -Wl,-l:libsqlite3.so.0
//      -Wl,--wrap=fstatat -Wl,--wrap=unlinkat -Wl,--wrap=linkat

#define _GNU_SOURCE

#include <errno.h>
#include <fcntl.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>

#include "fs/fake-db.h"

int fakefs_rebuild(struct fakefs_db *fs, int root_fd);

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

static void emit_hex(const void *data, size_t count) {
    const uint8_t *bytes = data;
    for (size_t i = 0; i < count; i++)
        printf("%02x", bytes[i]);
}

// ---- the scripted host filesystem -----------------------------------------

// Paths are stored in the repaired form the rebuild passes to the host
// (`fix_path` has already dropped the leading slash).
struct host_file {
    const char *path;
    unsigned long long inode;
};

static const struct host_file HOST[] = {
    {"a", 501},
    {"b", 502},
    {"c", 503},
    {"c/d", 504},
    // "missing" is deliberately absent.
    {"e", 505},
    {"f", 506},
    {"g", 507},
};

static const struct host_file *host_lookup(const char *path) {
    for (size_t i = 0; i < sizeof(HOST) / sizeof(HOST[0]); i++)
        if (strcmp(HOST[i].path, path) == 0)
            return &HOST[i];
    return NULL;
}

struct host_op {
    const char *kind;
    char *source;
    char *destination;
};

static struct host_op OPS[64];
static size_t op_count;

static void record_op(const char *kind, const char *source, const char *destination) {
    if (op_count >= sizeof(OPS) / sizeof(OPS[0])) {
        fprintf(stderr, "too many host operations\n");
        abort();
    }
    OPS[op_count].kind = kind;
    OPS[op_count].source = source == NULL ? NULL : strdup(source);
    OPS[op_count].destination = destination == NULL ? NULL : strdup(destination);
    op_count++;
}

int __wrap_fstatat(int dirfd, const char *path, struct stat *buf, int flags) {
    (void) dirfd;
    (void) flags;
    const struct host_file *file = host_lookup(path);
    if (file == NULL) {
        errno = ENOENT;
        return -1;
    }
    memset(buf, 0, sizeof(*buf));
    buf->st_ino = file->inode;
    return 0;
}

int __wrap_unlinkat(int dirfd, const char *path, int flags) {
    (void) dirfd;
    (void) flags;
    record_op("unlink", path, NULL);
    return 0;
}

int __wrap_linkat(int olddirfd, const char *oldpath, int newdirfd, const char *newpath, int flags) {
    (void) olddirfd;
    (void) newdirfd;
    (void) flags;
    record_op("link", oldpath, newpath);
    return 0;
}

// ---- output helpers --------------------------------------------------------

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

static void print_objects(sqlite3 *db) {
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(db, "select type, name from sqlite_master order by type, name",
                             -1, &stmt, NULL), db);
    printf("O");
    while (sqlite3_step(stmt) == SQLITE_ROW)
        printf(" %s:%s", sqlite3_column_text(stmt, 0), sqlite3_column_text(stmt, 1));
    putchar('\n');
    check(sqlite3_finalize(stmt), db);
}

// ---- corpus ----------------------------------------------------------------

// Inserted in this order, exactly as fake-db.c's `path_create` and `path_link`
// do: a fresh path takes a new rowid, a link reuses an existing inode.
struct path_op {
    const char *kind; // "create" or "link"
    const char *path;
    const char *alias;
};

static const struct path_op PATHS[] = {
    {"create", "/a", NULL},
    {"link", "/a", "/b"},
    {"create", "/c", NULL},
    {"create", "/c/d", NULL},
    {"create", "/missing", NULL},
    {"create", "/e", NULL},
    {"link", "/e", "/f"},
    {"link", "/e", "/g"},
};

// `struct ish_stat` blobs, as hex text; decoded before binding.
static const char *STAT_HEX[] = {
    "a4810000e80300006400000000000000", // 0100644
    "41ed0000e80300006400000000000000", // 0040755
    "a4810000e80300006400000000000000",
};

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

// `create` = insert a stat row then map the path to `last_insert_rowid()`;
// `link` = map the alias to the existing inode.
static void build_corpus(sqlite3 *db) {
    sqlite3_stmt *create_stat, *create_path, *link_path, *get_inode;
    check(sqlite3_prepare_v2(db, "insert into stats (stat) values (?)", -1, &create_stat, NULL), db);
    check(sqlite3_prepare_v2(db, "insert or replace into paths values (?, last_insert_rowid())",
                             -1, &create_path, NULL), db);
    check(sqlite3_prepare_v2(db, "insert or replace into paths (path, inode) values (?, ?)",
                             -1, &link_path, NULL), db);
    check(sqlite3_prepare_v2(db, "select inode from paths where path = ?", -1, &get_inode, NULL), db);

    uint8_t stat_bytes[16];
    size_t stat_index = 0;
    for (size_t i = 0; i < sizeof(PATHS) / sizeof(PATHS[0]); i++) {
        const char *path = PATHS[i].kind[0] == 'c' ? PATHS[i].path : PATHS[i].alias;
        if (strcmp(PATHS[i].kind, "create") == 0) {
            unhex(STAT_HEX[stat_index % (sizeof(STAT_HEX) / sizeof(STAT_HEX[0]))], stat_bytes,
                  sizeof(stat_bytes));
            stat_index++;
            check(sqlite3_bind_blob(create_stat, 1, stat_bytes, sizeof(stat_bytes), SQLITE_TRANSIENT), db);
            check(sqlite3_step(create_stat), db);
            check(sqlite3_reset(create_stat), db);
            check(sqlite3_bind_blob(create_path, 1, path, (int) strlen(path), SQLITE_TRANSIENT), db);
            check(sqlite3_step(create_path), db);
            check(sqlite3_reset(create_path), db);
        } else {
            check(sqlite3_bind_blob(get_inode, 1, PATHS[i].path, (int) strlen(PATHS[i].path),
                                    SQLITE_TRANSIENT), db);
            if (sqlite3_step(get_inode) != SQLITE_ROW)
                die("corpus link source is missing");
            int64_t inode = sqlite3_column_int64(get_inode, 0);
            check(sqlite3_reset(get_inode), db);
            check(sqlite3_bind_blob(link_path, 1, path, (int) strlen(path), SQLITE_TRANSIENT), db);
            check(sqlite3_bind_int64(link_path, 2, inode), db);
            check(sqlite3_step(link_path), db);
            check(sqlite3_reset(link_path), db);
        }
    }
    check(sqlite3_finalize(create_stat), db);
    check(sqlite3_finalize(create_path), db);
    check(sqlite3_finalize(link_path), db);
    check(sqlite3_finalize(get_inode), db);
}

// Re-emit the corpus so the Rust side can build the same tables with the same
// public API calls.
static void print_corpus(sqlite3 *db) {
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(db, "select inode, stat from stats order by inode", -1, &stmt, NULL), db);
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        printf("S %llx ", (unsigned long long) sqlite3_column_int64(stmt, 0));
        emit_hex(sqlite3_column_blob(stmt, 1), (size_t) sqlite3_column_bytes(stmt, 1));
        putchar('\n');
    }
    check(sqlite3_finalize(stmt), db);
    check(sqlite3_prepare_v2(db, "select path, inode from paths order by rowid", -1, &stmt, NULL), db);
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        printf("P ");
        emit_hex(sqlite3_column_blob(stmt, 0), (size_t) sqlite3_column_bytes(stmt, 0));
        printf(" %llx\n", (unsigned long long) sqlite3_column_int64(stmt, 1));
    }
    check(sqlite3_finalize(stmt), db);
    for (size_t i = 0; i < sizeof(PATHS) / sizeof(PATHS[0]); i++) {
        printf("W %s ", PATHS[i].kind);
        if (strcmp(PATHS[i].kind, "create") == 0)
            emit_hex(PATHS[i].path, strlen(PATHS[i].path));
        else {
            emit_hex(PATHS[i].path, strlen(PATHS[i].path));
            putchar(' ');
            emit_hex(PATHS[i].alias, strlen(PATHS[i].alias));
        }
        putchar('\n');
    }
}

int main(void) {
    sqlite3 *db;
    check(sqlite3_open(":memory:", &db), db);
    exec_sql(db, "pragma foreign_keys=true;"
                 "create table meta (id integer unique default 0, db_inode integer);"
                 "insert into meta (db_inode) values (0);"
                 "create table stats (inode integer primary key, stat blob);"
                 "create table paths (path blob primary key, inode integer references stats(inode));"
                 "create index inode_to_path on paths (inode, path);"
                 "pragma user_version = 3;");
    build_corpus(db);
    print_corpus(db);

    // The scripted host filesystem, as the Rust host adapter receives it.
    for (size_t i = 0; i < sizeof(HOST) / sizeof(HOST[0]); i++)
        printf("F %s %llx\n", HOST[i].path, HOST[i].inode);

    printf("B %016llx\n", (unsigned long long) database_hash(db));

    struct fakefs_db fs = {.db = db};
    int err = fakefs_rebuild(&fs, -1);
    if (err != 0)
        die("fakefs_rebuild returned %d", err);

    for (size_t i = 0; i < op_count; i++) {
        printf("L %s %s", OPS[i].kind, OPS[i].source);
        if (OPS[i].destination != NULL)
            printf(" %s", OPS[i].destination);
        putchar('\n');
    }
    printf("A %016llx\n", (unsigned long long) database_hash(db));
    print_objects(db);

    check(sqlite3_close(db), db);
    return 0;
}
