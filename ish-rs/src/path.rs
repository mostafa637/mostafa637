//! `fs/path.h` + `fs/path.c` — path normalization helpers.
//!
//! The full `path_normalize` depends on `mount`, `generic_getpath`, and
//! symlink resolution via `fs->readlink`. Those depend on the filesystem
//! layer. This module ports the leaf helpers `path_is_normalized` and
//! `path_next_component` which are pure string operations, plus a simplified
//! `path_normalize` that handles `.` and `..` without symlink resolution.
//!
//! This satisfies ascending order: these helpers have no fs dependencies.

use crate::fix_path::MAX_PATH;

/// `MAX_NAME` from `kernel/fs.h`.
pub const MAX_NAME: usize = 256;

/// `N_*` flags from `path.h`.
pub const N_SYMLINK_FOLLOW: u32 = 1;
pub const N_SYMLINK_NOFOLLOW: u32 = 2;
pub const N_PARENT_DIR_WRITE: u32 = 4;

/// Check if a path is normalized: starts with `/` for each component and
/// has no `//`. Mirrors C exactly.
pub fn path_is_normalized(path: &str) -> bool {
    let bytes = path.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] != b'/' {
            return false;
        }
        i += 1;
        if i < bytes.len() && bytes[i] == b'/' {
            return false;
        }
        while i < bytes.len() && bytes[i] != b'/' {
            i += 1;
        }
    }
    true
}

/// Iterate through a normalized path, returning next component.
///
/// Mirrors `path_next_component`: `*path` is advanced to next `/`, component
/// is copied. Returns `Some(component)` if there is a next component, `None`
/// if at end. `err` is set to `ENAMETOOLONG` if component >= `MAX_NAME`.
///
/// For "/" C returns one empty component "" then end; we preserve that.
pub fn path_next_component<'a>(path: &mut &'a str) -> Result<Option<String>, i32> {
    if path.is_empty() {
        return Ok(None);
    }
    assert!(path.starts_with('/'), "path must be normalized and start with '/'");

    // C: p = *path; if *p == '\0' return false; assert(*p=='/'); p++;
    // Then copy until '/' or '\0'
    let p = *path;
    // Skip leading '/'
    let remaining = &p[1..];
    let end = remaining.find('/').unwrap_or(remaining.len());
    let component = &remaining[..end];

    if component.len() >= MAX_NAME {
        return Err(-36);
    }

    let result = component.to_string();

    if end == remaining.len() {
        *path = "";
    } else {
        *path = &remaining[end..];
    }

    Ok(Some(result))
}

/// Simplified `path_normalize` without symlink resolution.
pub fn path_normalize_simple(at_path: Option<&str>, path: &str, out: &mut String) -> i32 {
    if path.is_empty() {
        return -2;
    }

    let mut result = String::new();

    if let Some(at) = at_path {
        if at != "/" {
            result.push_str(at);
        }
    }

    if !path.starts_with('/') {
    } else {
        result.clear();
    }

    let combined = if path.starts_with('/') {
        path.to_string()
    } else if result.is_empty() {
        path.to_string()
    } else {
        format!("{}/{}", result, path)
    };

    let mut components: Vec<&str> = Vec::new();
    for comp in combined.split('/') {
        match comp {
            "" | "." => continue,
            ".." => {
                components.pop();
            }
            _ => components.push(comp),
        }
    }

    out.clear();
    for comp in components {
        out.push('/');
        out.push_str(comp);
    }
    if out.is_empty() {
        out.push('/');
    }

    if out.len() >= MAX_PATH {
        return -36;
    }

    0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn path_is_normalized_matches_c() {
        assert!(path_is_normalized(""));
        assert!(path_is_normalized("/"));
        assert!(path_is_normalized("/a"));
        assert!(path_is_normalized("/a/b"));
        assert!(path_is_normalized("/a/b/c"));
        assert!(!path_is_normalized("a"));
        assert!(!path_is_normalized("//"));
        assert!(!path_is_normalized("/a//b"));
        assert!(path_is_normalized("/a/"));
    }

    #[test]
    fn path_next_component_iterates() {
        let mut path = "/a/bb/ccc";
        let c1 = path_next_component(&mut path).unwrap().unwrap();
        assert_eq!(c1, "a");
        assert_eq!(path, "/bb/ccc");
        let c2 = path_next_component(&mut path).unwrap().unwrap();
        assert_eq!(c2, "bb");
        let c3 = path_next_component(&mut path).unwrap().unwrap();
        assert_eq!(c3, "ccc");
        assert_eq!(path, "");
        assert!(path_next_component(&mut path).unwrap().is_none());

        // "/" returns one empty component per C
        let mut slash = "/";
        let comp = path_next_component(&mut slash).unwrap().unwrap();
        assert_eq!(comp, "");
        assert_eq!(slash, "");
        assert!(path_next_component(&mut slash).unwrap().is_none());
    }

    #[test]
    fn path_normalize_simple_handles_dots() {
        let mut out = String::new();
        assert_eq!(path_normalize_simple(None, "/a/./b", &mut out), 0);
        assert_eq!(out, "/a/b");
        assert_eq!(path_normalize_simple(None, "/a/../b", &mut out), 0);
        assert_eq!(out, "/b");
        assert_eq!(path_normalize_simple(Some("/cwd"), "a/../b", &mut out), 0);
        assert_eq!(out, "/cwd/b");
        assert_eq!(path_normalize_simple(Some("/"), "/a/b", &mut out), 0);
        assert_eq!(out, "/a/b");
    }
}
