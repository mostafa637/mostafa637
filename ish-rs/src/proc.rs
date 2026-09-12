//! `fs/proc.h` — proc filesystem entry types and constants.

/// Proc entry type.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProcEntryType {
    File,
    Dir,
    Symlink,
}

/// Proc file mode bits (simplified).
pub const PROC_MODE_FILE: u32 = 0o100444;
pub const PROC_MODE_DIR: u32 = 0o040555;
pub const PROC_MODE_SYMLINK: u32 = 0o120777;

/// Proc entry descriptor.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ProcDirEntry {
    pub name: String,
    pub mode: u32,
}

impl ProcDirEntry {
    pub fn new(name: impl Into<String>, mode: u32) -> Self {
        Self {
            name: name.into(),
            mode,
        }
    }

    pub fn is_dir(&self) -> bool {
        (self.mode & 0o170000) == 0o040000
    }

    pub fn is_file(&self) -> bool {
        (self.mode & 0o170000) == 0o100000
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn proc_entry_type_checks() {
        let dir = ProcDirEntry::new("test", PROC_MODE_DIR);
        assert!(dir.is_dir());
        assert!(!dir.is_file());
        let file = ProcDirEntry::new("test", PROC_MODE_FILE);
        assert!(file.is_file());
    }
}
