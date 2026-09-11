//! Differential test of `stat.rs` against unmodified `fs/stat.h` and
//! `fs/stat.c`.
//!
//! `tools/stat-dump.c` includes the real C file and takes every size, offset and
//! field width in `tests/fixtures/stat_reference.txt` from `sizeof` and
//! `offsetof` on the real headers. This test replays three things from that
//! fixture:
//!
//! * **the layouts** — `size_of`, `align_of` and `offset_of!` of the Rust
//!   structs have to be the C compiler's numbers, including the two packed
//!   structures whose whole point is that their fields are unaligned;
//! * **the images** — every field of every struct is set to a distinct value,
//!   and the bytes the Rust serializer produces have to be the bytes the C
//!   struct held. Bytes that no field covers are the C's padding: the fixture
//!   shows them as the `0xAA` the oracle memset, and the port zeroes them,
//!   which is checked rather than assumed;
//! * **the conversion** — `stat_convert_newstat64` over five `statbuf`s,
//!   compared as a whole image and field by field, so a truncated `dev`, a lost
//!   high half of the inode, or a swapped timestamp names itself in the failure.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_stat_reference.sh
//! ```

use std::collections::BTreeMap;

use ish_emu::stat::{
    stat_convert_newstat64, NewStat, Newstat64, OldStat, Statbuf, Statfs, Statfs64, Statfsbuf,
    Statx, StatxTimestamp, STATX_BASIC_STATS,
};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/stat_reference.txt"
);

/// One field's position in a Rust struct, read from the struct itself.
struct Field {
    /// The field's name on the C side, as the fixture spells it.
    name: &'static str,
    offset: usize,
    width: usize,
}

/// A struct the fixture has records for: its Rust size and its Rust fields.
struct Layout {
    kind: &'static str,
    size: usize,
    align: usize,
    fields: Vec<Field>,
}

/// The width of a field, from the field's own type.
trait FieldWidth {
    const WIDTH: usize;
}

macro_rules! field_width {
    ($($type:ty),* $(,)?) => {
        $(impl FieldWidth for $type {
            const WIDTH: usize = std::mem::size_of::<$type>();
        })*
    };
}

field_width!(
    u16,
    u32,
    u64,
    i64,
    [u8; 8],
    [u32; 4],
    [u32; 24],
    [i64; 4],
    StatxTimestamp
);

/// The width of a value — called on a field, so the type is the field's.
fn width_of<T: FieldWidth>(_: T) -> usize {
    T::WIDTH
}

/// Every struct the fixture describes, with the offsets the *Rust* compiler
/// assigned. Comparing these with the fixture's `F` records is the layout test.
fn layouts() -> Vec<Layout> {
    // Reading a field out of a packed struct is a copy, so `width_of` can be
    // handed one even where a reference would be unaligned.
    let statbuf = Statbuf::default();
    let oldstat = OldStat::default();
    let newstat = NewStat::default();
    let newstat64 = Newstat64::default();
    let statfsbuf = Statfsbuf::default();
    let statfs = Statfs::default();
    let statfs64 = Statfs64::default();
    let timestamp = StatxTimestamp::default();
    let statx = Statx::default();

    vec![
        Layout {
            kind: "statbuf",
            size: std::mem::size_of::<Statbuf>(),
            align: std::mem::align_of::<Statbuf>(),
            fields: vec![
                Field {
                    name: "dev",
                    offset: std::mem::offset_of!(Statbuf, dev),
                    width: width_of(statbuf.dev),
                },
                Field {
                    name: "inode",
                    offset: std::mem::offset_of!(Statbuf, inode),
                    width: width_of(statbuf.inode),
                },
                Field {
                    name: "mode",
                    offset: std::mem::offset_of!(Statbuf, mode),
                    width: width_of(statbuf.mode),
                },
                Field {
                    name: "nlink",
                    offset: std::mem::offset_of!(Statbuf, nlink),
                    width: width_of(statbuf.nlink),
                },
                Field {
                    name: "uid",
                    offset: std::mem::offset_of!(Statbuf, uid),
                    width: width_of(statbuf.uid),
                },
                Field {
                    name: "gid",
                    offset: std::mem::offset_of!(Statbuf, gid),
                    width: width_of(statbuf.gid),
                },
                Field {
                    name: "rdev",
                    offset: std::mem::offset_of!(Statbuf, rdev),
                    width: width_of(statbuf.rdev),
                },
                Field {
                    name: "size",
                    offset: std::mem::offset_of!(Statbuf, size),
                    width: width_of(statbuf.size),
                },
                Field {
                    name: "blksize",
                    offset: std::mem::offset_of!(Statbuf, blksize),
                    width: width_of(statbuf.blksize),
                },
                Field {
                    name: "blocks",
                    offset: std::mem::offset_of!(Statbuf, blocks),
                    width: width_of(statbuf.blocks),
                },
                Field {
                    name: "atime",
                    offset: std::mem::offset_of!(Statbuf, atime),
                    width: width_of(statbuf.atime),
                },
                Field {
                    name: "atime_nsec",
                    offset: std::mem::offset_of!(Statbuf, atime_nsec),
                    width: width_of(statbuf.atime_nsec),
                },
                Field {
                    name: "mtime",
                    offset: std::mem::offset_of!(Statbuf, mtime),
                    width: width_of(statbuf.mtime),
                },
                Field {
                    name: "mtime_nsec",
                    offset: std::mem::offset_of!(Statbuf, mtime_nsec),
                    width: width_of(statbuf.mtime_nsec),
                },
                Field {
                    name: "ctime",
                    offset: std::mem::offset_of!(Statbuf, ctime),
                    width: width_of(statbuf.ctime),
                },
                Field {
                    name: "ctime_nsec",
                    offset: std::mem::offset_of!(Statbuf, ctime_nsec),
                    width: width_of(statbuf.ctime_nsec),
                },
            ],
        },
        Layout {
            kind: "oldstat",
            size: std::mem::size_of::<OldStat>(),
            align: std::mem::align_of::<OldStat>(),
            fields: vec![
                Field {
                    name: "dev",
                    offset: std::mem::offset_of!(OldStat, dev),
                    width: width_of(oldstat.dev),
                },
                Field {
                    name: "ino",
                    offset: std::mem::offset_of!(OldStat, ino),
                    width: width_of(oldstat.ino),
                },
                Field {
                    name: "mode",
                    offset: std::mem::offset_of!(OldStat, mode),
                    width: width_of(oldstat.mode),
                },
                Field {
                    name: "nlink",
                    offset: std::mem::offset_of!(OldStat, nlink),
                    width: width_of(oldstat.nlink),
                },
                Field {
                    name: "uid",
                    offset: std::mem::offset_of!(OldStat, uid),
                    width: width_of(oldstat.uid),
                },
                Field {
                    name: "gid",
                    offset: std::mem::offset_of!(OldStat, gid),
                    width: width_of(oldstat.gid),
                },
                Field {
                    name: "rdev",
                    offset: std::mem::offset_of!(OldStat, rdev),
                    width: width_of(oldstat.rdev),
                },
                Field {
                    name: "size",
                    offset: std::mem::offset_of!(OldStat, size),
                    width: width_of(oldstat.size),
                },
                Field {
                    name: "atime",
                    offset: std::mem::offset_of!(OldStat, atime),
                    width: width_of(oldstat.atime),
                },
                Field {
                    name: "mtime",
                    offset: std::mem::offset_of!(OldStat, mtime),
                    width: width_of(oldstat.mtime),
                },
                Field {
                    name: "ctime",
                    offset: std::mem::offset_of!(OldStat, ctime),
                    width: width_of(oldstat.ctime),
                },
            ],
        },
        Layout {
            kind: "newstat",
            size: std::mem::size_of::<NewStat>(),
            align: std::mem::align_of::<NewStat>(),
            fields: vec![
                Field {
                    name: "dev",
                    offset: std::mem::offset_of!(NewStat, dev),
                    width: width_of(newstat.dev),
                },
                Field {
                    name: "ino",
                    offset: std::mem::offset_of!(NewStat, ino),
                    width: width_of(newstat.ino),
                },
                Field {
                    name: "mode",
                    offset: std::mem::offset_of!(NewStat, mode),
                    width: width_of(newstat.mode),
                },
                Field {
                    name: "nlink",
                    offset: std::mem::offset_of!(NewStat, nlink),
                    width: width_of(newstat.nlink),
                },
                Field {
                    name: "uid",
                    offset: std::mem::offset_of!(NewStat, uid),
                    width: width_of(newstat.uid),
                },
                Field {
                    name: "gid",
                    offset: std::mem::offset_of!(NewStat, gid),
                    width: width_of(newstat.gid),
                },
                Field {
                    name: "rdev",
                    offset: std::mem::offset_of!(NewStat, rdev),
                    width: width_of(newstat.rdev),
                },
                Field {
                    name: "size",
                    offset: std::mem::offset_of!(NewStat, size),
                    width: width_of(newstat.size),
                },
                Field {
                    name: "blksize",
                    offset: std::mem::offset_of!(NewStat, blksize),
                    width: width_of(newstat.blksize),
                },
                Field {
                    name: "blocks",
                    offset: std::mem::offset_of!(NewStat, blocks),
                    width: width_of(newstat.blocks),
                },
                Field {
                    name: "atime",
                    offset: std::mem::offset_of!(NewStat, atime),
                    width: width_of(newstat.atime),
                },
                Field {
                    name: "atime_nsec",
                    offset: std::mem::offset_of!(NewStat, atime_nsec),
                    width: width_of(newstat.atime_nsec),
                },
                Field {
                    name: "mtime",
                    offset: std::mem::offset_of!(NewStat, mtime),
                    width: width_of(newstat.mtime),
                },
                Field {
                    name: "mtime_nsec",
                    offset: std::mem::offset_of!(NewStat, mtime_nsec),
                    width: width_of(newstat.mtime_nsec),
                },
                Field {
                    name: "ctime",
                    offset: std::mem::offset_of!(NewStat, ctime),
                    width: width_of(newstat.ctime),
                },
                Field {
                    name: "ctime_nsec",
                    offset: std::mem::offset_of!(NewStat, ctime_nsec),
                    width: width_of(newstat.ctime_nsec),
                },
                Field {
                    name: "pad",
                    offset: std::mem::offset_of!(NewStat, pad),
                    width: width_of(newstat.pad),
                },
            ],
        },
        Layout {
            kind: "newstat64",
            size: std::mem::size_of::<Newstat64>(),
            align: std::mem::align_of::<Newstat64>(),
            fields: vec![
                Field {
                    name: "dev",
                    offset: std::mem::offset_of!(Newstat64, dev),
                    width: width_of(newstat64.dev),
                },
                Field {
                    name: "_pad1",
                    offset: std::mem::offset_of!(Newstat64, _pad1),
                    width: width_of(newstat64._pad1),
                },
                Field {
                    name: "fucked_ino",
                    offset: std::mem::offset_of!(Newstat64, fucked_ino),
                    width: width_of(newstat64.fucked_ino),
                },
                Field {
                    name: "mode",
                    offset: std::mem::offset_of!(Newstat64, mode),
                    width: width_of(newstat64.mode),
                },
                Field {
                    name: "nlink",
                    offset: std::mem::offset_of!(Newstat64, nlink),
                    width: width_of(newstat64.nlink),
                },
                Field {
                    name: "uid",
                    offset: std::mem::offset_of!(Newstat64, uid),
                    width: width_of(newstat64.uid),
                },
                Field {
                    name: "gid",
                    offset: std::mem::offset_of!(Newstat64, gid),
                    width: width_of(newstat64.gid),
                },
                Field {
                    name: "rdev",
                    offset: std::mem::offset_of!(Newstat64, rdev),
                    width: width_of(newstat64.rdev),
                },
                Field {
                    name: "_pad2",
                    offset: std::mem::offset_of!(Newstat64, _pad2),
                    width: width_of(newstat64._pad2),
                },
                Field {
                    name: "size",
                    offset: std::mem::offset_of!(Newstat64, size),
                    width: width_of(newstat64.size),
                },
                Field {
                    name: "blksize",
                    offset: std::mem::offset_of!(Newstat64, blksize),
                    width: width_of(newstat64.blksize),
                },
                Field {
                    name: "blocks",
                    offset: std::mem::offset_of!(Newstat64, blocks),
                    width: width_of(newstat64.blocks),
                },
                Field {
                    name: "atime",
                    offset: std::mem::offset_of!(Newstat64, atime),
                    width: width_of(newstat64.atime),
                },
                Field {
                    name: "atime_nsec",
                    offset: std::mem::offset_of!(Newstat64, atime_nsec),
                    width: width_of(newstat64.atime_nsec),
                },
                Field {
                    name: "mtime",
                    offset: std::mem::offset_of!(Newstat64, mtime),
                    width: width_of(newstat64.mtime),
                },
                Field {
                    name: "mtime_nsec",
                    offset: std::mem::offset_of!(Newstat64, mtime_nsec),
                    width: width_of(newstat64.mtime_nsec),
                },
                Field {
                    name: "ctime",
                    offset: std::mem::offset_of!(Newstat64, ctime),
                    width: width_of(newstat64.ctime),
                },
                Field {
                    name: "ctime_nsec",
                    offset: std::mem::offset_of!(Newstat64, ctime_nsec),
                    width: width_of(newstat64.ctime_nsec),
                },
                Field {
                    name: "ino",
                    offset: std::mem::offset_of!(Newstat64, ino),
                    width: width_of(newstat64.ino),
                },
            ],
        },
        Layout {
            kind: "statfsbuf",
            size: std::mem::size_of::<Statfsbuf>(),
            align: std::mem::align_of::<Statfsbuf>(),
            fields: vec![
                Field {
                    name: "type",
                    offset: std::mem::offset_of!(Statfsbuf, type_),
                    width: width_of(statfsbuf.type_),
                },
                Field {
                    name: "bsize",
                    offset: std::mem::offset_of!(Statfsbuf, bsize),
                    width: width_of(statfsbuf.bsize),
                },
                Field {
                    name: "blocks",
                    offset: std::mem::offset_of!(Statfsbuf, blocks),
                    width: width_of(statfsbuf.blocks),
                },
                Field {
                    name: "bfree",
                    offset: std::mem::offset_of!(Statfsbuf, bfree),
                    width: width_of(statfsbuf.bfree),
                },
                Field {
                    name: "bavail",
                    offset: std::mem::offset_of!(Statfsbuf, bavail),
                    width: width_of(statfsbuf.bavail),
                },
                Field {
                    name: "files",
                    offset: std::mem::offset_of!(Statfsbuf, files),
                    width: width_of(statfsbuf.files),
                },
                Field {
                    name: "ffree",
                    offset: std::mem::offset_of!(Statfsbuf, ffree),
                    width: width_of(statfsbuf.ffree),
                },
                Field {
                    name: "fsid",
                    offset: std::mem::offset_of!(Statfsbuf, fsid),
                    width: width_of(statfsbuf.fsid),
                },
                Field {
                    name: "namelen",
                    offset: std::mem::offset_of!(Statfsbuf, namelen),
                    width: width_of(statfsbuf.namelen),
                },
                Field {
                    name: "frsize",
                    offset: std::mem::offset_of!(Statfsbuf, frsize),
                    width: width_of(statfsbuf.frsize),
                },
                Field {
                    name: "flags",
                    offset: std::mem::offset_of!(Statfsbuf, flags),
                    width: width_of(statfsbuf.flags),
                },
                Field {
                    name: "spare",
                    offset: std::mem::offset_of!(Statfsbuf, spare),
                    width: width_of(statfsbuf.spare),
                },
            ],
        },
        Layout {
            kind: "statfs_",
            size: std::mem::size_of::<Statfs>(),
            align: std::mem::align_of::<Statfs>(),
            fields: vec![
                Field {
                    name: "type",
                    offset: std::mem::offset_of!(Statfs, type_),
                    width: width_of(statfs.type_),
                },
                Field {
                    name: "bsize",
                    offset: std::mem::offset_of!(Statfs, bsize),
                    width: width_of(statfs.bsize),
                },
                Field {
                    name: "blocks",
                    offset: std::mem::offset_of!(Statfs, blocks),
                    width: width_of(statfs.blocks),
                },
                Field {
                    name: "bfree",
                    offset: std::mem::offset_of!(Statfs, bfree),
                    width: width_of(statfs.bfree),
                },
                Field {
                    name: "bavail",
                    offset: std::mem::offset_of!(Statfs, bavail),
                    width: width_of(statfs.bavail),
                },
                Field {
                    name: "files",
                    offset: std::mem::offset_of!(Statfs, files),
                    width: width_of(statfs.files),
                },
                Field {
                    name: "ffree",
                    offset: std::mem::offset_of!(Statfs, ffree),
                    width: width_of(statfs.ffree),
                },
                Field {
                    name: "fsid",
                    offset: std::mem::offset_of!(Statfs, fsid),
                    width: width_of(statfs.fsid),
                },
                Field {
                    name: "namelen",
                    offset: std::mem::offset_of!(Statfs, namelen),
                    width: width_of(statfs.namelen),
                },
                Field {
                    name: "frsize",
                    offset: std::mem::offset_of!(Statfs, frsize),
                    width: width_of(statfs.frsize),
                },
                Field {
                    name: "flags",
                    offset: std::mem::offset_of!(Statfs, flags),
                    width: width_of(statfs.flags),
                },
                Field {
                    name: "spare",
                    offset: std::mem::offset_of!(Statfs, spare),
                    width: width_of(statfs.spare),
                },
            ],
        },
        Layout {
            kind: "statfs64_",
            size: std::mem::size_of::<Statfs64>(),
            align: std::mem::align_of::<Statfs64>(),
            fields: vec![
                Field {
                    name: "type",
                    offset: std::mem::offset_of!(Statfs64, type_),
                    width: width_of(statfs64.type_),
                },
                Field {
                    name: "bsize",
                    offset: std::mem::offset_of!(Statfs64, bsize),
                    width: width_of(statfs64.bsize),
                },
                Field {
                    name: "blocks",
                    offset: std::mem::offset_of!(Statfs64, blocks),
                    width: width_of(statfs64.blocks),
                },
                Field {
                    name: "bfree",
                    offset: std::mem::offset_of!(Statfs64, bfree),
                    width: width_of(statfs64.bfree),
                },
                Field {
                    name: "bavail",
                    offset: std::mem::offset_of!(Statfs64, bavail),
                    width: width_of(statfs64.bavail),
                },
                Field {
                    name: "files",
                    offset: std::mem::offset_of!(Statfs64, files),
                    width: width_of(statfs64.files),
                },
                Field {
                    name: "ffree",
                    offset: std::mem::offset_of!(Statfs64, ffree),
                    width: width_of(statfs64.ffree),
                },
                Field {
                    name: "fsid",
                    offset: std::mem::offset_of!(Statfs64, fsid),
                    width: width_of(statfs64.fsid),
                },
                Field {
                    name: "namelen",
                    offset: std::mem::offset_of!(Statfs64, namelen),
                    width: width_of(statfs64.namelen),
                },
                Field {
                    name: "frsize",
                    offset: std::mem::offset_of!(Statfs64, frsize),
                    width: width_of(statfs64.frsize),
                },
                Field {
                    name: "flags",
                    offset: std::mem::offset_of!(Statfs64, flags),
                    width: width_of(statfs64.flags),
                },
                Field {
                    name: "pad",
                    offset: std::mem::offset_of!(Statfs64, pad),
                    width: width_of(statfs64.pad),
                },
            ],
        },
        Layout {
            kind: "statx_timestamp_",
            size: std::mem::size_of::<StatxTimestamp>(),
            align: std::mem::align_of::<StatxTimestamp>(),
            fields: vec![
                Field {
                    name: "sec",
                    offset: std::mem::offset_of!(StatxTimestamp, sec),
                    width: width_of(timestamp.sec),
                },
                Field {
                    name: "nsec",
                    offset: std::mem::offset_of!(StatxTimestamp, nsec),
                    width: width_of(timestamp.nsec),
                },
                Field {
                    name: "_pad",
                    offset: std::mem::offset_of!(StatxTimestamp, _pad),
                    width: width_of(timestamp._pad),
                },
            ],
        },
        Layout {
            kind: "statx_",
            size: std::mem::size_of::<Statx>(),
            align: std::mem::align_of::<Statx>(),
            fields: vec![
                Field {
                    name: "mask",
                    offset: std::mem::offset_of!(Statx, mask),
                    width: width_of(statx.mask),
                },
                Field {
                    name: "blksize",
                    offset: std::mem::offset_of!(Statx, blksize),
                    width: width_of(statx.blksize),
                },
                Field {
                    name: "attributes",
                    offset: std::mem::offset_of!(Statx, attributes),
                    width: width_of(statx.attributes),
                },
                Field {
                    name: "nlink",
                    offset: std::mem::offset_of!(Statx, nlink),
                    width: width_of(statx.nlink),
                },
                Field {
                    name: "uid",
                    offset: std::mem::offset_of!(Statx, uid),
                    width: width_of(statx.uid),
                },
                Field {
                    name: "gid",
                    offset: std::mem::offset_of!(Statx, gid),
                    width: width_of(statx.gid),
                },
                Field {
                    name: "mode",
                    offset: std::mem::offset_of!(Statx, mode),
                    width: width_of(statx.mode),
                },
                Field {
                    name: "_pad1",
                    offset: std::mem::offset_of!(Statx, _pad1),
                    width: width_of(statx._pad1),
                },
                Field {
                    name: "ino",
                    offset: std::mem::offset_of!(Statx, ino),
                    width: width_of(statx.ino),
                },
                Field {
                    name: "size",
                    offset: std::mem::offset_of!(Statx, size),
                    width: width_of(statx.size),
                },
                Field {
                    name: "blocks",
                    offset: std::mem::offset_of!(Statx, blocks),
                    width: width_of(statx.blocks),
                },
                Field {
                    name: "attributes_mask",
                    offset: std::mem::offset_of!(Statx, attributes_mask),
                    width: width_of(statx.attributes_mask),
                },
                Field {
                    name: "atime",
                    offset: std::mem::offset_of!(Statx, atime),
                    width: width_of(statx.atime),
                },
                Field {
                    name: "btime",
                    offset: std::mem::offset_of!(Statx, btime),
                    width: width_of(statx.btime),
                },
                Field {
                    name: "ctime",
                    offset: std::mem::offset_of!(Statx, ctime),
                    width: width_of(statx.ctime),
                },
                Field {
                    name: "mtime",
                    offset: std::mem::offset_of!(Statx, mtime),
                    width: width_of(statx.mtime),
                },
                Field {
                    name: "rdev_major",
                    offset: std::mem::offset_of!(Statx, rdev_major),
                    width: width_of(statx.rdev_major),
                },
                Field {
                    name: "rdev_minor",
                    offset: std::mem::offset_of!(Statx, rdev_minor),
                    width: width_of(statx.rdev_minor),
                },
                Field {
                    name: "dev_major",
                    offset: std::mem::offset_of!(Statx, dev_major),
                    width: width_of(statx.dev_major),
                },
                Field {
                    name: "dev_minor",
                    offset: std::mem::offset_of!(Statx, dev_minor),
                    width: width_of(statx.dev_minor),
                },
                Field {
                    name: "mnt_id",
                    offset: std::mem::offset_of!(Statx, mnt_id),
                    width: width_of(statx.mnt_id),
                },
                Field {
                    name: "dio_mem_align",
                    offset: std::mem::offset_of!(Statx, dio_mem_align),
                    width: width_of(statx.dio_mem_align),
                },
                Field {
                    name: "dio_offset_align",
                    offset: std::mem::offset_of!(Statx, dio_offset_align),
                    width: width_of(statx.dio_offset_align),
                },
                Field {
                    name: "_pad2",
                    offset: std::mem::offset_of!(Statx, _pad2),
                    width: width_of(statx._pad2),
                },
            ],
        },
    ]
}

/// The corpus values for one struct: the fixture's `V` records.
#[derive(Default)]
struct Values(BTreeMap<String, u64>);

impl Values {
    fn take(&mut self, name: &str) -> u64 {
        self.0
            .remove(name)
            .unwrap_or_else(|| panic!("the fixture has no value for `{name}`"))
    }

    fn finish(self, kind: &str) {
        assert!(
            self.0.is_empty(),
            "{kind}: the Rust struct has no field for {:.?}",
            self.0.keys().collect::<Vec<_>>()
        );
    }
}

fn hex(text: &str) -> u64 {
    let digits = text.strip_prefix("0x").unwrap_or(text);
    u64::from_str_radix(digits, 16).unwrap_or_else(|err| panic!("invalid hex `{text}`: {err}"))
}

fn u32_of(value: u64) -> u32 {
    u32::try_from(value).unwrap_or_else(|_| panic!("{value:#x} does not fit in 32 bits"))
}

fn u16_of(value: u64) -> u16 {
    u16::try_from(value).unwrap_or_else(|_| panic!("{value:#x} does not fit in 16 bits"))
}

fn decode_bytes(text: &str) -> Vec<u8> {
    assert_eq!(text.len() % 2, 0, "odd-length hex string `{text}`");
    (0..text.len())
        .step_by(2)
        .map(|index| u8::from_str_radix(&text[index..index + 2], 16).unwrap())
        .collect()
}

/// The C's padding byte. The oracle memsets each struct to this and then sets
/// every field, so bytes no field covers are still 0xAA in the fixture image.
const C_PADDING: u8 = 0xAA;

/// Compare a Rust image with the fixture's, letting the caller name the bytes
/// that are padding.
fn compare_image(
    kind: &str,
    rust: &[u8],
    c_reference: &[u8],
    is_padding: impl Fn(usize) -> bool,
) -> usize {
    assert_eq!(
        rust.len(),
        c_reference.len(),
        "{kind}: the Rust image is {} bytes, the C reference is {}",
        rust.len(),
        c_reference.len()
    );
    let mut padding = 0;
    for (offset, (got, want)) in rust.iter().zip(c_reference).enumerate() {
        if is_padding(offset) {
            padding += 1;
            assert_eq!(
                *want, C_PADDING,
                "{kind}: byte {offset} is not a field and the C did not leave 0xAA there \
                 either, so the field list in this test is wrong"
            );
            assert_eq!(
                *got, 0,
                "{kind}: byte {offset} is padding, and the port leaves padding zero"
            );
        } else {
            assert_eq!(
                got, want,
                "{kind}: byte {offset} is {got:#04x} in the Rust image and {want:#04x} in the C"
            );
        }
    }
    padding
}

#[test]
fn struct_layouts_and_images_match_the_c_reference() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|err| panic!("cannot read {FIXTURE}: {err}"));

    // The fixture's view: sizes, fields, values and images per struct name.
    let mut c_sizes: BTreeMap<String, usize> = BTreeMap::new();
    let mut c_fields: BTreeMap<String, Vec<(String, usize, usize)>> = BTreeMap::new();
    let mut c_values: BTreeMap<String, BTreeMap<String, u64>> = BTreeMap::new();
    let mut c_images: BTreeMap<String, Vec<u8>> = BTreeMap::new();
    let mut wart_seen = false;
    let mut statx_basic_stats = None;

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split(' ').collect();
        match fields[0] {
            "S" => {
                assert_eq!(fields.len(), 3, "malformed size record: {line}");
                let size = fields[2]
                    .strip_prefix("size=")
                    .expect("size record")
                    .parse()
                    .expect("size");
                c_sizes.insert(fields[1].to_owned(), size);
            }
            "F" => {
                assert_eq!(fields.len(), 5, "malformed field record: {line}");
                let offset: usize = fields[3]
                    .strip_prefix("off=")
                    .expect("offset")
                    .parse()
                    .expect("offset");
                let width: usize = fields[4]
                    .strip_prefix("size=")
                    .expect("width")
                    .parse()
                    .expect("width");
                c_fields.entry(fields[1].to_owned()).or_default().push((
                    fields[2].to_owned(),
                    offset,
                    width,
                ));
            }
            "V" => {
                assert_eq!(fields.len(), 3, "malformed value record: {line}");
                let (name, value) = fields[2]
                    .split_once('=')
                    .unwrap_or_else(|| panic!("malformed value record: {line}"));
                c_values
                    .entry(fields[1].to_owned())
                    .or_default()
                    .insert(name.to_owned(), hex(value));
            }
            "E" => {
                assert_eq!(fields.len(), 3, "malformed image record: {line}");
                let bytes = fields[2]
                    .strip_prefix("bytes=")
                    .expect("image record")
                    .to_owned();
                c_images.insert(fields[1].to_owned(), decode_bytes(&bytes));
            }
            "W" => {
                assert!(
                    line.contains("_pad1") && line.contains("_pad2"),
                    "unexpected wart record: {line}"
                );
                wart_seen = true;
            }
            "K" => {
                assert_eq!(fields.len(), 2, "malformed constant record: {line}");
                let (name, value) = fields[1].split_once('=').expect("constant record");
                assert_eq!(name, "statx_basic_stats");
                statx_basic_stats = Some(u32_of(hex(value)));
            }
            // The conversion inputs and results belong to the other test; this
            // one is about the structures.
            "I" | "C" => {}
            other => panic!("unknown fixture record `{other}`: {line}"),
        }
    }

    // ---- layouts ----------------------------------------------------------
    let mut covered = 0usize;
    for layout in layouts() {
        let kind = layout.kind;
        let c_size = *c_sizes
            .get(kind)
            .unwrap_or_else(|| panic!("{kind}: no size in the fixture"));
        assert_eq!(
            layout.size, c_size,
            "{kind}: the Rust struct is {} bytes, the C is {c_size}",
            layout.size
        );
        assert!(
            layout.align == 0 || layout.align.is_power_of_two(),
            "{kind}: alignment"
        );

        let fixture_fields = c_fields
            .get(kind)
            .unwrap_or_else(|| panic!("{kind}: no fields in the fixture"));
        assert_eq!(
            layout.fields.len(),
            fixture_fields.len(),
            "{kind}: {} fields in the fixture, {} in the Rust struct",
            fixture_fields.len(),
            layout.fields.len()
        );
        for (field, (name, c_offset, c_width)) in layout.fields.iter().zip(fixture_fields) {
            assert_eq!(
                field.name, name,
                "{kind}: field order differs from the fixture"
            );
            assert_eq!(
                field.offset, *c_offset,
                "{kind}.{name}: the Rust field is at {}, the C at {c_offset}",
                field.offset
            );
            assert_eq!(
                field.width, *c_width,
                "{kind}.{name}: the Rust field is {} bytes, the C field is {c_width}",
                field.width
            );
            assert!(
                field.offset + field.width <= layout.size,
                "{kind}.{name}: the field runs past the end of the struct"
            );
        }

        // ---- images -------------------------------------------------------
        let c_image = c_images
            .get(kind)
            .unwrap_or_else(|| panic!("{kind}: no image in the fixture"));
        let mut values = Values(
            c_values
                .get(kind)
                .unwrap_or_else(|| panic!("{kind}: no values in the fixture"))
                .clone(),
        );
        let rust_image = build_image(kind, &mut values);
        values.finish(kind);
        covered += compare_image(kind, &rust_image, c_image, |offset| {
            !layout
                .fields
                .iter()
                .any(|field| (field.offset..field.offset + field.width).contains(&offset))
        });
    }

    assert!(
        wart_seen,
        "the fixture no longer records the indeterminate padding"
    );
    assert_eq!(
        statx_basic_stats,
        Some(STATX_BASIC_STATS),
        "STATX_BASIC_STATS_ differs from the C"
    );
    println!(
        "{} struct layouts and every field byte matched C; {covered} padding bytes are zero",
        c_sizes.len()
    );
}

/// Build the byte image of one struct from the fixture's values.
///
/// Each arm is the type's own serializer — the code being tested — fed with the
/// same distinct values the C held. A field the arm forgets to set stays at its
/// default, which the image comparison reports as a byte mismatch.
fn build_image(kind: &str, values: &mut Values) -> Vec<u8> {
    match kind {
        "statbuf" => Statbuf {
            dev: values.take("dev"),
            inode: values.take("inode"),
            mode: u32_of(values.take("mode")),
            nlink: u32_of(values.take("nlink")),
            uid: u32_of(values.take("uid")),
            gid: u32_of(values.take("gid")),
            rdev: values.take("rdev"),
            size: values.take("size"),
            blksize: u32_of(values.take("blksize")),
            blocks: values.take("blocks"),
            atime: u32_of(values.take("atime")),
            atime_nsec: u32_of(values.take("atime_nsec")),
            mtime: u32_of(values.take("mtime")),
            mtime_nsec: u32_of(values.take("mtime_nsec")),
            ctime: u32_of(values.take("ctime")),
            ctime_nsec: u32_of(values.take("ctime_nsec")),
        }
        .to_le_bytes()
        .to_vec(),
        "oldstat" => OldStat {
            dev: u16_of(values.take("dev")),
            ino: u16_of(values.take("ino")),
            mode: u16_of(values.take("mode")),
            nlink: u16_of(values.take("nlink")),
            uid: u16_of(values.take("uid")),
            gid: u16_of(values.take("gid")),
            rdev: u16_of(values.take("rdev")),
            size: u32_of(values.take("size")),
            atime: u32_of(values.take("atime")),
            mtime: u32_of(values.take("mtime")),
            ctime: u32_of(values.take("ctime")),
        }
        .to_le_bytes()
        .to_vec(),
        "newstat" => NewStat {
            dev: u32_of(values.take("dev")),
            ino: u32_of(values.take("ino")),
            mode: u16_of(values.take("mode")),
            nlink: u16_of(values.take("nlink")),
            uid: u16_of(values.take("uid")),
            gid: u16_of(values.take("gid")),
            rdev: u32_of(values.take("rdev")),
            size: u32_of(values.take("size")),
            blksize: u32_of(values.take("blksize")),
            blocks: u32_of(values.take("blocks")),
            atime: u32_of(values.take("atime")),
            atime_nsec: u32_of(values.take("atime_nsec")),
            mtime: u32_of(values.take("mtime")),
            mtime_nsec: u32_of(values.take("mtime_nsec")),
            ctime: u32_of(values.take("ctime")),
            ctime_nsec: u32_of(values.take("ctime_nsec")),
            pad: [
                values.take("pad0") as u8,
                values.take("pad1") as u8,
                values.take("pad2") as u8,
                values.take("pad3") as u8,
                values.take("pad4") as u8,
                values.take("pad5") as u8,
                values.take("pad6") as u8,
                values.take("pad7") as u8,
            ],
        }
        .to_le_bytes()
        .to_vec(),
        "newstat64" => Newstat64 {
            dev: values.take("dev"),
            _pad1: u32_of(values.take("_pad1")),
            fucked_ino: u32_of(values.take("fucked_ino")),
            mode: u32_of(values.take("mode")),
            nlink: u32_of(values.take("nlink")),
            uid: u32_of(values.take("uid")),
            gid: u32_of(values.take("gid")),
            rdev: values.take("rdev"),
            _pad2: u32_of(values.take("_pad2")),
            size: values.take("size"),
            blksize: u32_of(values.take("blksize")),
            blocks: values.take("blocks"),
            atime: u32_of(values.take("atime")),
            atime_nsec: u32_of(values.take("atime_nsec")),
            mtime: u32_of(values.take("mtime")),
            mtime_nsec: u32_of(values.take("mtime_nsec")),
            ctime: u32_of(values.take("ctime")),
            ctime_nsec: u32_of(values.take("ctime_nsec")),
            ino: values.take("ino"),
        }
        .to_le_bytes()
        .to_vec(),
        "statfsbuf" => Statfsbuf {
            type_: values.take("type") as i64,
            bsize: values.take("bsize") as i64,
            blocks: values.take("blocks"),
            bfree: values.take("bfree"),
            bavail: values.take("bavail"),
            files: values.take("files"),
            ffree: values.take("ffree"),
            fsid: values.take("fsid"),
            namelen: values.take("namelen") as i64,
            frsize: values.take("frsize") as i64,
            flags: values.take("flags") as i64,
            spare: [
                values.take("spare0") as i64,
                values.take("spare1") as i64,
                values.take("spare2") as i64,
                values.take("spare3") as i64,
            ],
        }
        .to_le_bytes()
        .to_vec(),
        "statfs_" => Statfs {
            type_: u32_of(values.take("type")),
            bsize: u32_of(values.take("bsize")),
            blocks: u32_of(values.take("blocks")),
            bfree: u32_of(values.take("bfree")),
            bavail: u32_of(values.take("bavail")),
            files: u32_of(values.take("files")),
            ffree: u32_of(values.take("ffree")),
            fsid: values.take("fsid"),
            namelen: u32_of(values.take("namelen")),
            frsize: u32_of(values.take("frsize")),
            flags: u32_of(values.take("flags")),
            spare: [
                u32_of(values.take("spare0")),
                u32_of(values.take("spare1")),
                u32_of(values.take("spare2")),
                u32_of(values.take("spare3")),
            ],
        }
        .to_le_bytes()
        .to_vec(),
        "statfs64_" => Statfs64 {
            type_: u32_of(values.take("type")),
            bsize: u32_of(values.take("bsize")),
            blocks: values.take("blocks"),
            bfree: values.take("bfree"),
            bavail: values.take("bavail"),
            files: values.take("files"),
            ffree: values.take("ffree"),
            fsid: values.take("fsid"),
            namelen: u32_of(values.take("namelen")),
            frsize: u32_of(values.take("frsize")),
            flags: u32_of(values.take("flags")),
            pad: [
                u32_of(values.take("pad0")),
                u32_of(values.take("pad1")),
                u32_of(values.take("pad2")),
                u32_of(values.take("pad3")),
            ],
        }
        .to_le_bytes()
        .to_vec(),
        "statx_timestamp_" => StatxTimestamp {
            sec: values.take("sec") as i64,
            nsec: u32_of(values.take("nsec")),
            _pad: u32_of(values.take("_pad")),
        }
        .to_le_bytes()
        .to_vec(),
        "statx_" => {
            let mut pad2 = [0u32; 24];
            for (index, slot) in pad2.iter_mut().enumerate() {
                *slot = u32_of(values.take(&format!("_pad2{index}")));
            }
            Statx {
                mask: u32_of(values.take("mask")),
                blksize: u32_of(values.take("blksize")),
                attributes: values.take("attributes"),
                nlink: u32_of(values.take("nlink")),
                uid: u32_of(values.take("uid")),
                gid: u32_of(values.take("gid")),
                mode: u16_of(values.take("mode")),
                _pad1: u16_of(values.take("_pad1")),
                ino: values.take("ino"),
                size: values.take("size"),
                blocks: values.take("blocks"),
                attributes_mask: values.take("attributes_mask"),
                atime: StatxTimestamp {
                    sec: values.take("atime_sec") as i64,
                    nsec: u32_of(values.take("atime_nsec")),
                    _pad: u32_of(values.take("atime__pad")),
                },
                btime: StatxTimestamp {
                    sec: values.take("btime_sec") as i64,
                    nsec: u32_of(values.take("btime_nsec")),
                    _pad: u32_of(values.take("btime__pad")),
                },
                ctime: StatxTimestamp {
                    sec: values.take("ctime_sec") as i64,
                    nsec: u32_of(values.take("ctime_nsec")),
                    _pad: u32_of(values.take("ctime__pad")),
                },
                mtime: StatxTimestamp {
                    sec: values.take("mtime_sec") as i64,
                    nsec: u32_of(values.take("mtime_nsec")),
                    _pad: u32_of(values.take("mtime__pad")),
                },
                rdev_major: u32_of(values.take("rdev_major")),
                rdev_minor: u32_of(values.take("rdev_minor")),
                dev_major: u32_of(values.take("dev_major")),
                dev_minor: u32_of(values.take("dev_minor")),
                mnt_id: values.take("mnt_id"),
                dio_mem_align: u32_of(values.take("dio_mem_align")),
                dio_offset_align: u32_of(values.take("dio_offset_align")),
                _pad2: pad2,
            }
            .to_le_bytes()
            .to_vec()
        }
        other => panic!("unknown struct `{other}` in the fixture"),
    }
}

#[test]
fn stat_convert_newstat64_matches_the_c_reference() {
    let fixture = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|err| panic!("cannot read {FIXTURE}: {err}"));

    // The fixture's inputs and outputs, per case.
    let mut inputs: Vec<(String, Statbuf)> = Vec::new();
    let mut c_images: BTreeMap<String, Vec<u8>> = BTreeMap::new();
    let mut c_fields: BTreeMap<String, BTreeMap<String, u64>> = BTreeMap::new();

    for line in fixture.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split(' ').collect();
        let pairs = |text: &str| -> BTreeMap<String, u64> {
            text.split(' ')
                .map(|pair| {
                    let (name, value) = pair
                        .split_once('=')
                        .unwrap_or_else(|| panic!("malformed pair `{pair}` in: {line}"));
                    (name.to_owned(), hex(value))
                })
                .collect()
        };
        match fields[0] {
            "I" => {
                assert_eq!(fields.len(), 18, "malformed input record: {line}");
                let values = pairs(&fields[2..].join(" "));
                inputs.push((
                    fields[1].to_owned(),
                    Statbuf {
                        dev: values["dev"],
                        inode: values["inode"],
                        mode: u32_of(values["mode"]),
                        nlink: u32_of(values["nlink"]),
                        uid: u32_of(values["uid"]),
                        gid: u32_of(values["gid"]),
                        rdev: values["rdev"],
                        size: values["size"],
                        blksize: u32_of(values["blksize"]),
                        blocks: values["blocks"],
                        atime: u32_of(values["atime"]),
                        atime_nsec: u32_of(values["atime_nsec"]),
                        mtime: u32_of(values["mtime"]),
                        mtime_nsec: u32_of(values["mtime_nsec"]),
                        ctime: u32_of(values["ctime"]),
                        ctime_nsec: u32_of(values["ctime_nsec"]),
                    },
                ));
            }
            "C" if fields[2].starts_with("bytes=") => {
                let bytes = fields[2].strip_prefix("bytes=").expect("image record");
                c_images.insert(fields[1].to_owned(), decode_bytes(bytes));
            }
            "C" if fields[2] == "fields" => {
                c_fields.insert(fields[1].to_owned(), pairs(&fields[3..].join(" ")));
            }
            // The layout records belong to the other test; they are read there.
            "S" | "F" | "V" | "E" | "W" | "K" => {}
            other => panic!("unknown fixture record `{other}`: {line}"),
        }
    }

    assert!(!inputs.is_empty(), "the fixture has no conversion inputs");
    let mut converted = 0usize;
    for (label, input) in &inputs {
        let image = c_images
            .get(label)
            .unwrap_or_else(|| panic!("{label}: no conversion image in the fixture"));
        let expected = c_fields
            .get(label)
            .unwrap_or_else(|| panic!("{label}: no converted fields in the fixture"));
        let out = stat_convert_newstat64(input);

        // Field by field first, so a wrong field names itself, then the image,
        // which is what actually reaches the guest.
        let checks: [(&str, u64); 18] = [
            ("dev", out.dev),
            ("fucked_ino", u64::from(out.fucked_ino)),
            ("ino", out.ino),
            ("mode", u64::from(out.mode)),
            ("nlink", u64::from(out.nlink)),
            ("uid", u64::from(out.uid)),
            ("gid", u64::from(out.gid)),
            ("rdev", out.rdev),
            ("size", out.size),
            ("blksize", u64::from(out.blksize)),
            ("blocks", out.blocks),
            ("atime", u64::from(out.atime)),
            ("atime_nsec", u64::from(out.atime_nsec)),
            ("mtime", u64::from(out.mtime)),
            ("mtime_nsec", u64::from(out.mtime_nsec)),
            ("ctime", u64::from(out.ctime)),
            ("ctime_nsec", u64::from(out.ctime_nsec)),
            ("ino_again", out.ino),
        ];
        for (name, got) in checks {
            let name = if name == "ino_again" { "ino" } else { name };
            assert_eq!(
                got, expected[name],
                "{label}: newstat64.{name} is {got:#x}, the C says {:#x}",
                expected[name]
            );
        }

        // The C leaves `_pad1` and `_pad2` alone; the oracle zeroes them (see
        // the `W` record) and the port writes zero, so they compare like any
        // other byte here.
        let rust_image = out.to_le_bytes();
        compare_image(label, &rust_image, image, |_| false);
        converted += 1;
    }

    assert_eq!(converted, 5, "the fixture's conversion corpus changed size");
    println!("{converted} statbufs converted to exactly the C's newstat64 bytes");
}
