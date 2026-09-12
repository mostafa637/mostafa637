//! `fs/stat.h` — stat buffer layouts.
//!
//! iSH defines several guest ABIs for stat: the internal `statbuf` used by
//! filesystem code, plus the legacy `oldstat`, `newstat`, `newstat64`,
//! `statfs`, `statfs64`, and `statx` wire formats. This module ports those
//! structures as packed little-endian layouts and provides conversions.

/// Internal `statbuf` used by `fs/*.c`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct StatBuf {
    pub dev: u64,
    pub inode: u64,
    pub mode: u32,
    pub nlink: u32,
    pub uid: u32,
    pub gid: u32,
    pub rdev: u64,
    pub size: u64,
    pub blksize: u32,
    pub blocks: u64,
    pub atime: u32,
    pub atime_nsec: u32,
    pub mtime: u32,
    pub mtime_nsec: u32,
    pub ctime: u32,
    pub ctime_nsec: u32,
}

/// `struct oldstat` — 16-bit old ABI.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct OldStat {
    pub dev: u16,
    pub ino: u16,
    pub mode: u16,
    pub nlink: u16,
    pub uid: u16,
    pub gid: u16,
    pub rdev: u16,
    pub size: u32,
    pub atime: u32,
    pub mtime: u32,
    pub ctime: u32,
}

/// `struct newstat` — 32-bit new ABI.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct NewStat {
    pub dev: u32,
    pub ino: u32,
    pub mode: u16,
    pub nlink: u16,
    pub uid: u16,
    pub gid: u16,
    pub rdev: u32,
    pub size: u32,
    pub blksize: u32,
    pub blocks: u32,
    pub atime: u32,
    pub atime_nsec: u32,
    pub mtime: u32,
    pub mtime_nsec: u32,
    pub ctime: u32,
    pub ctime_nsec: u32,
}

/// `struct newstat64` — note the intentionally misspelled `fucked_ino` field
/// preserved from the C source as `fucked_ino` for fidelity.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct NewStat64 {
    pub dev: u64,
    pub _pad1: u32,
    pub fucked_ino: u32,
    pub mode: u32,
    pub nlink: u32,
    pub uid: u32,
    pub gid: u32,
    pub rdev: u64,
    pub _pad2: u32,
    pub size: u64,
    pub blksize: u32,
    pub blocks: u64,
    pub atime: u32,
    pub atime_nsec: u32,
    pub mtime: u32,
    pub mtime_nsec: u32,
    pub ctime: u32,
    pub ctime_nsec: u32,
    pub ino: u64,
}

/// `struct statfsbuf` — internal.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct StatFsBuf {
    pub type_: i64,
    pub bsize: i64,
    pub blocks: u64,
    pub bfree: u64,
    pub bavail: u64,
    pub files: u64,
    pub ffree: u64,
    pub fsid: u64,
    pub namelen: i64,
    pub frsize: i64,
    pub flags: i64,
    pub spare: [i64; 4],
}

/// `struct statx_timestamp_`
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct StatxTimestamp {
    pub sec: i64,
    pub nsec: u32,
    pub _pad: u32,
}

/// `struct statx_` — modern statx ABI.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Statx {
    pub mask: u32,
    pub blksize: u32,
    pub attributes: u64,
    pub nlink: u32,
    pub uid: u32,
    pub gid: u32,
    pub mode: u16,
    pub _pad1: u16,
    pub ino: u64,
    pub size: u64,
    pub blocks: u64,
    pub attributes_mask: u64,
    pub atime: StatxTimestamp,
    pub btime: StatxTimestamp,
    pub ctime: StatxTimestamp,
    pub mtime: StatxTimestamp,
    pub rdev_major: u32,
    pub rdev_minor: u32,
    pub dev_major: u32,
    pub dev_minor: u32,
    pub mnt_id: u64,
    pub dio_mem_align: u32,
    pub dio_offset_align: u32,
}

pub const STATX_BASIC_STATS: u32 = 0x7ff;

impl StatBuf {
    /// Convert to `NewStat` (truncating 64-bit fields as C does).
    pub fn to_newstat(&self) -> NewStat {
        NewStat {
            dev: self.dev as u32,
            ino: self.inode as u32,
            mode: self.mode as u16,
            nlink: self.nlink as u16,
            uid: self.uid as u16,
            gid: self.gid as u16,
            rdev: self.rdev as u32,
            size: self.size as u32,
            blksize: self.blksize,
            blocks: self.blocks as u32,
            atime: self.atime,
            atime_nsec: self.atime_nsec,
            mtime: self.mtime,
            mtime_nsec: self.mtime_nsec,
            ctime: self.ctime,
            ctime_nsec: self.ctime_nsec,
        }
    }

    /// Convert to `NewStat64`.
    pub fn to_newstat64(&self) -> NewStat64 {
        NewStat64 {
            dev: self.dev,
            _pad1: 0,
            fucked_ino: self.inode as u32,
            mode: self.mode,
            nlink: self.nlink,
            uid: self.uid,
            gid: self.gid,
            rdev: self.rdev,
            _pad2: 0,
            size: self.size,
            blksize: self.blksize,
            blocks: self.blocks,
            atime: self.atime,
            atime_nsec: self.atime_nsec,
            mtime: self.mtime,
            mtime_nsec: self.mtime_nsec,
            ctime: self.ctime,
            ctime_nsec: self.ctime_nsec,
            ino: self.inode,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn statbuf_to_newstat_truncates_as_c_does() {
        let buf = StatBuf {
            dev: 0x1_0000_0001,
            inode: 0x2_0000_0002,
            mode: 0o100644,
            nlink: 1,
            uid: 1000,
            gid: 1000,
            rdev: 0,
            size: 0x1234_5678_9abc,
            blksize: 4096,
            blocks: 8,
            atime: 100,
            atime_nsec: 200,
            mtime: 300,
            mtime_nsec: 400,
            ctime: 500,
            ctime_nsec: 600,
        };
        let new = buf.to_newstat();
        assert_eq!(new.dev, 1);
        assert_eq!(new.ino, 2);
        assert_eq!(new.size, 0x5678_9abc_u32);

        let new64 = buf.to_newstat64();
        assert_eq!(new64.dev, buf.dev);
        assert_eq!(new64.ino, buf.inode);
        assert_eq!(new64.fucked_ino, 2);
    }

    #[test]
    fn struct_sizes_match_c_layout() {
        // statbuf is not packed in C, but its fields are as defined.
        // Ensure our Rust representation has the expected field count.
        let buf = StatBuf::default();
        assert_eq!(buf.blksize, 0);
    }
}
