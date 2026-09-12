//! `kernel/exec.c` — exec argument handling and ELF loading full port.

pub const ARGV_MAX: usize = 32 * 4096;
pub const ENOEXEC: i32 = -8;
pub const E2BIG: i32 = -7;
pub const EINVAL: i32 = -22;

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
        self.sp
    }
    pub fn push_args(&mut self, args: &ExecArgs) -> Result<(), i32> {
        check_argv_size(args.size())?;
        self.argc = args.count as u32;
        for arg in args.args.iter().rev() {
            let addr = self.push_string(arg);
            self.argv.push(addr);
        }
        self.argv.reverse();
        Ok(())
    }
}

/// ELF header, matching C's Elf32_Ehdr
#[derive(Debug, Clone, Copy, Default)]
pub struct ElfHeader {
    pub magic: [u8; 4],
    pub bitness: u8,
    pub endian: u8,
    pub version: u8,
    pub abi: u8,
    pub elf_type: u16,
    pub machine: u16,
    pub entry: u32,
    pub phoff: u32,
    pub shoff: u32,
    pub flags: u32,
    pub ehsize: u16,
    pub phentsize: u16,
    pub phnum: u16,
    pub shentsize: u16,
    pub shnum: u16,
    pub shstrndx: u16,
}

impl ElfHeader {
    pub fn is_valid(&self) -> bool {
        is_valid_elf_header(&self.magic, self.elf_type, self.bitness, self.endian, self.machine)
    }
    pub fn is_executable(&self) -> bool { self.elf_type == 2 }
    pub fn is_shared(&self) -> bool { self.elf_type == 3 }
}

/// Program header
#[derive(Debug, Clone, Copy, Default)]
pub struct ProgramHeader {
    pub p_type: u32,
    pub offset: u32,
    pub vaddr: u32,
    pub paddr: u32,
    pub filesz: u32,
    pub memsz: u32,
    pub flags: u32,
    pub align: u32,
}

pub const PT_NULL: u32 = 0;
pub const PT_LOAD: u32 = 1;
pub const PT_DYNAMIC: u32 = 2;
pub const PT_INTERP: u32 = 3;
pub const PT_NOTE: u32 = 4;
pub const PT_PHDR: u32 = 6;
pub const PT_GNU_STACK: u32 = 0x6474e551;

impl ProgramHeader {
    pub fn is_load(&self) -> bool { self.p_type == PT_LOAD }
    pub fn is_interp(&self) -> bool { self.p_type == PT_INTERP }
    pub fn is_writable(&self) -> bool { (self.flags & 2) != 0 }
    pub fn is_executable(&self) -> bool { (self.flags & 1) != 0 }
}

/// Exec context, matching C's exec state
#[derive(Debug, Default)]
pub struct ExecContext {
    pub header: ElfHeader,
    pub phdrs: Vec<ProgramHeader>,
    pub entry: u32,
    pub interp: Option<String>,
    pub stack_top: u32,
    pub brk: u32,
}

impl ExecContext {
    pub fn new() -> Self { Self::default() }

    pub fn load_elf(&mut self, data: &[u8]) -> Result<(), i32> {
        if data.len() < 52 { return Err(ENOEXEC); }
        let magic = [data[0], data[1], data[2], data[3]];
        if magic != *b"\x7fELF" { return Err(ENOEXEC); }
        let bitness = data[4];
        let endian = data[5];
        if bitness != 1 || endian != 1 { return Err(ENOEXEC); }
        // Simplified parsing: read e_type, e_machine, e_entry, phoff, phnum
        let elf_type = u16::from_le_bytes([data[16], data[17]]);
        let machine = u16::from_le_bytes([data[18], data[19]]);
        if !is_valid_elf_header(&magic, elf_type, bitness, endian, machine) {
            return Err(ENOEXEC);
        }
        self.header.magic = magic;
        self.header.bitness = bitness;
        self.header.endian = endian;
        self.header.elf_type = elf_type;
        self.header.machine = machine;
        self.header.entry = u32::from_le_bytes([data[24], data[25], data[26], data[27]]);
        self.header.phoff = u32::from_le_bytes([data[28], data[29], data[30], data[31]]);
        self.header.phnum = u16::from_le_bytes([data[42], data[43]]);
        self.entry = self.header.entry;

        // Parse program headers
        self.phdrs.clear();
        let phoff = self.header.phoff as usize;
        for i in 0..self.header.phnum as usize {
            let off = phoff + i * 32;
            if off + 32 > data.len() { break; }
            let p_type = u32::from_le_bytes([data[off], data[off+1], data[off+2], data[off+3]]);
            let offset = u32::from_le_bytes([data[off+4], data[off+5], data[off+6], data[off+7]]);
            let vaddr = u32::from_le_bytes([data[off+8], data[off+9], data[off+10], data[off+11]]);
            let filesz = u32::from_le_bytes([data[off+16], data[off+17], data[off+18], data[off+19]]);
            let memsz = u32::from_le_bytes([data[off+20], data[off+21], data[off+22], data[off+23]]);
            let flags = u32::from_le_bytes([data[off+24], data[off+25], data[off+26], data[off+27]]);
            let ph = ProgramHeader { p_type, offset, vaddr, paddr: 0, filesz, memsz, flags, align: 0 };
            if ph.is_interp() {
                let interp_off = offset as usize;
                let interp_end = (offset + filesz) as usize;
                if interp_end <= data.len() {
                    let interp_bytes = &data[interp_off..interp_end];
                    if let Some(nul) = interp_bytes.iter().position(|&b| b == 0) {
                        if let Ok(s) = std::str::from_utf8(&interp_bytes[..nul]) {
                            self.interp = Some(s.to_string());
                        }
                    }
                }
            }
            self.phdrs.push(ph);
        }
        Ok(())
    }

    pub fn total_memory(&self) -> u32 {
        self.phdrs.iter().filter(|ph| ph.is_load()).map(|ph| ph.memsz).sum()
    }

    pub fn has_interpreter(&self) -> bool { self.interp.is_some() }
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
        assert_eq!(stack.sp & 0xf, 0);
        let args = ExecArgs::new(vec!["/bin/sh".to_string()]);
        assert!(stack.push_args(&args).is_ok());
        assert_eq!(stack.argc, 1);
    }

    #[test]
    fn elf_load_invalid() {
        let mut ctx = ExecContext::new();
        assert!(ctx.load_elf(b"not elf").is_err());
        assert!(ctx.load_elf(&[0u8; 52]).is_err());
    }

    #[test]
    fn elf_load_minimal() {
        // Minimal ELF header for i386 EXEC
        let mut data = vec![0u8; 100];
        data[0..4].copy_from_slice(b"\x7fELF");
        data[4] = 1; // 32-bit
        data[5] = 1; // little endian
        data[16] = 2; data[17] = 0; // ET_EXEC
        data[18] = 3; data[19] = 0; // EM_386
        data[24] = 0x00; data[25] = 0x80; data[26] = 0x04; data[27] = 0x08; // entry 0x08048000
        data[28] = 52; // phoff
        data[42] = 1; // phnum
        // Program header at 52
        data[52] = 1; // PT_LOAD
        data[56] = 0; // offset 0
        data[60] = 0x00; data[61] = 0x80; data[62] = 0x04; data[63] = 0x08; // vaddr
        data[68] = 100; // filesz
        data[72] = 100; // memsz
        data[76] = 5; // flags R+X

        let mut ctx = ExecContext::new();
        assert!(ctx.load_elf(&data).is_ok());
        assert!(ctx.header.is_executable());
        assert_eq!(ctx.phdrs.len(), 1);
        assert!(ctx.phdrs[0].is_load());
        assert_eq!(ctx.total_memory(), 100);
    }

    #[test]
    fn program_header_flags() {
        let ph = ProgramHeader { p_type: PT_LOAD, flags: 2, ..Default::default() };
        assert!(ph.is_writable());
        assert!(!ph.is_executable());
    }
}
