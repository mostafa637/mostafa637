//! Differential test of `mount.rs` against unmodified `fs/mount.c`.
//!
//! `tools/mount-dump.c` includes the real `fs/mount.c` (and links the real
//! `fs/path.c`, whose predicate `mount_find` asserts), so every answer in
//! `tests/fixtures/mount_reference.txt` comes from the C's own functions. The
//! fixture is not just a corpus but a *script*: it builds a table, registers
//! filesystems, mounts a fake filesystem at a series of points, looks paths up,
//! takes and gives back references, removes mounts, and asks about parameter
//! strings. This test performs the same operations through [`MountTable`] and
//! compares each record as it goes, which is what makes the record order part
//! of the contract: a lookup has to find the same mount, a reference has to move
//! the same counter, and a mount point has to sit at the same place in the list.
//!
//! The C's fake filesystem records what it was asked to do; so does the port's,
//! into a `Mutex` because an `fs_ops` member is a plain `fn` pointer. The
//! fixture's `F` block is that log, and it has to match entry for entry — the
//! order of the `mount` and `umount` calls, their arguments, and how many there
//! were.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_mount_reference.sh
//! ```

use std::collections::BTreeMap;
use std::rc::Rc;
use std::sync::Mutex;

use ish_emu::mount::{
    mount_param_flag, FsOps, Mount, MountTable, EINVAL, MAX_FILESYSTEMS, MS_FLAGS, MS_NODEV,
    MS_NOEXEC, MS_NOSUID, MS_READONLY, MS_SILENT, MS_SUPPORTED, SEEDED_FILESYSTEMS,
};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/mount_reference.txt"
);

/// The shape of the script, so a regenerated fixture that lost records is a
/// failure rather than a test that checks less.
const CONSTANTS: usize = 8;
const SEEDED: usize = 4;
const REGISTRATIONS: usize = 6;
const MOUNTS: usize = 8;
const LOOKUPS: usize = 17;
const RETAINS: usize = 1;
const RELEASES: usize = 18;
const REMOVALS: usize = 2;
const UMOUNTS: usize = 4;
const PARAMETERS: usize = 12;
const LIST_ENTRIES: usize = 37;
const BLOCKS: usize = 7;
const CALLBACKS: usize = 11;

/// The descriptor number the fake filesystem's `mount` op opens its root on,
/// as in the oracle: the list records prove the callback's write arrived.
const FAKE_ROOT_FD: i32 = 7;

/// The fake filesystem's callback log, in call order.
static CALLBACKS_LOG: Mutex<Vec<String>> = Mutex::new(Vec::new());

fn hex(data: &[u8]) -> String {
    if data.is_empty() {
        return "-".to_owned();
    }
    data.iter().map(|byte| format!("{byte:02x}")).collect()
}

fn logged(line: String) {
    CALLBACKS_LOG
        .lock()
        .expect("the log is never poisoned")
        .push(line);
}

/// The fake filesystem's `mount`: everything succeeds except the point `/fail`.
fn fake_mount(mount: &Mount) -> i32 {
    let ret = if mount.point() == b"/fail" { EINVAL } else { 0 };
    logged(format!(
        "mount point={} source={} info={} ret={ret}",
        hex(mount.point()),
        hex(mount.source()),
        hex(mount.info())
    ));
    if ret == 0 {
        mount.set_root_fd(FAKE_ROOT_FD);
    }
    ret
}

/// The fake filesystem's `umount`, whose return value C ignores.
fn fake_umount(mount: &Mount) -> i32 {
    logged(format!("umount point={}", hex(mount.point())));
    0
}

/// A copy of the fake filesystem under another name.
///
/// The C's `fs_ops` values are `'static` file-scope constants; the port's
/// callbacks are `fn` pointers, so one can be built per name here and leaked,
/// which is what a `static` would have been.
fn named(name: &str) -> &'static FsOps {
    Box::leak(Box::new(FsOps {
        name: Box::leak(name.to_string().into_boxed_str()),
        magic: 0x66616b65,
        mount: Some(fake_mount),
        umount: Some(fake_umount),
        statfs: None,
        readlink: None,
        link: None,
        unlink: None,
        rmdir: None,
        rename: None,
        symlink: None,
        mknod: None,
        mkdir: None,
        stat: None,
        setattr: None,
        utime: None,
        inode_orphaned: None,
    }))
}

/// The fixture's hex encoding, where `-` is the empty byte string.
fn bytes(field: &str) -> Vec<u8> {
    if field == "-" {
        return Vec::new();
    }
    assert_eq!(field.len() % 2, 0, "odd hex length in `{field}`");
    (0..field.len() / 2)
        .map(|index| u8::from_str_radix(&field[index * 2..index * 2 + 2], 16).expect("hex"))
        .collect()
}

/// The value of a `key=value` field of a record.
fn value<'a>(line: &'a str, key: &str) -> &'a str {
    let wanted = format!("{key}=");
    line.split(' ')
        .find_map(|part| part.strip_prefix(wanted.as_str()))
        .unwrap_or_else(|| panic!("no `{key}=` in `{line}`"))
}

#[test]
fn the_mount_table_matches_the_c() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|err| panic!("cannot read {FIXTURE}: {err}"));

    let table = MountTable::new();
    let fake = named("fake");
    // The C's table starts with the four filesystems of kernel/fs.h, whose
    // values live in files that are not ported yet; the fixture's `S` records
    // name them and the registered stand-ins take their places, which is why a
    // registration lands in slot 4.
    for name in SEEDED_FILESYSTEMS {
        table.register(named(name));
    }

    // Outstanding references, by mount point: C's `struct mount *` handles. A
    // `Q` record leaves one behind and an `R` record gives it back.
    let mut handles: BTreeMap<Vec<u8>, Vec<Rc<Mount>>> = BTreeMap::new();
    let mut expected_callbacks: Vec<String> = Vec::new();
    let mut callback_count: Option<usize> = None;
    let mut list_index = 0usize;

    let mut constants = 0usize;
    let mut seeded = 0usize;
    let mut registrations = 0usize;
    let mut mounts = 0usize;
    let mut lookups = 0usize;
    let mut retains = 0usize;
    let mut releases = 0usize;
    let mut removals = 0usize;
    let mut umounts = 0usize;
    let mut parameters = 0usize;
    let mut list_entries = 0usize;
    let mut blocks = 0usize;

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split(' ').collect();
        match fields[0] {
            "K" => {
                let (name, raw) = line[2..].split_once('=').expect("name=value");
                let raw: i32 = raw.parse().expect("integer");
                let ported = match name {
                    "MAX_FILESYSTEMS" => MAX_FILESYSTEMS as i32,
                    "MS_READONLY" => MS_READONLY,
                    "MS_NOSUID" => MS_NOSUID,
                    "MS_NODEV" => MS_NODEV,
                    "MS_NOEXEC" => MS_NOEXEC,
                    "MS_SILENT" => MS_SILENT,
                    "MS_SUPPORTED" => MS_SUPPORTED,
                    "MS_FLAGS" => MS_FLAGS,
                    other => panic!("the fixture has a constant the port does not define: {other}"),
                };
                assert_eq!(ported, raw, "the C's {name}");
                constants += 1;
            }
            "S" => {
                assert_eq!(fields.len(), 3, "malformed seeded record: {line}");
                let slot: usize = fields[1].parse().expect("slot");
                let name = value(line, "name");
                assert_eq!(slot, seeded, "the seeded records are out of order: {line}");
                assert_eq!(
                    SEEDED_FILESYSTEMS[slot], name,
                    "the C seeds slot {slot} with a different filesystem"
                );
                seeded += 1;
            }
            "G" => {
                assert_eq!(fields.len(), 3, "malformed registration record: {line}");
                let name = value(line, "name");
                let slot: usize = value(line, "slot").parse().expect("slot");
                assert_eq!(
                    table.filesystems().len(),
                    slot,
                    "the C registered {name} in slot {slot}, the port's table is a different length"
                );
                // The registered filesystems are in the table's own order, so
                // the port registers the stand-in this record names.
                table.register(named(name));
                assert_eq!(table.filesystems().len(), slot + 1);
                registrations += 1;
            }
            "M" => {
                let fs = value(line, "fs");
                assert_eq!(
                    fs, "fake",
                    "the fixture mounts a filesystem the port cannot"
                );
                let source = bytes(value(line, "source"));
                let point = bytes(value(line, "point"));
                let info = bytes(value(line, "info"));
                let flags: i32 = value(line, "flags").parse().expect("flags");
                let ret: i32 = value(line, "ret").parse().expect("ret");
                assert_eq!(
                    table.do_mount(fake, &source, &point, &info, flags),
                    ret,
                    "do_mount {line}"
                );
                mounts += 1;
            }
            "Q" => {
                let path = bytes(value(line, "path"));
                let point = bytes(value(line, "point"));
                let source = bytes(value(line, "source"));
                let refcount: u32 = value(line, "refcount").parse().expect("refcount");
                let mount = table.find(&path);
                assert_eq!(mount.point(), point, "mount_find {line}");
                assert_eq!(mount.source(), source, "mount_find {line}");
                assert_eq!(mount.refcount(), refcount, "mount_find {line}");
                handles.entry(point).or_default().push(mount);
                lookups += 1;
            }
            "T" => {
                let point = bytes(value(line, "point"));
                let refcount: u32 = value(line, "refcount").parse().expect("refcount");
                let held = handles
                    .get(&point)
                    .and_then(|stack| stack.last())
                    .unwrap_or_else(|| panic!("mount_retain with nothing to retain: {line}"))
                    .clone();
                let extra = table.retain(&held);
                assert_eq!(extra.refcount(), refcount, "mount_retain {line}");
                handles.entry(point).or_default().push(extra);
                retains += 1;
            }
            "R" => {
                let point = bytes(value(line, "point"));
                let refcount: u32 = value(line, "refcount").parse().expect("refcount");
                let given_back = handles
                    .get_mut(&point)
                    .and_then(|stack| stack.pop())
                    .unwrap_or_else(|| panic!("mount_release with nothing to release: {line}"));
                table.release(given_back);
                // The mount is still in the list, which is what C's stale
                // pointer relies on: the count is readable after the release.
                let still_mounted = table
                    .mounts()
                    .into_iter()
                    .find(|mount| mount.point() == point)
                    .unwrap_or_else(|| panic!("the mount left the list on release: {line}"));
                assert_eq!(still_mounted.refcount(), refcount, "mount_release {line}");
                releases += 1;
            }
            "X" => {
                let point = bytes(value(line, "point"));
                let refcount: u32 = value(line, "refcount").parse().expect("refcount");
                let ret: i32 = value(line, "ret").parse().expect("ret");
                // A referenced mount is removed through the reference the C is
                // holding; an unreferenced one through the list.
                let mount = if refcount > 0 {
                    handles
                        .get(&point)
                        .and_then(|stack| stack.last())
                        .cloned()
                        .unwrap_or_else(|| panic!("no reference to remove: {line}"))
                } else {
                    table
                        .mounts()
                        .into_iter()
                        .find(|mount| mount.point() == point)
                        .unwrap_or_else(|| panic!("no mount to remove: {line}"))
                };
                assert_eq!(mount.refcount(), refcount, "mount_remove {line}");
                assert_eq!(table.mount_remove(&mount), ret, "mount_remove {line}");
                removals += 1;
            }
            "U" => {
                let point = bytes(value(line, "point"));
                let ret: i32 = value(line, "ret").parse().expect("ret");
                assert_eq!(table.do_umount(&point), ret, "do_umount {line}");
                umounts += 1;
            }
            "P" => {
                let info = bytes(value(line, "info"));
                let flag = bytes(value(line, "flag"));
                let ret: i32 = value(line, "ret").parse().expect("ret");
                assert_eq!(
                    i32::from(mount_param_flag(&info, &flag)),
                    ret,
                    "mount_param_flag {line}"
                );
                parameters += 1;
            }
            "L" => {
                let index: usize = fields[1].parse().expect("index");
                assert_eq!(index, list_index, "the list is out of order: {line}");
                let mounts = table.mounts();
                let mount = mounts
                    .get(index)
                    .unwrap_or_else(|| panic!("the port's list is shorter than the C's: {line}"));
                assert_eq!(
                    hex(mount.point()),
                    value(line, "point"),
                    "list entry {index}"
                );
                assert_eq!(
                    hex(mount.source()),
                    value(line, "source"),
                    "list entry {index}"
                );
                assert_eq!(hex(mount.info()), value(line, "info"), "list entry {index}");
                assert_eq!(
                    mount.flags(),
                    value(line, "flags").parse::<i32>().expect("flags"),
                    "list entry {index}"
                );
                assert_eq!(
                    mount.refcount(),
                    value(line, "refcount").parse::<u32>().expect("refcount"),
                    "list entry {index}"
                );
                assert_eq!(
                    mount.root_fd(),
                    value(line, "root_fd").parse::<i32>().expect("root_fd"),
                    "list entry {index}"
                );
                list_index += 1;
                list_entries += 1;
            }
            "E" => {
                let count: usize = value(line, "count").parse().expect("count");
                assert_eq!(list_index, count, "the block's count disagrees: {line}");
                assert_eq!(
                    table.mounts().len(),
                    count,
                    "the port's list is a different length: {line}"
                );
                list_index = 0;
                blocks += 1;
            }
            "F" => {
                if let Some(raw) = line.strip_prefix("F count=") {
                    callback_count = Some(raw.parse().expect("count"));
                } else {
                    let (index, text) = line[2..].split_once(' ').expect("index and text");
                    let index: usize = index.parse().expect("index");
                    assert_eq!(index, expected_callbacks.len(), "the log is out of order");
                    expected_callbacks.push(text.to_owned());
                }
            }
            other => panic!("unknown record type `{other}` in {line}"),
        }
    }

    // The counters, so a fixture that lost records fails here instead of
    // quietly checking less.
    assert_eq!(constants, CONSTANTS);
    assert_eq!(seeded, SEEDED);
    assert_eq!(registrations, REGISTRATIONS);
    assert_eq!(mounts, MOUNTS);
    assert_eq!(lookups, LOOKUPS);
    assert_eq!(retains, RETAINS);
    assert_eq!(releases, RELEASES);
    assert_eq!(removals, REMOVALS);
    assert_eq!(umounts, UMOUNTS);
    assert_eq!(parameters, PARAMETERS);
    assert_eq!(list_entries, LIST_ENTRIES);
    assert_eq!(blocks, BLOCKS);
    assert_eq!(list_index, 0, "a block of list records was left open");

    // The fake filesystem saw exactly what the C's did, in the same order.
    assert_eq!(callback_count, Some(CALLBACKS));
    let log = CALLBACKS_LOG
        .lock()
        .expect("the log is never poisoned")
        .clone();
    assert_eq!(log, expected_callbacks, "the filesystem callback log");

    // Every reference the script took was given back.
    for (point, stack) in &handles {
        assert!(
            stack.is_empty(),
            "{} references to {:?} were never released",
            stack.len(),
            String::from_utf8_lossy(point)
        );
    }
}
