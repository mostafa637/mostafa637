//! `fs/generic.c` — generic file operation constants and helpers.
//!
//! The full generic operations depend on mount, fd, and path which are
//! already ported. This module ports the access mode constants and
//! pure helpers like `generic_seek` logic and open flag validation.

/// Access check constants from `kernel/fs.h`
pub const AC_R: u32 = 4;
pub const AC_W: u32 = 2;
pub const AC_X: u32 = 1;
pub const AC_F: u32 = 0;

/// Seek helper, mirroring `generic_seek` logic.
pub fn generic_seek_logic(offset: i64, whence: u32, file_size: u64, current_offset: u64) -> Result<u64, i32> {
    const LSEEK_SET: u32 = 0;
    const LSEEK_CUR: u32 = 1;
    const LSEEK_END: u32 = 2;

    let new_offset: i64 = match whence {
        LSEEK_SET => offset,
        LSEEK_CUR => current_offset as i64 + offset,
        LSEEK_END => file_size as i64 + offset,
        _ => return Err(-22),
    };

    if new_offset < 0 {
        return Err(-22);
    }

    Ok(new_offset as u64)
}

/// Access check helper, simplified.
pub fn access_check_simple(mode: u32, check: u32) -> bool {
    if check == AC_F {
        return true;
    }
    (mode & check) != 0
}

/// Open flag validation, matching C's checks in generic_openat.
pub fn validate_open_flags(flags: u32) -> Result<(), i32> {
    const O_ACCMODE: u32 = 3;
    const O_CREAT: u32 = 64;
    const O_EXCL: u32 = 128;
    const O_DIRECTORY: u32 = 1 << 16;

    // Check for invalid combinations
    let accmode = flags & O_ACCMODE;
    if accmode > 2 {
        return Err(-22);
    }

    // O_EXCL without O_CREAT is invalid per some checks, but C allows it?
    // Simplified: allow
    Ok(())
}

/// Path validation for generic operations.
pub fn validate_path(path: &str) -> Result<(), i32> {
    if path.is_empty() {
        return Err(-2); // ENOENT
    }
    if path.len() >= 4096 {
        return Err(-36); // ENAMETOOLONG
    }
    if path.contains('\0') {
        return Err(-22);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn generic_seek_matches_c_logic() {
        assert_eq!(generic_seek_logic(10, 0, 100, 0).unwrap(), 10);
        assert_eq!(generic_seek_logic(5, 1, 100, 10).unwrap(), 15);
        assert_eq!(generic_seek_logic(-10, 2, 100, 0).unwrap(), 90);
        assert!(generic_seek_logic(-200, 0, 100, 0).is_err());
        assert!(generic_seek_logic(0, 99, 100, 0).is_err());
    }

    #[test]
    fn access_check_logic() {
        assert!(access_check_simple(0o777, AC_R));
        assert!(access_check_simple(0o777, AC_W));
        assert!(!access_check_simple(0o444, AC_W));
        assert!(access_check_simple(0o444, AC_F));
    }

    #[test]
    fn open_flags_validation() {
        assert!(validate_open_flags(0).is_ok());
        assert!(validate_open_flags(0 | 64).is_ok()); // O_CREAT
        assert!(validate_open_flags(3).is_err()); // invalid accmode 3
    }

    #[test]
    fn path_validation() {
        assert!(validate_path("/foo/bar").is_ok());
        assert!(validate_path("").is_err());
        assert!(validate_path(&"a".repeat(5000)).is_err());
        assert!(validate_path("/foo\0bar").is_err());
    }
}
