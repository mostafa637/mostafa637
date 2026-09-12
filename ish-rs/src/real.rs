//! `fs/real.c` — real filesystem constants and path helpers.

/// Realfs file type mapping helpers.
pub fn real_mode_to_guest_type(mode: u32) -> u32 {
    // S_IFMT mask
    mode & 0o170000
}

/// Check if path is safe for realfs (no null bytes, etc.)
pub fn is_safe_real_path(path: &str) -> bool {
    !path.contains('\0') && path.len() < 4096
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
}
