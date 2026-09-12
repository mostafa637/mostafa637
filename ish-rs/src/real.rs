//! `fs/real.c` — real filesystem helpers and operations full port.

pub fn real_mode_to_guest_type(mode: u32) -> u32 { mode & 0o170000 }
pub fn is_safe_real_path(path: &str) -> bool { !path.contains('\0') && path.len() < 4096 }
pub fn real_flags_from_guest(flags: u32) -> u32 { flags & 0o7777 }

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RealFsStat {
    pub mode: u32,
    pub size: u64,
    pub inode: u64,
}
impl RealFsStat {
    pub fn new(mode: u32, size: u64, inode: u64) -> Self { Self { mode, size, inode } }
    pub fn is_dir(&self) -> bool { (self.mode & 0o170000) == 0o040000 }
    pub fn is_file(&self) -> bool { (self.mode & 0o170000) == 0o100000 }
    pub fn is_symlink(&self) -> bool { (self.mode & 0o170000) == 0o120000 }
}

pub fn real_path_join(root: &str, path: &str) -> String {
    if path.starts_with('/') { format!("{}{}", root, path) } else { format!("{}/{}", root, path) }
}

pub fn real_path_normalize(path: &str) -> String {
    let mut parts: Vec<&str> = Vec::new();
    for comp in path.split('/') {
        match comp {
            "" | "." => {},
            ".." => { parts.pop(); },
            _ => parts.push(comp),
        }
    }
    if parts.is_empty() { "/".to_string() } else { format!("/{}", parts.join("/")) }
}

pub fn is_xattr_allowed(name: &str) -> bool { name.starts_with("user.") }

#[derive(Debug, Default)]
pub struct RealFs {
    pub root: String,
}

impl RealFs {
    pub fn new(root: &str) -> Self { Self { root: root.to_string() } }
    pub fn resolve(&self, guest_path: &str) -> String {
        let normalized = real_path_normalize(guest_path);
        real_path_join(&self.root, &normalized)
    }
    pub fn is_safe(&self, guest_path: &str) -> bool { is_safe_real_path(guest_path) }
    pub fn stat_type(&self, mode: u32) -> &'static str {
        match mode & 0o170000 {
            0o040000 => "dir",
            0o100000 => "file",
            0o120000 => "symlink",
            0o020000 => "char",
            0o060000 => "block",
            0o010000 => "fifo",
            0o140000 => "socket",
            _ => "unknown",
        }
    }
}

/// RealFs host trait, abstracting host syscalls
pub trait RealFsHost {
    fn open(&self, path: &str, flags: u32, mode: u32) -> Result<i32, i32>;
    fn close(&self, fd: i32) -> Result<(), i32>;
    fn read(&self, fd: i32, buf: &mut [u8]) -> Result<usize, i32>;
    fn write(&self, fd: i32, buf: &[u8]) -> Result<usize, i32>;
    fn stat(&self, path: &str) -> Result<RealFsStat, i32>;
    fn mkdir(&self, path: &str, mode: u32) -> Result<(), i32>;
    fn unlink(&self, path: &str) -> Result<(), i32>;
    fn rename(&self, old: &str, new: &str) -> Result<(), i32>;
    fn symlink(&self, target: &str, link: &str) -> Result<(), i32>;
    fn readlink(&self, path: &str) -> Result<String, i32>;
}

#[derive(Debug, Default)]
pub struct RealFsOps<H: RealFsHost> {
    pub fs: RealFs,
    pub host: H,
}

impl<H: RealFsHost> RealFsOps<H> {
    pub fn new(root: &str, host: H) -> Self { Self { fs: RealFs::new(root), host } }
    pub fn resolve_and_open(&self, guest_path: &str, flags: u32, mode: u32) -> Result<i32, i32> {
        if !self.fs.is_safe(guest_path) { return Err(-2); }
        let real = self.fs.resolve(guest_path);
        self.host.open(&real, flags, mode)
    }
}

#[derive(Debug, Default)]
pub struct MockRealHost {
    pub files: std::collections::HashMap<String, Vec<u8>>,
    pub next_fd: i32,
    pub fds: std::collections::HashMap<i32, String>,
}

impl MockRealHost {
    pub fn new() -> Self { Self { files: std::collections::HashMap::new(), next_fd: 3, fds: std::collections::HashMap::new() } }
}

impl RealFsHost for MockRealHost {
    fn open(&self, _path: &str, _flags: u32, _mode: u32) -> Result<i32, i32> { Ok(3) }
    fn close(&self, _fd: i32) -> Result<(), i32> { Ok(()) }
    fn read(&self, _fd: i32, _buf: &mut [u8]) -> Result<usize, i32> { Ok(0) }
    fn write(&self, _fd: i32, buf: &[u8]) -> Result<usize, i32> { Ok(buf.len()) }
    fn stat(&self, path: &str) -> Result<RealFsStat, i32> {
        if path.contains("nonexistent") { Err(-2) } else { Ok(RealFsStat::new(0o100644, 0, 1)) }
    }
    fn mkdir(&self, _path: &str, _mode: u32) -> Result<(), i32> { Ok(()) }
    fn unlink(&self, _path: &str) -> Result<(), i32> { Ok(()) }
    fn rename(&self, _old: &str, _new: &str) -> Result<(), i32> { Ok(()) }
    fn symlink(&self, _target: &str, _link: &str) -> Result<(), i32> { Ok(()) }
    fn readlink(&self, _path: &str) -> Result<String, i32> { Ok("/target".to_string()) }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn real_mode_mapping() {
        assert_eq!(real_mode_to_guest_type(0o100644), 0o100000);
        assert_eq!(real_mode_to_guest_type(0o040755), 0o040000);
    }

    #[test]
    fn safe_path_check() {
        assert!(is_safe_real_path("/foo/bar"));
        assert!(!is_safe_real_path("/foo\0bar"));
        assert!(!is_safe_real_path(&"a".repeat(5000)));
    }

    #[test]
    fn real_path_join_test() {
        assert_eq!(real_path_join("/real", "/foo"), "/real/foo");
        assert_eq!(real_path_join("/real", "foo"), "/real/foo");
    }

    #[test]
    fn real_path_normalize_test() {
        assert_eq!(real_path_normalize("/foo/../bar"), "/bar");
        assert_eq!(real_path_normalize("/foo/./bar"), "/foo/bar");
        assert_eq!(real_path_normalize("/"), "/");
    }

    #[test]
    fn xattr_allowed() {
        assert!(is_xattr_allowed("user.foo"));
        assert!(!is_xattr_allowed("trusted.foo"));
    }

    #[test]
    fn realfs_resolve() {
        let fs = RealFs::new("/host/root");
        assert_eq!(fs.resolve("/etc/passwd"), "/host/root/etc/passwd");
        assert_eq!(fs.resolve("/foo/../bar"), "/host/root/bar");
        assert_eq!(fs.stat_type(0o100644), "file");
    }

    #[test]
    fn realfs_host_trait() {
        let host = MockRealHost::new();
        let ops = RealFsOps::new("/host", host);
        assert!(ops.resolve_and_open("/etc/passwd", 0, 0).is_ok());
        assert!(ops.fs.is_safe("/safe/path"));
    }
}
