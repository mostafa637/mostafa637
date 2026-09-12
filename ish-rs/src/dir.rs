//! `fs/dir.c` — directory iteration and getdents full port.

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct DirPos {
    pub index: usize,
    pub offset: u64,
}

impl DirPos {
    pub fn new(index: usize, offset: u64) -> Self { Self { index, offset } }
    pub fn next(&mut self) { self.index += 1; self.offset += 1; }
}

pub const DT_UNKNOWN: u8 = 0;
pub const DT_FIFO: u8 = 1;
pub const DT_CHR: u8 = 2;
pub const DT_DIR: u8 = 4;
pub const DT_BLK: u8 = 6;
pub const DT_REG: u8 = 8;
pub const DT_LNK: u8 = 10;
pub const DT_SOCK: u8 = 12;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Dirent {
    pub ino: u64,
    pub off: u64,
    pub reclen: u16,
    pub name: String,
    pub d_type: u8,
}

impl Dirent {
    pub fn new(ino: u64, off: u64, name: impl Into<String>, d_type: u8) -> Self {
        let name_str = name.into();
        let reclen = (19 + name_str.len() + 1) as u16;
        Self { ino, off, reclen, name: name_str, d_type }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Dirent64 {
    pub ino: u64,
    pub off: u64,
    pub reclen: u16,
    pub d_type: u8,
    pub name: String,
}

impl Dirent64 {
    pub fn new(ino: u64, off: u64, name: impl Into<String>, d_type: u8) -> Self {
        let name_str = name.into();
        let reclen = (24 + name_str.len() + 1) as u16;
        Self { ino, off, reclen, d_type, name: name_str }
    }
}

/// `struct dir_entry` from fs/fd.h
#[derive(Debug, Clone)]
pub struct DirEntry {
    pub inode: u64,
    pub name: String,
}

impl DirEntry {
    pub fn new(inode: u64, name: &str) -> Self { Self { inode, name: name.to_string() } }
}

/// Directory fd state, matching fd->offset + ops->telldir/seekdir
#[derive(Debug, Default)]
pub struct DirFd {
    pub offset: u64,
    pub dir_pos: Option<u64>, // index from telldir
    pub entries: Vec<DirEntry>,
}

impl DirFd {
    pub fn new(entries: Vec<DirEntry>) -> Self { Self { offset: 0, dir_pos: None, entries } }

    pub fn telldir(&self) -> u64 {
        if let Some(pos) = self.dir_pos { pos } else { self.offset }
    }

    pub fn seekdir(&mut self, off: u64) {
        self.offset = off;
        self.dir_pos = Some(off);
    }

    pub fn readdir(&mut self) -> Option<DirEntry> {
        let idx = self.telldir() as usize;
        if idx >= self.entries.len() { return None; }
        let entry = self.entries[idx].clone();
        self.seekdir((idx + 1) as u64);
        Some(entry)
    }
}

/// fill_dirent_32 matching C
pub fn fill_dirent_32(inode: u64, offset: u64, name: &str, type_: u8) -> (Vec<u8>, usize) {
    // struct linux_dirent_: inode(4) offset(4) reclen(2) name[] + null + type
    let reclen = 10 + name.len() + 2; // offsetof(name) = 10
    let mut buf = vec![0u8; reclen];
    buf[0..4].copy_from_slice(&(inode as u32).to_le_bytes());
    buf[4..8].copy_from_slice(&(offset as u32).to_le_bytes());
    buf[8..10].copy_from_slice(&(reclen as u16).to_le_bytes());
    buf[10..10+name.len()].copy_from_slice(name.as_bytes());
    buf[10+name.len()] = 0;
    buf[reclen-1] = type_;
    (buf, reclen)
}

/// fill_dirent_64 matching C
pub fn fill_dirent_64(inode: u64, offset: u64, name: &str, type_: u8) -> (Vec<u8>, usize) {
    // struct linux_dirent64_: inode(8) offset(8) reclen(2) type(1) name[]
    let reclen = 19 + name.len() + 1; // offsetof(name) = 19
    let mut buf = vec![0u8; reclen];
    buf[0..8].copy_from_slice(&inode.to_le_bytes());
    buf[8..16].copy_from_slice(&offset.to_le_bytes());
    buf[16..18].copy_from_slice(&(reclen as u16).to_le_bytes());
    buf[18] = type_;
    buf[19..19+name.len()].copy_from_slice(name.as_bytes());
    buf[19+name.len()] = 0;
    (buf, reclen)
}

/// sys_getdents_common matching C
pub fn sys_getdents_common<F>(dir_fd: &mut DirFd, count: usize, fill_dirent: F) -> Result<(Vec<u8>, usize), i32>
where F: Fn(u64, u64, &str, u8) -> (Vec<u8>, usize) {
    if dir_fd.entries.is_empty() { return Err(-20); } // ENOTDIR
    let orig_count = count;
    let mut remaining = count;
    let mut result = Vec::new();
    let mut ptr = dir_fd.telldir();

    loop {
        ptr = dir_fd.telldir();
        let entry = match dir_fd.readdir() {
            Some(e) => e,
            None => break,
        };
        let offset = dir_fd.telldir();
        let (dirent_data, reclen) = fill_dirent(entry.inode, offset, &entry.name, DT_UNKNOWN);
        if reclen > remaining { break; }
        result.extend_from_slice(&dirent_data);
        remaining -= reclen;
    }
    dir_fd.seekdir(ptr);
    Ok((result, orig_count - remaining))
}

pub fn sys_getdents(dir_fd: &mut DirFd, count: usize) -> Result<(Vec<u8>, usize), i32> {
    sys_getdents_common(dir_fd, count, fill_dirent_32)
}

pub fn sys_getdents64(dir_fd: &mut DirFd, count: usize) -> Result<(Vec<u8>, usize), i32> {
    sys_getdents_common(dir_fd, count, fill_dirent_64)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dir_pos_next() {
        let mut pos = DirPos::new(0, 0);
        pos.next();
        assert_eq!(pos.index, 1);
        assert_eq!(pos.offset, 1);
    }

    #[test]
    fn dirent_creation() {
        let d = Dirent::new(1, 0, "foo", DT_REG);
        assert_eq!(d.ino, 1);
        assert_eq!(d.name, "foo");
        assert_eq!(d.d_type, DT_REG);
        let d64 = Dirent64::new(2, 10, "bar", DT_DIR);
        assert_eq!(d64.ino, 2);
        assert_eq!(d64.d_type, DT_DIR);
    }

    #[test]
    fn fill_dirent_32_64() {
        let (buf32, len32) = fill_dirent_32(1, 2, "foo", DT_REG);
        assert_eq!(len32, 10 + 3 + 2);
        assert_eq!(buf32.len(), len32);
        let (buf64, len64) = fill_dirent_64(1, 2, "foo", DT_REG);
        assert_eq!(len64, 19 + 3 + 1);
        assert_eq!(buf64.len(), len64);
        assert_eq!(buf64[18], DT_REG);
    }

    #[test]
    fn dir_fd_telldir_seekdir_readdir() {
        let entries = vec![DirEntry::new(1, "a"), DirEntry::new(2, "b"), DirEntry::new(3, "c")];
        let mut fd = DirFd::new(entries);
        assert_eq!(fd.telldir(), 0);
        let e = fd.readdir().unwrap();
        assert_eq!(e.name, "a");
        assert_eq!(fd.telldir(), 1);
        fd.seekdir(0);
        assert_eq!(fd.telldir(), 0);
    }

    #[test]
    fn sys_getdents_common_test() {
        let entries = vec![DirEntry::new(1, "foo"), DirEntry::new(2, "bar")];
        let mut fd = DirFd::new(entries);
        let (data, written) = sys_getdents(&mut fd, 1024).unwrap();
        assert!(written > 0);
        assert!(!data.is_empty());
        // After reading all, next call should return 0 (at end)
        let (data_empty, written_empty) = sys_getdents(&mut fd, 1024).unwrap();
        assert_eq!(written_empty, 0);
        assert!(data_empty.is_empty());
        // Seek back to start and try 64
        fd.seekdir(0);
        let (data64, written64) = sys_getdents64(&mut fd, 1024).unwrap();
        assert!(written64 > 0);
        assert!(!data64.is_empty());
    }

    #[test]
    fn sys_getdents_small_buffer() {
        let entries = vec![DirEntry::new(1, "very_long_filename_here"), DirEntry::new(2, "short")];
        let mut fd = DirFd::new(entries);
        // Small buffer that fits none
        let (data, written) = sys_getdents(&mut fd, 5).unwrap();
        assert_eq!(written, 0);
        assert!(data.is_empty());
    }
}
