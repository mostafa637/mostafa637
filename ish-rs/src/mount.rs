//! `kernel/fs.h` mount constants and structures.
//!
//! The full mount table depends on `fs/fd`, `fs/inode`, etc. This module
//! ports the leaf constants and the `fs_ops` / `mount` descriptors.

/// Mount flags from `calls.h`
pub const MS_RDONLY: u32 = 1 << 0;
pub const MS_NOSUID: u32 = 1 << 1;
pub const MS_NODEV: u32 = 1 << 2;
pub const MS_NOEXEC: u32 = 1 << 3;
pub const MS_SILENT: u32 = 1 << 15;

/// Open flags re-exported for mount parsing (same as fd.rs)
pub use crate::fd::{O_ACCMODE, O_CREAT, O_DIRECTORY, O_RDONLY, O_RDWR, O_WRONLY};

/// File system magic numbers (partial, from various fs headers)
pub const TMPFS_MAGIC: u32 = 0x01021994;
pub const PROC_SUPER_MAGIC: u32 = 0x9fa0;

/// Mount point descriptor, simplified.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MountPoint {
    pub point: String,
    pub source: String,
    pub fs_name: String,
    pub flags: u32,
}

impl MountPoint {
    pub fn new(point: impl Into<String>, source: impl Into<String>, fs_name: impl Into<String>, flags: u32) -> Self {
        Self {
            point: point.into(),
            source: source.into(),
            fs_name: fs_name.into(),
            flags,
        }
    }

    pub fn is_readonly(&self) -> bool {
        (self.flags & MS_RDONLY) != 0
    }
}

/// Check if mount info string contains a flag, matching C `mount_param_flag`.
pub fn mount_param_flag(info: &str, flag: &str) -> bool {
    for param in info.split(',') {
        if param == flag {
            return true;
        }
    }
    false
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn mount_flags_match_c() {
        assert_eq!(MS_RDONLY, 1);
        assert_eq!(MS_NOSUID, 2);
        assert_eq!(MS_NODEV, 4);
        assert_eq!(MS_NOEXEC, 8);
    }

    #[test]
    fn mount_param_flag_matches_c() {
        assert!(mount_param_flag("foo,bar,baz", "bar"));
        assert!(!mount_param_flag("foo,bar,baz", "qux"));
        assert!(mount_param_flag("single", "single"));
        assert!(!mount_param_flag("", "foo"));
    }

    #[test]
    fn mount_point_readonly() {
        let mp = MountPoint::new("/", "/dev/root", "fakefs", MS_RDONLY);
        assert!(mp.is_readonly());
        let mp2 = MountPoint::new("/", "/dev/root", "fakefs", 0);
        assert!(!mp2.is_readonly());
    }
}
