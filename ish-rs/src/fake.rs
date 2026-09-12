//! `fs/fake.c` — fake filesystem constants and inode helpers.

/// Fakefs inode type
pub const FAKEFS_MAGIC: u32 = 0x66616b65; // "fake"

/// Ish stat, matching C `struct ish_stat`
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct IshStat {
    pub mode: u32,
    pub uid: u32,
    pub gid: u32,
    pub rdev: u32,
}

impl IshStat {
    pub fn new(mode: u32, uid: u32, gid: u32, rdev: u32) -> Self {
        Self { mode, uid, gid, rdev }
    }

    pub fn is_dir(&self) -> bool { (self.mode & 0o170000) == 0o040000 }
    pub fn is_file(&self) -> bool { (self.mode & 0o170000) == 0o100000 }
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
    }
}
