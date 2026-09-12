//! `kernel/exec.c` — exec argument handling and ELF loading.

pub const ARGV_MAX: usize = 32 * 4096;
pub const ENOEXEC: i32 = -8;
pub const E2BIG: i32 = -7;

pub fn align_stack(sp: u32) -> u32 { sp & !0xf }

pub fn check_argv_size(size: usize) -> Result<(), i32> {
    if size > ARGV_MAX { Err(E2BIG) } else { Ok(()) }
}

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
        self.args.iter().map(|s| s.len() + 1).sum::<usize>() + (self.count + 1) * 4
    }
    pub fn is_too_large(&self) -> bool { self.size() > ARGV_MAX }
}

pub fn is_valid_elf_header(magic: &[u8; 4], elf_type: u16, bitness: u8, endian: u8, machine: u16) -> bool {
    magic == b"\x7fELF" && (elf_type == 2 || elf_type == 3) && bitness == 1 && endian == 1 && machine == 3
}

/// Stack layout for exec, matching C's stack setup
#[derive(Debug, Clone, Default)]
pub struct ExecStack {
    pub sp: u32,
    pub argc: u32,
    pub argv: Vec<u32>,
    pub envp: Vec<u32>,
}

impl ExecStack {
    pub fn new(sp: u32) -> Self { Self { sp: align_stack(sp), argc: 0, argv: Vec::new(), envp: Vec::new() } }

    pub fn push_string(&mut self, s: &str) -> u32 {
        let len = s.len() + 1;
        self.sp -= len as u32;
        self.sp = align_stack(self.sp);
        // In real C, it would copy string to user memory at sp
        self.sp
    }

    pub fn push_args(&mut self, args: &ExecArgs) -> Result<(), i32> {
        check_argv_size(args.size())?;
        self.argc = args.count as u32;
        // Simulate pushing args onto stack
        for arg in args.args.iter().rev() {
            let addr = self.push_string(arg);
            self.argv.push(addr);
        }
        self.argv.reverse();
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

    #[test]
    fn exec_args_size() {
        let args = ExecArgs::new(vec!["/bin/sh".to_string(), "-c".to_string(), "echo hi".to_string()]);
        assert_eq!(args.count, 3);
        assert!(!args.is_too_large());
    }

    #[test]
    fn elf_header_validation() {
        assert!(is_valid_elf_header(b"\x7fELF", 2, 1, 1, 3));
        assert!(!is_valid_elf_header(b"\x00ELF", 2, 1, 1, 3));
    }

    #[test]
    fn exec_stack_push() {
        let mut stack = ExecStack::new(0x10000);
        let addr = stack.push_string("hello");
        assert!(addr < 0x10000);
        assert_eq!(stack.sp & 0xf, 0); // aligned
        let args = ExecArgs::new(vec!["/bin/sh".to_string()]);
        assert!(stack.push_args(&args).is_ok());
        assert_eq!(stack.argc, 1);
    }
}
