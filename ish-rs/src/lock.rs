//! `fs/lock.h` + `fs/lock.c` — file locking constants and logic.

pub const LOCK_SH: u32 = 1;
pub const LOCK_EX: u32 = 2;
pub const LOCK_NB: u32 = 4;
pub const LOCK_UN: u32 = 8;

pub const F_RDLCK: u32 = 0;
pub const F_WRLCK: u32 = 1;
pub const F_UNLCK: u32 = 2;

pub const F_GETLK: u32 = 5;
pub const F_SETLK: u32 = 6;
pub const F_SETLKW: u32 = 7;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct FileLock {
    pub lock_type: u32,
    pub whence: u32,
    pub start: i64,
    pub end: i64,
    pub pid: u32,
    pub owner: usize, // simplified owner id
}

impl FileLock {
    pub fn new(lock_type: u32, start: i64, end: i64, owner: usize) -> Self {
        Self { lock_type, whence: 0, start, end, pid: 0, owner }
    }

    pub fn is_unlock(&self) -> bool { self.lock_type == F_UNLCK }
    pub fn is_read(&self) -> bool { self.lock_type == F_RDLCK }
    pub fn is_write(&self) -> bool { self.lock_type == F_WRLCK }
}

/// Check if two locks overlap, matching C `file_locks_overlap`
pub fn file_locks_overlap(a: &FileLock, b: &FileLock) -> bool {
    a.end >= b.start && b.end >= a.start
}

/// Check if two locks conflict, matching C `file_locks_conflict`
pub fn file_locks_conflict(a: &FileLock, b: &FileLock) -> bool {
    if a.owner == b.owner { return false; }
    if !file_locks_overlap(a, b) { return false; }
    if a.lock_type == F_WRLCK || b.lock_type == F_WRLCK { return true; }
    false
}

/// Check if two locks are adjacent, matching C `file_locks_adjacent`
pub fn file_locks_adjacent(a: &FileLock, b: &FileLock) -> bool {
    a.end == b.start - 1 || b.end == a.start - 1
}

/// Simplified inode lock table
#[derive(Debug, Default)]
pub struct InodeLocks {
    pub locks: Vec<FileLock>,
}

impl InodeLocks {
    pub fn new() -> Self { Self { locks: Vec::new() } }

    pub fn test(&self, request: &FileLock) -> Option<&FileLock> {
        self.locks.iter().find(|l| file_locks_conflict(l, request))
    }

    pub fn acquire(&mut self, mut request: FileLock) -> Result<(), i32> {
        // Check conflicts first
        if request.lock_type != F_UNLCK {
            for lock in &self.locks {
                if file_locks_conflict(lock, &request) {
                    return Err(-11); // EAGAIN
                }
            }
        }

        // Remove/merge logic simplified from C
        let mut i = 0;
        while i < self.locks.len() {
            if self.locks[i].owner != request.owner {
                i += 1;
                continue;
            }

            if request.lock_type == self.locks[i].lock_type {
                if file_locks_overlap(&self.locks[i], &request) || file_locks_adjacent(&request, &self.locks[i]) {
                    // Merge
                    if self.locks[i].start < request.start {
                        request.start = self.locks[i].start;
                    }
                    if self.locks[i].end > request.end {
                        request.end = self.locks[i].end;
                    }
                    self.locks.remove(i);
                    continue;
                }
            } else {
                if file_locks_overlap(&self.locks[i], &request) {
                    // Subtract logic simplified
                    if request.start > self.locks[i].start && request.end < self.locks[i].end {
                        // Split
                        let mut lock2 = self.locks[i];
                        lock2.start = request.end + 1;
                        self.locks[i].end = request.start - 1;
                        self.locks.insert(i + 1, lock2);
                        i += 2;
                        continue;
                    } else if request.start <= self.locks[i].start && request.end >= self.locks[i].end {
                        self.locks.remove(i);
                        continue;
                    } else if self.locks[i].start < request.start {
                        self.locks[i].end = request.start - 1;
                    } else if self.locks[i].end > request.end {
                        self.locks[i].start = request.end + 1;
                    }
                }
            }
            i += 1;
        }

        if request.lock_type != F_UNLCK {
            self.locks.push(request);
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn lock_constants_match_c() {
        assert_eq!(LOCK_SH, 1);
        assert_eq!(LOCK_EX, 2);
        assert_eq!(LOCK_UN, 8);
        assert_eq!(F_RDLCK, 0);
        assert_eq!(F_WRLCK, 1);
        assert_eq!(F_UNLCK, 2);
    }

    #[test]
    fn file_locks_overlap_and_conflict_test() {
        let a = FileLock::new(F_WRLCK, 0, 10, 1);
        let b = FileLock::new(F_WRLCK, 5, 15, 2);
        assert!(file_locks_overlap(&a, &b));
        assert!(file_locks_conflict(&a, &b));

        let c = FileLock::new(F_RDLCK, 0, 10, 1);
        let d = FileLock::new(F_RDLCK, 5, 15, 2);
        assert!(file_locks_overlap(&c, &d));
        assert!(!file_locks_conflict(&c, &d)); // read+read no conflict

        let e = FileLock::new(F_WRLCK, 0, 10, 1);
        let f = FileLock::new(F_WRLCK, 0, 10, 1); // same owner
        assert!(!file_locks_conflict(&e, &f));

        let g = FileLock::new(F_WRLCK, 0, 10, 1);
        let h = FileLock::new(F_WRLCK, 20, 30, 2);
        assert!(!file_locks_overlap(&g, &h));
        assert!(!file_locks_conflict(&g, &h));
    }

    #[test]
    fn file_locks_adjacent_test() {
        let a = FileLock::new(F_WRLCK, 0, 10, 1);
        let b = FileLock::new(F_WRLCK, 11, 20, 1);
        assert!(file_locks_adjacent(&a, &b));
        assert!(file_locks_adjacent(&b, &a));
        let c = FileLock::new(F_WRLCK, 0, 10, 1);
        let d = FileLock::new(F_WRLCK, 12, 20, 1);
        assert!(!file_locks_adjacent(&c, &d));
    }

    #[test]
    fn inode_locks_acquire_and_test() {
        let mut locks = InodeLocks::new();
        let req1 = FileLock::new(F_WRLCK, 0, 10, 1);
        assert!(locks.acquire(req1).is_ok());
        assert_eq!(locks.locks.len(), 1);

        // Conflicting lock from different owner
        let req2 = FileLock::new(F_WRLCK, 5, 15, 2);
        assert!(locks.test(&req2).is_some());
        assert!(locks.acquire(req2).is_err());

        // Non-conflicting read locks
        let mut locks2 = InodeLocks::new();
        locks2.acquire(FileLock::new(F_RDLCK, 0, 10, 1)).unwrap();
        assert!(locks2.acquire(FileLock::new(F_RDLCK, 5, 15, 2)).is_ok());

        // Merge adjacent same owner same type
        let mut locks3 = InodeLocks::new();
        locks3.acquire(FileLock::new(F_WRLCK, 0, 10, 1)).unwrap();
        locks3.acquire(FileLock::new(F_WRLCK, 11, 20, 1)).unwrap();
        assert_eq!(locks3.locks.len(), 1);
        assert_eq!(locks3.locks[0].start, 0);
        assert_eq!(locks3.locks[0].end, 20);

        // Unlock
        locks3.acquire(FileLock::new(F_UNLCK, 0, 20, 1)).unwrap();
        assert_eq!(locks3.locks.len(), 0);
    }
}
