//! `fs/mount.c` and the `struct mount`/`struct fs_ops` half of `kernel/fs.h` —
//! the mount table the whole filesystem layer looks paths up in.
//!
//! iSH has no filesystem tree of its own: it has a list of mount points on the
//! host's, and every path a guest hands the kernel is answered by the longest
//! mount point that prefixes it. `find_mount_and_trim_path` (fs/generic.c) walks
//! that list, strips the mount point off the front of the path, and hands the
//! rest to the filesystem's own `open`/`stat`/`unlink` implementation — so the
//! order of this list *is* the semantics: it must stay in descending order of
//! mount-point length, or a short mount point mounted later would shadow a long
//! one mounted earlier.
//!
//! ```text
//!   mounts:  /var/db/fakefs   (longest first)
//!            /var/db
//!            /mnt
//!            /
//! ```
//!
//! # What the port keeps
//!
//! * **The list order, and the insertion rule.** `do_mount` walks the list and
//!   inserts the new mount before the first entry whose point is no longer than
//!   its own — which appends when the new point is the shortest so far. A mount
//!   of `/` therefore lands at the end and a mount of `/a/b/c` at the front, and
//!   `mount_find` can stop at the first prefix match.
//! * **The longest-prefix rule, and its boundary.** `mount_find` matches a mount
//!   point only when the character after it is `/` or the end of the path, so a
//!   mount of `/mnt` never claims `/mnt2/x`. The port keeps the same test on the
//!   same byte, which is why it works in bytes and not on `str`. It is also why
//!   the root filesystem is mounted at the *empty* point by `kernel/init.c`:
//!   `strncmp(path, "", 0)` matches every path, and every path `mount_find` is
//!   given is normalized, so every lookup ends at the root at the latest.
//! * **The refcount, and that it is separate from being in the list.** A mount
//!   in the list has `refcount == 0` until somebody looks it up; `mount_remove`
//!   refuses to free a mount with references (`EBUSY`) precisely so a lookup
//!   that is still in flight keeps it alive. The port keeps a `Cell<u32>` and
//!   the same rules, and additionally holds an [`std::rc::Rc`] for the list and
//!   for every reference, so a mistake is a leak rather than a use after free.
//! * **What the callbacks see, and when.** `do_mount` calls the filesystem's
//!   `mount` op *before* the mount is in the list and with no lock held by the
//!   port's own bookkeeping, and `mount_remove` calls `umount` before removing
//!   the entry — both are observable through the callbacks, and the
//!   differential test records their order from the C's own functions.
//! * **`mount_param_flag`'s prefix match.** C compares `strncmp(info, flag,
//!   strlen(flag))`, so a flag matches a *prefix* of a field: `readonly` matches
//!   `readonlyx=1`. The port keeps that.
//!
//! # Deliberate differences
//!
//! * **`mount_param_flag` terminates.** C advances by `strcspn(info, ",")`,
//!   which is zero when `info` is *at* a comma, so any parameter string with a
//!   comma in it that does not match before the first comma spins forever (the
//!   `FIXME: this is shit` on the function is about exactly this). A faithful
//!   port would have to hang, so [`mount_param_flag`] skips the comma the C
//!   forgot to skip and keeps searching — the behavior the loop was clearly
//!   meant to have. Every input on which C terminates gives the same answer; the
//!   inputs on which it does not are pinned by a unit test.
//! * **The table is an object, not a static.** C has file-scope `mounts` and
//!   `mounts_lock`, which every module reaches for directly. [`MountTable`] is
//!   the same table as a value the embedder owns and passes to the filesystem
//!   layer — the shape `FutexTable` and `TaskTable` already have, and the only
//!   one that works while the payload a filesystem stores in its mount
//!   (`fakefs_db`, i.e. a SQLite connection) is neither `Send` nor `Sync`.
//! * **`fs_register`'s table starts empty.** C seeds `filesystems[]` with four
//!   entries that are compiled-in values of `struct fs_ops`; those values live in
//!   files that are not ported yet, so [`SEEDED_FILESYSTEMS`] carries their
//!   names and order and [`MountTable::new`] starts with nothing registered. The
//!   differential test reads the four names out of the C sources and stands in
//!   for them, which is also why a registration in the fixture lands in slot 4.
//! * **`fs_ops`' descriptor half is not here yet.** `kernel/fs.h`'s `fs_ops` has
//!   six members that take or return a `struct fd *` — `open`, `close`, `fstat`,
//!   `fsetattr`, `getpath` and `flock` — and `struct fd` arrives with `fs/fd.c`.
//!   They are absent rather than stubbed, and nothing in this file calls them.
//! * **The allocator's failures are not modelled.** `do_mount` returns `_ENOMEM`
//!   when `malloc` fails; in the port that is an abort, as it is everywhere else
//!   in this crate.
//! * **`mount_find` panics where C has undefined behaviour.** C's loop walks off
//!   the end of the list into the sentinel head if nothing matches and then
//!   increments a refcount inside it; [`MountTable`] panics instead.
//! * **`root_fd` starts at -1, not uninitialized.** C leaves the field
//!   uninitialized and lets the filesystem's `mount` op fill it in; the port
//!   starts at -1, the value a failed `open` would have left, so a filesystem
//!   that never sets it fails cleanly instead of reading a stale descriptor
//!   number — which in C could be `0` and close the guest's stdin.
//! * **`STRACE`/`FIXME` output is not ported.** C prints the flags it does not
//!   understand with `FIXME`; there is no tracer here, so only the return value
//!   survives. (`sys_mount` and `sys_umount2` themselves are still pending: they
//!   need `generic_statat`, `path_normalize` and `current->fs`.)

use std::any::Any;
use std::cell::{Cell, RefCell};
use std::fmt;
use std::rc::Rc;

use crate::dev::DevT;
use crate::fake_db::FakeDb;
use crate::path::path_is_normalized;
use crate::stat::{Statbuf, Statfsbuf};
use crate::sync::{Lock, LockGuard, Timespec};

/// `MAX_FILESYSTEMS` from `fs/mount.c`: the size of the filesystem table.
pub const MAX_FILESYSTEMS: usize = 10;

/// The filesystems `fs/mount.c` seeds its table with, in order.
///
/// These are the four `extern const struct fs_ops` values `kernel/fs.h`
/// declares. The port has no value for any of them yet — `real` arrives with
/// `fs/real.c`, `proc` with `fs/proc.c`, `devpts` with `fs/pty.c` and `tmpfs`
/// with `fs/tmp.c` — so this list is the record of what [`MountTable::new`]
/// will hold once they are ported, and `tests/mount_differential.rs` checks the
/// names against the C's own initializer.
pub const SEEDED_FILESYSTEMS: [&str; 4] = ["real", "proc", "devpts", "tmpfs"];

/// `MS_READONLY_` from `kernel/calls.h`.
pub const MS_READONLY: i32 = 1 << 0;
/// `MS_NOSUID_` from `kernel/calls.h`.
pub const MS_NOSUID: i32 = 1 << 1;
/// `MS_NODEV_` from `kernel/calls.h`.
pub const MS_NODEV: i32 = 1 << 2;
/// `MS_NOEXEC_` from `kernel/calls.h`.
pub const MS_NOEXEC: i32 = 1 << 3;
/// `MS_SILENT_` from `kernel/calls.h`.
pub const MS_SILENT: i32 = 1 << 15;

/// `MS_SUPPORTED` from `fs/mount.c`: the flags `sys_mount` accepts. Anything
/// else is `_EINVAL`.
pub const MS_SUPPORTED: i32 = MS_READONLY | MS_NOSUID | MS_NODEV | MS_NOEXEC | MS_SILENT;

/// `MS_FLAGS` from `fs/mount.c`: the subset of `MS_SUPPORTED` that is stored in
/// `mount->flags` — `MS_SILENT` is a property of the mount call, not of the
/// mount.
pub const MS_FLAGS: i32 = MS_READONLY | MS_NOSUID | MS_NODEV | MS_NOEXEC;

/// `_EBUSY` from `kernel/errno.h`: `mount_remove` was called on a referenced
/// mount.
pub const EBUSY: i32 = -16;
/// `_EINVAL` from `kernel/errno.h`: `do_umount` was called on a point that is
/// not a mount.
pub const EINVAL: i32 = -22;

/// `struct attr` from `kernel/fs.h`: one attribute of a file, tagged by which
/// one it is.
///
/// C spells this as a tagged union with a `make_attr(uid, thing)` macro; the
/// payload widths are the C's own (`mode_t_` is a `word_t`, so 16 bits).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Attr {
    /// `attr_uid`, `uid_t_`.
    Uid(u32),
    /// `attr_gid`, `uid_t_`.
    Gid(u32),
    /// `attr_mode`, `mode_t_`.
    Mode(u16),
    /// `attr_size`, `off_t_`.
    Size(i64),
}

/// `struct fs_ops` from `kernel/fs.h`: what a filesystem can do.
///
/// Every operation is optional unless the comment in the C says otherwise; the
/// two the C marks required — `stat`, and `getpath`, which is one of the six
/// members that wait for `fs/fd.c` — are the reason `stat` is here and not in
/// the fd half. A missing mutating op is `_EPERM` in C's `generic_*` helpers
/// (fs/generic.c), not here: this table only records which ones exist.
///
/// The port passes paths as byte slices without their terminator (the C's
/// `const char *`), and the filesystem implementations compare them with the
/// same byte-for-byte rules.
///
/// The member types are named because a six-argument function pointer is what
/// `clippy::type_complexity` exists for; each alias is one member of the C
/// struct with its `struct mount *` first argument spelled out.
pub type MountOp = fn(&Mount) -> i32;
/// `int (*)(struct mount *)`: the `umount` member, whose result C discards.
pub type UmountOp = fn(&Mount) -> i32;
/// `int (*)(struct mount *, struct statfsbuf *)`.
pub type StatfsOp = fn(&Mount, &mut Statfsbuf) -> i32;
/// `ssize_t (*)(struct mount *, const char *, char *, size_t)`.
pub type ReadlinkOp = fn(&Mount, &[u8], &mut [u8]) -> isize;
/// `int (*)(struct mount *, const char *)`, and the other one-path members.
pub type PathOp = fn(&Mount, &[u8]) -> i32;
/// `int (*)(struct mount *, const char *, const char *)`.
pub type TwoPathOp = fn(&Mount, &[u8], &[u8]) -> i32;
/// `int (*)(struct mount *, const char *, mode_t_, dev_t_)`.
pub type MknodOp = fn(&Mount, &[u8], u16, DevT) -> i32;
/// `int (*)(struct mount *, const char *, mode_t_)`.
pub type MkdirOp = fn(&Mount, &[u8], u16) -> i32;
/// `int (*)(struct mount *, const char *, struct statbuf *)`.
pub type StatOp = fn(&Mount, &[u8], &mut Statbuf) -> i32;
/// `int (*)(struct mount *, const char *, struct attr)`.
pub type SetattrOp = fn(&Mount, &[u8], Attr) -> i32;
/// `int (*)(struct mount *, const char *, struct timespec, struct timespec)`.
pub type UtimeOp = fn(&Mount, &[u8], Timespec, Timespec) -> i32;
/// `void (*)(struct mount *, ino_t)`.
pub type InodeOrphanedOp = fn(&Mount, u64);

#[derive(Clone, Copy)]
pub struct FsOps {
    /// `name`: the string `sys_mount` matches the guest's filesystem type on.
    pub name: &'static str,
    /// `magic`: the `statfs` filesystem type, `0` when the filesystem has none.
    pub magic: i32,

    /// `mount`: called by `do_mount` before the mount is reachable.
    pub mount: Option<MountOp>,
    /// `umount`: called by `mount_remove` before the mount leaves the list.
    pub umount: Option<UmountOp>,
    /// `statfs`.
    pub statfs: Option<StatfsOp>,

    /// `readlink`: C returns the number of bytes written, without a terminator,
    /// or a negative error; `buf` is the caller's `MAX_PATH`-sized scratch.
    pub readlink: Option<ReadlinkOp>,
    /// `link`.
    pub link: Option<TwoPathOp>,
    /// `unlink`.
    pub unlink: Option<PathOp>,
    /// `rmdir`.
    pub rmdir: Option<PathOp>,
    /// `rename`.
    pub rename: Option<TwoPathOp>,
    /// `symlink`.
    pub symlink: Option<TwoPathOp>,
    /// `mknod`.
    pub mknod: Option<MknodOp>,
    /// `mkdir`.
    pub mkdir: Option<MkdirOp>,

    /// `stat` (required).
    pub stat: Option<StatOp>,
    /// `setattr`.
    pub setattr: Option<SetattrOp>,
    /// `utime`.
    pub utime: Option<UtimeOp>,

    /// `inode_orphaned`: called by `fs/inode.c` when the last reference to an
    /// inode of this filesystem goes away.
    pub inode_orphaned: Option<InodeOrphanedOp>,
}

impl fmt::Debug for FsOps {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        // Function pointers have no useful `Debug`, and a whole table of
        // `Some(0x…)` is noise; the name and magic are what identifies one.
        f.debug_struct("FsOps")
            .field("name", &self.name)
            .field("magic", &self.magic)
            .finish_non_exhaustive()
    }
}

/// The union at the end of `struct mount`: `void *data` or `struct fakefs_db
/// fakefs`, whichever the mounted filesystem stores.
///
/// C picks the arm with the filesystem: tmpfs keeps a pointer to its root
/// directory in `data`, and the fake filesystem keeps its SQLite handle *by
/// value* in `fakefs`. Reading the wrong arm is undefined behaviour in C; here
/// the arms are a Rust enum, and the payload types are the ported ones.
pub enum MountData {
    /// `NULL`: what `do_mount` leaves, and what a filesystem that stores
    /// nothing keeps.
    None,
    /// C's `void *data`: whatever the filesystem's `mount` op allocated. The
    /// port keeps it owned and typed, so the filesystem downcasts it back.
    Opaque(Box<dyn Any>),
    /// C's `struct fakefs_db fakefs`: the fake filesystem's database.
    Fakefs(FakeDb),
}

/// `struct mount` from `kernel/fs.h`: one mounted filesystem.
///
/// Only [`MountTableGuard::do_mount`] creates one, exactly as only C's
/// `do_mount` does, and the only way to hold one is through the [`Rc`] the table
/// hands out — a lookup's reference, which must be given back with
/// [`MountTableGuard::release`].
pub struct Mount {
    point: Vec<u8>,
    source: Vec<u8>,
    info: Vec<u8>,
    flags: i32,
    fs: &'static FsOps,
    refcount: Cell<u32>,
    root_fd: Cell<i32>,
    data: RefCell<MountData>,
}

impl Mount {
    /// `mount->point`: the normalized absolute path this mount answers for.
    #[must_use]
    pub fn point(&self) -> &[u8] {
        &self.point
    }

    /// `mount->source`: the guest-visible device or directory the mount came
    /// from (`/dev/disk`, or whatever `mount(2)` was told).
    #[must_use]
    pub fn source(&self) -> &[u8] {
        &self.source
    }

    /// `mount->info`: the option string `mount(2)` received.
    #[must_use]
    pub fn info(&self) -> &[u8] {
        &self.info
    }

    /// `mount->flags`: the `MS_*` flags the mount was created with.
    #[must_use]
    pub fn flags(&self) -> i32 {
        self.flags
    }

    /// `mount->fs`: the filesystem this mount belongs to.
    #[must_use]
    pub fn fs(&self) -> &'static FsOps {
        self.fs
    }

    /// `mount->refcount`: how many lookups are outstanding.
    #[must_use]
    pub fn refcount(&self) -> u32 {
        self.refcount.get()
    }

    /// `mount->root_fd`: the host descriptor the filesystem's `mount` op opened
    /// its root with, or -1.
    #[must_use]
    pub fn root_fd(&self) -> i32 {
        self.root_fd.get()
    }

    /// `mount->root_fd = fd`, as `realfs_mount` does.
    pub fn set_root_fd(&self, fd: i32) {
        self.root_fd.set(fd);
    }

    /// `mount->data = thing`.
    pub fn set_data(&self, data: MountData) {
        *self.data.borrow_mut() = data;
    }

    /// Read `mount->data` out, leaving `NULL` behind: how C's `umount` ops hand
    /// back the allocation they stored.
    pub fn take_data(&self) -> MountData {
        std::mem::replace(&mut *self.data.borrow_mut(), MountData::None)
    }

    /// `mount->data` as the filesystem's own code sees it — `struct tmpfs_data
    /// *root = mount->data;` — without moving it out.
    ///
    /// # Panics
    ///
    /// If the closure re-enters this mount's data; C would have the same
    /// problem with a pointer, but not a panic to report it.
    pub fn with_data<R>(&self, f: impl FnOnce(&mut MountData) -> R) -> R {
        f(&mut self.data.borrow_mut())
    }
}

impl fmt::Debug for Mount {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("Mount")
            .field("point", &String::from_utf8_lossy(&self.point))
            .field("source", &String::from_utf8_lossy(&self.source))
            .field("info", &String::from_utf8_lossy(&self.info))
            .field("flags", &self.flags)
            .field("fs", &self.fs.name)
            .field("refcount", &self.refcount.get())
            .field("root_fd", &self.root_fd.get())
            .finish_non_exhaustive()
    }
}

/// C's `mounts` list and `mounts_lock`: the mount points, longest first.
///
/// The embedder owns one of these and hands it to everything that walks the
/// filesystem; see the module documentation for why it is not a `static`.
pub struct MountTable {
    lock: Lock,
    mounts: RefCell<Vec<Rc<Mount>>>,
    filesystems: RefCell<Vec<&'static FsOps>>,
}

impl Default for MountTable {
    fn default() -> Self {
        Self::new()
    }
}

impl MountTable {
    /// An empty table with no filesystems registered.
    ///
    /// C's table starts with [`SEEDED_FILESYSTEMS`]; the port's starts empty
    /// until those four exist, and the end-to-end path (`kernel/fs.c`'s mount
    /// of `/`) registers them.
    #[must_use]
    pub fn new() -> Self {
        Self {
            lock: Lock::new(),
            mounts: RefCell::new(Vec::new()),
            filesystems: RefCell::new(Vec::new()),
        }
    }

    /// `fs_register`: add a filesystem to the table.
    ///
    /// C fills the first `NULL` slot of a fixed array and asserts when there is
    /// none; the port pushes and panics at the same limit.
    ///
    /// # Panics
    ///
    /// When [`MAX_FILESYSTEMS`] filesystems are already registered.
    pub fn register(&self, fs: &'static FsOps) {
        let mut filesystems = self.filesystems.borrow_mut();
        assert!(
            filesystems.len() < MAX_FILESYSTEMS,
            "reached filesystem limit"
        );
        filesystems.push(fs);
    }

    /// The registered filesystems, in registration order — C's `filesystems[]`
    /// with its `NULL`s trimmed.
    #[must_use]
    pub fn filesystems(&self) -> Vec<&'static FsOps> {
        self.filesystems.borrow().clone()
    }

    /// Take `mounts_lock`, the way C's callers do when they have several list
    /// operations to do at once.
    #[must_use]
    pub fn lock(&self) -> MountTableGuard<'_> {
        MountTableGuard {
            table: self,
            _guard: self.lock.lock(),
        }
    }

    /// `mount_find`, with the lock taken and released around it.
    ///
    /// # Panics
    ///
    /// See [`MountTableGuard::find`].
    #[must_use]
    pub fn find(&self, path: &[u8]) -> Rc<Mount> {
        self.lock().find(path)
    }

    /// `mount_retain`, with the lock taken and released around it.
    #[must_use]
    pub fn retain(&self, mount: &Rc<Mount>) -> Rc<Mount> {
        self.lock().retain(mount)
    }

    /// `mount_release`, with the lock taken and released around it.
    pub fn release(&self, mount: Rc<Mount>) {
        self.lock().release(mount);
    }

    /// `do_mount`, with the lock taken and released around it.
    ///
    /// C requires the caller to hold `mounts_lock` around `do_mount`; the port's
    /// guard is that requirement, and this one-shot form is for callers (like
    /// the ported `sys_mount`) that have nothing else to do inside the lock.
    pub fn do_mount(
        &self,
        fs: &'static FsOps,
        source: &[u8],
        point: &[u8],
        info: &[u8],
        flags: i32,
    ) -> i32 {
        self.lock().do_mount(fs, source, point, info, flags)
    }

    /// `do_umount`, with the lock taken and released around it.
    pub fn do_umount(&self, point: &[u8]) -> i32 {
        self.lock().do_umount(point)
    }

    /// `mount_remove`, with the lock taken and released around it.
    pub fn mount_remove(&self, mount: &Rc<Mount>) -> i32 {
        self.lock().mount_remove(mount)
    }

    /// The live mounts in list order, for C's unlocked walks over `&mounts`
    /// (`contains_mount_point`).
    #[must_use]
    pub fn mounts(&self) -> Vec<Rc<Mount>> {
        self.lock().mounts()
    }
}

/// `mounts_lock`, held: the guard whose methods are the C functions that
/// `kernel/fs.h` says "must hold mounts_lock while calling".
///
/// Holding one of these means holding the same lock C's callers do, so a
/// sequence like fs/generic.c's trim-then-stat or kernel/exit.c's unmount-all
/// can be ported with the same bracketing. Taking the lock again while it is
/// held is a deadlock, exactly as it is with C's non-recursive `pthread_mutex`.
pub struct MountTableGuard<'a> {
    table: &'a MountTable,
    // Held for its `Drop` alone: the lock is what makes the methods above
    // atomic against every other caller, exactly as C's `mounts_lock` does.
    // C frees a mount while holding it too (`mount_remove`), so the port's
    // `release` drops the last `Rc` in here rather than after unlocking.
    _guard: LockGuard<'a>,
}

impl MountTableGuard<'_> {
    /// `mount_find`: the longest mount point that prefixes `path`, retained.
    ///
    /// The returned reference must be given back with [`Self::release`] (C's
    /// "returns a reference, which must be released").
    ///
    /// # Panics
    ///
    /// If `path` is not normalized (C asserts), if no filesystem is mounted (C
    /// asserts), or if no mount point matches it (C walks off the list into its
    /// own head).
    #[must_use]
    pub fn find(&self, path: &[u8]) -> Rc<Mount> {
        assert!(
            path_is_normalized(path),
            "mount_find needs a normalized path, got {:?}",
            String::from_utf8_lossy(path)
        );
        let mounts = self.table.mounts.borrow();
        assert!(!mounts.is_empty(), "there is no root filesystem mounted");
        let mount = mounts
            .iter()
            .find(|mount| {
                let n = mount.point.len();
                path.starts_with(&mount.point) && (path.len() == n || path[n] == b'/')
            })
            .unwrap_or_else(|| {
                panic!("no mount point matches {:?}", String::from_utf8_lossy(path))
            });
        mount.refcount.set(mount.refcount.get() + 1);
        Rc::clone(mount)
    }

    /// `mount_retain`: one more reference, which the caller owns.
    #[must_use]
    pub fn retain(&self, mount: &Rc<Mount>) -> Rc<Mount> {
        mount.refcount.set(mount.refcount.get() + 1);
        Rc::clone(mount)
    }

    /// `mount_release`: give a reference back.
    ///
    /// The C takes a pointer and only decrements; the port takes the [`Rc`] the
    /// reference *is*, so the mount is freed when the last one goes away.
    ///
    /// # Panics
    ///
    /// If the mount has no references to give back. C's `unsigned refcount`
    /// wraps instead, which is what a use-after-release looks like there.
    pub fn release(&self, mount: Rc<Mount>) {
        let refcount = mount.refcount.get();
        assert!(refcount > 0, "releasing a mount with no references");
        mount.refcount.set(refcount - 1);
        drop(mount);
    }

    /// `do_mount`: call the filesystem's `mount` op, then put the new mount in
    /// its place in the list.
    ///
    /// Returns the filesystem's error, or 0.
    pub fn do_mount(
        &self,
        fs: &'static FsOps,
        source: &[u8],
        point: &[u8],
        info: &[u8],
        flags: i32,
    ) -> i32 {
        let mount = Rc::new(Mount {
            point: point.to_vec(),
            source: source.to_vec(),
            info: info.to_vec(),
            flags,
            fs,
            refcount: Cell::new(0),
            // C leaves this uninitialized; -1 is what a failed `open` leaves,
            // and what every reader already treats as "no descriptor".
            root_fd: Cell::new(-1),
            data: RefCell::new(MountData::None),
        });

        if let Some(mount_op) = fs.mount {
            let err = mount_op(&mount);
            if err < 0 {
                // C frees point and source here and leaks info; dropping the
                // mount releases all three.
                return err;
            }
        }

        // the list must stay in descending order of mount point length
        let mut mounts = self.table.mounts.borrow_mut();
        let at = mounts
            .iter()
            .position(|other| other.point.len() <= mount.point.len())
            .unwrap_or(mounts.len());
        mounts.insert(at, mount);
        0
    }

    /// `mount_remove`: unmount and free a mount that nothing references.
    ///
    /// Returns `EBUSY` if any reference is outstanding.
    ///
    /// # Panics
    ///
    /// If the mount is not in the table; C's `list_remove` would unlink
    /// whatever the stale links pointed at.
    pub fn mount_remove(&self, mount: &Rc<Mount>) -> i32 {
        if mount.refcount.get() != 0 {
            return EBUSY;
        }

        // C ignores what `umount` returns; so does the port.
        if let Some(umount) = mount.fs.umount {
            let _ = umount(mount);
        }

        let mut mounts = self.table.mounts.borrow_mut();
        let at = mounts
            .iter()
            .position(|other| Rc::ptr_eq(other, mount))
            .expect("mount_remove on a mount that is not in the table");
        mounts.remove(at);
        0
    }

    /// `do_umount`: remove the mount whose point is exactly `point`.
    pub fn do_umount(&self, point: &[u8]) -> i32 {
        let mount = self
            .table
            .mounts
            .borrow()
            .iter()
            .find(|mount| mount.point == point)
            .map(Rc::clone);
        match mount {
            Some(mount) => self.mount_remove(&mount),
            None => EINVAL,
        }
    }

    /// The live mounts in list order — C's `list_for_each_entry(&mounts, ...)`.
    #[must_use]
    pub fn mounts(&self) -> Vec<Rc<Mount>> {
        self.table.mounts.borrow().clone()
    }
}

impl fmt::Debug for MountTable {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("MountTable")
            .field("mounts", &self.mounts.borrow())
            .field("filesystems", &self.filesystems.borrow().len())
            .finish_non_exhaustive()
    }
}

/// `mount_param_flag` from `fs/mount.c`: whether a mount option string carries
/// a flag.
///
/// Flags live in a comma-separated parameter list (`"rw,size=64k,nosuid"`), and
/// C tests each field with a prefix match. See the module documentation for the
/// one deviation: C hangs on a parameter string whose comma it reaches without
/// having matched, because it never skips the comma, so the port skips it.
#[must_use]
pub fn mount_param_flag(info: &[u8], flag: &[u8]) -> bool {
    let mut info = info;
    while !info.is_empty() {
        if info.starts_with(flag) {
            return true;
        }
        // `info += strcspn(info, ",")`: the next field, or the end.
        match info.iter().position(|&byte| byte == b',') {
            // `strcspn` is 0 at a comma, which is where C stops advancing.
            Some(0) => info = &info[1..],
            Some(at) => info = &info[at..],
            None => return false,
        }
    }
    false
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A filesystem that records nothing and does nothing, for the tests that
    /// only care about the table.
    static EMPTY_FS: FsOps = FsOps {
        name: "empty",
        magic: 0,
        mount: None,
        umount: None,
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
    };

    /// A table with the root mounted the way `kernel/init.c` does it — at the
    /// *empty* mount point, which is why `mount_find` finds something for every
    /// normalized path: `strncmp(path, "", 0)` matches anything, and every
    /// absolute path starts with a slash.
    fn table_with_root() -> MountTable {
        let table = MountTable::new();
        table.register(&EMPTY_FS);
        assert_eq!(table.do_mount(&EMPTY_FS, b"/dev/disk", b"", b"", 0), 0);
        table
    }

    #[test]
    fn a_mount_lands_in_descending_order_of_point_length() {
        let table = table_with_root();
        for point in [
            &b"/mnt"[..],
            &b"/var/db/fakefs"[..],
            &b"/mnt/db"[..],
            &b"/var"[..],
        ] {
            assert_eq!(table.do_mount(&EMPTY_FS, b"src", point, b"", 0), 0);
        }
        let points: Vec<String> = table
            .mounts()
            .iter()
            .map(|mount| String::from_utf8_lossy(mount.point()).into_owned())
            .collect();
        // longest first, and the empty root last: a new mount of the same
        // length goes before the ones already there.
        assert_eq!(points, ["/var/db/fakefs", "/mnt/db", "/var", "/mnt", ""]);
    }

    #[test]
    fn a_mount_point_matches_only_at_a_component_boundary() {
        let table = table_with_root();
        assert_eq!(table.do_mount(&EMPTY_FS, b"src", b"/mnt", b"", 0), 0);

        let found = table.find(b"/mnt/x");
        assert_eq!(found.point(), b"/mnt");
        assert_eq!(found.refcount(), 1);
        table.release(found);

        let found = table.find(b"/mnt");
        assert_eq!(found.point(), b"/mnt");
        table.release(found);

        // `/mnt2` is not under `/mnt`, so the chain falls through to the root
        // at the empty point.
        let found = table.find(b"/mnt2/x");
        assert_eq!(found.point(), b"");
        table.release(found);

        // ... and so does a path with no mount of its own at all.
        let found = table.find(b"/elsewhere");
        assert_eq!(found.point(), b"");
        table.release(found);
    }

    #[test]
    fn a_referenced_mount_cannot_be_removed() {
        let table = table_with_root();
        assert_eq!(table.do_mount(&EMPTY_FS, b"src", b"/mnt", b"", 0), 0);

        let found = table.find(b"/mnt/x");
        assert_eq!(table.mount_remove(&found), EBUSY);
        assert_eq!(table.do_umount(b"/mnt"), EBUSY);
        table.release(found);

        assert_eq!(table.do_umount(b"/mnt"), 0);
        assert_eq!(table.do_umount(b"/mnt"), EINVAL);
        assert_eq!(table.mounts().len(), 1);
        assert_eq!(table.mounts()[0].point(), b"");
    }

    #[test]
    fn a_fresh_mount_has_the_arguments_it_was_given_and_no_root() {
        // C leaves `root_fd` uninitialized and leans on the filesystem's own
        // `mount` op to fill it in; the port starts at -1, which is what a
        // failed `open` leaves, so a filesystem that never sets it fails
        // cleanly instead of reading a stale descriptor number.
        let table = MountTable::new();
        assert_eq!(table.do_mount(&EMPTY_FS, b"src", b"", b"ro", 0), 0);
        let mount = &table.mounts()[0];
        assert_eq!(mount.point(), b"");
        assert_eq!(mount.source(), b"src");
        assert_eq!(mount.info(), b"ro");
        assert_eq!(mount.flags(), 0);
        assert_eq!(mount.refcount(), 0);
        assert_eq!(mount.root_fd(), -1);
    }

    #[test]
    fn a_mount_that_fails_is_not_in_the_table() {
        static FAILING_FS: FsOps = FsOps {
            mount: Some(|_mount| EINVAL),
            ..EMPTY_FS
        };
        let table = table_with_root();
        assert_eq!(table.do_mount(&FAILING_FS, b"src", b"/mnt", b"", 0), EINVAL);
        assert_eq!(table.mounts().len(), 1);
    }

    #[test]
    fn the_filesystem_table_registers_up_to_its_limit() {
        let table = MountTable::new();
        assert!(table.filesystems().is_empty());
        for _ in 0..MAX_FILESYSTEMS {
            table.register(&EMPTY_FS);
        }
        assert_eq!(table.filesystems().len(), MAX_FILESYSTEMS);
        // The table is full, so the next registration is C's assert.
        let table = std::panic::AssertUnwindSafe(&table);
        let overflow = std::panic::catch_unwind(|| table.register(&EMPTY_FS));
        assert!(overflow.is_err(), "the eleventh register must panic");
    }

    #[test]
    fn the_seeded_filesystems_are_the_cs_four() {
        assert_eq!(SEEDED_FILESYSTEMS, ["real", "proc", "devpts", "tmpfs"]);
    }

    #[test]
    fn a_parameter_list_is_matched_by_prefix() {
        assert!(mount_param_flag(b"ro,nosuid", b"ro"));
        assert!(mount_param_flag(b"ro,nosuid", b"nosuid"));
        // a prefix match, as C's strncmp is
        assert!(mount_param_flag(b"readonlyx=1", b"readonly"));
        assert!(!mount_param_flag(b"ro,nosuid", b"noexec"));
        // C's loop ends when strcspn runs off the end of a comma-free string
        assert!(!mount_param_flag(b"ro", b"nosuid"));
        assert!(!mount_param_flag(b"", b"ro"));
        // an empty flag matches any non-empty string, as strncmp(.., 0) does
        assert!(mount_param_flag(b"ro", b""));
        assert!(!mount_param_flag(b"", b""));
    }

    #[test]
    fn the_parameter_scan_terminates_where_c_would_spin() {
        // C never skips the comma it lands on, so every one of these would
        // hang; the port keeps searching, which is what the loop meant to do.
        assert!(mount_param_flag(b"ro,nosuid", b"nosuid"));
        assert!(mount_param_flag(b"a=1,b=2,c=3", b"c=3"));
        assert!(!mount_param_flag(b"a=1,b=2", b"absent"));
        // ... and a flag that starts with the comma still matches, as it does
        // in C before the loop gives up.
        assert!(mount_param_flag(b"a,b", b",b"));
    }
}
