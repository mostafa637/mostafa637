//! Pure-Rust host-inode rebuild for iSH fakefs metadata.
//!
//! This module translates `fs/fake-rebuild.c`. iSH stores guest metadata under
//! synthetic inode numbers in the fakefs SQLite database. If a filesystem is
//! copied or restored, host inode numbers can change; rebuilding maps every
//! surviving metadata path to its current host inode and attempts to restore
//! host hard links that shared an old fakefs inode.
//!
//! The database side runs against the vendored pure-Rust GraphiteSQL engine.
//! Host operations are deliberately supplied by [`RebuildHost`], which keeps
//! the filesystem boundary explicit and lets the C behavior be checked with a
//! deterministic local host oracle.

use std::collections::BTreeMap;

use graphitesql::{Connection, Value};

use crate::fake_db::{
    blob_from_value, blob_literal, execute, integer_from_value, query_rows, sqlite_integer,
    FakeDbError,
};

/// Host operations used by [`crate::fake_db::FakeDb::rebuild_with_host`].
///
/// Every path passed to this trait has first gone through [`fix_path`], just as
/// `fake-rebuild.c` passes `fix_path(path)` to `fstatat`, `unlinkat`, and
/// `linkat`. An error from `inode_for_path` skips that metadata path. Errors
/// from `unlink_path` and `link_path` are intentionally ignored after being
/// counted in [`RebuildReport`], matching the unchecked C calls.
pub trait RebuildHost {
    /// Host-specific error type. Rebuild follows C by handling errors locally.
    type Error;

    /// Return the current host inode for a fixed, root-relative path.
    fn inode_for_path(&mut self, path: &[u8]) -> Result<u64, Self::Error>;

    /// Remove a fixed, root-relative path before a hard-link repair.
    fn unlink_path(&mut self, path: &[u8]) -> Result<(), Self::Error>;

    /// Create a fixed, root-relative hard link from `source` to `destination`.
    fn link_path(&mut self, source: &[u8], destination: &[u8]) -> Result<(), Self::Error>;
}

/// Observable counters from one fakefs metadata rebuild.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct RebuildReport {
    /// Rows read from the old `paths` table.
    pub paths_scanned: usize,
    /// Paths skipped because their host `fstatat` equivalent failed.
    pub paths_skipped_missing_host: usize,
    /// Paths skipped because their old fakefs inode had no stat record.
    pub paths_skipped_missing_stat: usize,
    /// Repeated old inodes for which C attempts an unlink/link restoration.
    pub hardlink_repairs_attempted: usize,
    /// Failed unlink attempts that C deliberately ignores.
    pub ignored_unlink_failures: usize,
    /// Failed hard-link attempts that C deliberately ignores.
    pub ignored_link_failures: usize,
    /// New `stats` and `paths` mappings written to the rebuilt database.
    pub metadata_paths_written: usize,
}

/// Rust representation of `fs/fix_path.h`.
///
/// C sees SQLite paths through a NUL-terminated `char *`, so an interior NUL
/// ends the host-facing part of a malformed legacy row. An originally empty
/// path maps to `.`, while `/` maps to the empty string because C checks for
/// emptiness before it strips a leading slash.
#[must_use]
pub fn fix_path(path: &[u8]) -> &[u8] {
    let path = c_path(path);
    if path.is_empty() {
        b"."
    } else if path[0] == b'/' {
        &path[1..]
    } else {
        path
    }
}

/// Execute the database half of C `fakefs_rebuild` over one owned connection.
///
/// `FakeDb::rebuild_with_host` keeps this internal entry point transactional;
/// it is crate-visible only so that the database module owns connection
/// borrowing while this module owns the filesystem rebuild algorithm.
pub(crate) fn rebuild<H: RebuildHost>(
    connection: &mut Connection,
    host: &mut H,
) -> Result<RebuildReport, FakeDbError> {
    execute(connection, "BEGIN")?;
    let result = (|| {
        // Keep the same copy-then-empty order as fake-rebuild.c. The old tables
        // isolate reads from paths and stats inserted during the loop.
        execute(
            connection,
            "CREATE TABLE paths_old (path BLOB PRIMARY KEY, inode INTEGER)",
        )?;
        execute(
            connection,
            "CREATE TABLE stats_old (inode INTEGER PRIMARY KEY, stat BLOB)",
        )?;
        execute(connection, "INSERT INTO paths_old SELECT * FROM paths")?;
        execute(connection, "INSERT INTO stats_old SELECT * FROM stats")?;
        execute(connection, "DELETE FROM paths")?;
        execute(connection, "DELETE FROM stats")?;

        let old_paths = query_rows(connection, "SELECT path, inode FROM paths_old")?;
        let mut canonical_paths = BTreeMap::<u64, Vec<u8>>::new();
        let mut report = RebuildReport::default();

        for row in old_paths {
            let (path, old_inode) = path_row(&row)?;
            report.paths_scanned += 1;
            let host_path = fix_path(&path);
            let real_inode = match host.inode_for_path(host_path) {
                Ok(inode) => inode,
                Err(_) => {
                    // fstatat failure makes C skip this path without creating a
                    // canonical hard-link entry or any new database metadata.
                    report.paths_skipped_missing_host += 1;
                    continue;
                }
            };

            if let Some(canonical_path) = canonical_paths.get(&old_inode) {
                // C ignores both return values and continues to write metadata
                // using the inode observed before this repair attempt.
                report.hardlink_repairs_attempted += 1;
                if host.unlink_path(host_path).is_err() {
                    report.ignored_unlink_failures += 1;
                }
                if host.link_path(fix_path(canonical_path), host_path).is_err() {
                    report.ignored_link_failures += 1;
                }
            } else {
                canonical_paths.insert(old_inode, path.clone());
            }

            let Some(stat) = read_old_stat(connection, old_inode)? else {
                // Like C, retain the canonical hard-link entry above even when
                // metadata is malformed or missing, then skip database writes.
                report.paths_skipped_missing_stat += 1;
                continue;
            };
            write_rebuilt_metadata(connection, &path, real_inode, &stat)?;
            report.metadata_paths_written += 1;
        }

        execute(connection, "DROP TABLE paths_old")?;
        execute(connection, "DROP TABLE stats_old")?;
        execute(connection, "COMMIT")?;
        Ok(report)
    })();
    if result.is_err() {
        // C dies on SQL failures. Returning an error is friendlier in Rust, but
        // preserving its transactional database outcome is essential; host
        // hard-link changes made before the SQL error deliberately remain.
        let _ = connection.execute("ROLLBACK");
    }
    result
}

fn c_path(path: &[u8]) -> &[u8] {
    let length = path
        .iter()
        .position(|byte| *byte == 0)
        .unwrap_or(path.len());
    &path[..length]
}

fn path_row(row: &[Value]) -> Result<(Vec<u8>, u64), FakeDbError> {
    if row.len() != 2 {
        return Err(FakeDbError::Database(format!(
            "SQLite rebuild path row has {} columns instead of two",
            row.len()
        )));
    }
    // fake-rebuild.c uses sqlite3_column_text plus strlen, so preserve that
    // legacy C-string boundary even for a malformed database constructed
    // outside FakeDb's no-interior-NUL public API.
    let path = c_path(blob_from_value(&row[0])?).to_vec();
    let inode = integer_from_value(&row[1])? as u64;
    Ok((path, inode))
}

fn read_old_stat(connection: &Connection, inode: u64) -> Result<Option<Vec<u8>>, FakeDbError> {
    let rows = query_rows(
        connection,
        &format!(
            "SELECT stat FROM stats_old WHERE inode = {}",
            sqlite_integer(inode)
        ),
    )?;
    let Some(row) = rows.first() else {
        return Ok(None);
    };
    if row.len() != 1 {
        return Err(FakeDbError::Database(format!(
            "SQLite rebuild stat row has {} columns instead of one",
            row.len()
        )));
    }
    Ok(Some(blob_from_value(&row[0])?.to_vec()))
}

fn write_rebuilt_metadata(
    connection: &mut Connection,
    path: &[u8],
    inode: u64,
    stat: &[u8],
) -> Result<(), FakeDbError> {
    let inode = sqlite_integer(inode);
    execute(
        connection,
        &format!(
            "REPLACE INTO stats (inode, stat) VALUES ({inode}, {})",
            blob_literal(stat)
        ),
    )?;
    // C uses `sqlite3_bind_blob(..., strlen(path), ...)` here, not the source
    // blob length. `path_row` already applied exactly that C-string truncation.
    execute(
        connection,
        &format!(
            "INSERT INTO paths (path, inode) VALUES ({}, {inode})",
            blob_literal(path)
        ),
    )
}

/// A standard-library rooted host adapter for Unix/iOS targets.
///
/// It applies `std::fs` operations beneath `root` with byte-preserving Unix
/// paths. The rebuild algorithm has already called [`fix_path`], so the adapter
/// receives the same relative names that C gives its `*at` syscalls.
#[cfg(unix)]
#[derive(Debug, Clone)]
pub struct RootedHostFs {
    root: std::path::PathBuf,
}

#[cfg(unix)]
impl RootedHostFs {
    /// Bind host rebuild operations to `root`.
    #[must_use]
    pub fn new(root: impl AsRef<std::path::Path>) -> Self {
        Self {
            root: root.as_ref().to_path_buf(),
        }
    }

    fn resolve(&self, path: &[u8]) -> std::io::Result<std::path::PathBuf> {
        use std::os::unix::ffi::OsStringExt as _;

        // `fstatat(root_fd, "", ...)`, unlike `fstatat(root_fd, ".", ...)`,
        // fails. `fix_path("/")` deliberately produces this case.
        if path.is_empty() {
            return Err(std::io::Error::new(
                std::io::ErrorKind::NotFound,
                "empty path is invalid for fstatat/unlinkat/linkat",
            ));
        }
        Ok(self.root.join(std::ffi::OsString::from_vec(path.to_vec())))
    }
}

#[cfg(unix)]
impl RebuildHost for RootedHostFs {
    type Error = std::io::Error;

    fn inode_for_path(&mut self, path: &[u8]) -> Result<u64, Self::Error> {
        use std::os::unix::fs::MetadataExt as _;

        Ok(std::fs::metadata(self.resolve(path)?)?.ino())
    }

    fn unlink_path(&mut self, path: &[u8]) -> Result<(), Self::Error> {
        std::fs::remove_file(self.resolve(path)?)
    }

    fn link_path(&mut self, source: &[u8], destination: &[u8]) -> Result<(), Self::Error> {
        std::fs::hard_link(self.resolve(source)?, self.resolve(destination)?)
    }
}

#[cfg(all(test, unix))]
mod tests {
    use super::*;
    use crate::fake_db::{FakeDb, IshStat};
    use std::os::unix::fs::MetadataExt as _;
    use std::sync::atomic::{AtomicU64, Ordering};

    #[test]
    fn rooted_host_adapter_relinks_files_and_keeps_c_prelink_metadata_inodes() {
        let root = temporary_path("root");
        let database_path = temporary_path("database.sqlite");
        std::fs::create_dir(&root).unwrap();
        std::fs::write(root.join("a"), b"canonical").unwrap();
        std::fs::write(root.join("b"), b"separate inode before rebuild").unwrap();
        let original_a_inode = std::fs::metadata(root.join("a")).unwrap().ino();
        let original_b_inode = std::fs::metadata(root.join("b")).unwrap().ino();
        assert_ne!(original_a_inode, original_b_inode);

        let stat = IshStat {
            mode: 0o100644,
            uid: 1000,
            gid: 100,
            rdev: 0,
        };
        let db = FakeDb::create(&database_path).unwrap();
        db.path_create(b"/a", stat).unwrap();
        db.path_link(b"/a", b"/b").unwrap();
        let mut host = RootedHostFs::new(&root);
        let report = db.rebuild_with_host(&mut host).unwrap();

        assert_eq!(
            std::fs::metadata(root.join("a")).unwrap().ino(),
            original_a_inode
        );
        assert_eq!(
            std::fs::metadata(root.join("b")).unwrap().ino(),
            original_a_inode
        );
        // Like fake-rebuild.c, the database retains b's inode as observed
        // before unlink/link rather than re-statting it after the repair.
        assert_eq!(db.path_get_inode(b"/a").unwrap(), original_a_inode);
        assert_eq!(db.path_get_inode(b"/b").unwrap(), original_b_inode);
        assert_eq!(
            report,
            RebuildReport {
                paths_scanned: 2,
                paths_skipped_missing_host: 0,
                paths_skipped_missing_stat: 0,
                hardlink_repairs_attempted: 1,
                ignored_unlink_failures: 0,
                ignored_link_failures: 0,
                metadata_paths_written: 2,
            }
        );
        drop(db);
        remove_database_files(&database_path);
        std::fs::remove_dir_all(root).unwrap();
    }

    fn temporary_path(suffix: &str) -> std::path::PathBuf {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        std::env::temp_dir().join(format!(
            "ish-rs-fake-rebuild-{}-{}-{suffix}",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ))
    }

    fn remove_database_files(path: &std::path::Path) {
        let path = path.to_str().unwrap();
        for suffix in ["", "-journal", "-wal"] {
            let _ = std::fs::remove_file(format!("{path}{suffix}"));
        }
    }
}
