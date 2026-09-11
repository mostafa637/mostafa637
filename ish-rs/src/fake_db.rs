//! Pure-Rust SQLite fake-filesystem metadata database.
//!
//! iSH's `fs/fake-db.c` stores guest-visible inode metadata separately from
//! host filesystem metadata. This module preserves that API's observable
//! rules—byte paths, inode allocation, links, prefix renames, orphan cleanup,
//! and transactions—using the vendored [`graphitesql`] SQLite-3 implementation.
//!
//! `graphitesql` is a from-scratch, pure-Rust implementation of SQLite's SQL
//! surface and version-3 database-file format. Its narrowed vendored profile
//! has no native bindings, C/C++ source, unsafe code, or external Cargo
//! dependencies. Local differential tests compare both metadata operations and
//! legacy-schema migration with unmodified iSH C, using C SQLite only as an
//! oracle.

use std::cell::RefCell;
use std::collections::BTreeSet;
use std::error::Error;
use std::fmt;
use std::path::Path;
use std::rc::Rc;

use graphitesql::{Connection, Value};

type ConnectionSlot = Rc<RefCell<Option<Connection>>>;
const SQLITE_SCHEMA_VERSION: u64 = 3;

// Ordered exactly as the `migrations` array in upstream fs/fake-migrate.c.
// These SQL statements run inside one transaction in `migrate_ish_schema`.
const ISH_SCHEMA_MIGRATIONS: [&str; 3] = [
    "CREATE INDEX inode_to_path ON paths (inode, path)",
    "CREATE TABLE paths_new (path BLOB PRIMARY KEY, inode INTEGER REFERENCES stats(inode));\
     INSERT INTO paths_new SELECT * FROM paths WHERE EXISTS (SELECT 1 FROM stats WHERE inode = paths.inode);\
     DROP TABLE paths;\
     ALTER TABLE paths_new RENAME TO paths;\
     CREATE INDEX inode_to_path ON paths (inode, path);\
     DELETE FROM stats WHERE NOT EXISTS (SELECT 1 FROM paths WHERE inode = stats.inode);\
     CREATE TRIGGER delete_path AFTER DELETE ON paths \
       WHEN NOT EXISTS (SELECT 1 FROM paths WHERE inode = OLD.inode) \
       BEGIN \
         DELETE FROM stats \
         WHERE NOT EXISTS (SELECT 1 FROM paths WHERE inode = OLD.inode) \
           AND inode = OLD.inode; \
       END;",
    "DROP TRIGGER delete_path",
];

/// An error from the pure-Rust SQLite metadata database.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FakeDbError {
    /// The pure-Rust SQLite engine rejected an operation.
    Database(String),
    /// A path had an interior NUL, which C's `strlen` interface cannot carry.
    InteriorNulPath,
    /// A filesystem path could not be represented by the SQLite library's
    /// UTF-8 file-VFS API.
    NonUtf8DatabasePath,
    /// A metadata record was not the 16-byte `struct ish_stat` layout.
    InvalidStatBlob {
        /// Actual record length.
        len: usize,
    },
    /// `path_link`, `path_unlink` or a required inode lookup was absent.
    MissingPath,
    /// The opened SQLite metadata schema is older than the migrations this
    /// port can complete. Current upstream iSH metadata uses version 3.
    UnsupportedSchemaVersion {
        /// SQLite `PRAGMA user_version` found in the database file.
        found: u64,
    },
    /// A caller tried to use the database while an explicit transaction owns
    /// its single SQLite connection.
    TransactionActive,
    /// A transaction was already committed, rolled back, or otherwise closed.
    TransactionFinished,
    /// SQLite's positive implicit rowid range has been exhausted.
    InodeExhausted,
}

impl FakeDbError {
    fn database(error: impl fmt::Display) -> Self {
        Self::Database(error.to_string())
    }
}

impl fmt::Display for FakeDbError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Database(message) => write!(formatter, "fakefs SQLite error: {message}"),
            Self::InteriorNulPath => formatter.write_str("fakefs path contains an interior NUL"),
            Self::NonUtf8DatabasePath => {
                formatter.write_str("fakefs database path is not valid UTF-8")
            }
            Self::InvalidStatBlob { len } => {
                write!(
                    formatter,
                    "fakefs stat record has {len} bytes instead of 16"
                )
            }
            Self::MissingPath => formatter.write_str("fakefs metadata path does not exist"),
            Self::UnsupportedSchemaVersion { found } => {
                write!(
                    formatter,
                    "unsupported fakefs SQLite schema version {found}"
                )
            }
            Self::TransactionActive => {
                formatter.write_str("fakefs database is owned by an active transaction")
            }
            Self::TransactionFinished => {
                formatter.write_str("fakefs transaction has already finished")
            }
            Self::InodeExhausted => formatter.write_str("fakefs inode range is exhausted"),
        }
    }
}

impl Error for FakeDbError {}

/// The fixed 16-byte `struct ish_stat` value from `fs/fake-db.h`.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct IshStat {
    /// Guest file mode, including type bits.
    pub mode: u32,
    /// Guest owner UID.
    pub uid: u32,
    /// Guest owner GID.
    pub gid: u32,
    /// Guest device number for character/block nodes.
    pub rdev: u32,
}

impl IshStat {
    /// `sizeof(struct ish_stat)` on iSH's supported little-endian targets.
    pub const SIZE: usize = 16;

    /// Serialize the four i386-compatible words stored by `fake-db.c`.
    #[must_use]
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        bytes[0..4].copy_from_slice(&self.mode.to_le_bytes());
        bytes[4..8].copy_from_slice(&self.uid.to_le_bytes());
        bytes[8..12].copy_from_slice(&self.gid.to_le_bytes());
        bytes[12..16].copy_from_slice(&self.rdev.to_le_bytes());
        bytes
    }

    fn from_le_bytes(bytes: &[u8]) -> Result<Self, FakeDbError> {
        if bytes.len() != Self::SIZE {
            return Err(FakeDbError::InvalidStatBlob { len: bytes.len() });
        }
        Ok(Self {
            mode: u32::from_le_bytes(bytes[0..4].try_into().unwrap()),
            uid: u32::from_le_bytes(bytes[4..8].try_into().unwrap()),
            gid: u32::from_le_bytes(bytes[8..12].try_into().unwrap()),
            rdev: u32::from_le_bytes(bytes[12..16].try_into().unwrap()),
        })
    }
}

/// One `path_read_stat` result.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct MetadataRow {
    /// The fake inode associated with the path.
    pub inode: u64,
    /// The guest-visible stat metadata.
    pub stat: IshStat,
}

/// Persistent, SQLite-3-file-compatible fake filesystem metadata storage.
///
/// `create` writes a SQLite version-3 database using pure Rust. `open` also
/// upgrades compatible historical iSH metadata schemas v0–v2 through the same
/// durable v3 state as upstream `fakefs_migrate`. Host-filesystem rebuild and
/// fakefs integration remain separate conversion work.
pub struct FakeDb {
    connection: ConnectionSlot,
}

impl FakeDb {
    /// Create a SQLite-backed metadata database at `path`, or open and migrate
    /// it when it already holds a compatible historical iSH schema.
    pub fn create(path: impl AsRef<Path>) -> Result<Self, FakeDbError> {
        let path = path.as_ref();
        let sqlite_path = database_path(path)?;
        let exists = path.exists();
        let connection = if exists {
            Connection::open(sqlite_path)
        } else {
            Connection::create(sqlite_path)
        }
        .map_err(FakeDbError::database)?;
        let db = Self::from_connection(connection);
        if exists {
            db.migrate_schema()?;
            db.validate_schema()?;
        } else {
            db.initialize_schema()?;
        }
        db.clear_orphans()?;
        Ok(db)
    }

    /// Open an existing SQLite-backed fakefs metadata database.
    pub fn open(path: impl AsRef<Path>) -> Result<Self, FakeDbError> {
        let connection =
            Connection::open(database_path(path.as_ref())?).map_err(FakeDbError::database)?;
        let db = Self::from_connection(connection);
        db.migrate_schema()?;
        db.validate_schema()?;
        db.clear_orphans()?;
        Ok(db)
    }

    /// Create an in-memory pure-Rust SQLite database for a caller or
    /// differential test. No native SQLite library is loaded by this method.
    pub fn open_in_memory() -> Result<Self, FakeDbError> {
        let db = Self::from_connection(Connection::open_memory().map_err(FakeDbError::database)?);
        db.initialize_schema()?;
        Ok(db)
    }

    fn from_connection(connection: Connection) -> Self {
        Self {
            connection: Rc::new(RefCell::new(Some(connection))),
        }
    }

    fn initialize_schema(&self) -> Result<(), FakeDbError> {
        self.with_connection(|connection| {
            connection
                .execute("PRAGMA foreign_keys=ON")
                .map_err(FakeDbError::database)?;
            // Keep the same schema and user-version baseline used by the
            // current upstream fakefs migration path. Every value below is
            // fixed source text; guest-controlled paths and blobs use hex
            // literals in the individual metadata operations.
            connection
                .execute_batch(
                    "BEGIN;\
                     CREATE TABLE meta (id INTEGER UNIQUE DEFAULT 0, db_inode INTEGER);\
                     INSERT INTO meta (db_inode) VALUES (0);\
                     CREATE TABLE stats (inode INTEGER PRIMARY KEY, stat BLOB);\
                     CREATE TABLE paths (path BLOB PRIMARY KEY, inode INTEGER REFERENCES stats(inode));\
                     CREATE INDEX inode_to_path ON paths (inode, path);\
                     PRAGMA user_version=3;\
                     COMMIT;",
                )
                .map_err(FakeDbError::database)
        })
    }

    /// Port `fakefs_migrate`: upgrade an old iSH SQLite metadata schema before
    /// preparing the fakefs operations.
    fn migrate_schema(&self) -> Result<(), FakeDbError> {
        self.with_connection(migrate_ish_schema)
    }

    fn validate_schema(&self) -> Result<(), FakeDbError> {
        self.with_connection(|connection| {
            connection
                .execute("PRAGMA foreign_keys=ON")
                .map_err(FakeDbError::database)?;
            let version = sqlite_schema_version(connection)?;
            // Upstream leaves a newer user_version untouched. Resolve the
            // required current tables so that a newer compatible schema keeps
            // working, while a failed old-schema migration is never accepted.
            if version < SQLITE_SCHEMA_VERSION {
                return Err(FakeDbError::UnsupportedSchemaVersion { found: version });
            }
            // Ask the engine to resolve the columns rather than accepting an
            // arbitrary SQLite file that only happens to have the same PRAGMA.
            query_rows(connection, "SELECT db_inode FROM meta LIMIT 1")?;
            query_rows(connection, "SELECT inode, stat FROM stats LIMIT 1")?;
            query_rows(connection, "SELECT path, inode FROM paths LIMIT 1")?;
            Ok(())
        })
    }

    fn take_connection(&self) -> Result<Connection, FakeDbError> {
        take_connection(&self.connection)
    }

    fn put_connection(&self, connection: Connection) -> Result<(), FakeDbError> {
        put_connection(&self.connection, connection)
    }

    fn with_connection<T>(
        &self,
        operation: impl FnOnce(&mut Connection) -> Result<T, FakeDbError>,
    ) -> Result<T, FakeDbError> {
        let mut connection = self.take_connection()?;
        let result = operation(&mut connection);
        self.put_connection(connection)?;
        result
    }

    fn begin_compatible_transaction(&self) -> Result<FakeDbTransaction, FakeDbError> {
        let mut connection = self.take_connection()?;
        match connection.execute("BEGIN") {
            Ok(_) => Ok(FakeDbTransaction {
                slot: Rc::clone(&self.connection),
                connection: RefCell::new(Some(connection)),
            }),
            Err(error) => {
                let error = FakeDbError::database(error);
                self.put_connection(connection)?;
                Err(error)
            }
        }
    }

    /// Start the C `db_begin_read` compatibility transaction.
    ///
    /// SQLite's deferred read transaction can later promote to a write, so the
    /// same owned pure-Rust SQLite transaction backs both begin modes.
    pub fn begin_read(&self) -> Result<FakeDbTransaction, FakeDbError> {
        self.begin_compatible_transaction()
    }

    /// Start the C `db_begin_write` compatibility transaction.
    pub fn begin_write(&self) -> Result<FakeDbTransaction, FakeDbError> {
        self.begin_compatible_transaction()
    }

    fn autocommit<T>(
        &self,
        operation: impl FnOnce(&FakeDbTransaction) -> Result<T, FakeDbError>,
    ) -> Result<T, FakeDbError> {
        let transaction = self.begin_write()?;
        match operation(&transaction) {
            Ok(value) => {
                transaction.commit()?;
                Ok(value)
            }
            Err(error) => {
                // Dropping an unfinished transaction rolls it back and returns
                // its connection to the database pool.
                drop(transaction);
                Err(error)
            }
        }
    }

    /// Autocommit form of [`FakeDbTransaction::path_get_inode`].
    pub fn path_get_inode(&self, path: &[u8]) -> Result<u64, FakeDbError> {
        self.autocommit(|transaction| transaction.path_get_inode(path))
    }

    /// Autocommit form of [`FakeDbTransaction::path_read_stat`].
    pub fn path_read_stat(&self, path: &[u8]) -> Result<Option<MetadataRow>, FakeDbError> {
        self.autocommit(|transaction| transaction.path_read_stat(path))
    }

    /// Autocommit form of [`FakeDbTransaction::path_create`].
    pub fn path_create(&self, path: &[u8], stat: IshStat) -> Result<u64, FakeDbError> {
        self.autocommit(|transaction| transaction.path_create(path, stat))
    }

    /// Autocommit form of [`FakeDbTransaction::inode_read_stat_if_exist`].
    pub fn inode_read_stat_if_exist(&self, inode: u64) -> Result<Option<IshStat>, FakeDbError> {
        self.autocommit(|transaction| transaction.inode_read_stat_if_exist(inode))
    }

    /// Autocommit form of [`FakeDbTransaction::inode_write_stat`].
    pub fn inode_write_stat(&self, inode: u64, stat: IshStat) -> Result<(), FakeDbError> {
        self.autocommit(|transaction| transaction.inode_write_stat(inode, stat))
    }

    /// Autocommit form of [`FakeDbTransaction::path_link`].
    pub fn path_link(&self, src: &[u8], dst: &[u8]) -> Result<(), FakeDbError> {
        self.autocommit(|transaction| transaction.path_link(src, dst))
    }

    /// Autocommit form of [`FakeDbTransaction::path_unlink`].
    pub fn path_unlink(&self, path: &[u8]) -> Result<u64, FakeDbError> {
        self.autocommit(|transaction| transaction.path_unlink(path))
    }

    /// Autocommit form of [`FakeDbTransaction::path_rename`].
    pub fn path_rename(&self, src: &[u8], dst: &[u8]) -> Result<(), FakeDbError> {
        self.autocommit(|transaction| transaction.path_rename(src, dst))
    }

    /// Autocommit form of [`FakeDbTransaction::paths_for_inode`].
    pub fn paths_for_inode(&self, inode: u64) -> Result<Vec<Vec<u8>>, FakeDbError> {
        self.autocommit(|transaction| transaction.paths_for_inode(inode))
    }

    /// Autocommit form of [`FakeDbTransaction::try_cleanup_inode`].
    pub fn try_cleanup_inode(&self, inode: u64) -> Result<(), FakeDbError> {
        self.autocommit(|transaction| transaction.try_cleanup_inode(inode))
    }

    /// Delete every stat record with no path mapping, as C initialization does.
    pub fn clear_orphans(&self) -> Result<(), FakeDbError> {
        self.autocommit(FakeDbTransaction::clear_orphans)
    }

    /// Hash the ordered logical stat and path tables like the local C harness.
    pub fn logical_hash(&self) -> Result<u64, FakeDbError> {
        self.autocommit(|transaction| transaction.logical_hash())
    }
}

/// A transaction over [`FakeDb`] metadata.
///
/// The transaction owns the one pure-Rust SQLite connection while it is active.
/// This mirrors C's fakefs transaction ownership: operations through the parent
/// [`FakeDb`] are rejected until this value is committed, rolled back, or dropped.
pub struct FakeDbTransaction {
    slot: ConnectionSlot,
    connection: RefCell<Option<Connection>>,
}

impl FakeDbTransaction {
    fn path(path: &[u8]) -> Result<&[u8], FakeDbError> {
        if path.contains(&0) {
            Err(FakeDbError::InteriorNulPath)
        } else {
            Ok(path)
        }
    }

    fn with_connection<T>(
        &self,
        operation: impl FnOnce(&mut Connection) -> Result<T, FakeDbError>,
    ) -> Result<T, FakeDbError> {
        let mut connection = self.connection.borrow_mut();
        let connection = connection
            .as_mut()
            .ok_or(FakeDbError::TransactionFinished)?;
        operation(connection)
    }

    fn finish(mut self, statement: &str) -> Result<(), FakeDbError> {
        let mut connection = self
            .connection
            .get_mut()
            .take()
            .ok_or(FakeDbError::TransactionFinished)?;
        let result = connection.execute(statement).map_err(FakeDbError::database);
        if result.is_err() {
            // There is no live owner left after this consuming call. Keep the
            // database usable even if the engine rejected the finishing SQL.
            let _ = connection.execute("ROLLBACK");
        }
        put_connection(&self.slot, connection)?;
        result.map(|_| ())
    }

    /// Commit the metadata transaction.
    pub fn commit(self) -> Result<(), FakeDbError> {
        self.finish("COMMIT")
    }

    /// Roll back the metadata transaction.
    pub fn rollback(self) -> Result<(), FakeDbError> {
        self.finish("ROLLBACK")
    }

    /// `path_get_inode`.
    pub fn path_get_inode(&self, path: &[u8]) -> Result<u64, FakeDbError> {
        let path = Self::path(path)?;
        self.with_connection(|connection| {
            let rows = query_rows(
                connection,
                &format!("SELECT inode FROM paths WHERE path={}", blob_literal(path)),
            )?;
            let Some(inode) = first_integer(rows)? else {
                return Ok(0);
            };
            Ok(inode as u64)
        })
    }

    /// `path_read_stat`.
    pub fn path_read_stat(&self, path: &[u8]) -> Result<Option<MetadataRow>, FakeDbError> {
        let inode = self.path_get_inode(path)?;
        if inode == 0 {
            return Ok(None);
        }
        let Some(stat) = self.inode_read_stat_if_exist(inode)? else {
            // The C natural join also hides a malformed dangling path mapping.
            return Ok(None);
        };
        Ok(Some(MetadataRow { inode, stat }))
    }

    /// `path_create`.
    pub fn path_create(&self, path: &[u8], stat: IshStat) -> Result<u64, FakeDbError> {
        let path = Self::path(path)?;
        let stat = stat.to_le_bytes();
        self.with_connection(|connection| {
            execute(
                connection,
                &format!("INSERT INTO stats (stat) VALUES ({})", blob_literal(&stat)),
            )?;
            let inode = connection.last_insert_rowid();
            if inode <= 0 {
                return Err(FakeDbError::InodeExhausted);
            }
            execute(
                connection,
                &format!(
                    "INSERT OR REPLACE INTO paths (path, inode) VALUES ({}, {})",
                    blob_literal(path),
                    inode
                ),
            )?;
            Ok(inode as u64)
        })
    }

    /// `inode_read_stat_if_exist`.
    pub fn inode_read_stat_if_exist(&self, inode: u64) -> Result<Option<IshStat>, FakeDbError> {
        self.with_connection(|connection| {
            let rows = query_rows(
                connection,
                &format!(
                    "SELECT stat FROM stats WHERE inode={}",
                    sqlite_integer(inode)
                ),
            )?;
            let Some(row) = rows.into_iter().next() else {
                return Ok(None);
            };
            let value = one_column(row)?;
            Ok(Some(stat_from_value(&value)?))
        })
    }

    /// Safe form of C's `inode_read_stat_or_die`.
    pub fn inode_read_stat_or_error(&self, inode: u64) -> Result<IshStat, FakeDbError> {
        self.inode_read_stat_if_exist(inode)?
            .ok_or(FakeDbError::MissingPath)
    }

    /// `inode_write_stat`.
    pub fn inode_write_stat(&self, inode: u64, stat: IshStat) -> Result<(), FakeDbError> {
        let stat = stat.to_le_bytes();
        self.with_connection(|connection| {
            // C uses UPDATE ... WHERE inode = ?, so a stale inode is a
            // no-op—not an implicit INSERT into stats.
            execute(
                connection,
                &format!(
                    "UPDATE stats SET stat={} WHERE inode={}",
                    blob_literal(&stat),
                    sqlite_integer(inode)
                ),
            )
        })
    }

    /// `path_link`.
    pub fn path_link(&self, src: &[u8], dst: &[u8]) -> Result<(), FakeDbError> {
        let src = Self::path(src)?;
        let dst = Self::path(dst)?;
        let inode = self.path_get_inode(src)?;
        if inode == 0 {
            return Err(FakeDbError::MissingPath);
        }
        self.with_connection(|connection| {
            execute(
                connection,
                &format!(
                    "INSERT OR REPLACE INTO paths (path, inode) VALUES ({}, {})",
                    blob_literal(dst),
                    sqlite_integer(inode)
                ),
            )
        })
    }

    /// `path_unlink`.
    pub fn path_unlink(&self, path: &[u8]) -> Result<u64, FakeDbError> {
        let path = Self::path(path)?;
        let inode = self.path_get_inode(path)?;
        if inode == 0 {
            return Err(FakeDbError::MissingPath);
        }
        self.with_connection(|connection| {
            execute(
                connection,
                &format!("DELETE FROM paths WHERE path={}", blob_literal(path)),
            )?;
            Ok(inode)
        })
    }

    /// `path_rename` with C's exact path-component boundary rule.
    ///
    /// Upstream's SQLite condition selects `src` and descendants beginning
    /// `src/`; it does not rename a sibling such as `/apple` when `src` is
    /// `/app`. Replacing an existing destination leaves its old stat record
    /// orphaned until a cleanup call, as in C.
    pub fn path_rename(&self, src: &[u8], dst: &[u8]) -> Result<(), FakeDbError> {
        let src = Self::path(src)?;
        let dst = Self::path(dst)?;
        let mut moved = self
            .all_paths()?
            .into_iter()
            .filter(|(old_path, _)| rename_matches(old_path, src))
            .map(|(old_path, inode)| {
                let mut new_path = dst.to_vec();
                new_path.extend_from_slice(&old_path[src.len()..]);
                (old_path, new_path, inode)
            })
            .collect::<Vec<_>>();
        // SQLite's transformed source paths are injective. Sorting keeps the
        // replacement result stable independently of the table scan plan.
        moved.sort_unstable_by(|left, right| left.0.cmp(&right.0));
        self.with_connection(|connection| {
            for (old_path, _, _) in &moved {
                execute(
                    connection,
                    &format!("DELETE FROM paths WHERE path={}", blob_literal(old_path)),
                )?;
            }
            for (_, new_path, inode) in moved {
                execute(
                    connection,
                    &format!(
                        "INSERT OR REPLACE INTO paths (path, inode) VALUES ({}, {})",
                        blob_literal(&new_path),
                        sqlite_integer(inode)
                    ),
                )?;
            }
            Ok(())
        })
    }

    /// The `path_from_inode` query used by `fakefs_open_inode`.
    pub fn paths_for_inode(&self, inode: u64) -> Result<Vec<Vec<u8>>, FakeDbError> {
        let mut paths = self
            .all_paths()?
            .into_iter()
            .filter_map(|(path, mapped_inode)| (mapped_inode == inode).then_some(path))
            .collect::<Vec<_>>();
        // C's `inode_to_path` index returns this query in byte-path order.
        paths.sort_unstable();
        Ok(paths)
    }

    /// The cleanup query prepared by C `fake_db_init`.
    pub fn try_cleanup_inode(&self, inode: u64) -> Result<(), FakeDbError> {
        if self.paths_for_inode(inode)?.is_empty() {
            self.with_connection(|connection| {
                execute(
                    connection,
                    &format!("DELETE FROM stats WHERE inode={}", sqlite_integer(inode)),
                )
            })?;
        }
        Ok(())
    }

    /// Delete every stat record with no path mapping, as C initialization does.
    pub fn clear_orphans(&self) -> Result<(), FakeDbError> {
        let referenced = self
            .all_paths()?
            .into_iter()
            .map(|(_, inode)| inode)
            .collect::<BTreeSet<_>>();
        let stale = self
            .all_stats()?
            .into_iter()
            .map(|(inode, _)| inode)
            .filter(|inode| !referenced.contains(inode))
            .collect::<Vec<_>>();
        self.with_connection(|connection| {
            for inode in stale {
                execute(
                    connection,
                    &format!("DELETE FROM stats WHERE inode={}", sqlite_integer(inode)),
                )?;
            }
            Ok(())
        })
    }

    /// Hash the ordered logical stat and path tables like the local C harness.
    pub fn logical_hash(&self) -> Result<u64, FakeDbError> {
        let mut hash = 0xcbf2_9ce4_8422_2325u64;
        hash_bytes(&mut hash, b"stats");
        for (inode, stat) in self.all_stats()? {
            hash_u64(&mut hash, inode);
            let stat = stat.to_le_bytes();
            hash_u32(&mut hash, stat.len() as u32);
            hash_bytes(&mut hash, &stat);
        }
        hash_bytes(&mut hash, b"paths");
        for (path, inode) in self.all_paths()? {
            hash_u32(&mut hash, path.len() as u32);
            hash_bytes(&mut hash, &path);
            hash_u64(&mut hash, inode);
        }
        Ok(hash)
    }

    fn all_stats(&self) -> Result<Vec<(u64, IshStat)>, FakeDbError> {
        self.with_connection(|connection| {
            let rows = query_rows(connection, "SELECT inode, stat FROM stats")?;
            let mut stats = Vec::with_capacity(rows.len());
            for row in rows {
                if row.len() != 2 {
                    return Err(FakeDbError::Database(
                        "SQLite stats query returned the wrong column count".into(),
                    ));
                }
                stats.push((
                    integer_from_value(&row[0])? as u64,
                    stat_from_value(&row[1])?,
                ));
            }
            stats.sort_unstable_by_key(|(inode, _)| *inode);
            Ok(stats)
        })
    }

    fn all_paths(&self) -> Result<Vec<(Vec<u8>, u64)>, FakeDbError> {
        self.with_connection(|connection| {
            let rows = query_rows(connection, "SELECT path, inode FROM paths")?;
            let mut paths = Vec::with_capacity(rows.len());
            for row in rows {
                if row.len() != 2 {
                    return Err(FakeDbError::Database(
                        "SQLite paths query returned the wrong column count".into(),
                    ));
                }
                paths.push((
                    blob_from_value(&row[0])?.to_vec(),
                    integer_from_value(&row[1])? as u64,
                ));
            }
            paths.sort_unstable_by(|left, right| left.0.cmp(&right.0));
            Ok(paths)
        })
    }
}

impl Drop for FakeDbTransaction {
    fn drop(&mut self) {
        let Some(mut connection) = self.connection.get_mut().take() else {
            return;
        };
        // C callers must explicitly roll back, while Rust's RAII path must not
        // accidentally commit a partially failed metadata operation.
        let _ = connection.execute("ROLLBACK");
        let _ = put_connection(&self.slot, connection);
    }
}

/// Rust equivalent of upstream `fakefs_migrate` for schema versions 0–3.
///
/// The ordered C statement sequence is retained in [`ISH_SCHEMA_MIGRATIONS`],
/// including the temporary v2 delete trigger and its immediate v3 removal.
/// That keeps the transactional statement sequence aligned with upstream even
/// though no regular fakefs operation can observe the trigger in between.
fn migrate_ish_schema(connection: &mut Connection) -> Result<(), FakeDbError> {
    connection
        .execute("PRAGMA foreign_keys=ON")
        .map_err(FakeDbError::database)?;
    let version = sqlite_schema_version(connection)?;
    if version >= SQLITE_SCHEMA_VERSION {
        return Ok(());
    }

    execute(connection, "BEGIN")?;
    let result = (|| {
        let first_migration = usize::try_from(version).map_err(|_| {
            FakeDbError::Database(format!(
                "SQLite user_version {version} cannot index migrations"
            ))
        })?;
        for migration in &ISH_SCHEMA_MIGRATIONS[first_migration..] {
            connection
                .execute_batch(migration)
                .map_err(FakeDbError::database)?;
        }
        execute(
            connection,
            &format!("PRAGMA user_version={SQLITE_SCHEMA_VERSION}"),
        )?;
        execute(connection, "COMMIT")
    })();
    if result.is_err() {
        // C's `EXEC` aborts the process on a migration error. The Rust API
        // returns it, but must leave the owned connection usable instead.
        let _ = connection.execute("ROLLBACK");
    }
    result
}

fn sqlite_schema_version(connection: &Connection) -> Result<u64, FakeDbError> {
    let version = scalar_integer(connection, "PRAGMA user_version")?
        .ok_or_else(|| FakeDbError::Database("SQLite user_version is absent".into()))?;
    u64::try_from(version).map_err(|_| {
        FakeDbError::Database(format!("SQLite user_version {version} cannot be negative"))
    })
}

fn database_path(path: &Path) -> Result<&str, FakeDbError> {
    path.to_str().ok_or(FakeDbError::NonUtf8DatabasePath)
}

fn take_connection(slot: &ConnectionSlot) -> Result<Connection, FakeDbError> {
    slot.borrow_mut()
        .take()
        .ok_or(FakeDbError::TransactionActive)
}

fn put_connection(slot: &ConnectionSlot, database: Connection) -> Result<(), FakeDbError> {
    let mut connection = slot.borrow_mut();
    if connection.is_some() {
        return Err(FakeDbError::Database(
            "attempted to return a second SQLite connection".into(),
        ));
    }
    *connection = Some(database);
    Ok(())
}

fn execute(connection: &mut Connection, sql: &str) -> Result<(), FakeDbError> {
    connection
        .execute(sql)
        .map(|_| ())
        .map_err(FakeDbError::database)
}

fn query_rows(connection: &Connection, sql: &str) -> Result<Vec<Vec<Value>>, FakeDbError> {
    connection
        .query(sql)
        .map(|result| result.rows)
        .map_err(FakeDbError::database)
}

fn scalar_integer(connection: &Connection, sql: &str) -> Result<Option<i64>, FakeDbError> {
    first_integer(query_rows(connection, sql)?)
}

fn first_integer(rows: Vec<Vec<Value>>) -> Result<Option<i64>, FakeDbError> {
    let Some(row) = rows.into_iter().next() else {
        return Ok(None);
    };
    integer_from_value(&one_column(row)?).map(Some)
}

fn one_column(row: Vec<Value>) -> Result<Value, FakeDbError> {
    match row.len() {
        1 => Ok(row.into_iter().next().unwrap()),
        columns => Err(FakeDbError::Database(format!(
            "SQLite query returned {columns} columns instead of one"
        ))),
    }
}

fn integer_from_value(value: &Value) -> Result<i64, FakeDbError> {
    match value {
        Value::Integer(value) => Ok(*value),
        other => Err(FakeDbError::Database(format!(
            "SQLite metadata integer had unexpected value {other:?}"
        ))),
    }
}

fn blob_from_value(value: &Value) -> Result<&[u8], FakeDbError> {
    match value {
        Value::Blob(value) => Ok(value),
        other => Err(FakeDbError::Database(format!(
            "SQLite metadata blob had unexpected value {other:?}"
        ))),
    }
}

fn stat_from_value(value: &Value) -> Result<IshStat, FakeDbError> {
    IshStat::from_le_bytes(blob_from_value(value)?)
}

fn sqlite_integer(value: u64) -> i64 {
    // `fake-db.c` passes inode_t through sqlite3_bind_int64. Preserve that C
    // cast for out-of-range caller input even though normal implicit rowids are
    // positive signed SQLite integers.
    value as i64
}

fn blob_literal(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut literal = String::with_capacity(3 + bytes.len() * 2);
    literal.push_str("X'");
    for byte in bytes {
        literal.push(HEX[usize::from(*byte >> 4)] as char);
        literal.push(HEX[usize::from(*byte & 0x0f)] as char);
    }
    literal.push('\'');
    literal
}

fn rename_matches(path: &[u8], src: &[u8]) -> bool {
    path == src || (path.starts_with(src) && path.get(src.len()) == Some(&b'/'))
}

fn hash_bytes(hash: &mut u64, bytes: &[u8]) {
    for byte in bytes {
        *hash ^= u64::from(*byte);
        *hash = hash.wrapping_mul(0x0000_0100_0000_01b3);
    }
}

fn hash_u32(hash: &mut u64, value: u32) {
    hash_bytes(hash, &value.to_le_bytes());
}

fn hash_u64(hash: &mut u64, value: u64) {
    hash_bytes(hash, &value.to_le_bytes());
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::{AtomicU64, Ordering};

    #[test]
    fn sqlite_metadata_round_trips_transactions_links_and_prefix_renames() {
        let db = FakeDb::open_in_memory().unwrap();
        let first = IshStat {
            mode: 0o100644,
            uid: 1000,
            gid: 100,
            rdev: 0,
        };
        let child = IshStat {
            mode: 0o040755,
            uid: 2000,
            gid: 200,
            rdev: 0,
        };
        let transaction = db.begin_write().unwrap();
        assert_eq!(transaction.path_create(b"/a", first).unwrap(), 1);
        assert_eq!(transaction.path_create(b"/a/child", child).unwrap(), 2);
        transaction.path_link(b"/a", b"/b").unwrap();
        transaction.commit().unwrap();

        let transaction = db.begin_write().unwrap();
        transaction.path_rename(b"/a", b"/x").unwrap();
        transaction.commit().unwrap();
        assert_eq!(db.path_get_inode(b"/a").unwrap(), 0);
        assert_eq!(db.path_read_stat(b"/x").unwrap().unwrap().stat, first);
        assert_eq!(db.path_read_stat(b"/x/child").unwrap().unwrap().stat, child);
        assert_eq!(db.path_get_inode(b"/b").unwrap(), 1);
        assert_eq!(
            db.paths_for_inode(1).unwrap(),
            vec![b"/b".to_vec(), b"/x".to_vec()]
        );

        let transaction = db.begin_write().unwrap();
        assert_eq!(transaction.path_create(b"/rolled", first).unwrap(), 3);
        transaction.rollback().unwrap();
        assert_eq!(db.path_get_inode(b"/rolled").unwrap(), 0);
    }

    #[test]
    fn stat_layout_orphan_cleanup_persistence_and_sqlite_header_are_pure_rust() {
        let stat = IshStat {
            mode: 0x1122_3344,
            uid: 0x5566_7788,
            gid: 0x99aa_bbcc,
            rdev: 0xddee_ff00,
        };
        assert_eq!(
            stat.to_le_bytes(),
            [
                0x44, 0x33, 0x22, 0x11, 0x88, 0x77, 0x66, 0x55, 0xcc, 0xbb, 0xaa, 0x99, 0x00, 0xff,
                0xee, 0xdd,
            ]
        );
        let path = temporary_database_path();
        {
            let db = FakeDb::create(&path).unwrap();
            let inode = db.path_create(b"/one", stat).unwrap();
            assert_eq!(db.path_unlink(b"/one").unwrap(), inode);
            assert_eq!(db.inode_read_stat_if_exist(inode).unwrap(), Some(stat));
            db.try_cleanup_inode(inode).unwrap();
            assert_eq!(db.inode_read_stat_if_exist(inode).unwrap(), None);
            let before_stale_write = db.logical_hash().unwrap();
            db.inode_write_stat(99, stat).unwrap();
            assert_eq!(db.inode_read_stat_if_exist(99).unwrap(), None);
            assert_eq!(db.logical_hash().unwrap(), before_stale_write);
            assert_eq!(
                db.path_create(b"bad\0path", stat),
                Err(FakeDbError::InteriorNulPath)
            );
        }
        assert_eq!(&std::fs::read(&path).unwrap()[..16], b"SQLite format 3\0");
        let reopened = FakeDb::open(&path).unwrap();
        assert_eq!(reopened.path_get_inode(b"/one").unwrap(), 0);
        drop(reopened);
        remove_database_files(&path);
    }

    fn temporary_database_path() -> std::path::PathBuf {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        std::env::temp_dir().join(format!(
            "ish-rs-graphitesql-{}-{}.sqlite",
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
