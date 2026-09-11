#ifndef SQLITE3_H
#define SQLITE3_H
// Just enough of sqlite3's public API for fs/fake-db.h to declare
// `struct fakefs_db`, which fs/fd.h drags in and kernel/memory.c includes for
// fd_close. All three types are only ever used as pointers there, and nothing
// in the memory reference generator calls into sqlite - so opaque declarations
// are enough and no sqlite dependency is introduced.
typedef struct sqlite3 sqlite3;
typedef struct sqlite3_stmt sqlite3_stmt;
typedef struct sqlite3_mutex sqlite3_mutex;
#endif
