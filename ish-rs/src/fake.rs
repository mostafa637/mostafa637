//! `fs/fake.c` — fake filesystem constants and inode helpers with DB logic.

pub const FAKEFS_MAGIC: u32 = 0x66616b65;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct IshStat {
    pub mode: u32,
    pub uid: u32,
    pub gid: u32,
    pub rdev: u32,
}

impl IshStat {
    pub fn new(mode: u32, uid: u32, gid: u32, rdev: u32) -> Self { Self { mode, uid, gid, rdev } }
    pub fn is_dir(&self) -> bool { (self.mode & 0o170000) == 0o040000 }
    pub fn is_file(&self) -> bool { (self.mode & 0o170000) == 0o100000 }
    pub fn is_symlink(&self) -> bool { (self.mode & 0o170000) == 0o120000 }
}

/// Fakefs path operations, simplified from C
#[derive(Debug, Default)]
pub struct FakeFs {
    pub inodes: std::collections::HashMap<String, (u64, IshStat)>,
    pub next_inode: u64,
}

impl FakeFs {
    pub fn new() -> Self { Self { inodes: std::collections::HashMap::new(), next_inode: 1 } }

    pub fn path_get_inode(&self, path: &str) -> u64 {
        self.inodes.get(path).map(|(ino, _)| *ino).unwrap_or(0)
    }

    pub fn path_create(&mut self, path: &str, stat: IshStat) -> u64 {
        if self.inodes.contains_key(path) {
            return self.inodes[path].0;
        }
        let ino = self.next_inode;
        self.next_inode += 1;
        self.inodes.insert(path.to_string(), (ino, stat));
        ino
    }

    pub fn path_link(&mut self, src: &str, dst: &str) -> Result<(), i32> {
        if let Some((ino, stat)) = self.inodes.get(src).cloned() {
            if self.inodes.contains_key(dst) {
                return Err(-17); // EEXIST
            }
            self.inodes.insert(dst.to_string(), (ino, stat));
            Ok(())
        } else {
            Err(-2) // ENOENT
        }
    }

    pub fn path_unlink(&mut self, path: &str) -> Result<u64, i32> {
        if let Some((ino, _)) = self.inodes.remove(path) {
            Ok(ino)
        } else {
            Err(-2)
        }
    }

    pub fn path_rename(&mut self, src: &str, dst: &str) -> Result<(), i32> {
        if let Some(entry) = self.inodes.remove(src) {
            self.inodes.insert(dst.to_string(), entry);
            Ok(())
        } else {
            Err(-2)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ish_stat_type_checks() {
        let dir = IshStat::new(0o040755, 1000, 1000, 0);
        assert!(dir.is_dir());
        assert!(!dir.is_file());
        let file = IshStat::new(0o100644, 1000, 1000, 0);
        assert!(file.is_file());
        let link = IshStat::new(0o120777, 1000, 1000, 0);
        assert!(link.is_symlink());
    }

    #[test]
    fn fakefs_path_ops() {
        let mut fs = FakeFs::new();
        assert_eq!(fs.path_get_inode("/foo"), 0);
        let ino = fs.path_create("/foo", IshStat::new(0o100644, 1000, 1000, 0));
        assert_eq!(ino, 1);
        assert_eq!(fs.path_get_inode("/foo"), 1);
        assert!(fs.path_link("/foo", "/bar").is_ok());
        assert_eq!(fs.path_get_inode("/bar"), 1);
        assert!(fs.path_rename("/bar", "/baz").is_ok());
        assert_eq!(fs.path_get_inode("/baz"), 1);
        assert!(fs.path_get_inode("/bar") == 0);
        assert_eq!(fs.path_unlink("/foo").unwrap(), 1);
        assert_eq!(fs.path_get_inode("/foo"), 0);
    }
}
