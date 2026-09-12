//! `fs/inode.h` + `fs/inode.c` — inode data cache.
//!
//! The C implementation maintains a hash table of `inode_data` with explicit
//! refcounting and locking. This Rust port provides the data structure and
//! retain/release logic, depending only on `refcount`, `sync`, and `list`
//! (already ported). The mount pointer is represented as an id until the
//! mount table is ported.

use crate::refcount::RefCount;
use crate::sync::{Cond, Lock};

/// `ino_t` from C.
pub type Ino = u64;

/// Placeholder for mount id.
pub type MountId = u32;

/// `struct inode_data` — simplified.
pub struct InodeData {
    pub refcount: RefCount,
    pub number: Ino,
    pub mount: MountId,
    pub socket_id: u32,
    pub lock: Lock,
    pub posix_unlock: Cond,
    // posix_locks and chain are intrusive lists in C; omitted for now
    // until `list.rs` intrusive support is extended.
}

impl InodeData {
    pub fn new(mount: MountId, number: Ino) -> Self {
        Self {
            refcount: RefCount::new(),
            number,
            mount,
            socket_id: 0,
            lock: Lock::new(),
            posix_unlock: Cond::new(),
        }
    }

    pub fn retain(&self) -> usize {
        self.refcount.retain()
    }

    pub fn release(&self) -> bool {
        self.refcount.release()
    }
}

/// Inode hash table, mirroring C's `inodes_hash`.
pub struct InodeCache {
    buckets: Vec<Vec<InodeData>>,
    lock: Lock,
}

impl InodeCache {
    pub const HASH_SIZE: usize = 1 << 10;

    pub fn new() -> Self {
        let mut buckets = Vec::with_capacity(Self::HASH_SIZE);
        for _ in 0..Self::HASH_SIZE {
            buckets.push(Vec::new());
        }
        Self {
            buckets,
            lock: Lock::new(),
        }
    }

    fn hash(ino: Ino) -> usize {
        (ino as usize) % Self::HASH_SIZE
    }

    /// `inode_get_unlocked` — get or create, retaining.
    pub fn get_unlocked(&mut self, mount: MountId, ino: Ino) -> &mut InodeData {
        let idx = Self::hash(ino);
        // Search
        if let Some(pos) = self.buckets[idx]
            .iter()
            .position(|inode| inode.mount == mount && inode.number == ino)
        {
            self.buckets[idx][pos].retain();
            return &mut self.buckets[idx][pos];
        }
        // Create
        let inode = InodeData::new(mount, ino);
        self.buckets[idx].push(inode);
        let last = self.buckets[idx].len() - 1;
        &mut self.buckets[idx][last]
    }

    /// Check if orphaned (no entry).
    pub fn is_orphaned(&self, mount: MountId, ino: Ino) -> bool {
        let idx = Self::hash(ino);
        !self.buckets[idx]
            .iter()
            .any(|inode| inode.mount == mount && inode.number == ino)
    }
}

impl Default for InodeCache {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn inode_new_has_refcount_one() {
        let inode = InodeData::new(1, 42);
        assert_eq!(inode.refcount.get(), 1);
        assert_eq!(inode.number, 42);
    }

    #[test]
    fn inode_cache_get_or_create() {
        let mut cache = InodeCache::new();
        {
            let inode = cache.get_unlocked(1, 100);
            assert_eq!(inode.number, 100);
            assert_eq!(inode.refcount.get(), 1);
        }
        {
            let inode = cache.get_unlocked(1, 100);
            assert_eq!(inode.refcount.get(), 2);
        }
        assert!(!cache.is_orphaned(1, 100));
        assert!(cache.is_orphaned(1, 101));
    }
}
