// Reference-output generator for the fake filesystem metadata database port.
//
// It links unmodified fs/fake-db.c against the platform SQLite library. The
// real import/migration/rebuild layers are outside this narrow metadata API, so
// the harness creates the current schema first and supplies no-op migration and
// rebuild hooks. Every metadata primitive then runs through the original C
// prepared statements, transactions, and change_prefix SQLite function.

#define _GNU_SOURCE

#include <assert.h>
#include <fcntl.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include "fs/fake-db.h"

static struct fakefs_db fs;

int fakefs_migrate(struct fakefs_db *UNUSED(fake), int UNUSED(root_fd)) { return 0; }
int fakefs_rebuild(struct fakefs_db *UNUSED(fake), int UNUSED(root_fd)) { return 0; }
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

static void emit_bytes(const void *data, size_t count) {
    const uint8_t *bytes = data;
    for (size_t i = 0; i < count; i++)
        printf("%02x", bytes[i]);
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

// Hash the logical tables, rather than SQLite pages or pointer identities.
static uint64_t database_hash(void) {
    uint64_t hash = UINT64_C(0xcbf29ce484222325);
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(fs.db, "select inode, stat from stats order by inode", -1,
                             &stmt, NULL), fs.db);
    hash_bytes(&hash, "stats", 5);
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        hash_u64(&hash, (uint64_t) sqlite3_column_int64(stmt, 0));
        const void *stat = sqlite3_column_blob(stmt, 1);
        int count = sqlite3_column_bytes(stmt, 1);
        assert(count >= 0);
        hash_u32(&hash, (uint32_t) count);
        hash_bytes(&hash, stat, (size_t) count);
    }
    check(sqlite3_finalize(stmt), fs.db);

    check(sqlite3_prepare_v2(fs.db, "select path, inode from paths order by path", -1,
                             &stmt, NULL), fs.db);
    hash_bytes(&hash, "paths", 5);
    while (sqlite3_step(stmt) == SQLITE_ROW) {
        const void *path = sqlite3_column_blob(stmt, 0);
        int count = sqlite3_column_bytes(stmt, 0);
        assert(count >= 0);
        hash_u32(&hash, (uint32_t) count);
        hash_bytes(&hash, path, (size_t) count);
        hash_u64(&hash, (uint64_t) sqlite3_column_int64(stmt, 1));
    }
    check(sqlite3_finalize(stmt), fs.db);
    return hash;
}

static void snapshot(void) {
    printf("S %016llx\n", (unsigned long long) database_hash());
}

#define OP(...) \
    do { \
        printf("O "); \
        printf(__VA_ARGS__); \
        putchar('\n'); \
    } while (0)

static void record_begin(bool write) {
    OP("B %u", write);
    if (write)
        db_begin_write(&fs);
    else
        db_begin_read(&fs);
    puts("R 0000000000000000");
    snapshot();
}

static void record_finish(bool commit) {
    OP("F %u", commit);
    if (commit)
        db_commit(&fs);
    else
        db_rollback(&fs);
    puts("R 0000000000000000");
    snapshot();
}

static void record_create(const char *path, struct ish_stat stat) {
    printf("O C ");
    emit_bytes(path, strlen(path));
    printf(" %08x %08x %08x %08x\n", stat.mode, stat.uid, stat.gid, stat.rdev);
    printf("R %016llx\n", (unsigned long long) path_create(&fs, path, &stat));
    snapshot();
}

static void record_get(const char *path) {
    printf("O G ");
    emit_bytes(path, strlen(path));
    putchar('\n');
    printf("R %016llx\n", (unsigned long long) path_get_inode(&fs, path));
    snapshot();
}

static uint64_t paths_for_inode_hash(inode_t inode) {
    uint64_t hash = UINT64_C(0xcbf29ce484222325);
    sqlite3_stmt *stmt = fs.stmt.path_from_inode;
    check(sqlite3_bind_int64(stmt, 1, (int64_t) inode), fs.db);
    while (db_exec(&fs, stmt)) {
        const void *path = sqlite3_column_blob(stmt, 0);
        int count = sqlite3_column_bytes(stmt, 0);
        assert(count >= 0);
        hash_u32(&hash, (uint32_t) count);
        hash_bytes(&hash, path, (size_t) count);
    }
    db_reset(&fs, stmt);
    return hash;
}

static void record_paths_for_inode(inode_t inode) {
    OP("Q %016llx", (unsigned long long) inode);
    printf("R %016llx\n", (unsigned long long) paths_for_inode_hash(inode));
    snapshot();
}

static void emit_read(bool exists, inode_t inode, struct ish_stat stat) {
    printf("V %u %016llx %08x %08x %08x %08x\n", exists,
           (unsigned long long) inode, stat.mode, stat.uid, stat.gid, stat.rdev);
}

static void record_path_read(const char *path) {
    printf("O P ");
    emit_bytes(path, strlen(path));
    putchar('\n');
    struct ish_stat stat = {0};
    inode_t inode = 0;
    bool exists = path_read_stat(&fs, path, &stat, &inode);
    puts("R 0000000000000000");
    emit_read(exists, inode, stat);
    snapshot();
}

static void record_inode_read(inode_t wanted) {
    OP("I %016llx", (unsigned long long) wanted);
    struct ish_stat stat = {0};
    bool exists = inode_read_stat_if_exist(&fs, wanted, &stat);
    puts("R 0000000000000000");
    emit_read(exists, wanted, stat);
    snapshot();
}

static void record_inode_write(inode_t inode, struct ish_stat stat) {
    OP("W %016llx %08x %08x %08x %08x", (unsigned long long) inode, stat.mode,
       stat.uid, stat.gid, stat.rdev);
    inode_write_stat(&fs, inode, &stat);
    puts("R 0000000000000000");
    snapshot();
}

static void record_link(const char *src, const char *dst) {
    printf("O L ");
    emit_bytes(src, strlen(src));
    putchar(' ');
    emit_bytes(dst, strlen(dst));
    putchar('\n');
    path_link(&fs, src, dst);
    puts("R 0000000000000000");
    snapshot();
}

static void record_unlink(const char *path) {
    printf("O U ");
    emit_bytes(path, strlen(path));
    putchar('\n');
    printf("R %016llx\n", (unsigned long long) path_unlink(&fs, path));
    snapshot();
}

static void record_rename(const char *src, const char *dst) {
    printf("O N ");
    emit_bytes(src, strlen(src));
    putchar(' ');
    emit_bytes(dst, strlen(dst));
    putchar('\n');
    path_rename(&fs, src, dst);
    puts("R 0000000000000000");
    snapshot();
}

static void record_cleanup(inode_t inode) {
    OP("X %016llx", (unsigned long long) inode);
    check(sqlite3_bind_int64(fs.stmt.try_cleanup_inode, 1, (int64_t) inode), fs.db);
    db_exec_reset(&fs, fs.stmt.try_cleanup_inode);
    puts("R 0000000000000000");
    snapshot();
}

// This is the orphan sweep in fake_db_init, run after the corpus has created
// an orphan through INSERT OR REPLACE so it has observable behavior.
static void record_clear_orphans(void) {
    OP("A");
    sqlite3_stmt *stmt;
    check(sqlite3_prepare_v2(fs.db,
                             "delete from stats where not exists "
                             "(select 1 from paths where inode = stats.inode)",
                             -1, &stmt, NULL),
          fs.db);
    db_exec_reset(&fs, stmt);
    check(sqlite3_finalize(stmt), fs.db);
    puts("R 0000000000000000");
    snapshot();
}

int main(void) {
    char db_path[] = "/tmp/ish-fake-db-dump-XXXXXX";
    int fd = mkstemp(db_path);
    assert(fd >= 0);
    close(fd);

    sqlite3 *initial;
    check(sqlite3_open_v2(db_path, &initial, SQLITE_OPEN_READWRITE, NULL), initial);
    exec_sql(initial,
             "create table meta (id integer unique default 0, db_inode integer);"
             "insert into meta (db_inode) values (0);"
             "create table stats (inode integer primary key, stat blob);"
             "create table paths (path blob primary key, inode integer references stats(inode));"
             "create index inode_to_path on paths (inode, path);"
             "pragma user_version=3;");
    check(sqlite3_close(initial), initial);

    assert(fake_db_init(&fs, db_path, -1) == 0);
    puts("# fake-db reference, generated from unmodified iSH fs/fake-db.c");
    puts("# state is an FNV-1a hash of ordered logical stats and paths rows");
    snapshot();

    struct ish_stat one = {0100644, 1000, 100, 0};
    struct ish_stat two = {0040755, 2000, 200, 0};
    struct ish_stat three = {0120777, 3000, 300, 0x12345678};
    struct ish_stat changed = {0100600, 4000, 400, 9};

    record_begin(true);
    record_create("/a", one);
    record_finish(true);
    record_get("/a");
    record_begin(false);
    record_path_read("/a");
    record_finish(true);

    record_begin(true);
    record_create("/a/child", two);
    // An existing destination makes path_rename exercise SQLite UPDATE OR
    // REPLACE and leaves inode 3 orphaned until the sweep below.
    record_create("/x", three);
    // This sibling is deliberately outside C's ["/a/", "/a0") range.
    record_create("/apple", one);
    record_link("/a", "/b");
    record_paths_for_inode(1);
    record_finish(true);
    record_begin(true);
    record_rename("/a", "/x");
    record_finish(true);
    record_paths_for_inode(1);
    record_get("/a");
    record_path_read("/x");
    record_path_read("/x/child");
    record_path_read("/b");
    record_path_read("/apple");

    record_begin(true);
    record_inode_write(2, changed);
    record_finish(true);
    record_inode_read(2);
    record_unlink("/b");
    record_cleanup(1); // /x still keeps inode 1 alive.
    record_unlink("/x");
    record_cleanup(1);
    record_inode_read(1);
    // UPDATE stats ... WHERE inode = ? is a C no-op for stale metadata.
    record_inode_write(99, three);
    record_inode_read(99);
    record_clear_orphans();
    record_inode_read(3);

    // The database transaction itself, not an in-memory shadow, rolls this
    // create back; the following lookup must return the C sentinel zero.
    record_begin(true);
    record_create("/rolled", three);
    record_finish(false);
    record_get("/rolled");

    assert(fake_db_deinit(&fs) == SQLITE_OK);
    unlink(db_path);
    return 0;
}
