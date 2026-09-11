//! Differential test of `inode.rs` against unmodified `fs/inode.c`.
//!
//! `tools/inode-dump.c` includes the real `fs/inode.c` (and the real
//! `fs/mount.c`, whose `mount_retain`/`mount_release`/refcount an inode moves),
//! so every answer in `tests/fixtures/inode_reference.txt` comes from the C's
//! own functions — and because the oracle includes the .c file, its walk
//! records are read straight out of the C's static `inodes_hash[]`. The fixture
//! is not just a corpus but a *script*: it mounts three filesystems, gets
//! inodes on them, takes and gives back references, writes socket ids and asks
//! about orphaned inodes. This test performs the same calls through
//! [`InodeTable`] and compares each record as it goes, which is what makes the
//! record order part of the contract: a second `get` has to find the same inode,
//! a release has to move the same counter, the inode has to leave the hash at
//! the same reference count, and the mount has to lose its reference at the same
//! moment.
//!
//! The C's fake filesystem records the orphan calls it receives; so does the
//! port's, into a `Mutex` because an `fs_ops` member is a plain `fn` pointer.
//! The fixture's `F` block is that log, and it has to match entry for entry —
//! which calls, in which order, and how many.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_inode_reference.sh
//! ```

use std::cell::RefCell;
use std::collections::BTreeMap;
use std::rc::Rc;
use std::sync::Mutex;

use ish_emu::inode::{InodeData, InodeTable, F_RDLCK_, F_UNLCK_, F_WRLCK_, INODES_HASH_SIZE};
use ish_emu::mount::{FsOps, Mount, MountTable};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/inode_reference.txt"
);

/// The shape of the script, so a regenerated fixture that lost records is a
/// failure rather than a test that checks less.
const CONSTANTS: usize = 4;
const MOUNTS_MADE: usize = 3;
const MOUNT_RECORDS: usize = 6;
const MOUNT_BLOCKS: usize = 2;
const GETS: usize = 9;
const UNLOCKED_GETS: usize = 1;
const RETAINS: usize = 1;
const RELEASES: usize = 11;
const CHECKS: usize = 3;
const WRITES: usize = 2;
const TABLE_ENTRIES: usize = 20;
const WALKS: usize = 10;
/// The C's `F count=`, and the number of hook calls it stands for.
const HOOK_CALLS: usize = 9;

/// The fake filesystem's orphan-hook log, in call order.
static HOOK_LOG: Mutex<Vec<String>> = Mutex::new(Vec::new());

thread_local! {
    /// The table the hook asks about the inode it is being told about, the way
    /// C's hook asks `inode_get_data` while `inode_release` holds
    /// `inodes_lock`. Set by the test before it replays the script.
    static HOOK_TABLE: RefCell<Option<Rc<InodeTable>>> = const { RefCell::new(None) };
}

fn hex(data: &[u8]) -> String {
    if data.is_empty() {
        return "-".to_owned();
    }
    data.iter().map(|byte| format!("{byte:02x}")).collect()
}

fn logged(line: String) {
    HOOK_LOG
        .lock()
        .expect("the log is never poisoned")
        .push(line);
}

fn hook_calls() -> usize {
    HOOK_LOG.lock().expect("the log is never poisoned").len()
}

/// The fake filesystem's `inode_orphaned`, which `fs/fake.c` uses to drop the
/// metadata of a file nothing holds open any more.
///
/// It asks the table whether the inode is still there, which is what makes the
/// order inside the port's `release` observable: the inode leaves the hash
/// *before* the hook runs, so the answer is always no.
fn fake_orphaned(mount: &Mount, inode: u64) {
    let in_table = HOOK_TABLE.with(|table| {
        table
            .borrow()
            .clone()
            .expect("the test sets the table before it replays the script")
            .contains(mount, inode)
    });
    logged(format!(
        "orphaned point={} number={inode} in_table={}",
        hex(mount.point()),
        u8::from(in_table)
    ));
}

/// The fake filesystem's `umount`, which this script never reaches; the oracle
/// has the same op, so a call would show up in the `F` block on both sides.
fn fake_umount(mount: &Mount) -> i32 {
    logged(format!("umount point={}", hex(mount.point())));
    0
}

/// The filesystem the oracle mounts.
///
/// The C's `struct fs_ops` values are `'static` file-scope constants; the
/// port's callbacks are `fn` pointers, so this is built once and leaked, which
/// is what a `static` would have been. It has no `mount` op, and the four
/// filesystems `fs/mount.c` seeds its *table* with play no part in the script:
/// `do_mount` is given the filesystem value directly and never looks at the
/// table.
fn fake() -> &'static FsOps {
    Box::leak(Box::new(FsOps {
        name: "fake",
        magic: 0x66616b65,
        mount: None,
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
        inode_orphaned: Some(fake_orphaned),
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

/// C's bare `struct mount *`: the mount as the list holds it, with no reference
/// taken, so that every change to the count is one the inode table made.
fn bare_mount(mounts: &MountTable, point: &[u8]) -> Rc<Mount> {
    mounts
        .mounts()
        .into_iter()
        .find(|mount| mount.point() == point)
        .unwrap_or_else(|| panic!("no mount at `{}`", String::from_utf8_lossy(point)))
}

/// Whether the table holds an inode for `(point, number)` — C's
/// `inode_get_data`, which the differential test can only ask by walking.
fn present(inodes: &InodeTable, point: &[u8], number: u64) -> bool {
    inodes
        .entries()
        .iter()
        .any(|entry| entry.number == number && entry.point == point)
}

#[test]
fn the_inode_table_matches_the_c() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|err| panic!("cannot read {FIXTURE}: {err}"));
    HOOK_LOG.lock().expect("the log is never poisoned").clear();

    let mounts = Rc::new(MountTable::new());
    let inodes = Rc::new(InodeTable::new(Rc::clone(&mounts)));
    HOOK_TABLE.with(|table| *table.borrow_mut() = Some(Rc::clone(&inodes)));

    // Outstanding references, by key, in the order the script took them: C's
    // `struct inode_data *` handles. A `G`, `U` or `T` record pushes one and an
    // `R` record gives one back.
    let mut handles: BTreeMap<(Vec<u8>, u64), Vec<Rc<InodeData>>> = BTreeMap::new();
    let mut expected_hooks: Vec<String> = Vec::new();
    let mut hook_record_count: Option<usize> = None;

    let mut mount_index = 0usize;
    let mut table_index = 0usize;

    let mut constants = 0usize;
    let mut mounts_made = 0usize;
    let mut mount_records = 0usize;
    let mut mount_blocks = 0usize;
    let mut gets = 0usize;
    let mut unlocked_gets = 0usize;
    let mut retains = 0usize;
    let mut releases = 0usize;
    let mut checks = 0usize;
    let mut writes = 0usize;
    let mut table_entries = 0usize;
    let mut walks = 0usize;

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split(' ').collect();
        match fields[0] {
            "K" => {
                let (name, raw) = line[2..].split_once('=').expect("name=value");
                let raw: usize = raw.parse().expect("integer");
                let ported = match name {
                    "INODES_HASH_SIZE" => INODES_HASH_SIZE,
                    "F_RDLCK_" => F_RDLCK_ as usize,
                    "F_WRLCK_" => F_WRLCK_ as usize,
                    "F_UNLCK_" => F_UNLCK_ as usize,
                    other => panic!("the fixture has a constant the port does not define: {other}"),
                };
                assert_eq!(ported, raw, "the C's {name}");
                constants += 1;
            }
            "M" => {
                assert_eq!(
                    value(line, "fs"),
                    "fake",
                    "the fixture mounts a filesystem the port cannot"
                );
                let source = bytes(value(line, "source"));
                let point = bytes(value(line, "point"));
                let info = bytes(value(line, "info"));
                let flags: i32 = value(line, "flags").parse().expect("flags");
                let ret: i32 = value(line, "ret").parse().expect("ret");
                assert_eq!(
                    mounts.do_mount(fake(), &source, &point, &info, flags),
                    ret,
                    "do_mount {line}"
                );
                mounts_made += 1;
            }
            "S" => {
                let point = bytes(value(line, "point"));
                let refcount: u32 = value(line, "refcount").parse().expect("refcount");
                let list = mounts.mounts();
                let mount = list
                    .get(mount_index)
                    .unwrap_or_else(|| panic!("the port's mount list is shorter: {line}"));
                assert_eq!(mount.point(), point, "mount record {mount_index}");
                assert_eq!(mount.refcount(), refcount, "mount record {mount_index}");
                mount_index += 1;
                mount_records += 1;
            }
            "Z" => {
                let count: usize = value(line, "count").parse().expect("count");
                assert_eq!(
                    mount_index, count,
                    "the block disagrees with itself: {line}"
                );
                assert_eq!(
                    mounts.mounts().len(),
                    count,
                    "the port's mount list is a different length: {line}"
                );
                mount_index = 0;
                mount_blocks += 1;
            }
            "G" | "U" => {
                let unlocked = fields[0] == "U";
                let point = bytes(value(line, "point"));
                let number: u64 = value(line, "number").parse().expect("number");
                let is_new = value(line, "new") == "1";
                let refcount: u32 = value(line, "refcount").parse().expect("refcount");
                let socket_id: u32 = value(line, "socket_id").parse().expect("socket_id");
                let mount_refcount: u32 = value(line, "mount_refcount")
                    .parse()
                    .expect("mount_refcount");

                // inode_get_data(mount, ino) == NULL is what the C's `new` is.
                assert_eq!(
                    present(&inodes, &point, number),
                    !is_new,
                    "the fixture's new= disagrees with the port's table: {line}"
                );
                let mount = bare_mount(&mounts, &point);
                let inode = if unlocked {
                    // generic_open's shape: the table lock is held across the
                    // open *and* the reference that follows.
                    let guard = inodes.lock();
                    let inode = guard.get_unlocked(&mount, number);
                    drop(guard);
                    inode
                } else {
                    inodes.get(&mount, number)
                };
                assert_eq!(inode.number(), number, "the inode's number: {line}");
                assert_eq!(inode.refcount(), refcount, "inode_get {line}");
                assert_eq!(inode.socket_id(), socket_id, "inode_get {line}");
                assert_eq!(mount.refcount(), mount_refcount, "inode_get {line}");
                handles.entry((point, number)).or_default().push(inode);
                if unlocked {
                    unlocked_gets += 1;
                } else {
                    gets += 1;
                }
            }
            "T" => {
                let point = bytes(value(line, "point"));
                let number: u64 = value(line, "number").parse().expect("number");
                let refcount: u32 = value(line, "refcount").parse().expect("refcount");
                let key = (point.clone(), number);
                let held = handles
                    .get(&key)
                    .and_then(|stack| stack.last())
                    .cloned()
                    .unwrap_or_else(|| panic!("inode_retain with nothing to retain: {line}"));
                let extra = inodes.retain(&held);
                assert_eq!(extra.refcount(), refcount, "inode_retain {line}");
                handles.entry(key).or_default().push(extra);
                retains += 1;
            }
            "R" => {
                let point = bytes(value(line, "point"));
                let number: u64 = value(line, "number").parse().expect("number");
                let refcount: u32 = value(line, "refcount").parse().expect("refcount");
                let gone = value(line, "gone") == "1";
                let mount_refcount: u32 = value(line, "mount_refcount")
                    .parse()
                    .expect("mount_refcount");
                let key = (point.clone(), number);
                let given_back = handles
                    .get_mut(&key)
                    .and_then(|stack| stack.pop())
                    .unwrap_or_else(|| panic!("inode_release with nothing to release: {line}"));
                // The count is read before the call: this is the count the
                // release takes one away from, and `gone` says whether it was
                // the last one, i.e. whether C frees the inode.
                assert_eq!(given_back.refcount(), refcount, "inode_release {line}");
                inodes.release(given_back);
                assert_eq!(
                    present(&inodes, &point, number),
                    !gone,
                    "inode_release {line}"
                );
                // The mount is still in the list, which is what C's stale
                // pointer relies on: the count is readable after the release.
                let mount = bare_mount(&mounts, &point);
                assert_eq!(mount.refcount(), mount_refcount, "inode_release {line}");
                releases += 1;
            }
            "C" => {
                let point = bytes(value(line, "point"));
                let number: u64 = value(line, "number").parse().expect("number");
                let called = value(line, "called") == "1";
                let before = hook_calls();
                inodes.check_orphaned(&bare_mount(&mounts, &point), number);
                assert_eq!(
                    hook_calls() != before,
                    called,
                    "inode_check_orphaned {line}"
                );
                checks += 1;
            }
            "W" => {
                let point = bytes(value(line, "point"));
                let number: u64 = value(line, "number").parse().expect("number");
                let socket_id: u32 = value(line, "socket_id").parse().expect("socket_id");
                let inode = handles
                    .get(&(point, number))
                    .and_then(|stack| stack.last())
                    .cloned()
                    .unwrap_or_else(|| {
                        panic!("socket id written to an inode nothing holds: {line}")
                    });
                inode.set_socket_id(socket_id);
                assert_eq!(inode.socket_id(), socket_id, "socket_id {line}");
                writes += 1;
            }
            "L" => {
                let index: usize = fields[1].parse().expect("index");
                assert_eq!(index, table_index, "the table walk is out of order: {line}");
                let entries = inodes.entries();
                let entry = entries
                    .get(index)
                    .unwrap_or_else(|| panic!("the port's table is shorter: {line}"));
                assert_eq!(
                    entry.bucket,
                    value(line, "bucket").parse::<usize>().expect("bucket"),
                    "table entry {index}"
                );
                assert_eq!(
                    entry.number,
                    value(line, "number").parse::<u64>().expect("number"),
                    "table entry {index}"
                );
                assert_eq!(
                    entry.refcount,
                    value(line, "refcount").parse::<u32>().expect("refcount"),
                    "table entry {index}"
                );
                assert_eq!(
                    entry.socket_id,
                    value(line, "socket_id").parse::<u32>().expect("socket_id"),
                    "table entry {index}"
                );
                assert_eq!(
                    hex(&entry.point),
                    value(line, "point"),
                    "table entry {index}"
                );
                table_index += 1;
                table_entries += 1;
            }
            "E" => {
                let count: usize = value(line, "count").parse().expect("count");
                assert_eq!(table_index, count, "the walk's count disagrees: {line}");
                assert_eq!(
                    inodes.entries().len(),
                    count,
                    "the port's table is a different length: {line}"
                );
                table_index = 0;
                walks += 1;
            }
            "F" => {
                if let Some(raw) = line.strip_prefix("F count=") {
                    hook_record_count = Some(raw.parse().expect("count"));
                } else {
                    let (index, text) = line[2..].split_once(' ').expect("index and text");
                    let index: usize = index.parse().expect("index");
                    assert_eq!(index, expected_hooks.len(), "the log is out of order");
                    expected_hooks.push(text.to_owned());
                }
            }
            other => panic!("unknown record type `{other}` in {line}"),
        }
    }

    // The counters, so a fixture that lost records fails here instead of
    // quietly checking less.
    assert_eq!(constants, CONSTANTS);
    assert_eq!(mounts_made, MOUNTS_MADE);
    assert_eq!(mount_records, MOUNT_RECORDS);
    assert_eq!(mount_blocks, MOUNT_BLOCKS);
    assert_eq!(gets, GETS);
    assert_eq!(unlocked_gets, UNLOCKED_GETS);
    assert_eq!(retains, RETAINS);
    assert_eq!(releases, RELEASES);
    assert_eq!(checks, CHECKS);
    assert_eq!(writes, WRITES);
    assert_eq!(table_entries, TABLE_ENTRIES);
    assert_eq!(walks, WALKS);
    assert_eq!(mount_index, 0, "a block of mount records was left open");
    assert_eq!(table_index, 0, "a walk was left open");

    // The filesystem saw exactly what the C's saw, in the same order.
    assert_eq!(hook_record_count, Some(HOOK_CALLS));
    assert_eq!(expected_hooks.len(), HOOK_CALLS);
    let log = HOOK_LOG.lock().expect("the log is never poisoned").clone();
    assert_eq!(log, expected_hooks, "the orphan hook log");

    // Every reference the script took was given back.
    for ((point, number), stack) in &handles {
        assert!(
            stack.is_empty(),
            "{} references to {} #{} were never released",
            stack.len(),
            String::from_utf8_lossy(point),
            number
        );
    }
    assert!(
        inodes.entries().is_empty(),
        "the table is not empty: {:?}",
        inodes.entries()
    );
    HOOK_TABLE.with(|table| *table.borrow_mut() = None);
}
