//! `fs/inode.c` + `fs/inode.h` — the inode table.
//!
//! An *inode* in iSH is not a file: it is the identity of one file within one
//! mounted filesystem, plus the POSIX file-lock state that has to outlive every
//! descriptor for it. The filesystem supplies the number (a `struct statbuf`'s
//! `inode`, or a host `st_ino`), and this file answers with a refcounted object
//! that is shared by every descriptor open on that file — which is what makes
//! `fcntl(F_SETLK)` in one process exclude another, and what tells a
//! filesystem's `inode_orphaned` hook that nobody has the file open any more and
//! its metadata can go.
//!
//! ```text
//!   inodes_hash[ ino % 1024 ] -> [ inode_data{mount=A, ino=7, refcount=3}
//!                                  inode_data{mount=A, ino=1031, refcount=1}
//!                                  inode_data{mount=B, ino=7, refcount=1} ]
//! ```
//!
//! The key is the `(mount, ino)` pair, and the mount is a *pointer* comparison in
//! C: a fakefs file with inode 7 is not the same inode as a realfs file with
//! inode 7, even if both mounts were made by mounting the same directory.
//!
//! `tools/inode-dump.c` includes this file's C original and drives it — that is
//! also how it can walk the C's static `inodes_hash[]` — and
//! `tests/inode_differential.rs` replays every record it prints through the
//! port, so the refcounts, the mount's counter, the bucket of each inode and the
//! filesystem's orphan callbacks below are the C's own answers.
//!
//! # What the port keeps
//!
//! * **The two locks, in the same order.** An inode carries its own lock around
//!   its refcount; the table carries `inodes_lock` around everything else. The
//!   table lock is the outer one: `inode_release` holds it while it removes the
//!   inode from the hash and calls the filesystem, and `inode_get_unlocked` —
//!   which is called with the table lock *already held* by `generic_open`, so
//!   that nothing can destroy the inode between the open and the reference —
//!   takes the inode's lock inside it.
//! * **The reference count is the mount's too.** Creating an inode retains its
//!   mount, and releasing the last reference to an inode releases the mount, so
//!   a filesystem cannot be unmounted out from under an open file. The
//!   differential test sees this in the mount's own counter.
//! * **The orphan callback's two paths, and the difference between them.**
//!   `inode_release` calls `inode_orphaned` only when the filesystem has one;
//!   `inode_check_orphaned` calls it unconditionally, so a filesystem without
//!   the hook crashes in C. The port panics with an explanation instead; the
//!   hook is what `fs/fake.c` uses to drop a metadata row after an `unlink`.
//! * **The hash, and its collisions.** C's `ino % 1024` bucket choice is kept,
//!   including the fact that inodes 1 and 1025 share a bucket, and the bucket's
//!   list is scanned in insertion order.
//!
//! # Deliberate differences
//!
//! * **The table is an object, not a static.** As in [`crate::mount`], C's
//!   `inodes_lock` and `inodes_hash[]` are file-scope; [`InodeTable`] is the same
//!   table as a value the embedder owns, and it holds the [`MountTable`] it
//!   retains mounts through.
//! * **A reference is an [`Rc`], not a count in a freed object.** C hands out
//!   `struct inode_data *` and frees the object when the count reaches zero
//!   *while holding `inodes_lock`*; the port hands out an `Rc` per reference and
//!   drops the last one at the same point in the same order, so a caller that
//!   forgot to release leaks instead of touching freed memory.
//! * **`posix_locks` and `posix_unlock` are not here.** The per-inode list of
//!   `struct file_lock`s and the condition variable `fcntl(F_SETLKW)` waits on
//!   are `fs/lock.c`'s (254 lines of `fcntl_getlk`/`fcntl_setlk`/
//!   `file_lock_remove_owned_by`, all of which need a `struct fd` and the
//!   current task) — but the *lock* they wait on is this file's, which is why
//!   [`InodeData::lock`] is public. `fcntl` arrives with `fs/fd.c`.
//! * **The root is mounted at the empty point.** That is `kernel/init.c`, and
//!   it is what makes `mount_find` total: `strncmp(path, "", 0)` matches every
//!   path, so the last mount in the list is a catch-all, and every inode the
//!   emulator makes belongs to some mount.
//! * **`socket_id` is stored but unused.** `fs/sock.c` gives a Unix socket
//!   inode a number so the guest can be told which socket a path bound to; the
//!   field is here, with the 0 a fresh inode starts at, and nothing writes it
//!   until `fs/sock.c` is ported.

use std::cell::{Cell, RefCell};
use std::rc::Rc;

use crate::mount::{Mount, MountTable};
use crate::sync::{Lock, LockGuard};

/// `INODES_HASH_SIZE` from `fs/inode.c`: the number of buckets.
pub const INODES_HASH_SIZE: usize = 1 << 10;

/// `F_RDLCK_` from `fs/inode.h`: a shared lock.
///
/// The three lock types are the guest's `F_RDLCK`, `F_WRLCK` and `F_UNLCK`
/// under private names, because `fs/lock.c` compares them against the `type`
/// of the `struct flock_` a guest passed to `fcntl`. Nothing here uses them
/// yet; they arrive with `fs/lock.c`, which needs a `struct fd`.
pub const F_RDLCK_: i32 = 0;
/// `F_WRLCK_` from `fs/inode.h`: an exclusive lock.
pub const F_WRLCK_: i32 = 1;
/// `F_UNLCK_` from `fs/inode.h`: no lock.
pub const F_UNLCK_: i32 = 2;

/// `struct inode_data` from `fs/inode.h`: one file's identity, shared by every
/// descriptor open on it.
///
/// Only [`InodeTable`] creates one, exactly as only C's `inode_get_unlocked`
/// does, and the only way to hold one is through the [`Rc`] the table hands
/// out — a reference, which must be given back with [`InodeTable::release`].
pub struct InodeData {
    refcount: Cell<u32>,
    number: u64,
    mount: Rc<Mount>,
    socket_id: Cell<u32>,
    lock: Lock,
}

impl InodeData {
    /// `inode->refcount`: how many references are outstanding.
    #[must_use]
    pub fn refcount(&self) -> u32 {
        self.refcount.get()
    }

    /// `inode->number`: the number the filesystem gave this file.
    #[must_use]
    pub fn number(&self) -> u64 {
        self.number
    }

    /// `inode->mount`: the filesystem this inode belongs to, with the inode's
    /// own reference to it.
    #[must_use]
    pub fn mount(&self) -> &Rc<Mount> {
        &self.mount
    }

    /// `inode->socket_id`, 0 until a Unix socket binds this inode.
    #[must_use]
    pub fn socket_id(&self) -> u32 {
        self.socket_id.get()
    }

    /// `inode->socket_id = id`, as `fs/sock.c` sets it.
    pub fn set_socket_id(&self, id: u32) {
        self.socket_id.set(id);
    }

    /// `inode->lock`: the lock around this inode's refcount, and the one
    /// `fs/lock.c`'s `wait_for(&inode->posix_unlock, &inode->lock, NULL)` waits
    /// on.
    #[must_use]
    pub fn lock(&self) -> &Lock {
        &self.lock
    }
}

impl std::fmt::Debug for InodeData {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("InodeData")
            .field("refcount", &self.refcount.get())
            .field("number", &self.number)
            .field("mount", &String::from_utf8_lossy(self.mount.point()))
            .field("socket_id", &self.socket_id.get())
            .finish_non_exhaustive()
    }
}

/// C's `inode->mount == mount && inode->number == ino`: the inode key.
///
/// The mount is compared by address, as C compares the pointers — a second
/// filesystem mounted over the same directory is a different mount with its own
/// inode numbers. Both the lookup and the create path use this, so that they
/// cannot disagree about what a key is.
fn same_key(inode: &InodeData, mount: &Mount, ino: u64) -> bool {
    std::ptr::eq(Rc::as_ptr(&inode.mount), mount) && inode.number == ino
}

/// One inode, as a walk over the table sees it — C's `struct inode_data` with
/// the bucket it is in, which is the one thing the walk adds to it.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct InodeEntry {
    /// The bucket the inode is in: `number % INODES_HASH_SIZE`.
    pub bucket: usize,
    /// The mount point of the filesystem the inode belongs to.
    pub point: Vec<u8>,
    /// `inode->number`: the number the filesystem gave the file.
    pub number: u64,
    /// `inode->refcount`.
    pub refcount: u32,
    /// `inode->socket_id`, 0 unless a Unix socket bound this inode.
    pub socket_id: u32,
}

/// C's `inodes_hash` and `inodes_lock`: every live inode, by `(mount, ino)`.
///
/// The embedder owns one of these and hands it to everything that opens a file;
/// it holds the mount table because creating and destroying an inode retains and
/// releases its mount, exactly as C's `inode_get_unlocked` and `inode_release`
/// call `mount_retain`/`mount_release`.
pub struct InodeTable {
    lock: Lock,
    buckets: RefCell<Vec<Vec<Rc<InodeData>>>>,
    mounts: Rc<MountTable>,
}

impl InodeTable {
    /// An empty table over `mounts`.
    #[must_use]
    pub fn new(mounts: Rc<MountTable>) -> Self {
        Self {
            lock: Lock::new(),
            buckets: RefCell::new((0..INODES_HASH_SIZE).map(|_| Vec::new()).collect()),
            mounts,
        }
    }

    /// The mount table this table retains mounts through.
    #[must_use]
    pub fn mounts(&self) -> &Rc<MountTable> {
        &self.mounts
    }

    /// Take `inodes_lock`, the way `generic_open` does when it has to hold it
    /// across an open *and* the reference that follows.
    #[must_use]
    pub fn lock(&self) -> InodeGuard<'_> {
        InodeGuard {
            table: self,
            _guard: self.lock.lock(),
        }
    }

    /// `inode_get`: find or create the inode for `(mount, ino)`, taking a
    /// reference.
    #[must_use]
    pub fn get(&self, mount: &Rc<Mount>, ino: u64) -> Rc<InodeData> {
        let guard = self.lock();
        let inode = guard.get_unlocked(mount, ino);
        drop(guard);
        inode
    }

    /// C's `inode_get_data(mount, ino) != NULL`: whether the table holds this
    /// key.
    ///
    /// This takes no lock — C's `inode_get_data` takes none either, which is
    /// what lets a filesystem's `inode_orphaned` hook ask *while* the table lock
    /// is held whether the inode it is being told about is still there. It is
    /// not: [`Self::release`] unlinks the inode before it calls the hook, and
    /// [`Self::check_orphaned`] only calls it for an inode that is already gone.
    #[must_use]
    pub fn contains(&self, mount: &Mount, ino: u64) -> bool {
        self.get_data(mount, ino).is_some()
    }

    /// `inode_check_orphaned`: call the filesystem's `inode_orphaned` for an
    /// inode nothing holds any more.
    ///
    /// # Panics
    ///
    /// If the filesystem has no `inode_orphaned` — C calls the NULL pointer
    /// here, where `inode_release` checks it first.
    pub fn check_orphaned(&self, mount: &Rc<Mount>, ino: u64) {
        let _guard = self.lock.lock();
        if !self.contains(mount, ino) {
            let orphaned = mount
                .fs()
                .inode_orphaned
                .expect("inode_check_orphaned with a filesystem that has no inode_orphaned");
            orphaned(mount, ino);
        }
    }

    /// `inode_retain`: one more reference, which the caller owns.
    ///
    /// The C takes only the inode's own lock, not the table's; so does this.
    #[must_use]
    pub fn retain(&self, inode: &Rc<InodeData>) -> Rc<InodeData> {
        let _inode_lock = inode.lock.lock();
        inode.refcount.set(inode.refcount.get() + 1);
        Rc::clone(inode)
    }

    /// `inode_release`: give a reference back, and destroy the inode when it was
    /// the last one.
    ///
    /// The order is C's: the inode leaves the hash and the filesystem is told
    /// while the table lock is still held, and the mount is released after it is
    /// dropped.
    ///
    /// # Panics
    ///
    /// If the inode has no references to give back. C's `unsigned refcount`
    /// wraps instead, which is what a double release looks like there.
    pub fn release(&self, inode: Rc<InodeData>) {
        let guard = self.lock.lock();
        // `if (--inode->refcount == 0)`, with the inode's lock taken under the
        // table's and given back before anything else happens.
        let destroyed = {
            let _inode_lock = inode.lock.lock();
            let refcount = inode.refcount.get();
            assert!(refcount > 0, "releasing an inode with no references");
            inode.refcount.set(refcount - 1);
            refcount == 1
        };
        if !destroyed {
            // the table lock drops here, as C unlocks it
            return;
        }

        self.bucket_mut(inode.number)
            .retain(|other| !Rc::ptr_eq(other, &inode));
        if let Some(orphaned) = inode.mount.fs().inode_orphaned {
            orphaned(&inode.mount, inode.number);
        }
        drop(guard);

        self.mounts.release(Rc::clone(&inode.mount));
        drop(inode);
    }

    /// Every inode in the table, one entry per inode, bucket by bucket.
    ///
    /// This has no C counterpart: C's table is reachable only through the
    /// static `inodes_hash`, and `inode_get_data` looks one key up — nothing
    /// walks it. The differential test needs the whole table to compare with
    /// the oracle's walk, and this is what it walks. The order is buckets
    /// ascending and, within a bucket, by `(number, mount point)`: C adds each
    /// new inode at the *head* of its bucket, and since the key is unique no
    /// lookup can observe that order, so it is not something to compare.
    #[must_use]
    pub fn entries(&self) -> Vec<InodeEntry> {
        let _guard = self.lock.lock();
        let buckets = self.buckets.borrow();
        let mut entries = Vec::new();
        for (bucket, inodes) in buckets.iter().enumerate() {
            let mut here: Vec<InodeEntry> = inodes
                .iter()
                .map(|inode| InodeEntry {
                    bucket,
                    point: inode.mount.point().to_vec(),
                    number: inode.number,
                    refcount: inode.refcount.get(),
                    socket_id: inode.socket_id.get(),
                })
                .collect();
            here.sort_by(|left, right| {
                (left.number, &left.point).cmp(&(right.number, &right.point))
            });
            entries.append(&mut here);
        }
        entries
    }

    /// `inode_get_data`: the inode for `(mount, ino)`, if it exists.
    fn get_data(&self, mount: &Mount, ino: u64) -> Option<Rc<InodeData>> {
        self.buckets.borrow()[Self::bucket_index(ino)]
            .iter()
            .find(|inode| same_key(inode, mount, ino))
            .map(Rc::clone)
    }

    fn bucket_index(ino: u64) -> usize {
        (ino % INODES_HASH_SIZE as u64) as usize
    }

    fn bucket_mut(&self, ino: u64) -> std::cell::RefMut<'_, Vec<Rc<InodeData>>> {
        std::cell::RefMut::map(self.buckets.borrow_mut(), |buckets| {
            &mut buckets[Self::bucket_index(ino)]
        })
    }
}

/// `inodes_lock`, held: the guard whose method is the C function that documents
/// it as having to be called with the lock held.
pub struct InodeGuard<'a> {
    table: &'a InodeTable,
    // Held for its `Drop` alone; see `crate::mount::MountTableGuard`.
    _guard: LockGuard<'a>,
}

impl InodeGuard<'_> {
    /// `inode_get_unlocked`: find or create the inode for `(mount, ino)` and
    /// take a reference to it, with `inodes_lock` already held.
    ///
    /// `fs/generic.c`'s `generic_open` is the one caller, and the comment in
    /// `fs/inode.h` is aimed at it: the lock has to be held from before the
    /// filesystem is asked to open the file until after its inode has a
    /// reference, or a concurrent close could destroy the inode in between.
    ///
    /// # Panics
    ///
    /// If the bucket is already borrowed, i.e. if a filesystem's open called
    /// back into the table; C would deadlock instead (the same lock is not
    /// recursive).
    #[must_use]
    pub fn get_unlocked(&self, mount: &Rc<Mount>, ino: u64) -> Rc<InodeData> {
        let mut buckets = self.table.buckets.borrow_mut();
        let bucket = &mut buckets[InodeTable::bucket_index(ino)];
        let inode = match bucket.iter().find(|inode| same_key(inode, mount, ino)) {
            Some(inode) => Rc::clone(inode),
            None => {
                let inode = Rc::new(InodeData {
                    refcount: Cell::new(0),
                    number: ino,
                    // C's `mount_retain(mount)`: the inode holds a reference to
                    // the filesystem it belongs to.
                    mount: self.table.mounts.retain(mount),
                    socket_id: Cell::new(0),
                    lock: Lock::new(),
                });
                bucket.push(Rc::clone(&inode));
                inode
            }
        };
        drop(buckets);

        // inode_retain(inode), with the table lock still held
        {
            let _inode_lock = inode.lock.lock();
            inode.refcount.set(inode.refcount.get() + 1);
        }
        inode
    }
}

impl std::fmt::Debug for InodeTable {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("InodeTable")
            .field("inodes", &self.entries())
            .finish_non_exhaustive()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::mount::FsOps;

    /// A filesystem with a root and an orphan hook, so the tests can see both
    /// ends of the inode lifecycle.
    static ORPHANS: std::sync::Mutex<Vec<u64>> = std::sync::Mutex::new(Vec::new());

    fn orphaned(_mount: &Mount, ino: u64) {
        ORPHANS.lock().expect("never poisoned").push(ino);
    }

    static FAKE_FS: FsOps = FsOps {
        name: "fake",
        magic: 0x66616b65,
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
        inode_orphaned: Some(orphaned),
    };

    /// A filesystem with no orphan hook at all: `inode_check_orphaned` on it is
    /// C's NULL call.
    static BARE_FS: FsOps = FsOps {
        inode_orphaned: None,
        ..FAKE_FS
    };

    fn table_with_root() -> (Rc<MountTable>, InodeTable) {
        let mounts = Rc::new(MountTable::new());
        mounts.register(&FAKE_FS);
        assert_eq!(mounts.do_mount(&FAKE_FS, b"src", b"", b"", 0), 0);
        let inodes = InodeTable::new(Rc::clone(&mounts));
        (mounts, inodes)
    }

    #[test]
    fn an_inode_is_shared_by_its_key_and_retains_its_mount() {
        let (mounts, inodes) = table_with_root();
        let mount = mounts.find(b"/x");

        let first = inodes.get(&mount, 1);
        assert_eq!(first.refcount(), 1);
        assert_eq!(first.number(), 1);
        assert_eq!(first.socket_id(), 0);
        assert_eq!(mount.refcount(), 2, "the inode retains its mount");

        let second = inodes.get(&mount, 1);
        assert!(Rc::ptr_eq(&first, &second));
        assert_eq!(second.refcount(), 2);

        // a different mount with the same number is a different inode
        mounts.do_mount(&FAKE_FS, b"other", b"/mnt", b"", 0);
        let other_mount = mounts.find(b"/mnt/x");
        let other = inodes.get(&other_mount, 1);
        assert!(!Rc::ptr_eq(&first, &other));
        assert_eq!(other.refcount(), 1);

        // ... and the table has both, with their own mounts
        let walked = inodes.entries();
        assert_eq!(walked.len(), 2, "the table has one inode per (mount, ino)");
        assert_eq!(walked[0].bucket, 1);
        assert_eq!(walked[0].number, 1);
        assert_eq!(walked[0].point, b"");
        assert_eq!(walked[1].number, 1);
        assert_eq!(walked[1].point, b"/mnt");

        inodes.release(second);
        inodes.release(first);
        assert_eq!(mount.refcount(), 1, "the mount was released with the inode");
        inodes.release(other);
        assert_eq!(other_mount.refcount(), 1);
        assert!(inodes.entries().is_empty());
        mounts.release(mount);
        mounts.release(other_mount);
    }

    #[test]
    fn the_last_release_tells_the_filesystem_and_stops_at_the_first_release() {
        ORPHANS.lock().expect("never poisoned").clear();
        let (mounts, inodes) = table_with_root();
        let mount = mounts.find(b"/x");

        let inode = inodes.get(&mount, 7);
        let held = inodes.retain(&inode);
        assert_eq!(inode.refcount(), 2);
        inodes.release(held);
        assert!(ORPHANS.lock().expect("never poisoned").is_empty());
        assert_eq!(inodes.entries().len(), 1);

        inodes.release(inode);
        assert_eq!(*ORPHANS.lock().expect("never poisoned"), vec![7]);
        assert!(inodes.entries().is_empty());
        assert_eq!(mount.refcount(), 1);
        mounts.release(mount);
    }

    #[test]
    fn the_orphan_check_only_reports_an_inode_that_is_gone() {
        ORPHANS.lock().expect("never poisoned").clear();
        let (mounts, inodes) = table_with_root();
        let mount = mounts.find(b"/x");

        let inode = inodes.get(&mount, 9);
        inodes.check_orphaned(&mount, 9);
        assert!(ORPHANS.lock().expect("never poisoned").is_empty());

        inodes.release(inode);
        assert_eq!(*ORPHANS.lock().expect("never poisoned"), vec![9]);
        ORPHANS.lock().expect("never poisoned").clear();

        // it was never there, or it has gone: the filesystem is told either way
        inodes.check_orphaned(&mount, 9);
        assert_eq!(*ORPHANS.lock().expect("never poisoned"), vec![9]);
        mounts.release(mount);
    }

    #[test]
    fn a_filesystem_without_an_orphan_hook_is_a_panic_not_a_crash() {
        let mounts = Rc::new(MountTable::new());
        mounts.register(&BARE_FS);
        mounts.do_mount(&BARE_FS, b"src", b"", b"", 0);
        let inodes = InodeTable::new(Rc::clone(&mounts));
        let mount = mounts.find(b"/x");

        // inode_release checks for the hook before calling it ...
        let inode = inodes.get(&mount, 1);
        inodes.release(inode);

        // ... while inode_check_orphaned calls it unconditionally, so for an
        // inode that is gone C jumps through a NULL pointer.
        let for_check = Rc::clone(&mount);
        let checked = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            inodes.check_orphaned(&for_check, 1);
        }));
        assert!(checked.is_err(), "a missing hook must be a panic");
        mounts.release(mount);
    }

    #[test]
    fn two_mounts_of_one_point_are_two_key_spaces() {
        // The key of an inode is the mount, not the mount point: a second
        // filesystem mounted over the same directory has its own inode numbers,
        // and its inode 1 is not the first mount's inode 1. C compares the
        // `struct mount *`s; so does the port, by address.
        let (mounts, inodes) = table_with_root();
        assert_eq!(mounts.do_mount(&FAKE_FS, b"second", b"", b"", 0), 0);
        let both = mounts.mounts();
        assert_eq!(both.len(), 2, "two mounts at the same point");
        let (one_mount, other_mount) = (Rc::clone(&both[0]), Rc::clone(&both[1]));
        assert_eq!(one_mount.point(), other_mount.point());

        let first = inodes.get(&one_mount, 1);
        let second = inodes.get(&other_mount, 1);
        assert!(!Rc::ptr_eq(&first, &second));
        assert_eq!(first.refcount(), 1);
        assert_eq!(second.refcount(), 1);
        assert!(inodes.contains(&one_mount, 1) && inodes.contains(&other_mount, 1));

        inodes.release(first);
        inodes.release(second);
        assert!(inodes.entries().is_empty());
        // the mounts are still in the list, and the inode's reference to them
        // is what the two releases above took back
        assert_eq!(one_mount.refcount(), 0);
        assert_eq!(other_mount.refcount(), 0);
    }

    #[test]
    fn inodes_that_share_a_bucket_are_still_distinct() {
        let (mounts, inodes) = table_with_root();
        let mount = mounts.find(b"/x");
        let one = inodes.get(&mount, 1);
        let collision = inodes.get(&mount, 1 + INODES_HASH_SIZE as u64);
        assert!(!Rc::ptr_eq(&one, &collision));
        assert_eq!(collision.number(), 1025);
        // both are in bucket 1, and the walk visits them in number order
        let walked = inodes.entries();
        assert_eq!(walked.len(), 2);
        assert_eq!(walked[0].bucket, 1, "inode 1 is in bucket 1");
        assert_eq!(walked[0].number, 1);
        assert_eq!(walked[1].number, 1 + INODES_HASH_SIZE as u64);
        assert_eq!(walked[1].bucket, 1);
        inodes.release(one);
        inodes.release(collision);
        mounts.release(mount);
    }
}
