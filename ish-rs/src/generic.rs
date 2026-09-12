//! `fs/generic.c` — generic file operation constants and helpers.
//!
//! The full generic operations depend on mount, fd, and path which are
//! already ported as leaf modules. This module ports the access mode
//! constants and pure helpers like `generic_seek` logic.

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
        _ => return Err(-22), // EINVAL
    };

    if new_offset < 0 {
        return Err(-22);
    }

    Ok(new_offset as u64)
}

/// Access check helper, simplified.
pub fn access_check(mode: u32, check: u32) -> bool {
    // Simplified: if file mode has required bits
    if check == AC_F {
        return true;
    }
    (mode & check) != 0
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
        assert!(access_check(0o777, AC_R));
        assert!(access_check(0o777, AC_W));
        assert!(!access_check(0o444, AC_W));
        assert!(access_check(0o444, AC_F));
    }
}
