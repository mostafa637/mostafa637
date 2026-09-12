//! `kernel/exec.c` — exec argument handling constants and helpers.

/// `ARGV_MAX` — max size of argv+envp
pub const ARGV_MAX: usize = 32 * 4096;

/// Exec error codes
pub const ENOEXEC: i32 = -8;
pub const E2BIG: i32 = -7;

/// Stack alignment for exec, matching C's `align_stack`.
pub fn align_stack(sp: u32) -> u32 {
    sp & !0xf
}

/// Check if a string length is within ARGV_MAX.
pub fn check_argv_size(size: usize) -> Result<(), i32> {
    if size > ARGV_MAX {
        Err(E2BIG)
    } else {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn align_stack_matches_c() {
        assert_eq!(align_stack(0x1234), 0x1230);
        assert_eq!(align_stack(0x1000), 0x1000);
    }

    #[test]
    fn argv_max_check() {
        assert!(check_argv_size(100).is_ok());
        assert!(check_argv_size(ARGV_MAX + 1).is_err());
    }
}
