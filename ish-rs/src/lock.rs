//! `fs/lock.h` — file locking constants and structures.

/// `flock` operations (also in fd.rs, re-exported here for fs/lock.h fidelity)
pub const LOCK_SH: u32 = 1;
pub const LOCK_EX: u32 = 2;
pub const LOCK_NB: u32 = 4;
pub const LOCK_UN: u32 = 8;

/// `struct flock_` constants
pub const F_RDLCK: u32 = 0;
pub const F_WRLCK: u32 = 1;
pub const F_UNLCK: u32 = 2;

/// `fcntl` commands for file locking
pub const F_GETLK: u32 = 5;
pub const F_SETLK: u32 = 6;
pub const F_SETLKW: u32 = 7;

/// File lock type, mirroring C's `struct flock`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct FileLock {
    pub lock_type: u32,
    pub whence: u32,
    pub start: i64,
    pub len: i64,
    pub pid: u32,
}

impl FileLock {
    pub fn is_unlock(&self) -> bool {
        self.lock_type == F_UNLCK
    }

    pub fn is_read(&self) -> bool {
        self.lock_type == F_RDLCK
    }

    pub fn is_write(&self) -> bool {
        self.lock_type == F_WRLCK
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
    fn file_lock_helpers() {
        let l = FileLock {
            lock_type: F_WRLCK,
            ..Default::default()
        };
        assert!(l.is_write());
        assert!(!l.is_unlock());
    }
}
