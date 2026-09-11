//! C-derived regression coverage for the pure-Rust fakefs schema migration.
//!
//! The checked-in reference is generated locally from the unmodified upstream
//! `fs/fake-migrate.c`, not from the Rust port. The native SQLite runtime is
//! only used to build that local C oracle; this test creates and migrates its
//! SQLite version-3 files through the vendored pure-Rust GraphiteSQL engine.
//!
//! Regenerate the C oracle locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_fake_migrate_reference.sh
//! ```

use std::collections::BTreeMap;
use std::fmt::Write as _;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};

use graphitesql::{Connection, Value};
use ish_emu::fake_db::{FakeDb, IshStat};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/fake_migrate_reference.txt"
);

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct MigrationReference {
    legacy_version: u64,
    resulting_version: u64,
    logical_hash: u64,
    inode_index_count: u64,
    delete_trigger_count: u64,
    unlink_hash: Option<u64>,
}

fn decimal(text: &str) -> u64 {
    text.parse()
        .unwrap_or_else(|error| panic!("invalid decimal integer `{text}`: {error}"))
}

fn hexadecimal(text: &str) -> u64 {
    u64::from_str_radix(text, 16)
        .unwrap_or_else(|error| panic!("invalid hexadecimal integer `{text}`: {error}"))
}

fn parse_reference() -> Vec<MigrationReference> {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|error| panic!("cannot read {FIXTURE}: {error}"));
    let mut migrations = BTreeMap::new();
    let mut unlink_hashes = BTreeMap::new();

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<_> = line.split_whitespace().collect();
        match fields.as_slice() {
            ["M", legacy, resulting, hash, index_count, trigger_count] => {
                let legacy_version = decimal(legacy);
                assert!(
                    migrations
                        .insert(
                            legacy_version,
                            MigrationReference {
                                legacy_version,
                                resulting_version: decimal(resulting),
                                logical_hash: hexadecimal(hash),
                                inode_index_count: decimal(index_count),
                                delete_trigger_count: decimal(trigger_count),
                                unlink_hash: None,
                            },
                        )
                        .is_none(),
                    "duplicate C migration record for schema v{legacy_version}"
                );
            }
            ["D", legacy, hash] => {
                let legacy_version = decimal(legacy);
                assert!(
                    unlink_hashes
                        .insert(legacy_version, hexadecimal(hash))
                        .is_none(),
                    "duplicate C unlink record for schema v{legacy_version}"
                );
            }
            _ => panic!("malformed fake-migrate reference record `{line}`"),
        }
    }

    assert_eq!(
        migrations.len(),
        3,
        "C corpus must cover legacy v0, v1, and v2"
    );
    let mut result = Vec::with_capacity(migrations.len());
    for (legacy_version, mut migration) in migrations {
        migration.unlink_hash = unlink_hashes.remove(&legacy_version);
        result.push(migration);
    }
    assert!(
        unlink_hashes.is_empty(),
        "C corpus contains unlink record(s) with no migration: {unlink_hashes:?}"
    );
    assert_eq!(
        result
            .iter()
            .map(|reference| reference.legacy_version)
            .collect::<Vec<_>>(),
        vec![0, 1, 2],
        "C corpus must cover exactly the three historical migrations"
    );
    assert_eq!(
        result
            .iter()
            .filter_map(|reference| reference.unlink_hash.map(|_| reference.legacy_version))
            .collect::<Vec<_>>(),
        vec![2],
        "only the v2 corpus should exercise removal of delete_path"
    );
    result
}

fn blob_literal(bytes: &[u8]) -> String {
    let mut literal = String::from("X'");
    for byte in bytes {
        write!(literal, "{byte:02x}").unwrap();
    }
    literal.push('\'');
    literal
}

/// Create exactly the historical database states consumed by the local C
/// harness. These statements set up input; only the checked-in C fixture
/// defines the expected post-migration behavior.
fn seed_legacy_schema(path: &Path, version: u64) {
    let kept = IshStat {
        mode: 0o100644,
        uid: 1000,
        gid: 100,
        rdev: 0,
    };
    let orphan = IshStat {
        mode: 0o040755,
        uid: 2000,
        gid: 200,
        rdev: 9,
    };
    let mut connection = Connection::create(path.to_str().expect("UTF-8 temporary path")).unwrap();
    let sql = match version {
        0 | 1 => format!(
            "CREATE TABLE meta (id INTEGER UNIQUE DEFAULT 0, db_inode INTEGER);\
             INSERT INTO meta (db_inode) VALUES (0);\
             CREATE TABLE stats (inode INTEGER PRIMARY KEY, stat BLOB);\
             CREATE TABLE paths (path BLOB PRIMARY KEY, inode INTEGER);\
             INSERT INTO stats (inode, stat) VALUES (5, {});\
             INSERT INTO stats (inode, stat) VALUES (6, {});\
             INSERT INTO paths (path, inode) VALUES (X'2f6b657074', 5);\
             INSERT INTO paths (path, inode) VALUES (X'2f64616e676c696e67', 99);\
             PRAGMA user_version={version};",
            blob_literal(&kept.to_le_bytes()),
            blob_literal(&orphan.to_le_bytes()),
        ),
        2 => format!(
            "CREATE TABLE meta (id INTEGER UNIQUE DEFAULT 0, db_inode INTEGER);\
             INSERT INTO meta (db_inode) VALUES (0);\
             CREATE TABLE stats (inode INTEGER PRIMARY KEY, stat BLOB);\
             CREATE TABLE paths (path BLOB PRIMARY KEY, inode INTEGER REFERENCES stats(inode));\
             CREATE INDEX inode_to_path ON paths (inode, path);\
             INSERT INTO stats (inode, stat) VALUES (12, {});\
             INSERT INTO paths (path, inode) VALUES (X'2f7632', 12);\
             CREATE TRIGGER delete_path AFTER DELETE ON paths \
               WHEN NOT EXISTS (SELECT 1 FROM paths WHERE inode = OLD.inode) \
               BEGIN \
                 DELETE FROM stats \
                 WHERE NOT EXISTS (SELECT 1 FROM paths WHERE inode = OLD.inode) \
                   AND inode = OLD.inode; \
               END;\
             PRAGMA user_version=2;",
            blob_literal(&orphan.to_le_bytes()),
        ),
        other => panic!("the C corpus has no legacy schema v{other}"),
    };
    connection.execute_batch(&sql).unwrap();
}

fn scalar(connection: &Connection, sql: &str) -> u64 {
    let rows = connection.query(sql).unwrap().rows;
    let [row] = rows.as_slice() else {
        panic!("expected exactly one scalar row from `{sql}`, got {rows:?}");
    };
    let [Value::Integer(value)] = row.as_slice() else {
        panic!("expected an integer scalar from `{sql}`, got {row:?}");
    };
    (*value)
        .try_into()
        .unwrap_or_else(|_| panic!("expected non-negative scalar from `{sql}`, got {value}"))
}

fn schema_scalar(path: &Path, sql: &str) -> u64 {
    let connection = Connection::open(path.to_str().expect("UTF-8 temporary path")).unwrap();
    scalar(&connection, sql)
}

fn temporary_database_path() -> PathBuf {
    static NEXT: AtomicU64 = AtomicU64::new(0);
    std::env::temp_dir().join(format!(
        "ish-rs-fake-migrate-differential-{}-{}.sqlite",
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

#[test]
fn pure_rust_schema_migration_matches_unmodified_c() {
    for reference in parse_reference() {
        let path = temporary_database_path();
        seed_legacy_schema(&path, reference.legacy_version);

        let db = FakeDb::open(&path).unwrap_or_else(|error| {
            panic!(
                "pure-Rust migration could not open C corpus schema v{}: {error}",
                reference.legacy_version
            )
        });
        assert_eq!(
            db.logical_hash().unwrap(),
            reference.logical_hash,
            "logical post-migration state differs from unmodified C for schema v{}",
            reference.legacy_version
        );
        if let Some(unlink_hash) = reference.unlink_hash {
            assert_eq!(db.path_unlink(b"/v2").unwrap(), 12);
            assert_eq!(
                db.logical_hash().unwrap(),
                unlink_hash,
                "v2 delete-trigger behavior differs from unmodified C"
            );
        }
        drop(db);

        assert_eq!(
            schema_scalar(&path, "PRAGMA user_version"),
            reference.resulting_version,
            "user_version differs from unmodified C for schema v{}",
            reference.legacy_version
        );
        assert_eq!(
            schema_scalar(
                &path,
                "SELECT count(*) FROM sqlite_master \
                 WHERE type = 'index' AND name = 'inode_to_path'"
            ),
            reference.inode_index_count,
            "inode_to_path index differs from unmodified C for schema v{}",
            reference.legacy_version
        );
        assert_eq!(
            schema_scalar(
                &path,
                "SELECT count(*) FROM sqlite_master \
                 WHERE type = 'trigger' AND name = 'delete_path'"
            ),
            reference.delete_trigger_count,
            "delete_path trigger differs from unmodified C for schema v{}",
            reference.legacy_version
        );
        remove_database_files(&path);
    }
}
