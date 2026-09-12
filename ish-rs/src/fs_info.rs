//! `kernel/fs.h` + `kernel/fs_info.c` — filesystem info (cwd, root, umask).
//!
//! `struct fs_info` holds the process's current working directory, root
//! directory, and umask. In C it uses explicit refcounting and fd retain/release.
//! This Rust port uses `Rc` for shared ownership and `RefCount` for the explicit
//! counter, depending only on `refcount`, `sync`, and a placeholder fd type.
//!
//! Full fd integration will come after `fd.rs` is ported; for now `FsInfo`
//! stores `Option<FdId>` as a stand-in for `struct fd *`.

use crate::refcount::RefCount;
use crate::sync::Lock;

/// Placeholder for fd identity until `fd.rs` is fully ported.
pub type FdId = u32;

/// `struct fs_info` — simplified.
pub struct FsInfo {
    pub refcount: RefCount,
    pub umask: u32,
    pub pwd: Option<FdId>,
    pub root: Option<FdId>,
    pub lock: Lock,
}

impl FsInfo {
    /// `fs_info_new`
    pub fn new() -> Self {
        Self {
            refcount: RefCount::new(),
            umask: 0,
            pwd: None,
            root: None,
            lock: Lock::new(),
        }
    }

    /// `fs_info_copy` — copies umask and retains pwd/root.
    pub fn copy(&self) -> Self {
        Self {
            refcount: RefCount::new(),
            umask: self.umask,
            pwd: self.pwd,
            root: self.root,
            lock: Lock::new(),
        }
    }

    /// `fs_info_release` — returns true if refcount hit zero.
    pub fn release(&self) -> bool {
        self.refcount.release()
    }

    pub fn retain(&self) -> usize {
        self.refcount.retain()
    }

    pub fn set_umask(&mut self, mask: u32) -> u32 {
        let old = self.umask;
        self.umask = mask & 0o777;
        old
    }

    pub fn chdir(&mut self, pwd: FdId) {
        self.pwd = Some(pwd);
    }
}

impl Default for FsInfo {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fs_info_new_has_refcount_one_and_zero_umask() {
        let fs = FsInfo::new();
        assert_eq!(fs.refcount.get(), 1);
        assert_eq!(fs.umask, 0);
        assert!(fs.pwd.is_none());
    }

    #[test]
    fn fs_info_copy_retains_pwd_root_and_umask() {
        let mut fs = FsInfo::new();
        fs.umask = 0o022;
        fs.pwd = Some(3);
        fs.root = Some(4);
        let copy = fs.copy();
        assert_eq!(copy.umask, 0o022);
        assert_eq!(copy.pwd, Some(3));
        assert_eq!(copy.root, Some(4));
        assert_eq!(copy.refcount.get(), 1);
    }

    #[test]
    fn fs_info_umask_truncates_to_0o777() {
        let mut fs = FsInfo::new();
        let old = fs.set_umask(0o7777);
        assert_eq!(old, 0);
        assert_eq!(fs.umask, 0o777);
    }
}
