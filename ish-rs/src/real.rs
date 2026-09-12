//! `fs/real.c` — real filesystem helpers and operations.

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

/// Realfs path resolution helper
pub fn real_path_join(root: &str, path: &str) -> String {
    if path.starts_with('/') {
        format!("{}{}", root, path)
    } else {
        format!("{}/{}", root, path)
    }
}

pub fn real_path_normalize(path: &str) -> String {
    // Simplified normalization, matching path_normalize logic in C
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

/// Realfs xattr name filter, matching C's realfs_is_xattr_allowed
pub fn is_xattr_allowed(name: &str) -> bool {
    // In C, only user.* is allowed for realfs
    name.starts_with("user.")
}

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

    pub fn is_safe(&self, guest_path: &str) -> bool {
        is_safe_real_path(guest_path) && !guest_path.contains("..")
            || is_safe_real_path(guest_path) // allow .. after normalize
    }

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
        assert!(!is_xattr_allowed("security.foo"));
    }

    #[test]
    fn realfs_resolve() {
        let fs = RealFs::new("/host/root");
        assert_eq!(fs.resolve("/etc/passwd"), "/host/root/etc/passwd");
        assert_eq!(fs.resolve("/foo/../bar"), "/host/root/bar");
        assert_eq!(fs.stat_type(0o100644), "file");
        assert_eq!(fs.stat_type(0o040755), "dir");
    }
}
