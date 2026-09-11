//! Differential replay of unmodified `fs/fake-rebuild.c`.
//!
//! `tools/fake-db-rebuild-dump.c` links the upstream rebuild pass against a
//! fully scripted host filesystem (`fstatat`, `unlinkat` and `linkat` are
//! linker-wrapped), so the fixture's inode map and operation log are
//! deterministic. The Rust side rebuilds the same metadata tables through
//! [`ish_emu::fake_db::FakeDb`], drives [`ish_emu::rebuild::fakefs_rebuild`]
//! with an adapter built from the same script, and compares the host calls, the
//! resulting `stats`/`paths` tables, and the surviving schema.
//!
//! Regenerate the C oracle locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_fake_db_rebuild_reference.sh
//! ```

use std::cell::RefCell;
use std::collections::BTreeMap;
use std::path::Path;

use ish_emu::fake_db::{FakeDb, IshStat};
use ish_emu::rebuild::{fakefs_rebuild, FakefsHost};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/fake_db_rebuild_reference.txt"
);

fn hex(text: &str) -> Vec<u8> {
    assert_eq!(text.len() % 2, 0, "odd-length hex record `{text}`");
    (0..text.len())
        .step_by(2)
        .map(|index| u8::from_str_radix(&text[index..index + 2], 16).unwrap())
        .collect()
}

fn hex64(text: &str) -> u64 {
    u64::from_str_radix(text, 16).unwrap()
}

fn stat(bytes: &[u8]) -> IshStat {
    assert_eq!(bytes.len(), 16, "ish_stat records are 16 bytes");
    IshStat {
        mode: u32::from_le_bytes(bytes[0..4].try_into().unwrap()),
        uid: u32::from_le_bytes(bytes[4..8].try_into().unwrap()),
        gid: u32::from_le_bytes(bytes[8..12].try_into().unwrap()),
        rdev: u32::from_le_bytes(bytes[12..16].try_into().unwrap()),
    }
}

/// The scripted host filesystem, recording every unlink/link the rebuild makes.
struct ScriptedHost {
    inodes: BTreeMap<Vec<u8>, u64>,
    log: RefCell<Vec<String>>,
}

impl ScriptedHost {
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

    fn database_inode(&self, _database: &Path) -> Option<u64> {
        None
    }
}

#[derive(Debug)]
enum Step {
    Create(Vec<u8>),
    Link(Vec<u8>, Vec<u8>),
}

#[derive(Debug, Default)]
struct Fixture {
    stats: Vec<(u64, IshStat)>,
    paths: Vec<(Vec<u8>, u64)>,
    steps: Vec<Step>,
    host: Vec<(Vec<u8>, u64)>,
    before: u64,
    operations: Vec<String>,
    after: u64,
}

impl Fixture {
    /// Number of `link` steps in the corpus.
    fn links(&self) -> usize {
        self.steps
            .iter()
            .filter(|step| matches!(step, Step::Link(..)))
            .count()
    }
}

fn parse_fixture() -> Fixture {
    let contents = std::fs::read_to_string(FIXTURE).unwrap();
    let mut fixture = Fixture::default();
    for line in contents.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let mut fields = line.split_whitespace();
        match fields.next().unwrap() {
            // Ordered by inode, which is also the order the create steps ran.
            "S" => fixture.stats.push((
                hex64(fields.next().unwrap()),
                stat(&hex(fields.next().unwrap())),
            )),
            "P" => fixture
                .paths
                .push((hex(fields.next().unwrap()), hex64(fields.next().unwrap()))),
            "W" => match fields.next().unwrap() {
                "create" => fixture
                    .steps
                    .push(Step::Create(hex(fields.next().unwrap()))),
                "link" => fixture.steps.push(Step::Link(
                    hex(fields.next().unwrap()),
                    hex(fields.next().unwrap()),
                )),
                other => panic!("unknown corpus step `{other}`"),
            },
            "F" => fixture.host.push((
                fields.next().unwrap().as_bytes().to_vec(),
                hex64(fields.next().unwrap()),
            )),
            "B" => fixture.before = hex64(fields.next().unwrap()),
            "L" => fixture
                .operations
                .push(fields.collect::<Vec<_>>().join(" ")),
            "A" => fixture.after = hex64(fields.next().unwrap()),
            "O" => {
                let objects = fields.collect::<Vec<_>>().join(" ");
                assert!(
                    !objects.contains("paths_old") && !objects.contains("stats_old"),
                    "the C rebuild left its scratch tables behind"
                );
            }
            other => panic!("unknown fixture record `{other}`"),
        }
    }
    // Every path was either created (one stat row each) or linked to another.
    assert_eq!(
        fixture.paths.len(),
        fixture.stats.len() + fixture.links(),
        "the corpus links and stat rows do not line up"
    );
    fixture
}

/// Replay the corpus with the same public API calls the C harness mirrors:
/// a create takes the next stat row (inodes are allocated in insertion order),
/// a link shares an existing path's inode.
fn build_database(fixture: &Fixture) -> FakeDb {
    let db = FakeDb::open_in_memory().unwrap();
    let transaction = db.begin_write().unwrap();
    let mut created = 0usize;
    for step in &fixture.steps {
        match step {
            Step::Create(path) => {
                let (inode, stat) = fixture.stats[created];
                let assigned = transaction.path_create(path, stat).unwrap();
                assert_eq!(
                    assigned,
                    inode,
                    "path_create did not allocate the C inode for {}",
                    String::from_utf8_lossy(path)
                );
                created += 1;
            }
            Step::Link(path, alias) => {
                transaction.path_link(path, alias).unwrap();
            }
        }
    }
    transaction.commit().unwrap();
    assert_eq!(created, fixture.stats.len());

    // Both harnesses must have built the same metadata tables.
    let mut expected = fixture.paths.clone();
    expected.sort();
    let mut actual: Vec<(Vec<u8>, u64)> = expected
        .iter()
        .map(|(path, _)| (path.clone(), db.path_get_inode(path).unwrap()))
        .collect();
    actual.sort();
    assert_eq!(
        actual, expected,
        "the corpus tables differ from the fixture"
    );
    assert_eq!(db.logical_hash().unwrap(), fixture.before);
    db
}

#[test]
fn rebuild_matches_the_c_reference() {
    let fixture = parse_fixture();
    let db = build_database(&fixture);
    let host = ScriptedHost {
        inodes: fixture.host.iter().cloned().collect(),
        log: RefCell::new(Vec::new()),
    };

    fakefs_rebuild(&db, &host).unwrap();

    assert_eq!(host.log(), fixture.operations, "host operations differ");
    assert_eq!(
        db.logical_hash().unwrap(),
        fixture.after,
        "rebuilt tables differ"
    );

    // Every path the host still had keeps its guest stat, keyed by the host's
    // current inode; the path the host lost is gone from the metadata.
    for (path, _) in &fixture.paths {
        let host_inode = host
            .inodes
            .get(path.strip_prefix(b"/").unwrap_or(path))
            .copied();
        match host_inode {
            Some(host_inode) => {
                assert_eq!(
                    db.path_get_inode(path).unwrap(),
                    host_inode,
                    "{} was not remapped to its host inode",
                    String::from_utf8_lossy(path)
                );
            }
            None => assert_eq!(
                db.path_get_inode(path).unwrap(),
                0,
                "{} should have been dropped",
                String::from_utf8_lossy(path)
            ),
        }
    }
}

/// The hardlink groups in the corpus must really have been re-linked, and the
/// stat rows must be the ones the old inodes carried.
#[test]
fn the_reference_really_restored_hardlinks() {
    let fixture = parse_fixture();
    assert!(
        fixture
            .operations
            .iter()
            .any(|operation| operation == "link a b"),
        "the corpus did not restore a hardlink: {:?}",
        fixture.operations
    );
    assert!(
        fixture
            .operations
            .iter()
            .filter(|operation| operation.starts_with("link e "))
            .count()
            == 2,
        "the triple hardlink group was not restored twice"
    );
    assert_eq!(fixture.before, 0x7227_d161_8dc2_509f);
    assert_ne!(fixture.before, fixture.after);
}
