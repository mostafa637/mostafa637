//! `fs/real.c` — real filesystem helpers.

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
}

/// Realfs path resolution helper
pub fn real_path_join(root: &str, path: &str) -> String {
    if path.starts_with('/') {
        format!("{}{}", root, path)
    } else {
        format!("{}/{}", root, path)
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
}
