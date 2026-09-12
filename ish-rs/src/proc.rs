//! `fs/proc.h` + `fs/proc.c` — proc filesystem entries and helpers.

pub const PROC_MODE_FILE: u32 = 0o100444;
pub const PROC_MODE_DIR: u32 = 0o040555;
pub const PROC_MODE_SYMLINK: u32 = 0o120777;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProcEntryType {
    File,
    Dir,
    Symlink,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ProcDirEntry {
    pub name: String,
    pub mode: u32,
}

impl ProcDirEntry {
    pub fn new(name: impl Into<String>, mode: u32) -> Self {
        Self { name: name.into(), mode }
    }
    pub fn is_dir(&self) -> bool { (self.mode & 0o170000) == 0o040000 }
    pub fn is_file(&self) -> bool { (self.mode & 0o170000) == 0o100000 }
    pub fn is_symlink(&self) -> bool { (self.mode & 0o170000) == 0o120000 }
}

/// Proc data buffer, matching C `struct proc_data`
#[derive(Debug, Clone, Default)]
pub struct ProcData {
    pub data: Vec<u8>,
}

impl ProcData {
    pub fn new() -> Self { Self { data: Vec::new() } }
    pub fn from_string(s: &str) -> Self { Self { data: s.as_bytes().to_vec() } }
    pub fn as_string(&self) -> String { String::from_utf8_lossy(&self.data).to_string() }
    pub fn len(&self) -> usize { self.data.len() }
    pub fn is_empty(&self) -> bool { self.data.is_empty() }

    pub fn write(&mut self, offset: usize, buf: &[u8]) -> usize {
        if offset > self.data.len() {
            self.data.resize(offset, 0);
        }
        let needed = offset + buf.len();
        if needed > self.data.len() {
            self.data.resize(needed, 0);
        }
        self.data[offset..offset + buf.len()].copy_from_slice(buf);
        buf.len()
    }

    pub fn read(&self, offset: usize, buf: &mut [u8]) -> usize {
        if offset >= self.data.len() {
            return 0;
        }
        let available = self.data.len() - offset;
        let to_read = available.min(buf.len());
        buf[..to_read].copy_from_slice(&self.data[offset..offset + to_read]);
        to_read
    }
}

/// Common proc entries
pub fn proc_root_entries() -> Vec<ProcDirEntry> {
    vec![
        ProcDirEntry::new("self", PROC_MODE_SYMLINK),
        ProcDirEntry::new("cpuinfo", PROC_MODE_FILE),
        ProcDirEntry::new("meminfo", PROC_MODE_FILE),
        ProcDirEntry::new("version", PROC_MODE_FILE),
        ProcDirEntry::new("uptime", PROC_MODE_FILE),
    ]
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
        let link = ProcDirEntry::new("self", PROC_MODE_SYMLINK);
        assert!(link.is_symlink());
    }

    #[test]
    fn proc_data_read_write() {
        let mut data = ProcData::from_string("hello world");
        assert_eq!(data.len(), 11);
        let mut buf = [0u8; 5];
        assert_eq!(data.read(0, &mut buf), 5);
        assert_eq!(&buf, b"hello");
        assert_eq!(data.write(6, b"Rust"), 4);
        assert_eq!(data.as_string(), "hello Rustd");
    }

    #[test]
    fn proc_root_entries_list() {
        let entries = proc_root_entries();
        assert!(entries.iter().any(|e| e.name == "self"));
        assert!(entries.iter().any(|e| e.name == "cpuinfo"));
    }
}
