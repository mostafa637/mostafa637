//! `fs/fake-migrate.c` — the fakefs SQLite schema migration ladder.
//!
//! The fakefs metadata database carries its schema generation in SQLite's
//! `user_version` pragma. [`fakefs_migrate`] reads that number, applies every
//! later migration in order, and writes the new version back — all inside one
//! transaction, so an interrupted migration leaves the old schema in place.
//!
//! The SQL text of each migration is byte-for-byte the text C compiles, and the
//! differential fixture carries that text as emitted by the C source itself
//! (`tools/fake-db-migrate-dump.c` includes `fs/fake-migrate.c`, so the strings
//! it prints are the ones the C compiler built). The table also keeps C's
//! function-pointer slot for migrations that need imperative work; no upstream
//! migration uses it, and the fixture asserts that.
//!
//! C's `fakefs_migrate` takes an unused `root_fd` for signature symmetry with
//! `fakefs_rebuild`; this port leaves the parameter out because nothing reads
//! it and no hook can reach it.

use graphitesql::Connection;

use crate::fake_db::{scalar_integer, FakeDbError};

/// A migration that needs code instead of (or in addition to) SQL.
pub type MigrationHook = fn(&mut Connection) -> Result<(), FakeDbError>;

/// One rung of the ladder, matching `struct migration` in C.
pub struct Migration {
    /// `sql`: executed as a script with `sqlite3_exec`, so several statements
    /// run in one step. `None` means "code only".
    pub sql: Option<&'static str>,
    /// `migrate`: C's optional function pointer. Upstream leaves every entry
    /// `None`.
    pub migrate: Option<MigrationHook>,
}

/// The migration ladder, in `user_version` order.
pub const MIGRATIONS: [Migration; 3] = [
    // version 1: add another index
    Migration {
        sql: Some("create index inode_to_path on paths (inode, path);"),
        migrate: None,
    },
    // version 2: add foreign key constraint on paths, create trigger to
    // automatically cleanup stats
    Migration {
        sql: Some(
            "create table paths_new (path blob primary key, inode integer references stats(inode));\
             insert into paths_new select * from paths where exists (select 1 from stats where inode = paths.inode);\
             drop table paths; alter table paths_new rename to paths;\
             create index inode_to_path on paths (inode, path);\
             delete from stats where not exists (select 1 from paths where inode = stats.inode);\
             create trigger delete_path after delete on paths when not exists (select 1 from paths where inode = old.inode) \
             begin delete from stats where not exists (select 1 from paths where inode = old.inode) and inode = old.inode; end;",
        ),
        migrate: None,
    },
    // version 3: the trigger was a mistake
    Migration {
        sql: Some("drop trigger delete_path"),
        migrate: None,
    },
];

/// `sizeof(migrations)/sizeof(migrations[0])`: the schema version this port's
/// ladder ends at, and the version [`crate::fake_db::FakeDb`] writes for a new
/// database.
pub const SCHEMA_VERSION: i32 = MIGRATIONS.len() as i32;

/// `fakefs_migrate`: bring `connection`'s fakefs schema up to
/// [`SCHEMA_VERSION`].
///
/// The ladder always runs `BEGIN` … `COMMIT`, even when the database is
/// already current: C does the same, so an already-migrated database is still
/// reopened through a transaction. A negative `user_version` would index out of
/// bounds in C; this port reports
/// [`FakeDbError::NegativeSchemaVersion`] instead, before writing anything.
pub fn fakefs_migrate(connection: &mut Connection) -> Result<(), FakeDbError> {
    // `sqlite3_column_int` in C: the pragma is read as a 32-bit signed value,
    // and a missing row behaves like zero.
    let mut version = scalar_integer(connection, "PRAGMA user_version")?.unwrap_or(0) as i32;
    if version < 0 {
        return Err(FakeDbError::NegativeSchemaVersion { found: version });
    }

    connection
        .execute_batch("BEGIN;")
        .map_err(FakeDbError::database)?;
    let outcome = migrate_versions(connection, &mut version);
    match outcome {
        Ok(()) => Ok(()),
        Err(error) => {
            // C `die`s mid-transaction; rolling back keeps the old schema.
            let _ = connection.execute_batch("ROLLBACK;");
            Err(error)
        }
    }
}

fn migrate_versions(connection: &mut Connection, version: &mut i32) -> Result<(), FakeDbError> {
    while *version < SCHEMA_VERSION {
        let migration = &MIGRATIONS[*version as usize];
        if let Some(sql) = migration.sql {
            connection
                .execute_batch(sql)
                .map_err(FakeDbError::database)?;
        }
        if let Some(migrate) = migration.migrate {
            migrate(connection)?;
        }
        *version += 1;
    }
    // Placeholders are not allowed in pragmas, so C formats the statement with
    // sqlite3_mprintf and executes it. A database that is *ahead* of this port
    // keeps its own number, exactly as in C.
    connection
        .execute(&format!("PRAGMA user_version = {version}"))
        .map(|_| ())
        .map_err(FakeDbError::database)?;
    connection
        .execute_batch("COMMIT;")
        .map_err(FakeDbError::database)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::fake_db::{blob_literal, query_rows, FakeDb, IshStat};
    use graphitesql::{Connection, Value};

    fn stat(mode: u32) -> IshStat {
        IshStat {
            mode,
            uid: 1000,
            gid: 100,
            rdev: 0,
        }
    }

    fn stat_literal(stat: IshStat) -> String {
        blob_literal(&stat.to_le_bytes())
    }

    /// Run one SQL script on the database, failing the test on any error.
    fn raw(db: &FakeDb, sql: &str) {
        db.with_connection(|connection| {
            connection.execute_batch(sql).map_err(FakeDbError::database)
        })
        .unwrap();
    }

    /// The schema generation 0 database, as historical iSH versions wrote it:
    /// no `inode_to_path` index and no foreign key on `paths`.
    fn version_zero_schema(connection: &mut Connection) {
        connection
            .execute_batch(
                "CREATE TABLE meta (id integer unique default 0, db_inode integer);\
                 INSERT INTO meta (db_inode) VALUES (0);\
                 CREATE TABLE stats (inode integer primary key, stat blob);\
                 CREATE TABLE paths (path blob primary key, inode integer);",
            )
            .unwrap();
    }

    fn fake_db_with_schema(setup: &str) -> FakeDb {
        let mut connection = Connection::open_memory().unwrap();
        version_zero_schema(&mut connection);
        connection.execute_batch(setup).unwrap();
        FakeDb::from_connection(connection)
    }

    fn user_version(db: &FakeDb) -> i64 {
        db.with_connection(|connection| scalar_integer(connection, "PRAGMA user_version"))
            .unwrap()
            .unwrap()
    }

    fn object_names(db: &FakeDb) -> Vec<(String, String)> {
        db.with_connection(|connection| {
            query_rows(
                connection,
                "SELECT type, name FROM sqlite_master ORDER BY type, name",
            )
        })
        .unwrap()
        .into_iter()
        .map(|row| match (&row[0], &row[1]) {
            (Value::Text(kind), Value::Text(name)) => {
                (kind.as_str().to_string(), name.as_str().to_string())
            }
            other => panic!("unexpected sqlite_master row {other:?}"),
        })
        .collect()
    }

    #[test]
    fn ladder_matches_the_c_source_shape() {
        assert_eq!(SCHEMA_VERSION, 3);
        assert_eq!(MIGRATIONS.len(), 3);
        // Upstream migrations are pure SQL; the hook slot exists for future
        // imperative migrations and is never populated by iSH.
        assert!(MIGRATIONS
            .iter()
            .all(|migration| migration.migrate.is_none()));
        assert_eq!(
            MIGRATIONS[0].sql.unwrap(),
            "create index inode_to_path on paths (inode, path);"
        );
        assert_eq!(MIGRATIONS[2].sql.unwrap(), "drop trigger delete_path");
        // Migration 2 is a single SQL script holding seven statements (one of
        // them the trigger body, which carries an extra semicolon).
        assert_eq!(
            MIGRATIONS[1].sql.unwrap().matches(';').count(),
            8,
            "migration 2 statement count changed"
        );
    }

    #[test]
    fn migrating_a_version_zero_database_adds_the_index_key_and_trigger_free_schema() {
        let db = fake_db_with_schema(&format!(
            "INSERT INTO stats (inode, stat) VALUES (1, {}), (2, {}), (3, {});\
             INSERT INTO paths (path, inode) VALUES (X'2f61', 1), (X'2f62', 2), (X'2f6f727068616e', 3);",
            stat_literal(stat(0o100644)),
            stat_literal(stat(0o040755)),
            stat_literal(stat(7)),
        ));
        // Inode 3 has no path: migration 2 deletes it.
        raw(&db, "DELETE FROM paths WHERE inode=3;");
        assert_eq!(user_version(&db), 0);

        db.migrate().unwrap();

        assert_eq!(user_version(&db), 3);
        // SQLite's implicit `sqlite_autoindex_paths_*` keeps its creation-time
        // name across `ALTER TABLE ... RENAME TO`; only the explicit objects
        // are pinned here, and the differential fixture pins the whole list.
        let objects: Vec<(String, String)> = object_names(&db)
            .into_iter()
            .filter(|(_, name)| !name.starts_with("sqlite_autoindex_"))
            .collect();
        assert_eq!(
            objects,
            vec![
                ("index".to_string(), "inode_to_path".to_string()),
                ("table".to_string(), "meta".to_string()),
                ("table".to_string(), "paths".to_string()),
                ("table".to_string(), "stats".to_string()),
            ],
            "the delete_path trigger must not survive version 3"
        );
        assert_eq!(db.path_get_inode(b"/a").unwrap(), 1);
        assert_eq!(db.path_get_inode(b"/b").unwrap(), 2);
        assert_eq!(db.inode_read_stat_if_exist(3).unwrap(), None);
        // The rebuilt `paths` table carries the foreign key from migration 2.
        let rejects_dangling = db
            .with_connection(|connection| {
                connection.execute_batch("PRAGMA foreign_keys=ON;").unwrap();
                connection
                    .execute("INSERT INTO paths (path, inode) VALUES (X'2f78', 999)")
                    .map_err(FakeDbError::database)
            })
            .is_err();
        assert!(rejects_dangling, "paths must reference stats(inode)");
    }

    #[test]
    fn migrating_from_version_two_drops_the_trigger() {
        // A version 2 database: foreign key present, trigger present.
        let db = fake_db_with_schema(&format!(
            "INSERT INTO stats (inode, stat) VALUES (1, {});\
             INSERT INTO paths (path, inode) VALUES (X'2f61', 1);",
            stat_literal(stat(0o100600))
        ));
        raw(
            &db,
            "CREATE TABLE paths_new (path blob primary key, inode integer references stats(inode));\
                     INSERT INTO paths_new SELECT * FROM paths;\
                     DROP TABLE paths;\
                     ALTER TABLE paths_new RENAME TO paths;\
                     CREATE INDEX inode_to_path ON paths (inode, path);\
                     CREATE TRIGGER delete_path AFTER DELETE ON paths \
                     WHEN NOT EXISTS (SELECT 1 FROM paths WHERE inode = old.inode) \
                     BEGIN DELETE FROM stats WHERE NOT EXISTS (SELECT 1 FROM paths WHERE inode = old.inode) \
                     AND inode = old.inode; END;\
             PRAGMA user_version = 2;",
        );
        assert_eq!(user_version(&db), 2);

        db.migrate().unwrap();
        assert_eq!(user_version(&db), 3);

        // The trigger would have deleted the now-orphaned stats row; version 3
        // removed it, so the stat survives like it did before version 2.
        raw(&db, "DELETE FROM paths WHERE path=X'2f61';");
        assert_eq!(
            db.inode_read_stat_if_exist(1).unwrap(),
            Some(stat(0o100600))
        );
    }

    #[test]
    fn current_and_newer_versions_are_left_alone() {
        let db = fake_db_with_schema(&format!(
            "INSERT INTO stats (inode, stat) VALUES (1, {});\
             INSERT INTO paths (path, inode) VALUES (X'2f61', 1);",
            stat_literal(stat(1))
        ));
        raw(&db, "PRAGMA user_version = 3;");
        // Version 3 databases already have the index; migration must not fail
        // on the already-existing object.
        raw(&db, "CREATE INDEX inode_to_path ON paths (inode, path);");
        let before = db.logical_hash().unwrap();
        db.migrate().unwrap();
        assert_eq!(user_version(&db), 3);
        assert_eq!(db.logical_hash().unwrap(), before);

        // A future schema keeps its own version number, as C writes the
        // unchanged value back out.
        raw(&db, "PRAGMA user_version = 7;");
        db.migrate().unwrap();
        assert_eq!(user_version(&db), 7);
        assert_eq!(db.logical_hash().unwrap(), before);
    }

    /// The one schema artifact that differs from real SQLite.
    ///
    /// Migration 2 builds `paths_new` and renames it; SQLite regenerates the
    /// implicit index name for the new table name, while the vendored pure-Rust
    /// engine keeps `sqlite_autoindex_paths_new_1`. No iSH code — and no guest
    /// syscall — can observe that name, so the port pins the explicit objects
    /// (tables, `inode_to_path`, triggers) and documents this one.
    #[test]
    fn implicit_index_name_keeps_its_creation_time_spelling() {
        let db = fake_db_with_schema("");
        db.migrate().unwrap();
        let names = object_names(&db);
        assert!(
            names
                .iter()
                .any(|(kind, name)| kind == "index" && name == "inode_to_path"),
            "the explicit index is the one that must exist: {names:?}"
        );
        assert!(
            names
                .iter()
                .any(|(_, name)| name.starts_with("sqlite_autoindex_paths")),
            "the implicit paths index must still exist: {names:?}"
        );
    }

    #[test]
    fn a_negative_version_is_reported_instead_of_indexing_out_of_bounds() {
        let db = fake_db_with_schema("");
        raw(&db, "PRAGMA user_version = -1;");
        assert_eq!(
            db.migrate().unwrap_err(),
            FakeDbError::NegativeSchemaVersion { found: -1 }
        );
        assert_eq!(user_version(&db), -1);
    }
}
