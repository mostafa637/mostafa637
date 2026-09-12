//! `fs/tmp.c` — tmpfs constants and directory entry helpers.

/// Tmpfs file type constants.
pub const TMPFS_TYPE_FILE: u32 = 1;
pub const TMPFS_TYPE_DIR: u32 = 2;
pub const TMPFS_TYPE_SYMLINK: u32 = 3;

/// Tmpfs directory entry, simplified.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TmpDirent {
    pub name: String,
    pub file_type: u32,
    pub inode: u64,
}

impl TmpDirent {
    pub fn new(name: impl Into<String>, file_type: u32, inode: u64) -> Self {
        Self {
            name: name.into(),
            file_type,
            inode,
        }
    }

    pub fn is_dir(&self) -> bool {
        self.file_type == TMPFS_TYPE_DIR
    }

    pub fn is_file(&self) -> bool {
        self.file_type == TMPFS_TYPE_FILE
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tmp_dirent_type_checks() {
        let dir = TmpDirent::new("a", TMPFS_TYPE_DIR, 1);
        assert!(dir.is_dir());
        assert!(!dir.is_file());
        let file = TmpDirent::new("b", TMPFS_TYPE_FILE, 2);
        assert!(file.is_file());
    }
}
