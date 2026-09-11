//! Differential replay of unmodified `fs/fake-db.c`.
//!
//! `tools/fake-db-dump.c` links the upstream C metadata implementation to its
//! local SQLite runtime, then records every return, metadata read and hash of
//! ordered logical tables. The Rust side uses the vendored pure-Rust
//! SQLite-3-compatible `graphitesql` library instead. Matching this fixture
//! therefore checks an independently implemented Rust SQLite engine rather
//! than merely comparing two calls to native SQLite.
//!
//! Regenerate the C oracle locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_fake_db_reference.sh
//! ```

use ish_emu::fake_db::{FakeDb, FakeDbTransaction, IshStat, MetadataRow};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/fake_db_reference.txt"
);

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct ReadValue {
    exists: bool,
    inode: u64,
    stat: IshStat,
}

#[derive(Debug)]
struct Pending {
    description: String,
    result: u64,
    value: Option<ReadValue>,
    saw_return: bool,
    saw_value: bool,
}

fn hex32(text: &str) -> u32 {
    u32::from_str_radix(text, 16).unwrap_or_else(|error| panic!("invalid u32 `{text}`: {error}"))
}

fn hex64(text: &str) -> u64 {
    u64::from_str_radix(text, 16).unwrap_or_else(|error| panic!("invalid u64 `{text}`: {error}"))
}

fn bytes(text: &str) -> Vec<u8> {
    assert_eq!(text.len() % 2, 0, "odd-length path record `{text}`");
    (0..text.len())
        .step_by(2)
        .map(|index| u8::from_str_radix(&text[index..index + 2], 16).unwrap())
        .collect()
}

fn stat(fields: &[&str]) -> IshStat {
    assert_eq!(fields.len(), 4, "expected four ish_stat words: {fields:?}");
    IshStat {
        mode: hex32(fields[0]),
        uid: hex32(fields[1]),
        gid: hex32(fields[2]),
        rdev: hex32(fields[3]),
    }
}

fn path_list_hash(paths: &[Vec<u8>]) -> u64 {
    let mut hash = 0xcbf2_9ce4_8422_2325u64;
    for path in paths {
        for byte in (path.len() as u32).to_le_bytes().iter().chain(path) {
            hash ^= u64::from(*byte);
            hash = hash.wrapping_mul(0x0000_0100_0000_01b3);
        }
    }
    hash
}

fn read_value(row: Option<MetadataRow>) -> ReadValue {
    match row {
        Some(row) => ReadValue {
            exists: true,
            inode: row.inode,
            stat: row.stat,
        },
        None => ReadValue {
            exists: false,
            inode: 0,
            stat: IshStat::default(),
        },
    }
}

// C keeps a transaction open over several fixture records. Route every
// primitive to that pure-Rust SQLite transaction when one exists; otherwise
// use FakeDb's autocommit counterpart.
macro_rules! metadata_call {
    ($active:expr, $db:expr, $method:ident($($argument:expr),* $(,)?)) => {{
        match $active.as_ref() {
            Some(transaction) => transaction.$method($($argument),*),
            None => $db.$method($($argument),*),
        }
    }};
}

fn run_operation(fields: &[&str], db: &FakeDb, active: &mut Option<FakeDbTransaction>) -> Pending {
    assert!(fields.len() >= 2, "short operation record: {fields:?}");
    let description = fields[1..].join(" ");
    let (result, value) = match fields[1] {
        "B" => {
            assert_eq!(fields.len(), 3);
            assert!(active.is_none(), "C fixture attempted a nested transaction");
            *active = Some(match fields[2] {
                "0" => db.begin_read().unwrap(),
                "1" => db.begin_write().unwrap(),
                invalid => panic!("invalid begin mode `{invalid}`"),
            });
            (0, None)
        }
        "F" => {
            assert_eq!(fields.len(), 3);
            let transaction = active.take().expect("finish without an active transaction");
            match fields[2] {
                "0" => transaction.rollback().unwrap(),
                "1" => transaction.commit().unwrap(),
                invalid => panic!("invalid finish mode `{invalid}`"),
            }
            (0, None)
        }
        "C" => {
            assert_eq!(fields.len(), 7);
            (
                metadata_call!(
                    active,
                    db,
                    path_create(&bytes(fields[2]), stat(&fields[3..7]))
                )
                .unwrap(),
                None,
            )
        }
        "G" => {
            assert_eq!(fields.len(), 3);
            (
                metadata_call!(active, db, path_get_inode(&bytes(fields[2]))).unwrap(),
                None,
            )
        }
        "Q" => {
            assert_eq!(fields.len(), 3);
            (
                path_list_hash(
                    &metadata_call!(active, db, paths_for_inode(hex64(fields[2]))).unwrap(),
                ),
                None,
            )
        }
        "P" => {
            assert_eq!(fields.len(), 3);
            (
                0,
                Some(read_value(
                    metadata_call!(active, db, path_read_stat(&bytes(fields[2]))).unwrap(),
                )),
            )
        }
        "I" => {
            assert_eq!(fields.len(), 3);
            let inode = hex64(fields[2]);
            let value = match metadata_call!(active, db, inode_read_stat_if_exist(inode)).unwrap() {
                Some(stat) => ReadValue {
                    exists: true,
                    inode,
                    stat,
                },
                // The C harness retains the requested inode in its record when
                // inode_read_stat_if_exist returns false.
                None => ReadValue {
                    exists: false,
                    inode,
                    stat: IshStat::default(),
                },
            };
            (0, Some(value))
        }
        "W" => {
            assert_eq!(fields.len(), 7);
            metadata_call!(
                active,
                db,
                inode_write_stat(hex64(fields[2]), stat(&fields[3..7]))
            )
            .unwrap();
            (0, None)
        }
        "L" => {
            assert_eq!(fields.len(), 4);
            metadata_call!(active, db, path_link(&bytes(fields[2]), &bytes(fields[3]))).unwrap();
            (0, None)
        }
        "U" => {
            assert_eq!(fields.len(), 3);
            (
                metadata_call!(active, db, path_unlink(&bytes(fields[2]))).unwrap(),
                None,
            )
        }
        "N" => {
            assert_eq!(fields.len(), 4);
            metadata_call!(
                active,
                db,
                path_rename(&bytes(fields[2]), &bytes(fields[3]))
            )
            .unwrap();
            (0, None)
        }
        "X" => {
            assert_eq!(fields.len(), 3);
            metadata_call!(active, db, try_cleanup_inode(hex64(fields[2]))).unwrap();
            (0, None)
        }
        "A" => {
            assert_eq!(fields.len(), 2);
            metadata_call!(active, db, clear_orphans()).unwrap();
            (0, None)
        }
        other => panic!("unknown fake-db operation `{other}`"),
    };
    Pending {
        description,
        result,
        value,
        saw_return: false,
        saw_value: false,
    }
}

#[test]
fn pure_rust_fake_db_matches_the_c_sqlite_metadata_reference() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|error| panic!("cannot read {FIXTURE}: {error}"));
    let db = FakeDb::open_in_memory().expect("pure-Rust SQLite opens an in-memory fakefs database");
    let mut active: Option<FakeDbTransaction> = None;
    let mut pending: Option<Pending> = None;
    let mut operations = 0usize;
    let mut reads = 0usize;
    let mut snapshots = 0usize;

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split(' ').collect();
        match fields[0] {
            "O" => {
                assert!(
                    pending.is_none(),
                    "operation before its preceding state snapshot"
                );
                pending = Some(run_operation(&fields, &db, &mut active));
            }
            "R" => {
                assert_eq!(fields.len(), 2, "malformed C return: {line}");
                let operation = pending.as_mut().expect("return without a C operation");
                assert!(
                    !operation.saw_return,
                    "duplicate return for {}",
                    operation.description
                );
                assert_eq!(
                    operation.result,
                    hex64(fields[1]),
                    "{}: C metadata return",
                    operation.description
                );
                operation.saw_return = true;
            }
            "V" => {
                assert_eq!(fields.len(), 7, "malformed C metadata read: {line}");
                let operation = pending
                    .as_mut()
                    .expect("metadata read without an operation");
                assert!(
                    operation.saw_return,
                    "metadata read before return for {}",
                    operation.description
                );
                assert!(
                    !operation.saw_value,
                    "duplicate metadata read for {}",
                    operation.description
                );
                let expected = ReadValue {
                    exists: match fields[1] {
                        "0" => false,
                        "1" => true,
                        invalid => panic!("invalid C exists field `{invalid}`"),
                    },
                    inode: hex64(fields[2]),
                    stat: stat(&fields[3..7]),
                };
                assert_eq!(
                    operation.value,
                    Some(expected),
                    "{}: C metadata read",
                    operation.description
                );
                operation.saw_value = true;
                reads += 1;
            }
            "S" => {
                assert_eq!(fields.len(), 2, "malformed C state hash: {line}");
                if let Some(operation) = pending.take() {
                    assert!(
                        operation.saw_return,
                        "missing return for {}",
                        operation.description
                    );
                    assert_eq!(
                        operation.value.is_some(),
                        operation.saw_value,
                        "missing or unexpected metadata read for {}",
                        operation.description
                    );
                    operations += 1;
                }
                let hash = match active.as_ref() {
                    Some(transaction) => transaction.logical_hash().unwrap(),
                    None => db.logical_hash().unwrap(),
                };
                assert_eq!(
                    hash,
                    hex64(fields[1]),
                    "logical metadata state after operation {operations}"
                );
                snapshots += 1;
            }
            other => panic!("unknown fake-db fixture record `{other}`: {line}"),
        }
    }

    assert!(active.is_none(), "fixture ended with an active transaction");
    assert!(
        pending.is_none(),
        "fixture ended before an operation's state snapshot"
    );
    assert_eq!(operations, 40);
    assert_eq!(reads, 9);
    assert_eq!(snapshots, operations + 1);
    println!(
        "{operations} pure-Rust metadata operations and {snapshots} C state snapshots matched"
    );
}
