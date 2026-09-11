//! `fs/fake-rebuild.c` — the fakefs metadata rebuild pass.
//!
//! iSH stores guest inode metadata in a SQLite database that lives *inside* the
//! fake filesystem, so when that directory is copied — or compressed,
//! transmitted and unpacked somewhere else — the database comes along but the
//! host inode numbers it recorded do not survive. `fake_db_init` detects that
//! by comparing the recorded `meta.db_inode` with the real inode of the
//! database file, and calls this rebuild pass when they disagree.
//!
//! The rebuild walks the stale `paths` table, asks the host for each path's
//! current inode, and writes a fresh `paths`/`stats` pair keyed by those real
//! inodes. Paths that shared one *old* inode were hardlinks in guest terms, and
//! are restored as host hardlinks by unlinking the later path and linking it to
//! the first path seen for that inode.
//!
//! The host calls (`fstatat`, `unlinkat`, `linkat`, all relative to the mount's
//! `root_fd`) are explicit through [`FakefsHost`] instead of being reached
//! through a raw file descriptor, so the C `root_fd` argument lives in the
//! adapter rather than in this function's signature.

use std::collections::BTreeMap;
use std::path::Path;

use graphitesql::{Connection, Value};

use crate::fake_db::{
    blob_from_value, blob_literal, integer_from_value, query_rows, sqlite_integer, FakeDb,
    FakeDbError,
};

/// The host filesystem behind a fakefs mount.
///
/// Every guest path handed to [`FakefsHost::file_inode`], [`FakefsHost::unlink`]
/// and [`FakefsHost::link`] has already been repaired by [`fix_path`], so
/// adapters see the relative, slash-free form that the C passes to
/// `fstatat`/`linkat` together with the mount's `root_fd`. Paths stay byte
/// slices because guest paths are not required to be UTF-8.
pub trait FakefsHost {
    /// `fstatat(root_fd, path, &stat, 0)`: the host inode of `path`, or `None`
    /// when the call fails. The C rebuild treats every failure as "skip this
    /// path", which is what keeps a rebuild working after files were deleted.
    fn file_inode(&self, path: &[u8]) -> Option<u64>;

    /// `unlinkat(root_fd, path, 0)`. The C rebuild ignores the result; `false`
    /// is reported so adapters can log a failure.
    fn unlink(&self, path: &[u8]) -> bool;

    /// `linkat(root_fd, source, root_fd, destination, 0)`. The C rebuild
    /// ignores the result; `false` is reported so adapters can log a failure.
    fn link(&self, source: &[u8], destination: &[u8]) -> bool;

    /// `stat(db_path)`: the real inode of the SQLite database file itself.
    /// `fake_db_init` records it in `meta` and rebuilds when it changes.
    fn database_inode(&self, database: &Path) -> Option<u64>;
}

/// `fs/fix_path.h`: the path form the host filesystem calls receive.
///
/// The empty guest path becomes `.` and a single leading `/` is dropped; any
/// other path is returned unchanged.
#[must_use]
pub fn fix_path(path: &[u8]) -> &[u8] {
    match path.first() {
        None => b".",
        Some(b'/') => &path[1..],
        Some(_) => path,
    }
}

/// `fakefs_rebuild`: rewrite `paths`/`stats` around the host's real inodes.
///
/// The C function takes the fakefs metadata database and the mount's root file
/// descriptor; here the host calls come from `host`. C only fails by aborting,
/// so this port reports SQL failures as [`FakeDbError::Database`] and rolls the
/// rebuild's own transaction back before returning, leaving the database usable
/// instead of half-rewritten.
pub fn fakefs_rebuild(db: &FakeDb, host: &dyn FakefsHost) -> Result<(), FakeDbError> {
    db.with_connection(|connection| rebuild(connection, host))
}

fn rebuild(connection: &mut Connection, host: &dyn FakefsHost) -> Result<(), FakeDbError> {
    match rewrite_tables(connection, host) {
        Ok(()) => Ok(()),
        Err(error) => {
            // The C `die`s inside the transaction and never commits. Rolling
            // back keeps a Rust caller's connection usable.
            let _ = connection.execute_batch("ROLLBACK;");
            Err(error)
        }
    }
}

fn rewrite_tables(connection: &mut Connection, host: &dyn FakefsHost) -> Result<(), FakeDbError> {
    // The C rebuild drives four prepared statements against the old tables
    // while it fills the new ones. `paths_old` is snapshotted here first: the
    // old tables are never written again during the walk, so their scan order —
    // which decides which path becomes the hardlink source — is preserved.
    connection
        .execute_batch(
            "BEGIN;\
             CREATE TABLE paths_old (path blob primary key, inode integer);\
             CREATE TABLE stats_old (inode integer primary key, stat blob);\
             INSERT INTO paths_old SELECT * FROM paths;\
             INSERT INTO stats_old SELECT * FROM stats;\
             DELETE FROM paths;\
             DELETE FROM stats;",
        )
        .map_err(FakeDbError::database)?;

    let paths_old = query_rows(connection, "SELECT path, inode FROM paths_old")?;
    let mut hardlinks: BTreeMap<u64, Vec<u8>> = BTreeMap::new();

    for row in paths_old {
        if row.len() != 2 {
            return Err(FakeDbError::Database(
                "SQLite paths_old query returned the wrong column count".into(),
            ));
        }
        // `sqlite3_column_text` plus `strlen` in C: a path is a byte string
        // that ends at its first NUL, and that same truncated byte string is
        // what gets bound back into the new paths table.
        let path = c_string(path_from_value(&row[0])?).to_vec();
        let old_inode = integer_from_value(&row[1])? as u64;

        let Some(real_inode) = host.file_inode(fix_path(&path)) else {
            continue;
        };

        // Restore hardlinks: the first path seen for an old inode is kept, and
        // every later path for the same old inode is relinked to it. C's 2000
        // bucket ad-hoc hash is keyed only by `inode`, with at most one entry
        // per inode, so a map keyed by that inode is the same table.
        match hardlinks.get(&old_inode) {
            Some(source) => {
                // The host calls always see the repaired path, exactly like the
                // C's `unlinkat(root_fd, fix_path(path), 0)` and
                // `linkat(root_fd, fix_path(entry->path), root_fd, fix_path(path), 0)`.
                host.unlink(fix_path(&path));
                host.link(fix_path(source), fix_path(&path));
            }
            None => {
                hardlinks.insert(old_inode, path.clone());
            }
        }

        // Copy the guest metadata across. A path mapping whose stat row is
        // missing has nothing to copy, so C skips that record.
        let Some(stat) = stat_blob(connection, old_inode)? else {
            continue;
        };

        connection
            .execute(&format!(
                "REPLACE INTO stats (inode, stat) VALUES ({}, {})",
                sqlite_integer(real_inode),
                blob_literal(&stat)
            ))
            .map_err(FakeDbError::database)?;
        connection
            .execute(&format!(
                "INSERT INTO paths (path, inode) VALUES ({}, {})",
                blob_literal(&path),
                sqlite_integer(real_inode)
            ))
            .map_err(FakeDbError::database)?;
    }

    connection
        .execute_batch("DROP TABLE paths_old; DROP TABLE stats_old; COMMIT;")
        .map_err(FakeDbError::database)
}

/// `select stat from stats_old where inode = ?`, returning the raw blob.
fn stat_blob(connection: &Connection, inode: u64) -> Result<Option<Vec<u8>>, FakeDbError> {
    let rows = query_rows(
        connection,
        &format!(
            "SELECT stat FROM stats_old WHERE inode={}",
            sqlite_integer(inode)
        ),
    )?;
    let Some(row) = rows.into_iter().next() else {
        return Ok(None);
    };
    if row.len() != 1 {
        return Err(FakeDbError::Database(
            "SQLite stats_old query returned the wrong column count".into(),
        ));
    }
    Ok(Some(blob_from_value(&row[0])?.to_vec()))
}

/// A stored path column, read with the C's byte-blob contract.
fn path_from_value(value: &Value) -> Result<&[u8], FakeDbError> {
    blob_from_value(value)
}

/// `sqlite3_column_text` / `strlen` semantics for a stored byte path.
fn c_string(bytes: &[u8]) -> &[u8] {
    match bytes.iter().position(|byte| *byte == 0) {
        Some(end) => &bytes[..end],
        None => bytes,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::fake_db::{query_rows, FakeDb, IshStat};
    use graphitesql::Connection;
    use std::cell::RefCell;
    use std::collections::BTreeMap;
    use std::path::{Path, PathBuf};

    /// Deterministic stand-in for a mount's host filesystem.
    ///
    /// `inodes` is the scripted result of `fstatat`; anything absent fails with
    /// the equivalent of `ENOENT`. Link and unlink calls are recorded in order
    /// so a test can assert the hardlink restoration sequence.
    #[derive(Default)]
    struct ScriptedHost {
        inodes: BTreeMap<Vec<u8>, u64>,
        log: RefCell<Vec<String>>,
    }

    impl ScriptedHost {
        fn new(entries: &[(&[u8], u64)]) -> Self {
            Self {
                inodes: entries
                    .iter()
                    .map(|(path, inode)| (path.to_vec(), *inode))
                    .collect(),
                log: RefCell::new(Vec::new()),
            }
        }

        fn log(&self) -> Vec<String> {
            self.log.borrow().clone()
        }
    }

    impl FakefsHost for ScriptedHost {
        fn file_inode(&self, path: &[u8]) -> Option<u64> {
            self.inodes.get(path).copied()
        }

        fn unlink(&self, path: &[u8]) -> bool {
            self.log
                .borrow_mut()
                .push(format!("unlink {}", String::from_utf8_lossy(path)));
            true
        }

        fn link(&self, source: &[u8], destination: &[u8]) -> bool {
            self.log.borrow_mut().push(format!(
                "link {} {}",
                String::from_utf8_lossy(source),
                String::from_utf8_lossy(destination)
            ));
            true
        }

        fn database_inode(&self, database: &Path) -> Option<u64> {
            let _ = database;
            None
        }
    }

    fn stat(mode: u32) -> IshStat {
        IshStat {
            mode,
            uid: 1000,
            gid: 100,
            rdev: 0,
        }
    }

    /// Build the corpus in `paths` order so the fake inode numbers and the
    /// table scan order both match the C harness. `create` adds a path with
    /// fresh metadata, `link` gives an existing path a second name, which is
    /// what produces the shared old inodes the rebuild has to restore.
    fn database(operations: &[Operation]) -> FakeDb {
        let db = FakeDb::open_in_memory().unwrap();
        let transaction = db.begin_write().unwrap();
        for operation in operations {
            match operation {
                Operation::Create(path, stat) => {
                    transaction.path_create(path, *stat).unwrap();
                }
                Operation::Link(path, alias) => {
                    transaction.path_link(path, alias).unwrap();
                }
            }
        }
        transaction.commit().unwrap();
        db
    }

    enum Operation<'a> {
        Create(&'a [u8], IshStat),
        Link(&'a [u8], &'a [u8]),
    }

    /// A database whose tables are written with arbitrary SQL, used for states
    /// the public metadata API cannot produce (a path mapping with no stat).
    fn raw_database(setup: &str) -> FakeDb {
        let mut connection = Connection::open_memory().unwrap();
        connection
            .execute_batch(&format!(
                "CREATE TABLE meta (id integer unique default 0, db_inode integer);\
                 CREATE TABLE stats (inode integer primary key, stat blob);\
                 CREATE TABLE paths (path blob primary key, inode integer references stats(inode));\
                 {setup}"
            ))
            .unwrap();
        FakeDb::from_connection(connection)
    }

    #[test]
    fn fix_path_matches_the_c_header() {
        assert_eq!(fix_path(b""), b".");
        assert_eq!(fix_path(b"/"), b"");
        assert_eq!(fix_path(b"/a/b"), b"a/b");
        assert_eq!(fix_path(b"a/b"), b"a/b");
        assert_eq!(fix_path(b"//a"), b"/a");
    }

    #[test]
    fn rebuild_rewrites_inodes_and_restores_hardlinks_in_scan_order() {
        let db = database(&[
            Operation::Create(b"/a", stat(0o100644)),
            Operation::Link(b"/a", b"/b"),
            Operation::Create(b"/c/d", stat(0o040755)),
            Operation::Create(b"/gone", stat(0o100644)),
        ]);
        // /a and /b were one guest inode before the copy; /c/d has its own;
        // /gone no longer exists on the host.
        let host = ScriptedHost::new(&[(b"a", 900), (b"b", 901), (b"c/d", 902)]);

        fakefs_rebuild(&db, &host).unwrap();

        assert_eq!(host.log(), vec!["unlink b", "link a b"]);
        assert_eq!(db.path_get_inode(b"/a").unwrap(), 900);
        // C stats each path *before* relinking it and then stores that inode,
        // so the linked alias keeps the inode the copy gave it. Pinned here
        // because it is observable to a guest that later re-reads the database.
        assert_eq!(db.path_get_inode(b"/b").unwrap(), 901);
        assert_eq!(db.path_get_inode(b"/c/d").unwrap(), 902);
        assert_eq!(db.path_get_inode(b"/gone").unwrap(), 0);
        assert_eq!(
            db.path_read_stat(b"/a").unwrap().unwrap().stat,
            stat(0o100644)
        );
        assert_eq!(
            db.path_read_stat(b"/c/d").unwrap().unwrap().stat,
            stat(0o040755)
        );
        assert_eq!(db.path_read_stat(b"/gone").unwrap(), None);
        // The stale inode metadata for the deleted path is gone as well.
        assert_eq!(db.inode_read_stat_if_exist(4).unwrap(), None);
    }

    #[test]
    fn rebuild_keeps_the_first_path_as_the_link_source() {
        let db = database(&[
            Operation::Create(b"/one", stat(1)),
            Operation::Link(b"/one", b"/two"),
            Operation::Link(b"/one", b"/three"),
        ]);
        let host = ScriptedHost::new(&[(b"one", 41), (b"two", 42), (b"three", 43)]);
        fakefs_rebuild(&db, &host).unwrap();
        assert_eq!(
            host.log(),
            vec![
                "unlink two",
                "link one two",
                "unlink three",
                "link one three"
            ]
        );
        // C stats each path before relinking it and then stores that inode, so
        // the linked paths keep their own pre-link inode numbers here.
        assert_eq!(db.path_get_inode(b"/one").unwrap(), 41);
        assert_eq!(db.path_get_inode(b"/two").unwrap(), 42);
        assert_eq!(db.path_get_inode(b"/three").unwrap(), 43);
    }

    #[test]
    fn rebuild_skips_paths_whose_stat_row_is_missing() {
        let db = raw_database(
            "INSERT INTO stats (inode, stat) VALUES (1, X'07000000');\
             INSERT INTO paths (path, inode) VALUES (X'2f70', 1), (X'2f6d', 2);",
        );
        let host = ScriptedHost::new(&[(b"p", 50), (b"m", 51)]);
        fakefs_rebuild(&db, &host).unwrap();
        assert!(host.log().is_empty());
        assert_eq!(db.path_get_inode(b"/p").unwrap(), 50);
        assert_eq!(db.path_get_inode(b"/m").unwrap(), 0);
    }

    #[test]
    fn rebuild_rejects_non_blob_path_records_and_rolls_back() {
        let db = raw_database("INSERT INTO paths (path, inode) VALUES (7, 1);");
        let host = ScriptedHost::new(&[(b"7", 200)]);
        let error = fakefs_rebuild(&db, &host).unwrap_err();
        assert!(matches!(error, FakeDbError::Database(_)), "{error:?}");
        // The rewrite transaction is rolled back, so the connection is usable
        // and the old tables are gone.
        assert_eq!(db.path_get_inode(b"/anything").unwrap(), 0);
        assert!(db.path_create(b"/new", stat(3)).is_ok());
    }

    /// The scratch tables the rebuild copies through must not survive it.
    #[test]
    fn rebuild_leaves_only_the_live_tables_behind() {
        let db = database(&[Operation::Create(b"/x", stat(1))]);
        let host = ScriptedHost::new(&[(b"x", 5)]);
        fakefs_rebuild(&db, &host).unwrap();
        let mut tables: Vec<String> = db
            .with_connection(|connection| {
                query_rows(
                    connection,
                    "SELECT name FROM sqlite_master WHERE type='table'",
                )
            })
            .unwrap()
            .into_iter()
            .map(|row| match &row[0] {
                graphitesql::Value::Text(name) => name.as_str().to_string(),
                other => panic!("unexpected sqlite_master row {other:?}"),
            })
            .collect();
        tables.sort();
        assert_eq!(tables, vec!["meta", "paths", "stats"]);
    }

    /// The `sqlite3_column_text` truncation the C rebuild inherits.
    #[test]
    fn paths_stop_at_an_embedded_nul() {
        assert_eq!(c_string(b"/a\0tail"), b"/a");
        assert_eq!(c_string(b"/a"), b"/a");
        assert_eq!(c_string(b""), b"");
    }

    /// A file-backed database so the rewrite also round-trips through the
    /// SQLite file format rather than only the in-memory pager.
    #[test]
    fn rebuild_persists_to_a_sqlite_file() {
        let path = temporary_database_path();
        {
            let db = FakeDb::create(&path).unwrap();
            let transaction = db.begin_write().unwrap();
            transaction.path_create(b"/x", stat(0o100600)).unwrap();
            transaction.path_create(b"/y", stat(0o100600)).unwrap();
            transaction.commit().unwrap();
        }
        let db = FakeDb::open(&path).unwrap();
        let host = ScriptedHost::new(&[(b"x", 700), (b"y", 701)]);
        fakefs_rebuild(&db, &host).unwrap();
        drop(db);

        let reopened = FakeDb::open(&path).unwrap();
        assert_eq!(reopened.path_get_inode(b"/x").unwrap(), 700);
        assert_eq!(reopened.path_get_inode(b"/y").unwrap(), 701);
        assert_eq!(
            reopened.path_read_stat(b"/y").unwrap().unwrap().stat,
            stat(0o100600)
        );
        drop(reopened);
        remove_database_files(&path);
    }

    fn temporary_database_path() -> PathBuf {
        use std::sync::atomic::{AtomicU64, Ordering};
        static NEXT: AtomicU64 = AtomicU64::new(0);
        std::env::temp_dir().join(format!(
            "ish-rs-rebuild-{}-{}.sqlite",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ))
    }

    fn remove_database_files(path: &Path) {
        let path = path.to_str().unwrap();
        for suffix in ["", "-journal", "-wal"] {
            let _ = std::fs::remove_file(format!("{path}{suffix}"));
        }
    }
}
