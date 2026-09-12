//! `fs/proc.c` — procfs constants and helpers.

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
}

impl ProcEntry {
    pub fn new(name: &str, entry_type: ProcEntryType, mode: u32) -> Self {
        Self { name: name.to_string(), entry_type, mode }
    }
    pub fn is_dir(&self) -> bool { matches!(self.entry_type, ProcEntryType::Dir | ProcEntryType::PidDir) }
    pub fn is_file(&self) -> bool { matches!(self.entry_type, ProcEntryType::File) }
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

/// Proc status fields, simplified
#[derive(Debug, Default)]
pub struct ProcStatus {
    pub pid: u32,
    pub ppid: u32,
    pub name: String,
    pub state: char,
}

impl ProcStatus {
    pub fn new(pid: u32, ppid: u32, name: &str, state: char) -> Self {
        Self { pid, ppid, name: name.to_string(), state }
    }
    pub fn format(&self) -> String {
        format!("Name:\t{}\nState:\t{} (state)\nTgid:\t{}\nPid:\t{}\nPPid:\t{}\n",
            self.name, self.state, self.pid, self.pid, self.ppid)
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
    }
}
