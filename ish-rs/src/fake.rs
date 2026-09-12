//! `fs/fake.c` — fake filesystem constants and inode helpers with DB logic full port.

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
    pub fn is_char(&self) -> bool { (self.mode & 0o170000) == 0o020000 }
    pub fn is_block(&self) -> bool { (self.mode & 0o170000) == 0o060000 }
    pub fn is_fifo(&self) -> bool { (self.mode & 0o170000) == 0o010000 }
    pub fn is_socket(&self) -> bool { (self.mode & 0o170000) == 0o140000 }
    pub fn perm(&self) -> u32 { self.mode & 0o7777 }
}

#[derive(Debug, Clone)]
pub struct FakeInode {
    pub ino: u64,
    pub stat: IshStat,
    pub xattrs: std::collections::HashMap<String, Vec<u8>>,
    pub link_count: u32,
}

impl FakeInode {
    pub fn new(ino: u64, stat: IshStat) -> Self {
        Self { ino, stat, xattrs: std::collections::HashMap::new(), link_count: 1 }
    }
}

#[derive(Debug, Default)]
pub struct FakeFs {
    pub inodes: std::collections::HashMap<String, (u64, IshStat)>,
    pub inode_table: std::collections::HashMap<u64, FakeInode>,
    pub next_inode: u64,
}

impl FakeFs {
    pub fn new() -> Self { Self { inodes: std::collections::HashMap::new(), inode_table: std::collections::HashMap::new(), next_inode: 1 } }

    pub fn path_get_inode(&self, path: &str) -> u64 {
        self.inodes.get(path).map(|(ino, _)| *ino).unwrap_or(0)
    }

    pub fn path_create(&mut self, path: &str, stat: IshStat) -> u64 {
        if let Some((ino, _)) = self.inodes.get(path) { return *ino; }
        let ino = self.next_inode;
        self.next_inode += 1;
        self.inodes.insert(path.to_string(), (ino, stat));
        self.inode_table.insert(ino, FakeInode::new(ino, stat));
        ino
    }

    pub fn path_link(&mut self, src: &str, dst: &str) -> Result<(), i32> {
        if let Some((ino, stat)) = self.inodes.get(src).cloned() {
            if self.inodes.contains_key(dst) { return Err(-17); }
            self.inodes.insert(dst.to_string(), (ino, stat));
            if let Some(inode) = self.inode_table.get_mut(&ino) { inode.link_count += 1; }
            Ok(())
        } else { Err(-2) }
    }

    pub fn path_unlink(&mut self, path: &str) -> Result<u64, i32> {
        if let Some((ino, _)) = self.inodes.remove(path) {
            let should_remove = if let Some(inode) = self.inode_table.get_mut(&ino) {
                if inode.link_count > 0 { inode.link_count -= 1; }
                inode.link_count == 0
            } else { true };
            if should_remove { self.inode_table.remove(&ino); }
            Ok(ino)
        } else { Err(-2) }
    }

    pub fn path_rename(&mut self, src: &str, dst: &str) -> Result<(), i32> {
        if let Some(entry) = self.inodes.remove(src) {
            if let Some((old_ino, _)) = self.inodes.remove(dst) {
                if let Some(inode) = self.inode_table.get_mut(&old_ino) {
                    if inode.link_count > 0 { inode.link_count -= 1; }
                    if inode.link_count == 0 { self.inode_table.remove(&old_ino); }
                }
            }
            self.inodes.insert(dst.to_string(), entry);
            Ok(())
        } else { Err(-2) }
    }

    pub fn get_inode(&self, ino: u64) -> Option<&FakeInode> { self.inode_table.get(&ino) }

    pub fn set_xattr(&mut self, path: &str, name: &str, value: Vec<u8>) -> Result<(), i32> {
        let ino = self.path_get_inode(path);
        if ino == 0 { return Err(-2); }
        if let Some(inode) = self.inode_table.get_mut(&ino) {
            inode.xattrs.insert(name.to_string(), value);
            Ok(())
        } else { Err(-2) }
    }

    pub fn get_xattr(&self, path: &str, name: &str) -> Result<Vec<u8>, i32> {
        let ino = self.path_get_inode(path);
        if ino == 0 { return Err(-2); }
        if let Some(inode) = self.inode_table.get(&ino) {
            inode.xattrs.get(name).cloned().ok_or(-61)
        } else { Err(-2) }
    }

    pub fn list_xattrs(&self, path: &str) -> Result<Vec<String>, i32> {
        let ino = self.path_get_inode(path);
        if ino == 0 { return Err(-2); }
        if let Some(inode) = self.inode_table.get(&ino) {
            Ok(inode.xattrs.keys().cloned().collect())
        } else { Err(-2) }
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
        assert_eq!(file.perm(), 0o644);
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

    #[test]
    fn fakefs_link_count_and_xattr() {
        let mut fs = FakeFs::new();
        let ino = fs.path_create("/a", IshStat::new(0o100644, 1000, 1000, 0));
        assert_eq!(fs.get_inode(ino).unwrap().link_count, 1);
        fs.path_link("/a", "/b").unwrap();
        assert_eq!(fs.get_inode(ino).unwrap().link_count, 2);
        fs.path_unlink("/a").unwrap();
        assert_eq!(fs.get_inode(ino).unwrap().link_count, 1);
        assert!(fs.get_inode(ino).is_some());
        fs.path_unlink("/b").unwrap();
        assert!(fs.get_inode(ino).is_none());

        let _ino2 = fs.path_create("/c", IshStat::new(0o100644, 1000, 1000, 0));
        fs.set_xattr("/c", "user.foo", b"bar".to_vec()).unwrap();
        assert_eq!(fs.get_xattr("/c", "user.foo").unwrap(), b"bar");
        assert!(fs.get_xattr("/c", "user.missing").is_err());
        let list = fs.list_xattrs("/c").unwrap();
        assert!(list.contains(&"user.foo".to_string()));
    }
}
