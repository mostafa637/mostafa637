#ifndef SQLITE3_H
#define SQLITE3_H

// A deliberately small, ABI-compatible slice of SQLite's public C header.
//
// Most C differential generators reach sqlite3 only through fs/fake-db.h and
// need opaque pointers. The fake-db generator additionally links the untouched
// fs/fake-db.c against the platform SQLite shared library, so it needs these
// declarations and constants without requiring a host sqlite3-dev package.
// This header is used solely to build the C behavioral oracle: the Rust
// fake_db module uses vendored pure-Rust graphitesql and never links SQLite.
// New C-oracle uses must add their exact public declarations here rather than
// depending on a generated or host-specific SQLite header.

#include <stddef.h>
#include <stdint.h>

typedef struct sqlite3 sqlite3;
typedef struct sqlite3_stmt sqlite3_stmt;
typedef struct sqlite3_mutex sqlite3_mutex;
typedef struct sqlite3_context sqlite3_context;
typedef struct sqlite3_value sqlite3_value;

typedef void (*sqlite3_destructor_type)(void *);
typedef void (*sqlite3_xfunc)(sqlite3_context *, int, sqlite3_value **);
typedef int (*sqlite3_callback)(void *, int, char **, char **);

#define SQLITE_OK 0
#define SQLITE_ROW 100
#define SQLITE_DONE 101
#define SQLITE_UTF8 1
#define SQLITE_DETERMINISTIC 0x00000800
#define SQLITE_OPEN_READWRITE 0x00000002
#define SQLITE_OPEN_CREATE 0x00000004
#define SQLITE_MUTEX_FAST 0
#define SQLITE_TRANSIENT ((sqlite3_destructor_type) -1)

int sqlite3_open_v2(const char *filename, sqlite3 **pp_db, int flags,
                    const char *z_vfs);
int sqlite3_close(sqlite3 *db);
int sqlite3_errcode(sqlite3 *db);
int sqlite3_extended_errcode(sqlite3 *db);
const char *sqlite3_errmsg(sqlite3 *db);
int sqlite3_busy_timeout(sqlite3 *db, int ms);
int sqlite3_create_function(sqlite3 *db, const char *z_function_name, int n_arg,
                            int e_text_rep, void *p_app, sqlite3_xfunc x_func,
                            void *x_step, void *x_final);
int sqlite3_prepare_v2(sqlite3 *db, const char *z_sql, int n_byte,
                       sqlite3_stmt **pp_stmt, const char **pz_tail);
int sqlite3_step(sqlite3_stmt *stmt);
int sqlite3_reset(sqlite3_stmt *stmt);
int sqlite3_finalize(sqlite3_stmt *stmt);
int sqlite3_bind_blob(sqlite3_stmt *stmt, int index, const void *value, int n,
                      sqlite3_destructor_type destructor);
int sqlite3_bind_int64(sqlite3_stmt *stmt, int index, int64_t value);
int sqlite3_column_int(sqlite3_stmt *stmt, int column);
int64_t sqlite3_column_int64(sqlite3_stmt *stmt, int column);
const unsigned char *sqlite3_column_text(sqlite3_stmt *stmt, int column);
const void *sqlite3_column_blob(sqlite3_stmt *stmt, int column);
int sqlite3_column_bytes(sqlite3_stmt *stmt, int column);
int64_t sqlite3_last_insert_rowid(sqlite3 *db);
sqlite3_mutex *sqlite3_mutex_alloc(int type);
void sqlite3_mutex_enter(sqlite3_mutex *mutex);
void sqlite3_mutex_leave(sqlite3_mutex *mutex);
const void *sqlite3_value_blob(sqlite3_value *value);
int sqlite3_value_bytes(sqlite3_value *value);
int64_t sqlite3_value_int64(sqlite3_value *value);
void *sqlite3_malloc(int n);
char *sqlite3_mprintf(const char *format, ...);
void sqlite3_free(void *ptr);
void sqlite3_result_blob(sqlite3_context *context, const void *value, int n,
                         sqlite3_destructor_type destructor);
int sqlite3_exec(sqlite3 *db, const char *sql, sqlite3_callback callback,
                 void *arg, char **error_message);

#endif
