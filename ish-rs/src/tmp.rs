//! `fs/tmp.c` — tmpfs constants and directory operations.

pub const TMPFS_TYPE_FILE: u32 = 1;
pub const TMPFS_TYPE_DIR: u32 = 2;
pub const TMPFS_TYPE_SYMLINK: u32 = 3;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TmpDirent {
    pub name: String,
    pub file_type: u32,
    pub inode: u64,
}

impl TmpDirent {
    pub fn new(name: impl Into<String>, file_type: u32, inode: u64) -> Self {
        Self { name: name.into(), file_type, inode }
    }
    pub fn is_dir(&self) -> bool { self.file_type == TMPFS_TYPE_DIR }
    pub fn is_file(&self) -> bool { self.file_type == TMPFS_TYPE_FILE }
    pub fn is_symlink(&self) -> bool { self.file_type == TMPFS_TYPE_SYMLINK }
}

/// Tmpfs directory, simplified
#[derive(Debug, Default)]
pub struct TmpDir {
    pub entries: Vec<TmpDirent>,
    pub next_inode: u64,
}

impl TmpDir {
    pub fn new() -> Self {
        Self { entries: Vec::new(), next_inode: 1 }
    }

    pub fn create(&mut self, name: &str, file_type: u32) -> Result<u64, i32> {
        if self.entries.iter().any(|e| e.name == name) {
            return Err(-17); // EEXIST
        }
        let inode = self.next_inode;
        self.next_inode += 1;
        self.entries.push(TmpDirent::new(name, file_type, inode));
        Ok(inode)
    }

    pub fn lookup(&self, name: &str) -> Option<&TmpDirent> {
        self.entries.iter().find(|e| e.name == name)
    }

    pub fn unlink(&mut self, name: &str) -> Result<(), i32> {
        if let Some(idx) = self.entries.iter().position(|e| e.name == name) {
            self.entries.remove(idx);
            Ok(())
        } else {
            Err(-2) // ENOENT
        }
    }

    pub fn readdir(&self, pos: usize) -> Option<&TmpDirent> {
        self.entries.get(pos)
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

    #[test]
    fn tmp_dir_create_lookup_unlink() {
        let mut dir = TmpDir::new();
        assert_eq!(dir.create("foo", TMPFS_TYPE_FILE).unwrap(), 1);
        assert_eq!(dir.create("bar", TMPFS_TYPE_DIR).unwrap(), 2);
        assert!(dir.create("foo", TMPFS_TYPE_FILE).is_err()); // EEXIST
        assert!(dir.lookup("foo").is_some());
        assert!(dir.lookup("baz").is_none());
        assert!(dir.unlink("foo").is_ok());
        assert!(dir.lookup("foo").is_none());
        assert!(dir.unlink("foo").is_err());
    }

    #[test]
    fn tmp_dir_readdir() {
        let mut dir = TmpDir::new();
        dir.create("a", TMPFS_TYPE_FILE).unwrap();
        dir.create("b", TMPFS_TYPE_FILE).unwrap();
        assert_eq!(dir.readdir(0).unwrap().name, "a");
        assert_eq!(dir.readdir(1).unwrap().name, "b");
        assert!(dir.readdir(2).is_none());
    }
}
