//! Pure-Rust fake-filesystem metadata database.
//!
//! iSH's `fs/fake-db.c` stores guest-visible inode metadata separately from
//! host filesystem metadata.  This module preserves that API's observable
//! rules—BLOB-like byte paths, inode allocation, links, prefix renames,
//! orphan cleanup and transactions—but uses the vendored [`redb`] embedded
//! database library instead of linking SQLite.
//!
//! `redb` is a pure-Rust ACID B-tree store.  Its source is vendored in
//! `vendor/redb` at v3.1.2 so this crate builds without crates.io access and
//! without a C/C++ database library.  The persistence format is consequently
//! `redb`, not SQLite's on-disk format; the local differential test compares
//! the guest-visible metadata behavior against unmodified iSH C/SQLite.

use std::collections::BTreeSet;
use std::error::Error;
use std::fmt;
use std::path::Path;

use redb::{Database, ReadableDatabase, ReadableTable, TableDefinition, WriteTransaction};

const FORMAT: TableDefinition<&str, u64> = TableDefinition::new("ish_fakefs_format");
const STATS: TableDefinition<u64, &[u8]> = TableDefinition::new("ish_fakefs_stats");
const PATHS: TableDefinition<&[u8], u64> = TableDefinition::new("ish_fakefs_paths");
const FORMAT_VERSION: u64 = 1;

/// An error from the pure-Rust metadata database.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FakeDbError {
    /// The `redb` storage engine rejected an operation.
    Database(String),
    /// A path had an interior NUL, which C's `strlen` interface cannot carry.
    InteriorNulPath,
    /// A metadata record was not the 16-byte `struct ish_stat` layout.
    InvalidStatBlob {
        /// Actual record length.
        len: usize,
    },
    /// `path_link`, `path_unlink` or a required inode lookup was absent.
    MissingPath,
    /// The database was opened with a newer incompatible metadata format.
    UnsupportedFormat {
        /// Version found in the database.
        found: u64,
    },
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
            Self::Database(message) => write!(formatter, "fakefs redb error: {message}"),
            Self::InteriorNulPath => formatter.write_str("fakefs path contains an interior NUL"),
            Self::InvalidStatBlob { len } => {
                write!(
                    formatter,
                    "fakefs stat record has {len} bytes instead of 16"
                )
            }
            Self::MissingPath => formatter.write_str("fakefs metadata path does not exist"),
            Self::UnsupportedFormat { found } => {
                write!(
                    formatter,
                    "unsupported fakefs metadata format version {found}"
                )
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

/// Persistent, pure-Rust fake filesystem metadata storage.
///
/// `create` and `open` use a redb file rather than an upstream `meta.db`
/// SQLite file.  The metadata operations intentionally keep iSH's behavior;
/// filesystem migration/import of old SQLite files belongs to the future fake
/// filesystem layer.
pub struct FakeDb {
    database: Database,
}

impl FakeDb {
    /// Create a redb-backed metadata database at `path`, or open it if it is
    /// already a valid redb file.  The iSH metadata tables are created once.
    pub fn create(path: impl AsRef<Path>) -> Result<Self, FakeDbError> {
        let database = Database::create(path).map_err(FakeDbError::database)?;
        let db = Self { database };
        db.initialize_or_validate()?;
        db.clear_orphans()?;
        Ok(db)
    }

    /// Open an existing redb-backed fakefs metadata database.
    pub fn open(path: impl AsRef<Path>) -> Result<Self, FakeDbError> {
        let database = Database::open(path).map_err(FakeDbError::database)?;
        let db = Self { database };
        db.validate_format()?;
        db.clear_orphans()?;
        Ok(db)
    }

    /// Create an in-memory pure-Rust database for a caller or differential
    /// test.  No native SQLite library is loaded by this method.
    pub fn open_in_memory() -> Result<Self, FakeDbError> {
        let database = Database::builder()
            .create_with_backend(redb::backends::InMemoryBackend::new())
            .map_err(FakeDbError::database)?;
        let db = Self { database };
        db.initialize_or_validate()?;
        Ok(db)
    }

    fn initialize_or_validate(&self) -> Result<(), FakeDbError> {
        let transaction = self.database.begin_write().map_err(FakeDbError::database)?;
        {
            let mut format = transaction
                .open_table(FORMAT)
                .map_err(FakeDbError::database)?;
            let version = {
                let value = format.get("version").map_err(FakeDbError::database)?;
                value.map(|value| value.value())
            };
            match version {
                Some(FORMAT_VERSION) => {}
                Some(found) => return Err(FakeDbError::UnsupportedFormat { found }),
                None => {
                    format
                        .insert("version", &FORMAT_VERSION)
                        .map_err(FakeDbError::database)?;
                }
            }
        }
        // Opening a table is the redb equivalent of creating the schema.  Keep
        // the handles scoped separately: redb enforces one open handle per
        // table in one transaction.
        {
            let _stats = transaction
                .open_table(STATS)
                .map_err(FakeDbError::database)?;
        }
        {
            let _paths = transaction
                .open_table(PATHS)
                .map_err(FakeDbError::database)?;
        }
        transaction.commit().map_err(FakeDbError::database)
    }

    fn validate_format(&self) -> Result<(), FakeDbError> {
        let transaction = self.database.begin_read().map_err(FakeDbError::database)?;
        let format = transaction
            .open_table(FORMAT)
            .map_err(FakeDbError::database)?;
        let version = format
            .get("version")
            .map_err(FakeDbError::database)?
            .map(|value| value.value());
        match version {
            Some(FORMAT_VERSION) => Ok(()),
            Some(found) => Err(FakeDbError::UnsupportedFormat { found }),
            None => Err(FakeDbError::UnsupportedFormat { found: 0 }),
        }
    }

    fn begin_compatible_transaction(&self) -> Result<FakeDbTransaction, FakeDbError> {
        // C's `begin deferred` allows fakefs_setattr to read a stat and then
        // promote the same transaction to a write. A redb WriteTransaction is
        // used for both C begin modes so that observable sequence remains
        // legal and atomic rather than rejecting that C-supported promotion.
        Ok(FakeDbTransaction {
            transaction: self.database.begin_write().map_err(FakeDbError::database)?,
        })
    }

    /// Start the C `db_begin_read` compatibility transaction.
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
                // Dropping an unfinished redb WriteTransaction aborts it.
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

    /// Remove all stat records not referenced by any path, like `fake_db_init`.
    pub fn clear_orphans(&self) -> Result<(), FakeDbError> {
        self.autocommit(|transaction| transaction.clear_orphans())
    }

    /// Hash logical metadata exactly as the local C reference harness does.
    pub fn logical_hash(&self) -> Result<u64, FakeDbError> {
        self.autocommit(|transaction| transaction.logical_hash())
    }
}

/// A transaction over [`FakeDb`] metadata.
///
/// It wraps redb's pure-Rust ACID write transaction. Both C `begin deferred`
/// and `begin immediate` map here because the former can be promoted to a
/// writer by iSH's own fake filesystem code.
pub struct FakeDbTransaction {
    transaction: WriteTransaction,
}

impl FakeDbTransaction {
    fn path(path: &[u8]) -> Result<&[u8], FakeDbError> {
        if path.contains(&0) {
            Err(FakeDbError::InteriorNulPath)
        } else {
            Ok(path)
        }
    }

    /// Finish a C `db_commit` equivalent.
    pub fn commit(self) -> Result<(), FakeDbError> {
        self.transaction.commit().map_err(FakeDbError::database)
    }

    /// Finish a C `db_rollback` equivalent.
    pub fn rollback(self) -> Result<(), FakeDbError> {
        self.transaction.abort().map_err(FakeDbError::database)
    }

    /// `path_get_inode`, including C's zero sentinel for a missing path.
    pub fn path_get_inode(&self, path: &[u8]) -> Result<u64, FakeDbError> {
        let path = Self::path(path)?;
        let paths = self
            .transaction
            .open_table(PATHS)
            .map_err(FakeDbError::database)?;
        let inode = {
            let value = paths.get(path).map_err(FakeDbError::database)?;
            value.map(|inode| inode.value())
        };
        Ok(inode.unwrap_or(0))
    }

    /// `path_read_stat`.
    pub fn path_read_stat(&self, path: &[u8]) -> Result<Option<MetadataRow>, FakeDbError> {
        let path = Self::path(path)?;
        let inode = {
            let paths = self
                .transaction
                .open_table(PATHS)
                .map_err(FakeDbError::database)?;
            let value = paths.get(path).map_err(FakeDbError::database)?;
            value.map(|value| value.value())
        };
        let Some(inode) = inode else {
            return Ok(None);
        };
        let stats = self
            .transaction
            .open_table(STATS)
            .map_err(FakeDbError::database)?;
        let Some(stat) = stats.get(&inode).map_err(FakeDbError::database)? else {
            // C's NATURAL JOIN hides dangling path entries rather than yielding
            // an inode with invalid stat data.
            return Ok(None);
        };
        Ok(Some(MetadataRow {
            inode,
            stat: IshStat::from_le_bytes(stat.value())?,
        }))
    }

    /// `path_create`, including SQLite's positive implicit rowid allocation.
    pub fn path_create(&self, path: &[u8], stat: IshStat) -> Result<u64, FakeDbError> {
        let path = Self::path(path)?;
        let inode = {
            let stats = self
                .transaction
                .open_table(STATS)
                .map_err(FakeDbError::database)?;
            let last = stats.last().map_err(FakeDbError::database)?;
            match last {
                Some((last, _)) => last
                    .value()
                    .checked_add(1)
                    .ok_or(FakeDbError::InodeExhausted)?,
                None => 1,
            }
        };
        let bytes = stat.to_le_bytes();
        {
            let mut stats = self
                .transaction
                .open_table(STATS)
                .map_err(FakeDbError::database)?;
            stats
                .insert(&inode, bytes.as_slice())
                .map_err(FakeDbError::database)?;
        }
        {
            let mut paths = self
                .transaction
                .open_table(PATHS)
                .map_err(FakeDbError::database)?;
            // `Table::insert` replaces the old path mapping, matching SQLite
            // INSERT OR REPLACE while deliberately leaving an old stat orphan.
            paths.insert(path, &inode).map_err(FakeDbError::database)?;
        }
        Ok(inode)
    }

    /// `inode_read_stat_if_exist`.
    pub fn inode_read_stat_if_exist(&self, inode: u64) -> Result<Option<IshStat>, FakeDbError> {
        let stats = self
            .transaction
            .open_table(STATS)
            .map_err(FakeDbError::database)?;
        let stat = stats.get(&inode).map_err(FakeDbError::database)?;
        match stat {
            Some(stat) => Ok(Some(IshStat::from_le_bytes(stat.value())?)),
            None => Ok(None),
        }
    }

    /// Safe form of C's `inode_read_stat_or_die`.
    pub fn inode_read_stat_or_error(&self, inode: u64) -> Result<IshStat, FakeDbError> {
        self.inode_read_stat_if_exist(inode)?
            .ok_or(FakeDbError::MissingPath)
    }

    /// `inode_write_stat`.
    pub fn inode_write_stat(&self, inode: u64, stat: IshStat) -> Result<(), FakeDbError> {
        let bytes = stat.to_le_bytes();
        let mut stats = self
            .transaction
            .open_table(STATS)
            .map_err(FakeDbError::database)?;
        let exists = {
            let value = stats.get(&inode).map_err(FakeDbError::database)?;
            value.is_some()
        };
        // C uses UPDATE ... WHERE inode = ?, so a stale inode is a no-op—not
        // an implicit INSERT into stats.
        if exists {
            stats
                .insert(&inode, bytes.as_slice())
                .map_err(FakeDbError::database)?;
        }
        Ok(())
    }

    /// `path_link`.
    pub fn path_link(&self, src: &[u8], dst: &[u8]) -> Result<(), FakeDbError> {
        let src = Self::path(src)?;
        let dst = Self::path(dst)?;
        let inode = self.path_get_inode(src)?;
        if inode == 0 {
            return Err(FakeDbError::MissingPath);
        }
        let mut paths = self
            .transaction
            .open_table(PATHS)
            .map_err(FakeDbError::database)?;
        paths.insert(dst, &inode).map_err(FakeDbError::database)?;
        Ok(())
    }

    /// `path_unlink`.
    pub fn path_unlink(&self, path: &[u8]) -> Result<u64, FakeDbError> {
        let path = Self::path(path)?;
        let inode = self.path_get_inode(path)?;
        if inode == 0 {
            return Err(FakeDbError::MissingPath);
        }
        let mut paths = self
            .transaction
            .open_table(PATHS)
            .map_err(FakeDbError::database)?;
        paths.remove(path).map_err(FakeDbError::database)?;
        Ok(inode)
    }

    /// `path_rename` with C's exact path-component boundary rule.
    ///
    /// Upstream's SQLite condition selects `src` and descendants beginning
    /// `src/`; it does not rename a sibling such as `/apple` when `src` is
    /// `/app`. `insert` replacement preserves `UPDATE OR REPLACE` behavior
    /// for an existing destination while stat records remain orphaned until a
    /// cleanup call, as in C.
    pub fn path_rename(&self, src: &[u8], dst: &[u8]) -> Result<(), FakeDbError> {
        let src = Self::path(src)?;
        let dst = Self::path(dst)?;
        let moved = {
            let paths = self
                .transaction
                .open_table(PATHS)
                .map_err(FakeDbError::database)?;
            let mut moved = Vec::new();
            for row in paths.iter().map_err(FakeDbError::database)? {
                let (old_path, inode) = row.map_err(FakeDbError::database)?;
                let old_path = old_path.value();
                if !rename_matches(old_path, src) {
                    continue;
                }
                let mut new_path = dst.to_vec();
                new_path.extend_from_slice(&old_path[src.len()..]);
                moved.push((old_path.to_vec(), new_path, inode.value()));
            }
            moved
        };
        let mut paths = self
            .transaction
            .open_table(PATHS)
            .map_err(FakeDbError::database)?;
        // SQLite's transformed source paths are injective. Remove all source
        // rows before replacing collisions at destinations, which gives the
        // same result for the legal rename inputs fed by fake.c.
        for (old_path, _, _) in &moved {
            paths
                .remove(old_path.as_slice())
                .map_err(FakeDbError::database)?;
        }
        for (_, new_path, inode) in moved {
            paths
                .insert(new_path.as_slice(), &inode)
                .map_err(FakeDbError::database)?;
        }
        Ok(())
    }

    /// The `path_from_inode` query used by `fakefs_open_inode`.
    pub fn paths_for_inode(&self, inode: u64) -> Result<Vec<Vec<u8>>, FakeDbError> {
        let paths = self
            .transaction
            .open_table(PATHS)
            .map_err(FakeDbError::database)?;
        let mut result = Vec::new();
        for row in paths.iter().map_err(FakeDbError::database)? {
            let (path, mapped_inode) = row.map_err(FakeDbError::database)?;
            if mapped_inode.value() == inode {
                result.push(path.value().to_vec());
            }
        }
        Ok(result)
    }

    /// The cleanup query prepared by C `fake_db_init`.
    pub fn try_cleanup_inode(&self, inode: u64) -> Result<(), FakeDbError> {
        if self.paths_for_inode(inode)?.is_empty() {
            let mut stats = self
                .transaction
                .open_table(STATS)
                .map_err(FakeDbError::database)?;
            stats.remove(&inode).map_err(FakeDbError::database)?;
        }
        Ok(())
    }

    /// Delete every stat record with no path mapping, as C init does.
    pub fn clear_orphans(&self) -> Result<(), FakeDbError> {
        let referenced = {
            let paths = self
                .transaction
                .open_table(PATHS)
                .map_err(FakeDbError::database)?;
            let mut referenced = BTreeSet::new();
            for row in paths.iter().map_err(FakeDbError::database)? {
                let (_, inode) = row.map_err(FakeDbError::database)?;
                referenced.insert(inode.value());
            }
            referenced
        };
        let stale = {
            let stats = self
                .transaction
                .open_table(STATS)
                .map_err(FakeDbError::database)?;
            let mut stale = Vec::new();
            for row in stats.iter().map_err(FakeDbError::database)? {
                let (inode, _) = row.map_err(FakeDbError::database)?;
                if !referenced.contains(&inode.value()) {
                    stale.push(inode.value());
                }
            }
            stale
        };
        let mut stats = self
            .transaction
            .open_table(STATS)
            .map_err(FakeDbError::database)?;
        for inode in stale {
            stats.remove(&inode).map_err(FakeDbError::database)?;
        }
        Ok(())
    }

    /// Hash the ordered logical stat and path tables like the local C harness.
    pub fn logical_hash(&self) -> Result<u64, FakeDbError> {
        let mut hash = 0xcbf2_9ce4_8422_2325u64;
        hash_bytes(&mut hash, b"stats");
        {
            let stats = self
                .transaction
                .open_table(STATS)
                .map_err(FakeDbError::database)?;
            for row in stats.iter().map_err(FakeDbError::database)? {
                let (inode, stat) = row.map_err(FakeDbError::database)?;
                hash_u64(&mut hash, inode.value());
                let stat = stat.value();
                hash_u32(&mut hash, stat.len() as u32);
                hash_bytes(&mut hash, stat);
            }
        }
        hash_bytes(&mut hash, b"paths");
        {
            let paths = self
                .transaction
                .open_table(PATHS)
                .map_err(FakeDbError::database)?;
            for row in paths.iter().map_err(FakeDbError::database)? {
                let (path, inode) = row.map_err(FakeDbError::database)?;
                let path = path.value();
                hash_u32(&mut hash, path.len() as u32);
                hash_bytes(&mut hash, path);
                hash_u64(&mut hash, inode.value());
            }
        }
        Ok(hash)
    }
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
    fn redb_metadata_round_trips_transactions_links_and_prefix_renames() {
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
    fn stat_layout_orphan_cleanup_and_persistence_are_pure_rust() {
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
        let reopened = FakeDb::open(&path).unwrap();
        assert_eq!(reopened.path_get_inode(b"/one").unwrap(), 0);
        drop(reopened);
        std::fs::remove_file(path).unwrap();
    }

    fn temporary_database_path() -> std::path::PathBuf {
        static NEXT: AtomicU64 = AtomicU64::new(0);
        std::env::temp_dir().join(format!(
            "ish-rs-redb-{}-{}.redb",
            std::process::id(),
            NEXT.fetch_add(1, Ordering::Relaxed)
        ))
    }
}
