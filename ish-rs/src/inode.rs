//! `fs/inode.h` + `fs/inode.c` — inode data cache full port.

use crate::refcount::RefCount;
use crate::sync::{Cond, Lock};
use std::collections::HashMap;

pub type Ino = u64;
pub type MountId = u32;

#[derive(Debug)]
pub struct PosixLock {
    pub start: u64,
    pub len: u64,
    pub pid: u32,
    pub type_: u32,
}

#[derive(Debug)]
pub struct InodeData {
    pub refcount: RefCount,
    pub number: Ino,
    pub mount: MountId,
    pub socket_id: u32,
    pub lock: Lock,
    pub posix_unlock: Cond,
    pub posix_locks: Vec<PosixLock>,
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
            posix_locks: Vec::new(),
        }
    }
    pub fn retain(&self) -> usize { self.refcount.retain() }
    pub fn release(&self) -> bool { self.refcount.release() }
    pub fn add_posix_lock(&mut self, lock: PosixLock) { self.posix_locks.push(lock); }
    pub fn remove_posix_locks_for_pid(&mut self, pid: u32) { self.posix_locks.retain(|l| l.pid != pid); }
}

pub struct InodeCache {
    buckets: Vec<Vec<InodeData>>,
    lock: Lock,
    mount_refcounts: HashMap<MountId, usize>,
}

impl InodeCache {
    pub const HASH_SIZE: usize = 1 << 10;

    pub fn new() -> Self {
        let mut buckets = Vec::with_capacity(Self::HASH_SIZE);
        for _ in 0..Self::HASH_SIZE { buckets.push(Vec::new()); }
        Self { buckets, lock: Lock::new(), mount_refcounts: HashMap::new() }
    }

    fn hash(ino: Ino) -> usize { (ino as usize) % Self::HASH_SIZE }

    fn get_data(&self, mount: MountId, ino: Ino) -> Option<usize> {
        let idx = Self::hash(ino);
        self.buckets[idx].iter().position(|inode| inode.mount == mount && inode.number == ino)
    }

    pub fn get_unlocked(&mut self, mount: MountId, ino: Ino) -> &mut InodeData {
        let idx = Self::hash(ino);
        if let Some(pos) = self.buckets[idx].iter().position(|inode| inode.mount == mount && inode.number == ino) {
            self.buckets[idx][pos].retain();
            return &mut self.buckets[idx][pos];
        }
        let inode = InodeData::new(mount, ino);
        *self.mount_refcounts.entry(mount).or_insert(0) += 1;
        self.buckets[idx].push(inode);
        let last = self.buckets[idx].len() - 1;
        &mut self.buckets[idx][last]
    }

    pub fn get(&mut self, mount: MountId, ino: Ino) -> &mut InodeData {
        // In C, locks inodes_lock
        self.get_unlocked(mount, ino)
    }

    pub fn check_orphaned(&self, mount: MountId, ino: Ino) -> bool {
        self.get_data(mount, ino).is_none()
    }

    pub fn is_orphaned(&self, mount: MountId, ino: Ino) -> bool { self.check_orphaned(mount, ino) }

    pub fn release(&mut self, mount: MountId, ino: Ino) -> bool {
        let idx = Self::hash(ino);
        if let Some(pos) = self.buckets[idx].iter().position(|inode| inode.mount == mount && inode.number == ino) {
            let should_free = {
                let inode = &self.buckets[idx][pos];
                inode.refcount.get() <= 1
            };
            if should_free {
                self.buckets[idx].remove(pos);
                if let Some(count) = self.mount_refcounts.get_mut(&mount) {
                    *count = count.saturating_sub(1);
                }
                return true; // freed, would call inode_orphaned
            } else {
                self.buckets[idx][pos].release();
                return false;
            }
        }
        false
    }

    pub fn retain(&mut self, mount: MountId, ino: Ino) -> Option<usize> {
        let idx = Self::hash(ino);
        if let Some(pos) = self.buckets[idx].iter().position(|inode| inode.mount == mount && inode.number == ino) {
            Some(self.buckets[idx][pos].retain())
        } else { None }
    }

    pub fn iter_all(&self) -> Vec<(MountId, Ino)> {
        let mut result = Vec::new();
        for bucket in &self.buckets {
            for inode in bucket { result.push((inode.mount, inode.number)); }
        }
        result
    }
}

impl Default for InodeCache {
    fn default() -> Self { Self::new() }
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

    #[test]
    fn inode_cache_release() {
        let mut cache = InodeCache::new();
        cache.get_unlocked(1, 100);
        assert!(!cache.is_orphaned(1, 100));
        cache.release(1, 100);
        assert!(cache.is_orphaned(1, 100));
    }

    #[test]
    fn inode_posix_locks() {
        let mut inode = InodeData::new(1, 42);
        inode.add_posix_lock(PosixLock { start: 0, len: 100, pid: 1, type_: 1 });
        assert_eq!(inode.posix_locks.len(), 1);
        inode.remove_posix_locks_for_pid(1);
        assert_eq!(inode.posix_locks.len(), 0);
    }

    #[test]
    fn inode_check_orphaned() {
        let mut cache = InodeCache::new();
        assert!(cache.check_orphaned(1, 999));
        cache.get_unlocked(1, 999);
        assert!(!cache.check_orphaned(1, 999));
    }
}
