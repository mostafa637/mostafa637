//! Differential replay of unmodified `fs/fake-migrate.c`.
//!
//! `tools/fake-db-migrate-dump.c` includes the upstream migration file, so the
//! migration SQL in the fixture is the text the C compiler built, and every
//! corpus case is a real ladder run over a database that starts at one of iSH's
//! historical schema generations. The Rust side rebuilds each starting database
//! with the vendored pure-Rust SQLite engine and runs
//! [`ish_emu::migrate::fakefs_migrate`] on it, so a match covers the SQL text,
//! the ladder's ordering rules, the resulting schema objects, the foreign key,
//! and the pre/post behavior of the version 2 `delete_path` trigger.
//!
//! Regenerate the C oracle locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_fake_db_migrate_reference.sh
//! ```

use graphitesql::{Connection, Value};
use ish_emu::migrate::{fakefs_migrate, MIGRATIONS};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/fake_db_migrate_reference.txt"
);

/// The delete probe both sides run; `/c/d` is the only path of its inode, so a
/// live `delete_path` trigger cascades into `stats`.
const PROBE_PATH: &str = "X'2f632f64'";

fn hex(text: &str) -> Vec<u8> {
    assert_eq!(text.len() % 2, 0, "odd-length hex record `{text}`");
    (0..text.len())
        .step_by(2)
        .map(|index| u8::from_str_radix(&text[index..index + 2], 16).unwrap())
        .collect()
}

fn blob_literal(bytes: &[u8]) -> String {
    const DIGITS: &[u8; 16] = b"0123456789abcdef";
    let mut literal = String::with_capacity(3 + bytes.len() * 2);
    literal.push_str("X'");
    for byte in bytes {
        literal.push(DIGITS[usize::from(*byte >> 4)] as char);
        literal.push(DIGITS[usize::from(*byte & 0x0f)] as char);
    }
    literal.push('\'');
    literal
}

fn text(value: &Value) -> String {
    match value {
        Value::Text(value) => value.as_str().to_string(),
        other => panic!("expected text, got {other:?}"),
    }
}

fn integer(value: &Value) -> i64 {
    match value {
        Value::Integer(value) => *value,
        other => panic!("expected integer, got {other:?}"),
    }
}

fn rows(connection: &Connection, sql: &str) -> Vec<Vec<Value>> {
    connection
        .query(sql)
        .unwrap_or_else(|error| panic!("`{sql}` failed: {error}"))
        .rows
}

fn execute_batch(connection: &mut Connection, sql: &str) {
    connection
        .execute_batch(sql)
        .unwrap_or_else(|error| panic!("`{sql}` failed: {error}"));
}

fn user_version(connection: &Connection) -> i64 {
    integer(&rows(connection, "PRAGMA user_version")[0][0])
}

/// `<type>:<name>` for every sqlite_master object, in the C harness's order.
fn objects(connection: &Connection) -> String {
    let mut objects: Vec<(String, String)> = rows(
        connection,
        "SELECT type, name FROM sqlite_master ORDER BY type, name",
    )
    .into_iter()
    .map(|row| (text(&row[0]), text(&row[1])))
    .collect();
    objects.sort();
    objects
        .into_iter()
        .map(|(kind, name)| format!("{kind}:{name}"))
        .collect::<Vec<_>>()
        .join(" ")
}

/// Drop SQLite's implicit `sqlite_autoindex_*` objects.
///
/// Their names are an engine artifact: real SQLite renames the implicit index
/// when `ALTER TABLE paths_new RENAME TO paths` runs, while the vendored pure-
/// Rust engine keeps the creation-time name. Nothing in iSH — and nothing in
/// the guest ABI — can observe that name, so the comparison pins every explicit
/// object (tables, the `inode_to_path` index, triggers) and ignores these.
fn strip_autoindexes(objects: &str) -> String {
    objects
        .split_whitespace()
        .filter(|object| !object.starts_with("index:sqlite_autoindex_"))
        .collect::<Vec<_>>()
        .join(" ")
}

/// `table.from->to` for the fakefs `paths` foreign keys.
fn foreign_keys(connection: &Connection) -> String {
    rows(connection, "PRAGMA foreign_key_list(paths)")
        .into_iter()
        .map(|row| format!("{}.{}->{}", text(&row[2]), text(&row[3]), text(&row[4])))
        .collect::<Vec<_>>()
        .join(" ")
}

/// The C harness's FNV-1a hash of the ordered logical tables.
fn database_hash(connection: &Connection) -> u64 {
    let mut hash = 0xcbf2_9ce4_8422_2325u64;
    let mut push = |bytes: &[u8]| {
        for byte in bytes {
            hash ^= u64::from(*byte);
            hash = hash.wrapping_mul(0x0000_0100_0000_01b3);
        }
    };
    push(b"stats");
    for row in rows(connection, "SELECT inode, stat FROM stats ORDER BY inode") {
        push(&integer(&row[0]).to_le_bytes());
        let Value::Blob(stat) = &row[1] else {
            panic!("stats.stat is not a blob");
        };
        push(&(stat.len() as u32).to_le_bytes());
        push(stat);
    }
    push(b"paths");
    for row in rows(connection, "SELECT path, inode FROM paths ORDER BY path") {
        let Value::Blob(path) = &row[0] else {
            panic!("paths.path is not a blob");
        };
        push(&(path.len() as u32).to_le_bytes());
        push(path);
        push(&integer(&row[1]).to_le_bytes());
    }
    hash
}

/// The historical schema generation `version` wrote for a new filesystem.
fn create_schema(connection: &mut Connection, version: i64) {
    execute_batch(
        connection,
        "CREATE TABLE meta (id integer unique default 0, db_inode integer);\
         INSERT INTO meta (db_inode) VALUES (0);\
         CREATE TABLE stats (inode integer primary key, stat blob);",
    );
    if version < 2 {
        execute_batch(
            connection,
            "CREATE TABLE paths (path blob primary key, inode integer);",
        );
        if version >= 1 {
            execute_batch(
                connection,
                "CREATE INDEX inode_to_path ON paths (inode, path);",
            );
        }
    } else {
        execute_batch(
            connection,
            "CREATE TABLE paths (path blob primary key, inode integer references stats(inode));\
             CREATE INDEX inode_to_path ON paths (inode, path);",
        );
        if version == 2 {
            execute_batch(
                connection,
                "CREATE TRIGGER delete_path AFTER DELETE ON paths \
                 WHEN NOT EXISTS (SELECT 1 FROM paths WHERE inode = old.inode) \
                 BEGIN DELETE FROM stats WHERE NOT EXISTS \
                 (SELECT 1 FROM paths WHERE inode = old.inode) AND inode = old.inode; END;",
            );
        }
    }
    execute_batch(connection, &format!("PRAGMA user_version = {version};"));
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct Row {
    inode: u64,
    stat: Vec<u8>,
}

#[derive(Debug)]
struct Case {
    start: i64,
    before_objects: String,
    before_probe: u64,
    version: i64,
    hash: u64,
    objects: String,
    foreign_keys: String,
    after_probe: u64,
}

struct Corpus {
    migrations: Vec<(usize, Option<Vec<u8>>, bool)>,
    stats: Vec<Row>,
    paths: Vec<(Vec<u8>, u64)>,
    cases: Vec<Case>,
}

fn parse_fixture() -> Corpus {
    let contents = std::fs::read_to_string(FIXTURE).unwrap();
    let mut corpus = Corpus {
        migrations: Vec::new(),
        stats: Vec::new(),
        paths: Vec::new(),
        cases: Vec::new(),
    };
    let mut pending: Option<Case> = None;
    for line in contents.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let mut fields = line.split_whitespace();
        match fields.next().unwrap() {
            "M" => {
                let index: usize = fields.next().unwrap().parse().unwrap();
                let sql = fields.next().unwrap();
                let sql = if sql == "-" { None } else { Some(hex(sql)) };
                let hook = fields.next().unwrap() == "hook";
                assert_eq!(index, corpus.migrations.len());
                corpus.migrations.push((index, sql, hook));
            }
            "S" => {
                let inode = u64::from_str_radix(fields.next().unwrap(), 16).unwrap();
                let stat = hex(fields.next().unwrap());
                assert_eq!(stat.len(), 16, "ish_stat records are 16 bytes");
                corpus.stats.push(Row { inode, stat });
            }
            "P" => {
                let path = hex(fields.next().unwrap());
                let inode = u64::from_str_radix(fields.next().unwrap(), 16).unwrap();
                corpus.paths.push((path, inode));
            }
            "C" => {
                assert!(pending.is_none(), "case records are consecutive");
                pending = Some(Case {
                    start: fields.next().unwrap().parse().unwrap(),
                    before_objects: String::new(),
                    before_probe: 0,
                    version: 0,
                    hash: 0,
                    objects: String::new(),
                    foreign_keys: String::new(),
                    after_probe: 0,
                });
            }
            field => {
                let case = pending.as_mut().expect("case field outside a case");
                let rest: Vec<&str> = fields.collect();
                match field {
                    // `B` repeats the version the start is parsed from.
                    "B" => {}
                    "BO" => case.before_objects = rest.join(" "),
                    "T0" => {
                        case.before_probe = u64::from_str_radix(rest[0], 16).unwrap();
                    }
                    "V" => case.version = rest[0].parse().unwrap(),
                    "H" => case.hash = u64::from_str_radix(rest[0], 16).unwrap(),
                    "O" => case.objects = rest.join(" "),
                    "K" => case.foreign_keys = rest.join(" "),
                    "T1" => {
                        case.after_probe = u64::from_str_radix(rest[0], 16).unwrap();
                        corpus.cases.push(pending.take().unwrap());
                    }
                    other => panic!("unknown fixture record `{other}`"),
                }
            }
        }
    }
    assert!(pending.is_none(), "fixture ended inside a case");
    corpus
}

/// Build the starting database for one corpus case.
fn start_database(corpus: &Corpus, start: i64) -> Connection {
    let mut connection = Connection::open_memory().unwrap();
    create_schema(&mut connection, start);
    for row in &corpus.stats {
        execute_batch(
            &mut connection,
            &format!(
                "INSERT INTO stats (inode, stat) VALUES ({}, {});",
                row.inode,
                blob_literal(&row.stat)
            ),
        );
    }
    for (path, inode) in &corpus.paths {
        execute_batch(
            &mut connection,
            &format!(
                "INSERT INTO paths (path, inode) VALUES ({}, {inode});",
                blob_literal(path)
            ),
        );
    }
    connection
}

fn delete_probe(connection: &mut Connection) -> u64 {
    execute_batch(
        connection,
        &format!("DELETE FROM paths WHERE path = {PROBE_PATH};"),
    );
    database_hash(connection)
}

#[test]
fn migration_table_is_the_c_table() {
    let corpus = parse_fixture();
    assert_eq!(corpus.migrations.len(), MIGRATIONS.len());
    assert_eq!(ish_emu::migrate::SCHEMA_VERSION as usize, MIGRATIONS.len());
    for (index, sql, hook) in &corpus.migrations {
        let migration = &MIGRATIONS[*index];
        match (&migration.sql, sql) {
            (Some(rust), Some(c)) => assert_eq!(
                rust.as_bytes(),
                c.as_slice(),
                "migration {index} SQL text differs from the C table"
            ),
            (None, None) => {}
            other => panic!("migration {index} presence differs: {other:?}"),
        }
        assert_eq!(
            migration.migrate.is_some(),
            *hook,
            "migration {index} hook presence differs from the C table"
        );
    }
}

#[test]
fn migration_ladder_matches_the_c_reference() {
    let corpus = parse_fixture();
    assert!(
        corpus.cases.len() >= 4,
        "expected a broad generation corpus"
    );
    for case in &corpus.cases {
        // The delete probe needs a live database, so the pre-migration probe
        // and the migration run each get their own copy of the corpus.
        let mut before = start_database(&corpus, case.start);
        assert_eq!(
            user_version(&before),
            case.start,
            "case {} starts at the wrong version",
            case.start
        );
        assert_eq!(
            strip_autoindexes(&objects(&before)),
            strip_autoindexes(&case.before_objects),
            "case {} starting schema differs",
            case.start
        );
        assert_eq!(
            delete_probe(&mut before),
            case.before_probe,
            "case {} pre-migration delete probe differs",
            case.start
        );

        let mut connection = start_database(&corpus, case.start);
        fakefs_migrate(&mut connection).unwrap();
        assert_eq!(
            user_version(&connection),
            case.version,
            "case {} user_version differs",
            case.start
        );
        assert_eq!(
            database_hash(&connection),
            case.hash,
            "case {} logical tables differ",
            case.start
        );
        assert_eq!(
            strip_autoindexes(&objects(&connection)),
            strip_autoindexes(&case.objects),
            "case {} schema objects differ",
            case.start
        );
        assert_eq!(
            foreign_keys(&connection),
            case.foreign_keys,
            "case {} paths foreign keys differ",
            case.start
        );
        assert_eq!(
            delete_probe(&mut connection),
            case.after_probe,
            "case {} post-migration delete probe differs",
            case.start
        );
    }
}

/// The corpus must include a database whose trigger is alive before the
/// migration and gone after it, otherwise the probe above proves nothing.
#[test]
fn the_reference_really_observed_the_version_two_trigger() {
    let corpus = parse_fixture();
    let trigger_case = corpus
        .cases
        .iter()
        .find(|case| case.before_objects.contains("trigger:delete_path"))
        .expect("no case starts at the version 2 schema");
    assert!(
        !trigger_case.objects.contains("trigger:delete_path"),
        "version 3 must drop the trigger"
    );
    assert_ne!(
        trigger_case.before_probe, trigger_case.after_probe,
        "the trigger must change the delete probe"
    );
    assert!(
        !corpus
            .cases
            .iter()
            .any(|case| case.objects.contains("trigger:delete_path")),
        "no migrated database may keep the trigger"
    );
    // Every migrated database ends at the ladder's current version, except the
    // generation that is already ahead of this port.
    assert!(corpus
        .cases
        .iter()
        .all(|case| case.version == 3 || case.start > 3));
}
