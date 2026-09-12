//! `fs/dir.c` — directory iteration helpers and getdents.

/// Directory iteration state
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

/// `struct dirent_` guest ABI (simplified)
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
        let reclen = (19 + name_str.len() + 1) as u16; // approximate
        Self { ino, off, reclen, name: name_str, d_type }
    }
}

/// `struct dirent64_` guest ABI
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
}
