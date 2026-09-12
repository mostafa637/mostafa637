//! `kernel/exec.c` — exec argument handling and ELF loading helpers.

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

/// Exec args, matching C `struct exec_args`
#[derive(Debug, Clone, Default)]
pub struct ExecArgs {
    pub count: usize,
    pub args: Vec<String>,
}

impl ExecArgs {
    pub fn new(args: Vec<String>) -> Self {
        let count = args.len();
        Self { count, args }
    }

    pub fn size(&self) -> usize {
        // Rough size: sum of string lengths + null terminators + pointers
        self.args.iter().map(|s| s.len() + 1).sum::<usize>() + (self.count + 1) * 4
    }

    pub fn is_too_large(&self) -> bool {
        self.size() > ARGV_MAX
    }
}

/// ELF header validation, matching C `read_header` checks
pub fn is_valid_elf_header(magic: &[u8; 4], elf_type: u16, bitness: u8, endian: u8, machine: u16) -> bool {
    magic == b"\x7fELF" && (elf_type == 2 || elf_type == 3) && bitness == 1 && endian == 1 && machine == 3
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

    #[test]
    fn exec_args_size() {
        let args = ExecArgs::new(vec!["/bin/sh".to_string(), "-c".to_string(), "echo hi".to_string()]);
        assert_eq!(args.count, 3);
        assert!(!args.is_too_large());
        assert!(args.size() < ARGV_MAX);
    }

    #[test]
    fn elf_header_validation() {
        assert!(is_valid_elf_header(b"\x7fELF", 2, 1, 1, 3));
        assert!(!is_valid_elf_header(b"\x00ELF", 2, 1, 1, 3));
        assert!(!is_valid_elf_header(b"\x7fELF", 1, 1, 1, 3)); // not EXEC or DYN
    }
}
