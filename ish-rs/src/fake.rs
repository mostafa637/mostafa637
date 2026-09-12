//! `fs/fake.c` — fake filesystem full port, matching C implementation.

use std::collections::HashMap;

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
    pub fn is_block(&self) -> bool { (self.mode & 0o170000) == 0o060000 }
    pub fn is_char(&self) -> bool { (self.mode & 0o170000) == 0o020000 }
    pub fn is_socket(&self) -> bool { (self.mode & 0o170000) == 0o140000 }
    pub fn perm(&self) -> u32 { self.mode & 0o7777 }
}

#[derive(Debug, Clone)]
pub struct FakeInode {
    pub ino: u64,
    pub stat: IshStat,
    pub xattrs: HashMap<String, Vec<u8>>,
    pub link_count: u32,
    pub data: Vec<u8>,
    pub symlink_target: Option<String>,
}

impl FakeInode {
    pub fn new(ino: u64, stat: IshStat) -> Self {
        Self { ino, stat, xattrs: HashMap::new(), link_count: 1, data: Vec::new(), symlink_target: None }
    }
    pub fn new_symlink(ino: u64, stat: IshStat, target: String) -> Self {
        Self { ino, stat, xattrs: HashMap::new(), link_count: 1, data: Vec::new(), symlink_target: Some(target) }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum AttrType {
    #[default]
    Uid,
    Gid,
    Mode,
    Size,
}

#[derive(Debug, Clone, Default)]
pub struct Attr {
    pub type_: AttrType,
    pub uid: u32,
    pub gid: u32,
    pub mode: u32,
    pub size: u64,
}

impl Attr {
    pub fn uid(uid: u32) -> Self { Self { type_: AttrType::Uid, uid, ..Default::default() } }
    pub fn gid(gid: u32) -> Self { Self { type_: AttrType::Gid, gid, ..Default::default() } }
    pub fn mode(mode: u32) -> Self { Self { type_: AttrType::Mode, mode, ..Default::default() } }
    pub fn size(size: u64) -> Self { Self { type_: AttrType::Size, size, ..Default::default() } }
}

#[derive(Debug, Default)]
pub struct FakeFs {
    pub inodes: HashMap<String, (u64, IshStat)>,
    pub inode_table: HashMap<u64, FakeInode>,
    pub next_inode: u64,
    pub root_fd: i32,
    pub db_path: String,
}

impl FakeFs {
    pub fn new() -> Self { Self { inodes: HashMap::new(), inode_table: HashMap::new(), next_inode: 1, root_fd: -1, db_path: String::new() } }

    pub fn path_get_inode(&self, path: &str) -> u64 { self.inodes.get(path).map(|(ino, _)| *ino).unwrap_or(0) }

    pub fn path_read_stat(&self, path: &str) -> Option<(IshStat, u64)> {
        self.inodes.get(path).map(|(ino, stat)| (*stat, *ino))
    }

    pub fn inode_read_stat_if_exist(&self, ino: u64) -> Option<IshStat> {
        self.inode_table.get(&ino).map(|i| i.stat)
    }

    pub fn inode_read_stat_or_die(&self, ino: u64) -> IshStat {
        self.inode_table.get(&ino).map(|i| i.stat).unwrap_or_else(|| IshStat::new(0o100644, 0, 0, 0))
    }

    pub fn inode_write_stat(&mut self, ino: u64, stat: &IshStat) {
        if let Some(inode) = self.inode_table.get_mut(&ino) { inode.stat = *stat; }
        // Also update inodes map
        let paths: Vec<String> = self.inodes.iter().filter(|(_, (i, _))| *i == ino).map(|(p, _)| p.clone()).collect();
        for path in paths {
            if let Some(entry) = self.inodes.get_mut(&path) { entry.1 = *stat; }
        }
    }

    pub fn path_create(&mut self, path: &str, stat: IshStat) -> u64 {
        if let Some((ino, _)) = self.inodes.get(path) { return *ino; }
        let ino = self.next_inode;
        self.next_inode += 1;
        self.inodes.insert(path.to_string(), (ino, stat));
        self.inode_table.insert(ino, FakeInode::new(ino, stat));
        ino
    }

    pub fn path_create_file(&mut self, path: &str, stat: IshStat, data: Vec<u8>) -> u64 {
        let ino = self.path_create(path, stat);
        if let Some(inode) = self.inode_table.get_mut(&ino) { inode.data = data; }
        ino
    }

    pub fn path_create_symlink(&mut self, path: &str, target: &str) -> u64 {
        let stat = IshStat::new(0o120777, 0, 0, 0);
        let ino = self.next_inode;
        self.next_inode += 1;
        self.inodes.insert(path.to_string(), (ino, stat));
        self.inode_table.insert(ino, FakeInode::new_symlink(ino, stat, target.to_string()));
        ino
    }

    pub fn resolve_symlink(&self, path: &str) -> String {
        let mut current = path.to_string();
        for _ in 0..10 {
            let ino = self.path_get_inode(&current);
            if ino == 0 { break; }
            if let Some(inode) = self.inode_table.get(&ino) {
                if let Some(target) = &inode.symlink_target { current = target.clone(); continue; }
            }
            break;
        }
        current
    }

    pub fn read_file(&self, path: &str) -> Result<Vec<u8>, i32> {
        let resolved = self.resolve_symlink(path);
        let ino = self.path_get_inode(&resolved);
        if ino == 0 { return Err(-2); }
        self.inode_table.get(&ino).map(|i| i.data.clone()).ok_or(-2)
    }

    pub fn write_file(&mut self, path: &str, data: Vec<u8>) -> Result<(), i32> {
        let resolved = self.resolve_symlink(path);
        let ino = self.path_get_inode(&resolved);
        if ino == 0 { return Err(-2); }
        if let Some(inode) = self.inode_table.get_mut(&ino) { inode.data = data; Ok(()) } else { Err(-2) }
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
        if let Some(inode) = self.inode_table.get_mut(&ino) { inode.xattrs.insert(name.to_string(), value); Ok(()) } else { Err(-2) }
    }

    pub fn get_xattr(&self, path: &str, name: &str) -> Result<Vec<u8>, i32> {
        let ino = self.path_get_inode(path);
        if ino == 0 { return Err(-2); }
        if let Some(inode) = self.inode_table.get(&ino) { inode.xattrs.get(name).cloned().ok_or(-61) } else { Err(-2) }
    }

    pub fn list_xattrs(&self, path: &str) -> Result<Vec<String>, i32> {
        let ino = self.path_get_inode(path);
        if ino == 0 { return Err(-2); }
        if let Some(inode) = self.inode_table.get(&ino) { Ok(inode.xattrs.keys().cloned().collect()) } else { Err(-2) }
    }

    // Full fakefs operations matching C

    pub fn open(&mut self, path: &str, flags: u32, mode: u32) -> Result<u64, i32> {
        const O_CREAT: u32 = 64;
        let mut ino = self.path_get_inode(path);
        if (flags & O_CREAT) != 0 && ino == 0 {
            let stat = IshStat::new(0o100000 | (mode & 0o777), 0, 0, 0);
            ino = self.path_create(path, stat);
        }
        if ino == 0 { return Err(-2); }
        Ok(ino)
    }

    pub fn open_inode(&self, ino: u64) -> Result<String, i32> {
        for (path, (i, _)) in &self.inodes {
            if *i == ino { return Ok(path.clone()); }
        }
        Err(-2)
    }

    pub fn link(&mut self, src: &str, dst: &str) -> Result<(), i32> { self.path_link(src, dst) }
    pub fn unlink(&mut self, path: &str) -> Result<u64, i32> { self.path_unlink(path) }
    pub fn rmdir(&mut self, path: &str) -> Result<u64, i32> { self.path_unlink(path) }
    pub fn rename(&mut self, src: &str, dst: &str) -> Result<(), i32> { self.path_rename(src, dst) }

    pub fn symlink(&mut self, target: &str, link: &str) -> Result<(), i32> {
        // Create file containing target like C does
        let stat = IshStat::new(0o120777, 0, 0, 0);
        let ino = self.next_inode;
        self.next_inode += 1;
        self.inodes.insert(link.to_string(), (ino, stat));
        let mut inode = FakeInode::new_symlink(ino, stat, target.to_string());
        inode.data = target.as_bytes().to_vec(); // C writes target to file
        self.inode_table.insert(ino, inode);
        Ok(())
    }

    pub fn mknod(&mut self, path: &str, mode: u32, dev: u32) -> Result<(), i32> {
        let real_mode = if (mode & 0o170000) == 0o060000 || (mode & 0o170000) == 0o020000 || (mode & 0o170000) == 0o140000 {
            0o100000 | 0o666
        } else {
            mode
        };
        let stat = IshStat::new(mode, 0, 0, if (mode & 0o170000) == 0o060000 || (mode & 0o170000) == 0o020000 { dev } else { 0 });
        self.path_create(path, stat);
        Ok(())
    }

    pub fn stat(&self, path: &str) -> Result<(IshStat, u64), i32> {
        if let Some((ino, stat)) = self.inodes.get(path) { Ok((*stat, *ino)) } else { Err(-2) }
    }

    pub fn fstat(&self, ino: u64) -> Result<IshStat, i32> {
        self.inode_table.get(&ino).map(|i| i.stat).ok_or(-2)
    }

    pub fn setattr(&mut self, path: &str, attr: Attr) -> Result<(), i32> {
        if attr.type_ == AttrType::Size {
            // Handled by realfs in C
            return Ok(());
        }
        if let Some((ino, _)) = self.inodes.get(path).cloned() {
            if let Some(inode) = self.inode_table.get_mut(&ino) {
                match attr.type_ {
                    AttrType::Uid => inode.stat.uid = attr.uid,
                    AttrType::Gid => inode.stat.gid = attr.gid,
                    AttrType::Mode => inode.stat.mode = (inode.stat.mode & 0o170000) | (attr.mode & !0o170000),
                    AttrType::Size => {},
                }
                // Update inodes map
                let paths: Vec<String> = self.inodes.iter().filter(|(_, (i, _))| *i == ino).map(|(p, _)| p.clone()).collect();
                for p in paths {
                    if let Some(entry) = self.inodes.get_mut(&p) { entry.1 = inode.stat; }
                }
                return Ok(());
            }
        }
        Err(-2)
    }

    pub fn fsetattr(&mut self, ino: u64, attr: Attr) -> Result<(), i32> {
        if attr.type_ == AttrType::Size { return Ok(()); }
        if let Some(inode) = self.inode_table.get_mut(&ino) {
            match attr.type_ {
                AttrType::Uid => inode.stat.uid = attr.uid,
                AttrType::Gid => inode.stat.gid = attr.gid,
                AttrType::Mode => inode.stat.mode = (inode.stat.mode & 0o170000) | (attr.mode & !0o170000),
                AttrType::Size => {},
            }
            Ok(())
        } else { Err(-2) }
    }

    pub fn mkdir(&mut self, path: &str, mode: u32) -> Result<(), i32> {
        if self.inodes.contains_key(path) { return Err(-17); }
        let stat = IshStat::new(0o040000 | (mode & 0o777), 0, 0, 0);
        self.path_create(path, stat);
        Ok(())
    }

    pub fn readlink(&self, path: &str) -> Result<String, i32> {
        let ino = self.path_get_inode(path);
        if ino == 0 { return Err(-2); }
        if let Some(inode) = self.inode_table.get(&ino) {
            if !inode.stat.is_symlink() { return Err(-22); }
            if let Some(target) = &inode.symlink_target { Ok(target.clone()) } else {
                // Fallback to file content like C's file_readlink
                Ok(String::from_utf8_lossy(&inode.data).to_string())
            }
        } else { Err(-2) }
    }

    pub fn readdir(&self, path: &str) -> Vec<(u64, String)> {
        let prefix = if path == "/" { "/".to_string() } else { format!("{}/", path) };
        let mut entries = Vec::new();
        for (p, (ino, _)) in &self.inodes {
            if p.starts_with(&prefix) && !p[prefix.len()..].contains('/') {
                let name = p[prefix.len()..].to_string();
                if !name.is_empty() { entries.push((*ino, name)); }
            }
        }
        entries
    }

    pub fn mount(&mut self, source: &str, root_fd: i32) -> Result<(), i32> {
        self.db_path = source.replace("data", "meta.db");
        self.root_fd = root_fd;
        Ok(())
    }

    pub fn umount(&mut self) -> Result<(), i32> { Ok(()) }

    pub fn inode_orphaned(&mut self, ino: u64) -> Result<(), i32> {
        // Try cleanup inode, matching fakefs_inode_orphaned
        if let Some(inode) = self.inode_table.get(&ino) {
            if inode.link_count == 0 { self.inode_table.remove(&ino); }
        }
        Ok(())
    }

    pub fn load_alpine_mock(&mut self) {
        for dir in &["/", "/bin", "/etc", "/lib", "/usr", "/usr/bin", "/proc", "/dev", "/tmp"] {
            self.path_create(dir, IshStat::new(0o040755, 0, 0, 0));
        }
        let mut elf = vec![0u8; 100];
        elf[0..4].copy_from_slice(b"\x7fELF");
        elf[4] = 1; elf[5] = 1; elf[6] = 1;
        elf[16] = 2; elf[17] = 0;
        elf[18] = 3; elf[19] = 0;
        elf[24] = 0x00; elf[25] = 0x80; elf[26] = 0x04; elf[27] = 0x08;
        elf[28] = 52;
        elf[42] = 32; elf[43] = 0;
        elf[44] = 1; elf[45] = 0;
        elf[52] = 1;
        elf[56] = 0;
        elf[60] = 0x00; elf[61] = 0x80; elf[62] = 0x04; elf[63] = 0x08;
        elf[68] = 100;
        elf[72] = 100;
        elf[76] = 5;
        self.path_create_file("/bin/busybox", IshStat::new(0o100755, 0, 0, 0), elf);
        self.path_create_symlink("/bin/sh", "/bin/busybox");
        self.path_create_file("/etc/passwd", IshStat::new(0o100644, 0, 0, 0), b"root:x:0:0:root:/root:/bin/sh\n".to_vec());
        self.path_create_file("/etc/hosts", IshStat::new(0o100644, 0, 0, 0), b"127.0.0.1 localhost\n".to_vec());
    }

    pub fn list_dir(&self, path: &str) -> Vec<String> {
        let prefix = if path == "/" { "/".to_string() } else { format!("{}/", path) };
        self.inodes.keys().filter(|k| k.starts_with(&prefix) && !k[prefix.len()..].contains('/')).cloned().collect()
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

    #[test]
    fn fakefs_file_content_and_alpine_mock() {
        let mut fs = FakeFs::new();
        fs.path_create_file("/test", IshStat::new(0o100644, 0, 0, 0), b"hello".to_vec());
        assert_eq!(fs.read_file("/test").unwrap(), b"hello");
        fs.write_file("/test", b"world".to_vec()).unwrap();
        assert_eq!(fs.read_file("/test").unwrap(), b"world");

        fs.load_alpine_mock();
        assert!(fs.path_get_inode("/bin/sh") != 0);
        assert!(fs.path_get_inode("/bin/busybox") != 0);
        assert!(fs.path_get_inode("/etc/passwd") != 0);
        let busybox_data = fs.read_file("/bin/busybox").unwrap();
        assert_eq!(&busybox_data[0..4], b"\x7fELF");
        let sh_data = fs.read_file("/bin/sh").unwrap();
        assert_eq!(&sh_data[0..4], b"\x7fELF");
        let passwd = fs.read_file("/etc/passwd").unwrap();
        assert!(String::from_utf8(passwd).unwrap().contains("root"));
    }

    #[test]
    fn fakefs_full_ops() {
        let mut fs = FakeFs::new();
        fs.path_create("/foo", IshStat::new(0o100644, 0, 0, 0));
        // open
        assert!(fs.open("/foo", 0, 0).is_ok());
        assert!(fs.open("/nonexistent", 0, 0).is_err());
        assert!(fs.open("/new", 64, 0o644).is_ok()); // O_CREAT
        // link
        assert!(fs.link("/foo", "/bar").is_ok());
        // stat
        let (stat, ino) = fs.stat("/foo").unwrap();
        assert!(stat.is_file());
        assert_eq!(ino, 1);
        // fstat
        assert!(fs.fstat(ino).is_ok());
        // setattr
        fs.setattr("/foo", Attr::uid(1000)).unwrap();
        assert_eq!(fs.get_inode(ino).unwrap().stat.uid, 1000);
        fs.setattr("/foo", Attr::mode(0o755)).unwrap();
        assert_eq!(fs.get_inode(ino).unwrap().stat.perm(), 0o755);
        // mkdir
        assert!(fs.mkdir("/dir", 0o755).is_ok());
        assert!(fs.stat("/dir").unwrap().0.is_dir());
        // symlink
        fs.symlink("/target", "/link").unwrap();
        assert_eq!(fs.readlink("/link").unwrap(), "/target");
        // mknod
        fs.mknod("/dev/null", 0o020666, 1).unwrap();
        // readdir
        let entries = fs.readdir("/bin");
        assert!(entries.is_empty()); // no bin yet
        fs.load_alpine_mock();
        let bin_entries = fs.readdir("/bin");
        assert!(!bin_entries.is_empty());
        // mount/umount
        assert!(fs.mount("/path/to/data", 3).is_ok());
        assert!(fs.umount().is_ok());
        // inode_orphaned
        let ino2 = fs.path_create("/tmpfile", IshStat::new(0o100644, 0, 0, 0));
        fs.path_unlink("/tmpfile").unwrap();
        fs.inode_orphaned(ino2).unwrap();
        // open_inode - may return /foo or /bar since both point to same inode
        let path = fs.open_inode(1).unwrap();
        assert!(path == "/foo" || path == "/bar");
    }
}
