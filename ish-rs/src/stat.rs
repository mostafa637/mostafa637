//! `fs/stat.h` and the conversion at the top of `fs/stat.c` — the stat
//! structures the kernel hands to the guest, and `stat_convert_newstat64`.
//!
//! The C file is mostly syscall glue that this port cannot reach yet: every
//! entry point there needs `path_normalize`, `find_mount_and_trim_path`,
//! `mount_release`, `f_get`, or the `user_*` bridge, and none of those exists in
//! this crate. What *is* portable is the part every one of those syscalls
//! funnels through: the layout of the three `stat64` families and the one
//! function that converts a filesystem's [`Statbuf`] into the guest's
//! `newstat64`. That is what is ported here, and it is the part a mistake in is
//! invisible until a guest program reads a directory listing and sees an inode
//! of zero.
//!
//! ```text
//! mount->fs->fstat(fd, &stat)      // filesystem fills a struct statbuf
//! newstat = stat_convert_newstat64(stat)   // this module
//! user_put(statbuf_addr, newstat)  // whole struct, padding included
//! ```
//!
//! # The structs
//!
//! `fs/stat.h` is a gallery of guest ABI decisions:
//!
//! * [`Statbuf`] is the *host* side: `dev`/`inode`/`rdev`/`size`/`blocks` are 64
//!   bits, and the compiler inserts four bytes of padding between `blksize` and
//!   `blocks`. The padding is part of the size the filesystems memcpy around, and
//!   [`Statbuf::to_le_bytes`] therefore keeps it (as zero).
//! * [`Newstat64`], [`Statfs`] and [`Statfs64`] are `__attribute__((packed))`,
//!   which is what lets the guest's 32-bit `struct stat64` and `struct statfs64`
//!   be filled in place: the packed `newstat64` puts an 8-byte `dev` at 0, a
//!   4-byte "fucked" inode at 12, and the real 64-bit inode at 88, exactly the
//!   offsets a 32-bit i386 glibc reads. [`Statfs`] does the same for `fsid` at
//!   28 and the unaligned tail after it.
//! * [`OldStat`] and [`NewStat`] are the pre-`stat64` ABIs, kept because the
//!   field lists are the same file's business.
//!
//! This replica uses `#[repr(C)]` and `#[repr(C, packed)]` so that `size_of` and
//! `offset_of!` answer the same numbers the C compiler answers — the differential
//! test compares both against the fixture — while serialization stays explicit
//! and byte-oriented, with no transmute.
//!
//! # Deliberate differences
//!
//! * **The two padding fields are written.** `stat_convert_newstat64` leaves
//!   `_pad1` and `_pad2` unassigned, so the C copies whatever was on the stack
//!   into the guest (at `-O0` it is the previous frame; at `-O2` it happened to
//!   be zeros). That is not reproducible and not an ABI; [`stat_convert_newstat64`]
//!   writes zero, [`Newstat64::default`] starts there, and the differential
//!   fixture records the wart rather than comparing it.
//! * **The conversion takes a reference.** C takes `struct statbuf` by value;
//!   nothing about the result changes, and the caller keeps its buffer.
//! * **`rdev` is copied, not decomposed.** The C copies all 64 bits and lets the
//!   guest decide; `sys_statx` is the one caller that splits it afterwards, and
//!   it is not ported here.
//! * **No syscall wrappers.** `sys_stat64`, `sys_lstat64`, `sys_fstatat64`,
//!   `sys_fstat64` and `sys_statx` all stay in C's half of the port until the
//!   path and fd layers land. `generic_statat` — `stat` via a mount — is
//!   likewise waiting on `path_normalize`.

/// The value of `STATX_BASIC_STATS_`: what `statx` says it filled in.
pub const STATX_BASIC_STATS: u32 = 0x7ff;

/// `struct statbuf` — what a filesystem's `stat`/`fstat` fills in.
///
/// Not a guest structure: the filesystems in `fs/` fill this, and the conversion
/// functions turn it into whichever wire format the guest asked for.
#[repr(C)]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Statbuf {
    pub dev: u64,
    pub inode: u64,
    pub mode: u32,
    pub nlink: u32,
    pub uid: u32,
    pub gid: u32,
    pub rdev: u64,
    pub size: u64,
    pub blksize: u32,
    // Four bytes of compiler padding live here, between the last 32-bit field
    // and `blocks`. The C compiles with them, so the port keeps the size.
    pub blocks: u64,
    pub atime: u32,
    pub atime_nsec: u32,
    pub mtime: u32,
    pub mtime_nsec: u32,
    pub ctime: u32,
    pub ctime_nsec: u32,
}

impl Statbuf {
    /// The struct's size, padding included: 88 bytes.
    pub const SIZE: usize = std::mem::size_of::<Statbuf>();

    /// The struct as it lies in host memory.
    ///
    /// Every field is written at its own offset and the padding is zeroed, so the
    /// image is deterministic where the C's is not.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        put(&mut bytes, 0, &self.dev.to_le_bytes());
        put(&mut bytes, 8, &self.inode.to_le_bytes());
        put(&mut bytes, 16, &self.mode.to_le_bytes());
        put(&mut bytes, 20, &self.nlink.to_le_bytes());
        put(&mut bytes, 24, &self.uid.to_le_bytes());
        put(&mut bytes, 28, &self.gid.to_le_bytes());
        put(&mut bytes, 32, &self.rdev.to_le_bytes());
        put(&mut bytes, 40, &self.size.to_le_bytes());
        put(&mut bytes, 48, &self.blksize.to_le_bytes());
        // 52..56 is the padding the C compiler also leaves alone.
        put(&mut bytes, 56, &self.blocks.to_le_bytes());
        put(&mut bytes, 64, &self.atime.to_le_bytes());
        put(&mut bytes, 68, &self.atime_nsec.to_le_bytes());
        put(&mut bytes, 72, &self.mtime.to_le_bytes());
        put(&mut bytes, 76, &self.mtime_nsec.to_le_bytes());
        put(&mut bytes, 80, &self.ctime.to_le_bytes());
        put(&mut bytes, 84, &self.ctime_nsec.to_le_bytes());
        bytes
    }
}

/// `struct oldstat` — the pre-`stat64` guest layout, 32 bytes of 16- and
/// 32-bit fields.
#[repr(C)]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
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

impl OldStat {
    /// The struct's size: 32 bytes.
    pub const SIZE: usize = std::mem::size_of::<OldStat>();

    /// The struct as it lies in host memory.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        put(&mut bytes, 0, &self.dev.to_le_bytes());
        put(&mut bytes, 2, &self.ino.to_le_bytes());
        put(&mut bytes, 4, &self.mode.to_le_bytes());
        put(&mut bytes, 6, &self.nlink.to_le_bytes());
        put(&mut bytes, 8, &self.uid.to_le_bytes());
        put(&mut bytes, 10, &self.gid.to_le_bytes());
        put(&mut bytes, 12, &self.rdev.to_le_bytes());
        put(&mut bytes, 16, &self.size.to_le_bytes());
        put(&mut bytes, 20, &self.atime.to_le_bytes());
        put(&mut bytes, 24, &self.mtime.to_le_bytes());
        put(&mut bytes, 28, &self.ctime.to_le_bytes());
        bytes
    }
}

/// `struct newstat` — the 32-bit `stat` layout with 64-bit time fields and a
/// trailing 8-byte pad that nothing in iSH writes, 64 bytes in all.
#[repr(C)]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
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
    pub pad: [u8; 8],
}

impl NewStat {
    /// The struct's size: 64 bytes.
    pub const SIZE: usize = std::mem::size_of::<NewStat>();

    /// The struct as it lies in host memory.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        put(&mut bytes, 0, &self.dev.to_le_bytes());
        put(&mut bytes, 4, &self.ino.to_le_bytes());
        put(&mut bytes, 8, &self.mode.to_le_bytes());
        put(&mut bytes, 10, &self.nlink.to_le_bytes());
        put(&mut bytes, 12, &self.uid.to_le_bytes());
        put(&mut bytes, 14, &self.gid.to_le_bytes());
        put(&mut bytes, 16, &self.rdev.to_le_bytes());
        put(&mut bytes, 20, &self.size.to_le_bytes());
        put(&mut bytes, 24, &self.blksize.to_le_bytes());
        put(&mut bytes, 28, &self.blocks.to_le_bytes());
        put(&mut bytes, 32, &self.atime.to_le_bytes());
        put(&mut bytes, 36, &self.atime_nsec.to_le_bytes());
        put(&mut bytes, 40, &self.mtime.to_le_bytes());
        put(&mut bytes, 44, &self.mtime_nsec.to_le_bytes());
        put(&mut bytes, 48, &self.ctime.to_le_bytes());
        put(&mut bytes, 52, &self.ctime_nsec.to_le_bytes());
        bytes[56..64].copy_from_slice(&self.pad);
        bytes
    }
}

/// `struct newstat64` — what `stat64`, `lstat64`, `fstatat64` and `fstat64`
/// write, packed, 96 bytes.
///
/// The pack is load-bearing: `dev` is 8 bytes at 0, the 32-bit view of the
/// inode sits at 12, `size` straddles 44..52, and the full 64-bit inode is last,
/// at 88.
#[repr(C, packed)]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Newstat64 {
    pub dev: u64,
    /// Never written by the C conversion; see the module notes.
    pub _pad1: u32,
    /// The low 32 bits of the inode, for the guests that read this field.
    pub fucked_ino: u32,
    pub mode: u32,
    pub nlink: u32,
    pub uid: u32,
    pub gid: u32,
    pub rdev: u64,
    /// Never written by the C conversion; see the module notes.
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

impl Newstat64 {
    /// The struct's size: 96 bytes, packed, so there is no padding at all.
    pub const SIZE: usize = std::mem::size_of::<Newstat64>();

    /// The struct as it lies in host memory.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        // Copying a field out of a packed struct first is what the compiler
        // requires; `put` takes it by value.
        let mut bytes = [0; Self::SIZE];
        put(&mut bytes, 0, &self.dev.to_le_bytes());
        put(&mut bytes, 8, &self._pad1.to_le_bytes());
        put(&mut bytes, 12, &self.fucked_ino.to_le_bytes());
        put(&mut bytes, 16, &self.mode.to_le_bytes());
        put(&mut bytes, 20, &self.nlink.to_le_bytes());
        put(&mut bytes, 24, &self.uid.to_le_bytes());
        put(&mut bytes, 28, &self.gid.to_le_bytes());
        put(&mut bytes, 32, &self.rdev.to_le_bytes());
        put(&mut bytes, 40, &self._pad2.to_le_bytes());
        put(&mut bytes, 44, &self.size.to_le_bytes());
        put(&mut bytes, 52, &self.blksize.to_le_bytes());
        put(&mut bytes, 56, &self.blocks.to_le_bytes());
        put(&mut bytes, 64, &self.atime.to_le_bytes());
        put(&mut bytes, 68, &self.atime_nsec.to_le_bytes());
        put(&mut bytes, 72, &self.mtime.to_le_bytes());
        put(&mut bytes, 76, &self.mtime_nsec.to_le_bytes());
        put(&mut bytes, 80, &self.ctime.to_le_bytes());
        put(&mut bytes, 84, &self.ctime_nsec.to_le_bytes());
        put(&mut bytes, 88, &self.ino.to_le_bytes());
        bytes
    }
}

/// `struct statfsbuf` — the host-side counterpart of the two guest `statfs`
/// layouts: all-`long` fields, 120 bytes.
#[repr(C)]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Statfsbuf {
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

impl Statfsbuf {
    /// The struct's size: 120 bytes.
    pub const SIZE: usize = std::mem::size_of::<Statfsbuf>();

    /// The struct as it lies in host memory.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        put(&mut bytes, 0, &self.type_.to_le_bytes());
        put(&mut bytes, 8, &self.bsize.to_le_bytes());
        put(&mut bytes, 16, &self.blocks.to_le_bytes());
        put(&mut bytes, 24, &self.bfree.to_le_bytes());
        put(&mut bytes, 32, &self.bavail.to_le_bytes());
        put(&mut bytes, 40, &self.files.to_le_bytes());
        put(&mut bytes, 48, &self.ffree.to_le_bytes());
        put(&mut bytes, 56, &self.fsid.to_le_bytes());
        put(&mut bytes, 64, &self.namelen.to_le_bytes());
        put(&mut bytes, 72, &self.frsize.to_le_bytes());
        put(&mut bytes, 80, &self.flags.to_le_bytes());
        let spare = self.spare;
        for (index, value) in spare.iter().enumerate() {
            put(&mut bytes, 88 + index * 8, &(*value).to_le_bytes());
        }
        bytes
    }
}

/// `struct statfs_` — the 32-bit guest `statfs`, packed, 64 bytes: the `fsid`
/// sits at offset 28, unaligned, and everything after it follows without gaps.
#[repr(C, packed)]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Statfs {
    pub type_: u32,
    pub bsize: u32,
    pub blocks: u32,
    pub bfree: u32,
    pub bavail: u32,
    pub files: u32,
    pub ffree: u32,
    pub fsid: u64,
    pub namelen: u32,
    pub frsize: u32,
    pub flags: u32,
    pub spare: [u32; 4],
}

impl Statfs {
    /// The struct's size: 64 bytes.
    pub const SIZE: usize = std::mem::size_of::<Statfs>();

    /// The struct as it lies in host memory.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        put(&mut bytes, 0, &self.type_.to_le_bytes());
        put(&mut bytes, 4, &self.bsize.to_le_bytes());
        put(&mut bytes, 8, &self.blocks.to_le_bytes());
        put(&mut bytes, 12, &self.bfree.to_le_bytes());
        put(&mut bytes, 16, &self.bavail.to_le_bytes());
        put(&mut bytes, 20, &self.files.to_le_bytes());
        put(&mut bytes, 24, &self.ffree.to_le_bytes());
        put(&mut bytes, 28, &self.fsid.to_le_bytes());
        put(&mut bytes, 36, &self.namelen.to_le_bytes());
        put(&mut bytes, 40, &self.frsize.to_le_bytes());
        put(&mut bytes, 44, &self.flags.to_le_bytes());
        let spare = self.spare;
        for (index, value) in spare.iter().enumerate() {
            put(&mut bytes, 48 + index * 4, &(*value).to_le_bytes());
        }
        bytes
    }
}

/// `struct statfs64_` — the `statfs64` guest layout, packed, 84 bytes: 64-bit
/// counts, a 32-bit `namelen`/`frsize`/`flags` tail, and a 16-byte pad.
#[repr(C, packed)]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Statfs64 {
    pub type_: u32,
    pub bsize: u32,
    pub blocks: u64,
    pub bfree: u64,
    pub bavail: u64,
    pub files: u64,
    pub ffree: u64,
    pub fsid: u64,
    pub namelen: u32,
    pub frsize: u32,
    pub flags: u32,
    pub pad: [u32; 4],
}

impl Statfs64 {
    /// The struct's size: 84 bytes.
    pub const SIZE: usize = std::mem::size_of::<Statfs64>();

    /// The struct as it lies in host memory.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        put(&mut bytes, 0, &self.type_.to_le_bytes());
        put(&mut bytes, 4, &self.bsize.to_le_bytes());
        put(&mut bytes, 8, &self.blocks.to_le_bytes());
        put(&mut bytes, 16, &self.bfree.to_le_bytes());
        put(&mut bytes, 24, &self.bavail.to_le_bytes());
        put(&mut bytes, 32, &self.files.to_le_bytes());
        put(&mut bytes, 40, &self.ffree.to_le_bytes());
        put(&mut bytes, 48, &self.fsid.to_le_bytes());
        put(&mut bytes, 56, &self.namelen.to_le_bytes());
        put(&mut bytes, 60, &self.frsize.to_le_bytes());
        put(&mut bytes, 64, &self.flags.to_le_bytes());
        let pad = self.pad;
        for (index, value) in pad.iter().enumerate() {
            put(&mut bytes, 68 + index * 4, &(*value).to_le_bytes());
        }
        bytes
    }
}

/// `struct statx_timestamp_` — seconds and nanoseconds for one of `statx`'s
/// four timestamps, 16 bytes with a trailing pad.
#[repr(C)]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct StatxTimestamp {
    pub sec: i64,
    pub nsec: u32,
    pub _pad: u32,
}

impl StatxTimestamp {
    /// The struct's size: 16 bytes.
    pub const SIZE: usize = std::mem::size_of::<StatxTimestamp>();

    /// The struct as it lies in host memory.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        put(&mut bytes, 0, &self.sec.to_le_bytes());
        put(&mut bytes, 8, &self.nsec.to_le_bytes());
        put(&mut bytes, 12, &self._pad.to_le_bytes());
        bytes
    }
}

/// `struct statx_` — what `sys_statx` writes, packed, 256 bytes.
#[repr(C, packed)]
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
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
    pub _pad2: [u32; 24],
}

impl Statx {
    /// The struct's size: 256 bytes.
    pub const SIZE: usize = std::mem::size_of::<Statx>();

    /// The struct as it lies in host memory.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        put(&mut bytes, 0, &self.mask.to_le_bytes());
        put(&mut bytes, 4, &self.blksize.to_le_bytes());
        put(&mut bytes, 8, &self.attributes.to_le_bytes());
        put(&mut bytes, 16, &self.nlink.to_le_bytes());
        put(&mut bytes, 20, &self.uid.to_le_bytes());
        put(&mut bytes, 24, &self.gid.to_le_bytes());
        put(&mut bytes, 28, &self.mode.to_le_bytes());
        put(&mut bytes, 30, &self._pad1.to_le_bytes());
        put(&mut bytes, 32, &self.ino.to_le_bytes());
        put(&mut bytes, 40, &self.size.to_le_bytes());
        put(&mut bytes, 48, &self.blocks.to_le_bytes());
        put(&mut bytes, 56, &self.attributes_mask.to_le_bytes());
        bytes[64..80].copy_from_slice(&self.atime.to_le_bytes());
        bytes[80..96].copy_from_slice(&self.btime.to_le_bytes());
        bytes[96..112].copy_from_slice(&self.ctime.to_le_bytes());
        bytes[112..128].copy_from_slice(&self.mtime.to_le_bytes());
        put(&mut bytes, 128, &self.rdev_major.to_le_bytes());
        put(&mut bytes, 132, &self.rdev_minor.to_le_bytes());
        put(&mut bytes, 136, &self.dev_major.to_le_bytes());
        put(&mut bytes, 140, &self.dev_minor.to_le_bytes());
        put(&mut bytes, 144, &self.mnt_id.to_le_bytes());
        put(&mut bytes, 152, &self.dio_mem_align.to_le_bytes());
        put(&mut bytes, 156, &self.dio_offset_align.to_le_bytes());
        let _pad2 = self._pad2;
        for (index, value) in _pad2.iter().enumerate() {
            put(&mut bytes, 160 + index * 4, &(*value).to_le_bytes());
        }
        bytes
    }
}

/// `stat_convert_newstat64` — turn a filesystem's [`Statbuf`] into the guest's
/// `newstat64`.
///
/// Every field is a copy except two, and both are easy to get wrong:
///
/// * the inode goes into *two* fields: the 32-bit `fucked_ino` (the low half,
///   truncated) and the full 64-bit `ino` at the end of the struct;
/// * `_pad1` and `_pad2` have no source at all — the C leaves them
///   uninitialized, and this port writes zero, which is the one part of the
///   result that is a decision rather than a copy.
///
/// The fields with no `statbuf` counterpart (`mode` is 32 bits here and 16 in
/// the old ABI, `blksize` and the four timestamps keep their full width) are
/// copied as they are; nothing is sign-extended or narrowed on the way.
pub fn stat_convert_newstat64(stat: &Statbuf) -> Newstat64 {
    Newstat64 {
        dev: stat.dev,
        _pad1: 0,
        fucked_ino: stat.inode as u32,
        ino: stat.inode,
        mode: stat.mode,
        nlink: stat.nlink,
        uid: stat.uid,
        gid: stat.gid,
        rdev: stat.rdev,
        _pad2: 0,
        size: stat.size,
        blksize: stat.blksize,
        blocks: stat.blocks,
        atime: stat.atime,
        atime_nsec: stat.atime_nsec,
        mtime: stat.mtime,
        mtime_nsec: stat.mtime_nsec,
        ctime: stat.ctime,
        ctime_nsec: stat.ctime_nsec,
    }
}

/// Write little-endian bytes into an image at a fixed offset.
///
/// The offsets are written out at every call site rather than derived, because
/// they *are* the guest ABI here: reading an offset next to the field name is
/// the only way to see at a glance that, say, `newstat64.size` is at 44 and not
/// 40. Each call also passes the field through `to_le_bytes`, so the width that
/// lands in the image is the field's own width and never a chosen one.
fn put(bytes: &mut [u8], offset: usize, encoded: &[u8]) {
    bytes[offset..offset + encoded.len()].copy_from_slice(encoded);
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn layouts_are_the_c_ones() {
        assert_eq!(Statbuf::SIZE, 88);
        assert_eq!(OldStat::SIZE, 32);
        assert_eq!(NewStat::SIZE, 64);
        assert_eq!(Newstat64::SIZE, 96);
        assert_eq!(Statfsbuf::SIZE, 120);
        assert_eq!(Statfs::SIZE, 64);
        assert_eq!(Statfs64::SIZE, 84);
        assert_eq!(StatxTimestamp::SIZE, 16);
        assert_eq!(Statx::SIZE, 256);

        // The packings, which are the only reason these are not larger.
        assert_eq!(std::mem::align_of::<Statbuf>(), 8);
        assert_eq!(std::mem::align_of::<Newstat64>(), 1);
        assert_eq!(std::mem::align_of::<Statfs>(), 1);
        assert_eq!(std::mem::align_of::<Statfs64>(), 1);
        assert_eq!(std::mem::align_of::<Statx>(), 1);
    }

    #[test]
    fn the_inode_goes_into_both_of_its_fields() {
        let stat = Statbuf {
            inode: 0xdead_beef_0000_0001,
            ..Statbuf::default()
        };
        let converted = stat_convert_newstat64(&stat);
        assert_eq!({ converted.fucked_ino }, 1);
        assert_eq!({ converted.ino }, 0xdead_beef_0000_0001);
    }

    #[test]
    fn the_unwritten_padding_is_zero() {
        // The C leaves `_pad1` and `_pad2` alone, which is not an ABI; the port
        // has to pick something, and zero is what the fixture was generated
        // with. A guest that depends on those bytes depends on nothing.
        let stat = Statbuf::default();
        let converted = stat_convert_newstat64(&stat);
        assert_eq!({ converted._pad1 }, 0);
        assert_eq!({ converted._pad2 }, 0);
    }

    #[test]
    fn statbuf_padding_is_not_a_field() {
        // Four bytes of the host struct are padding; the image zeroes them
        // rather than inventing content. Set everything around them and check
        // they stay zero.
        let stat = Statbuf {
            blksize: 0xffff_ffff,
            blocks: 0xffff_ffff_ffff_ffff,
            ..Statbuf::default()
        };
        let bytes = stat.to_le_bytes();
        assert_eq!(&bytes[52..56], &[0, 0, 0, 0]);
        assert_eq!(&bytes[48..52], &[0xff; 4]);
        assert_eq!(&bytes[56..64], &[0xff; 8]);
    }

    #[test]
    fn conversion_copies_every_field_it_has_a_source_for() {
        let stat = Statbuf {
            dev: 0x1122_3344_5566_7788,
            inode: 0x2233_4455_6677_8899,
            mode: 0x1122_3344,
            nlink: 0x2233_4455,
            uid: 0x3344_5566,
            gid: 0x4455_6677,
            rdev: 0x5566_7788_99aa_bbcc,
            size: 0x6677_8899_aabb_ccdd,
            blksize: 0x7788_99aa,
            blocks: 0x8899_aabb_ccdd_eeff,
            atime: 0x9900_1122,
            atime_nsec: 0xaa11_2233,
            mtime: 0xbb22_3344,
            mtime_nsec: 0xcc33_4455,
            ctime: 0xdd44_5566,
            ctime_nsec: 0xee55_6677,
        };
        let converted = stat_convert_newstat64(&stat);
        assert_eq!({ converted.dev }, stat.dev);
        assert_eq!({ converted.fucked_ino }, stat.inode as u32);
        assert_eq!({ converted.ino }, stat.inode);
        assert_eq!({ converted.mode }, stat.mode);
        assert_eq!({ converted.nlink }, stat.nlink);
        assert_eq!({ converted.uid }, stat.uid);
        assert_eq!({ converted.gid }, stat.gid);
        assert_eq!({ converted.rdev }, stat.rdev);
        assert_eq!({ converted.size }, stat.size);
        assert_eq!({ converted.blksize }, stat.blksize);
        assert_eq!({ converted.blocks }, stat.blocks);
        assert_eq!({ converted.atime }, stat.atime);
        assert_eq!({ converted.atime_nsec }, stat.atime_nsec);
        assert_eq!({ converted.mtime }, stat.mtime);
        assert_eq!({ converted.mtime_nsec }, stat.mtime_nsec);
        assert_eq!({ converted.ctime }, stat.ctime);
        assert_eq!({ converted.ctime_nsec }, stat.ctime_nsec);
    }

    #[test]
    fn packed_offsets_are_the_guest_abi() {
        // These four offsets are what make `stat64` on a 32-bit guest line up.
        assert_eq!(std::mem::offset_of!(Newstat64, dev), 0);
        assert_eq!(std::mem::offset_of!(Newstat64, fucked_ino), 12);
        assert_eq!(std::mem::offset_of!(Newstat64, size), 44);
        assert_eq!(std::mem::offset_of!(Newstat64, ino), 88);
        assert_eq!(std::mem::offset_of!(Statfs, fsid), 28);
        assert_eq!(std::mem::offset_of!(Statfs64, namelen), 56);
        assert_eq!(std::mem::offset_of!(Statx, mnt_id), 144);
        assert_eq!(std::mem::offset_of!(Statx, _pad2), 160);
    }

    #[test]
    fn images_place_fields_where_the_offsets_say() {
        let stat = Statbuf {
            blksize: 0x7788_99aa,
            blocks: 0x8899_aabb_ccdd_eeff,
            ..Statbuf::default()
        };
        let image = stat.to_le_bytes();
        assert_eq!(&image[48..52], &0x7788_99aau32.to_le_bytes());
        assert_eq!(&image[56..64], &0x8899_aabb_ccdd_eeffu64.to_le_bytes());

        let mut converted = stat_convert_newstat64(&stat);
        converted.ino = 0x5566_7788_99aa_bbcc;
        let image = converted.to_le_bytes();
        assert_eq!(&image[88..96], &0x5566_7788_99aa_bbccu64.to_le_bytes());
        assert_eq!(&image[44..52], &converted.size.to_le_bytes());
    }

    #[test]
    fn statx_basic_stats_is_the_c_constant() {
        assert_eq!(STATX_BASIC_STATS, 0x7ff);
    }
}
