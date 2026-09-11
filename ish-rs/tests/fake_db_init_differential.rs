//! C-derived regression coverage for fake_db_init host-inode integration.
//!
//! The checked-in fixture is generated locally from unmodified upstream
//! `fs/fake-db.c`, `fs/fake-migrate.c`, and `fs/fake-rebuild.c`. Its C oracle
//! uses platform SQLite only to establish the reference behavior; the Rust
//! implementation under test uses the vendored pure-Rust GraphiteSQL engine.
//!
//! Regenerate the C oracle locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_fake_db_init_reference.sh
//! ```

use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};

use graphitesql::{Connection, Value};
use ish_emu::fake_db::FakeDb;
use ish_emu::fake_rebuild::{RebuildHost, RebuildReport};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/fake_db_init_reference.txt"
);
const FNV_OFFSET: u64 = 0xcbf2_9ce4_8422_2325;
const FNV_PRIME: u64 = 0x0000_0100_0000_01b3;
const DATABASE_INODE: u64 = 0x4567;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct InitReference {
    database_hash: u64,
    host_hash: u64,
    stat_calls: usize,
    unlink_calls: usize,
    link_calls: usize,
    unchanged_stat_calls: usize,
    unchanged_unlink_calls: usize,
    unchanged_link_calls: usize,
    meta_matches_database_inode: bool,
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct HostFile {
    path: Vec<u8>,
    exists: bool,
    inode: u64,
    unlink_fails: bool,
}

#[derive(Debug, Clone, Default)]
struct MockHost {
    files: Vec<HostFile>,
    stat_calls: usize,
    unlink_calls: usize,
    link_calls: usize,
}

impl MockHost {
    fn c_oracle_input() -> Self {
        Self {
            files: vec![
                HostFile {
                    path: b".".to_vec(),
                    exists: true,
                    inode: 900,
                    unlink_fails: false,
                },
                HostFile {
                    path: b"a".to_vec(),
                    exists: true,
                    inode: 101,
                    unlink_fails: false,
                },
                HostFile {
                    path: b"b".to_vec(),
                    exists: true,
                    inode: 202,
                    unlink_fails: false,
                },
                HostFile {
                    path: b"broken".to_vec(),
                    exists: true,
                    inode: 404,
                    unlink_fails: true,
                },
                HostFile {
                    path: b"c".to_vec(),
                    exists: true,
                    inode: 303,
                    unlink_fails: false,
                },
                HostFile {
                    path: b"d".to_vec(),
                    exists: true,
                    inode: 606,
                    unlink_fails: false,
                },
                HostFile {
                    path: b"missing".to_vec(),
                    exists: false,
                    inode: 0,
                    unlink_fails: false,
                },
            ],
            ..Self::default()
        }
    }

    fn file(&self, path: &[u8]) -> Result<&HostFile, ()> {
        self.files.iter().find(|file| file.path == path).ok_or(())
    }

    fn file_mut(&mut self, path: &[u8]) -> Result<&mut HostFile, ()> {
        self.files
            .iter_mut()
            .find(|file| file.path == path)
            .ok_or(())
    }

    fn hash(&self) -> u64 {
        let mut hash = FNV_OFFSET;
        for file in &self.files {
            hash_bytes(&mut hash, &file.path);
            hash_byte(&mut hash, 0);
            hash_byte(&mut hash, u8::from(file.exists));
            hash_bytes(&mut hash, &file.inode.to_le_bytes());
        }
        hash
    }
}

impl RebuildHost for MockHost {
    type Error = ();

    fn inode_for_path(&mut self, path: &[u8]) -> Result<u64, Self::Error> {
        self.stat_calls += 1;
        let file = self.file(path)?;
        file.exists.then_some(file.inode).ok_or(())
    }

    fn unlink_path(&mut self, path: &[u8]) -> Result<(), Self::Error> {
        self.unlink_calls += 1;
        let file = self.file_mut(path)?;
        if !file.exists || file.unlink_fails {
            return Err(());
        }
        file.exists = false;
        Ok(())
    }

    fn link_path(&mut self, source: &[u8], destination: &[u8]) -> Result<(), Self::Error> {
        self.link_calls += 1;
        let source = self.file(source)?;
        if !source.exists {
            return Err(());
        }
        let inode = source.inode;
        let destination = self.file_mut(destination)?;
        if destination.exists {
            return Err(());
        }
        destination.exists = true;
        destination.inode = inode;
        Ok(())
    }
}

fn parse_hexadecimal(text: &str) -> u64 {
    u64::from_str_radix(text, 16)
        .unwrap_or_else(|error| panic!("invalid hexadecimal reference value `{text}`: {error}"))
}

fn parse_decimal(text: &str) -> usize {
    text.parse()
        .unwrap_or_else(|error| panic!("invalid decimal reference value `{text}`: {error}"))
}

fn parse_reference() -> InitReference {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|error| panic!("cannot read {FIXTURE}: {error}"));
    let mut database_hash = None;
    let mut host_hash = None;
    let mut calls = None;
    let mut unchanged_calls = None;
    let mut meta_matches_database_inode = None;

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<_> = line.split_whitespace().collect();
        match fields.as_slice() {
            ["D", hash] => {
                assert!(database_hash.replace(parse_hexadecimal(hash)).is_none());
            }
            ["H", hash] => {
                assert!(host_hash.replace(parse_hexadecimal(hash)).is_none());
            }
            ["C", stat, unlink, link] => {
                assert!(calls
                    .replace((
                        parse_decimal(stat),
                        parse_decimal(unlink),
                        parse_decimal(link)
                    ))
                    .is_none());
            }
            ["N", stat, unlink, link] => {
                assert!(unchanged_calls
                    .replace((
                        parse_decimal(stat),
                        parse_decimal(unlink),
                        parse_decimal(link)
                    ))
                    .is_none());
            }
            ["M", matches] => {
                assert!(meta_matches_database_inode
                    .replace(parse_decimal(matches) != 0)
                    .is_none());
            }
            _ => panic!("malformed fake-db-init reference record `{line}`"),
        }
    }

    let (stat_calls, unlink_calls, link_calls) = calls.expect("C oracle mismatch call counts");
    let (unchanged_stat_calls, unchanged_unlink_calls, unchanged_link_calls) =
        unchanged_calls.expect("C oracle matching-inode call counts");
    InitReference {
        database_hash: database_hash.expect("C oracle database hash"),
        host_hash: host_hash.expect("C oracle host hash"),
        stat_calls,
        unlink_calls,
        link_calls,
        unchanged_stat_calls,
        unchanged_unlink_calls,
        unchanged_link_calls,
        meta_matches_database_inode: meta_matches_database_inode.expect("C oracle metadata inode"),
    }
}

fn hash_byte(hash: &mut u64, byte: u8) {
    *hash ^= u64::from(byte);
    *hash = hash.wrapping_mul(FNV_PRIME);
}

fn hash_bytes(hash: &mut u64, bytes: &[u8]) {
    for byte in bytes {
        hash_byte(hash, *byte);
    }
}

fn seed_c_oracle_schema(path: &Path) {
    let mut connection = Connection::create(path.to_str().expect("UTF-8 temporary path")).unwrap();
    connection
        .execute_batch(
            "PRAGMA foreign_keys=OFF;\
             CREATE TABLE meta (id INTEGER UNIQUE DEFAULT 0, db_inode INTEGER);\
             INSERT INTO meta (db_inode) VALUES (0);\
             CREATE TABLE stats (inode INTEGER PRIMARY KEY, stat BLOB);\
             CREATE TABLE paths (path BLOB PRIMARY KEY, inode INTEGER REFERENCES stats(inode));\
             CREATE INDEX inode_to_path ON paths (inode, path);\
             INSERT INTO stats (inode, stat) VALUES (9, X'09000000010000000200000003000000');\
             INSERT INTO stats (inode, stat) VALUES (5, X'05000000060000000700000008000000');\
             INSERT INTO stats (inode, stat) VALUES (7, X'0700000008000000090000000a000000');\
             INSERT INTO paths (path, inode) VALUES (X'', 9);\
             INSERT INTO paths (path, inode) VALUES (X'2f61', 5);\
             INSERT INTO paths (path, inode) VALUES (X'2f62', 5);\
             INSERT INTO paths (path, inode) VALUES (X'2f62726f6b656e', 5);\
             INSERT INTO paths (path, inode) VALUES (X'2f63', 6);\
             INSERT INTO paths (path, inode) VALUES (X'2f64', 6);\
             INSERT INTO paths (path, inode) VALUES (X'2f6d697373696e67', 7);\
             PRAGMA user_version=3;\
             PRAGMA foreign_keys=ON;",
        )
        .unwrap();
}

fn temporary_database_path() -> PathBuf {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    std::env::temp_dir().join(format!(
        "ish-rs-fake-db-init-differential-{}-{}.sqlite",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ))
}

fn stored_database_inode(path: &Path) -> u64 {
    let connection = Connection::open(path.to_str().expect("UTF-8 temporary path")).unwrap();
    let rows = connection.query("SELECT db_inode FROM meta").unwrap().rows;
    let [row] = rows.as_slice() else {
        panic!("expected one metadata inode row, got {rows:?}");
    };
    let [Value::Integer(inode)] = row.as_slice() else {
        panic!("expected an integer metadata inode, got {row:?}");
    };
    *inode as u64
}

fn remove_database_files(path: &Path) {
    let path = path.to_str().expect("UTF-8 temporary path");
    for suffix in ["", "-journal", "-wal"] {
        let _ = std::fs::remove_file(format!("{path}{suffix}"));
    }
}

#[test]
fn pure_rust_fake_db_init_matches_unmodified_c() {
    let reference = parse_reference();
    let path = temporary_database_path();
    seed_c_oracle_schema(&path);
    let mut host = MockHost::c_oracle_input();

    let (db, initialization) =
        FakeDb::open_with_host_inode(&path, DATABASE_INODE, &mut host).unwrap();
    let report = initialization
        .rebuild
        .expect("mismatched metadata inode rebuild");
    assert_eq!(
        db.logical_hash().unwrap(),
        reference.database_hash,
        "initialized pure-Rust metadata differs from unmodified C fake_db_init"
    );
    assert_eq!(
        host.hash(),
        reference.host_hash,
        "pure-Rust fake_db_init hard-link repair differs from unmodified C"
    );
    assert_eq!(host.stat_calls, reference.stat_calls);
    assert_eq!(host.unlink_calls, reference.unlink_calls);
    assert_eq!(host.link_calls, reference.link_calls);
    assert_eq!(
        report,
        RebuildReport {
            paths_scanned: 7,
            paths_skipped_missing_host: 1,
            paths_skipped_missing_stat: 2,
            hardlink_repairs_attempted: 3,
            ignored_unlink_failures: 1,
            ignored_link_failures: 1,
            metadata_paths_written: 4,
        }
    );

    drop(db);
    let mut unchanged_host = host.clone();
    unchanged_host.stat_calls = 0;
    unchanged_host.unlink_calls = 0;
    unchanged_host.link_calls = 0;
    let (db, unchanged_initialization) =
        FakeDb::open_with_host_inode(&path, DATABASE_INODE, &mut unchanged_host).unwrap();
    assert!(unchanged_initialization.rebuild.is_none());
    assert_eq!(unchanged_host.stat_calls, reference.unchanged_stat_calls);
    assert_eq!(
        unchanged_host.unlink_calls,
        reference.unchanged_unlink_calls
    );
    assert_eq!(unchanged_host.link_calls, reference.unchanged_link_calls);

    drop(db);
    assert_eq!(
        stored_database_inode(&path) == DATABASE_INODE,
        reference.meta_matches_database_inode,
        "pure-Rust fake_db_init did not persist the current database inode"
    );
    remove_database_files(&path);
}
