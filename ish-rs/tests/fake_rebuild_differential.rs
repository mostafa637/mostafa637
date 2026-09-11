//! C-derived regression coverage for the pure-Rust fakefs inode rebuild.
//!
//! The checked-in fixture is generated locally from unmodified upstream
//! `fs/fake-rebuild.c`, linked only into a C oracle with a deterministic mock
//! `fstatat`/`unlinkat`/`linkat` host. The Rust side exercises the vendored
//! pure-Rust GraphiteSQL engine and the equivalent deterministic host adapter.
//!
//! Regenerate the C oracle locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_fake_rebuild_reference.sh
//! ```

use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};

use graphitesql::{Connection, Value};
use ish_emu::fake_db::FakeDb;
use ish_emu::fake_rebuild::{fix_path, RebuildHost, RebuildReport};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/fake_rebuild_reference.txt"
);
const FNV_OFFSET: u64 = 0xcbf2_9ce4_8422_2325;
const FNV_PRIME: u64 = 0x0000_0100_0000_01b3;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct RebuildReference {
    database_hash: u64,
    host_hash: u64,
    stat_calls: usize,
    unlink_calls: usize,
    link_calls: usize,
    paths_old_count: usize,
    stats_old_count: usize,
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct HostFile {
    path: Vec<u8>,
    exists: bool,
    inode: u64,
    unlink_fails: bool,
}

#[derive(Debug, Default)]
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
        if file.exists {
            Ok(file.inode)
        } else {
            Err(())
        }
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

fn parse_reference() -> RebuildReference {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|error| panic!("cannot read {FIXTURE}: {error}"));
    let mut database_hash = None;
    let mut host_hash = None;
    let mut calls = None;
    let mut tables = None;

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
            ["T", paths_old, stats_old] => {
                assert!(tables
                    .replace((parse_decimal(paths_old), parse_decimal(stats_old)))
                    .is_none());
            }
            _ => panic!("malformed fake-rebuild reference record `{line}`"),
        }
    }

    let (stat_calls, unlink_calls, link_calls) = calls.expect("C oracle call counts");
    let (paths_old_count, stats_old_count) = tables.expect("C oracle temporary-table counts");
    RebuildReference {
        database_hash: database_hash.expect("C oracle database hash"),
        host_hash: host_hash.expect("C oracle host hash"),
        stat_calls,
        unlink_calls,
        link_calls,
        paths_old_count,
        stats_old_count,
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
        "ish-rs-fake-rebuild-differential-{}-{}.sqlite",
        std::process::id(),
        NEXT.fetch_add(1, Ordering::Relaxed)
    ))
}

fn remove_database_files(path: &Path) {
    let path = path.to_str().expect("UTF-8 temporary path");
    for suffix in ["", "-journal", "-wal"] {
        let _ = std::fs::remove_file(format!("{path}{suffix}"));
    }
}

fn table_count(path: &Path, name: &str) -> usize {
    let connection = Connection::open(path.to_str().expect("UTF-8 temporary path")).unwrap();
    let rows = connection
        .query(&format!(
            "SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = '{name}'"
        ))
        .unwrap()
        .rows;
    let [row] = rows.as_slice() else {
        panic!("expected one SQLite table-count row, got {rows:?}");
    };
    let [Value::Integer(count)] = row.as_slice() else {
        panic!("expected an integer SQLite table count, got {row:?}");
    };
    (*count).try_into().unwrap()
}

#[test]
fn pure_rust_rebuild_matches_unmodified_c() {
    assert_eq!(fix_path(b""), b".");
    assert_eq!(fix_path(b"/a"), b"a");
    assert_eq!(fix_path(b"/"), b"");

    let reference = parse_reference();
    let path = temporary_database_path();
    seed_c_oracle_schema(&path);
    let db = FakeDb::open(&path).unwrap();
    let mut host = MockHost::c_oracle_input();

    let report = db.rebuild_with_host(&mut host).unwrap();
    assert_eq!(
        db.logical_hash().unwrap(),
        reference.database_hash,
        "rebuilt pure-Rust SQLite metadata differs from unmodified C"
    );
    assert_eq!(
        host.hash(),
        reference.host_hash,
        "pure-Rust hard-link repair differs from unmodified C"
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
    assert_eq!(table_count(&path, "paths_old"), reference.paths_old_count);
    assert_eq!(table_count(&path, "stats_old"), reference.stats_old_count);
    remove_database_files(&path);
}
