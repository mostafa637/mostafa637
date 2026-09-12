//! `fs/fix_path.h` — path fixing utilities.
//!
//! The C implementation is intentionally trivial: it strips a leading `/`
//! or returns `"."` for an empty string. This is used to turn an absolute
//! guest path into a relative host path for `openat(root_fd, ...)`.

/// Maximum path length, from `kernel/fs.h`.
pub const MAX_PATH: usize = 4096;
pub const MAX_NAME: usize = 256;

/// `fix_path` — mirrors C's inline function.
///
/// * `""` -> `"."`
/// * `"/foo"` -> `"foo"`
/// * `"/"` -> `""` (C returns pointer to NUL after the slash)
/// * `"foo"` -> `"foo"`
pub fn fix_path(path: &str) -> &str {
    if path.is_empty() {
        return ".";
    }
    if path.starts_with('/') {
        // SAFETY: stripping one byte from a valid UTF-8 string that starts
        // with '/' (ASCII) is still valid UTF-8.
        &path[1..]
    } else {
        path
    }
}

/// Byte-slice version, for `&[u8]` paths that may not be UTF-8 (iSH uses
/// `const char *` which is bytes). Returns normalized bytes.
pub fn fix_path_bytes(path: &[u8]) -> &[u8] {
    if path.is_empty() {
        return b".";
    }
    if path[0] == b'/' {
        &path[1..]
    } else {
        path
    }
}

/// Check if a path is safe (no NUL bytes, within length limits).
pub fn is_path_safe(path: &[u8]) -> bool {
    !path.contains(&0) && path.len() <= MAX_PATH
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fix_path_matches_c() {
        assert_eq!(fix_path(""), ".");
        assert_eq!(fix_path("/"), "");
        assert_eq!(fix_path("/foo"), "foo");
        assert_eq!(fix_path("/foo/bar"), "foo/bar");
        assert_eq!(fix_path("foo"), "foo");
        assert_eq!(fix_path("foo/bar"), "foo/bar");
        assert_eq!(fix_path("//double"), "/double"); // only first slash stripped
    }

    #[test]
    fn fix_path_bytes_matches_c() {
        assert_eq!(fix_path_bytes(b""), b".");
        assert_eq!(fix_path_bytes(b"/"), b"");
        assert_eq!(fix_path_bytes(b"/foo"), b"foo");
        assert_eq!(fix_path_bytes(b"foo"), b"foo");
    }

    #[test]
    fn is_path_safe_checks() {
        assert!(is_path_safe(b"/safe/path"));
        assert!(!is_path_safe(b"/unsafe/\0/path"));
    }
}
