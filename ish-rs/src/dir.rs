//! `fs/dir.c` — directory iteration helpers.

/// Directory iteration state, simplified.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct DirPos {
    pub index: usize,
    pub offset: u64,
}

impl DirPos {
    pub fn new(index: usize, offset: u64) -> Self {
        Self { index, offset }
    }

    pub fn next(&mut self) {
        self.index += 1;
        self.offset += 1;
    }
}

/// `getdents` entry types from `dirent.h`
pub const DT_UNKNOWN: u8 = 0;
pub const DT_FIFO: u8 = 1;
pub const DT_CHR: u8 = 2;
pub const DT_DIR: u8 = 4;
pub const DT_BLK: u8 = 6;
pub const DT_REG: u8 = 8;
pub const DT_LNK: u8 = 10;
pub const DT_SOCK: u8 = 12;

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
    fn dirent_type_constants() {
        assert_eq!(DT_DIR, 4);
        assert_eq!(DT_REG, 8);
    }
}
