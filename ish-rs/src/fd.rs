//! `fs/fd.h` — file descriptor constants and structures.

use crate::bits::bits_size;

/// `fd_t` — file descriptor number.
pub type FdT = i32;

/// `AT_FDCWD_`
pub const AT_FDCWD: FdT = -100;

/// Open flags from `kernel/fs.h`.
pub const O_ACCMODE: u32 = 3;
pub const O_RDONLY: u32 = 0;
pub const O_WRONLY: u32 = 1;
pub const O_RDWR: u32 = 2;
pub const O_CREAT: u32 = 1 << 6;
pub const O_EXCL: u32 = 1 << 7;
pub const O_NOCTTY: u32 = 1 << 8;
pub const O_TRUNC: u32 = 1 << 9;
pub const O_APPEND: u32 = 1 << 10;
pub const O_NONBLOCK: u32 = 1 << 11;
pub const O_DIRECTORY: u32 = 1 << 16;
pub const O_CLOEXEC: u32 = 1 << 19;

/// Generic ioctls
pub const FIONREAD: u32 = 0x541b;
pub const FIONBIO: u32 = 0x5421;
pub const FIONCLEX: u32 = 0x5450;
pub const FIOCLEX: u32 = 0x5451;

/// `lseek` whence
pub const LSEEK_SET: u32 = 0;
pub const LSEEK_CUR: u32 = 1;
pub const LSEEK_END: u32 = 2;

/// `flock` operations
pub const LOCK_SH: u32 = 1;
pub const LOCK_EX: u32 = 2;
pub const LOCK_NB: u32 = 4;
pub const LOCK_UN: u32 = 8;

/// `AT_*` flags
pub const AT_SYMLINK_NOFOLLOW: u32 = 0x100;
pub const AT_EMPTY_PATH: u32 = 0x1000;

/// `NAME_MAX`
pub const NAME_MAX: usize = 255;

/// File type bits (S_IFMT)
pub const S_IFMT: u32 = 0o170000;
pub const S_IFDIR: u32 = 0o040000;
pub const S_IFREG: u32 = 0o100000;
pub const S_IFLNK: u32 = 0o120000;
pub const S_IFCHR: u32 = 0o020000;
pub const S_IFBLK: u32 = 0o060000;
pub const S_IFIFO: u32 = 0o010000;
pub const S_IFSOCK: u32 = 0o140000;

/// Directory entry, matching C `struct dir_entry`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DirEntry {
    pub inode: u64,
    pub name: String,
}

impl DirEntry {
    pub fn new(inode: u64, name: impl Into<String>) -> Self {
        Self {
            inode,
            name: name.into(),
        }
    }
}

/// Fd table size helpers.
pub fn fdtable_cloexec_size(size: usize) -> usize {
    bits_size(size)
}

/// File descriptor flags.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct FdFlags {
    pub cloexec: bool,
    pub nonblock: bool,
}

impl FdFlags {
    pub fn from_open_flags(flags: u32) -> Self {
        Self {
            cloexec: (flags & O_CLOEXEC) != 0,
            nonblock: (flags & O_NONBLOCK) != 0,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn open_flags_match_c() {
        assert_eq!(O_ACCMODE, 3);
        assert_eq!(O_RDONLY, 0);
        assert_eq!(O_WRONLY, 1);
        assert_eq!(O_RDWR, 2);
        assert_eq!(O_CREAT, 64);
        assert_eq!(O_CLOEXEC, 1 << 19);
        assert_eq!(AT_FDCWD, -100);
    }

    #[test]
    fn fd_flags_from_open_flags() {
        let flags = FdFlags::from_open_flags(O_CLOEXEC | O_NONBLOCK);
        assert!(flags.cloexec);
        assert!(flags.nonblock);
        let flags2 = FdFlags::from_open_flags(O_RDONLY);
        assert!(!flags2.cloexec);
    }

    #[test]
    fn dir_entry_basic() {
        let e = DirEntry::new(1, "foo");
        assert_eq!(e.inode, 1);
        assert_eq!(e.name, "foo");
    }
}
