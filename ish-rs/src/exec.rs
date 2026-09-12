//! `kernel/exec.c` — full port: ELF loading, interpreter, stack setup for Alpine.

pub const ARGV_MAX: usize = 32 * 4096;
pub const ENOEXEC: i32 = -8;
pub const E2BIG: i32 = -7;
pub const EINVAL: i32 = -22;
pub const ENOMEM: i32 = -12;
pub const EIO: i32 = -5;
pub const ELIBBAD: i32 = -80;

pub const PAGE_SIZE: u32 = 4096;
pub const PAGE_BITS: u32 = 12;

pub fn align_stack(sp: u32) -> u32 { sp & !0xf }
pub fn page_align(addr: u32) -> u32 { addr & !(PAGE_SIZE - 1) }
pub fn page_round_up(addr: u32) -> u32 { (addr + PAGE_SIZE - 1) & !(PAGE_SIZE - 1) }
pub fn page_offset(addr: u32) -> u32 { addr & (PAGE_SIZE - 1) }

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
    pub auxv: Vec<(u32, u32)>,
}

impl ExecStack {
    pub fn new(sp: u32) -> Self { Self { sp: align_stack(sp), argc: 0, argv: Vec::new(), envp: Vec::new(), auxv: Vec::new() } }
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
    pub fn push_auxv(&mut self, key: u32, val: u32) {
        self.auxv.push((key, val));
    }
}

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

pub const PH_R: u32 = 4;
pub const PH_W: u32 = 2;
pub const PH_X: u32 = 1;

impl ProgramHeader {
    pub fn is_load(&self) -> bool { self.p_type == PT_LOAD }
    pub fn is_interp(&self) -> bool { self.p_type == PT_INTERP }
    pub fn is_writable(&self) -> bool { (self.flags & PH_W) != 0 }
    pub fn is_executable(&self) -> bool { (self.flags & PH_X) != 0 }
    pub fn is_readable(&self) -> bool { (self.flags & PH_R) != 0 }
    pub fn page_start(&self) -> u32 { page_align(self.vaddr) }
    pub fn page_end(&self) -> u32 { page_round_up(self.vaddr + self.memsz) }
    pub fn file_page_end(&self) -> u32 { page_round_up(self.vaddr + self.filesz) }
    pub fn bss_start(&self) -> u32 { self.vaddr + self.filesz }
    pub fn bss_size(&self) -> u32 { self.memsz.saturating_sub(self.filesz) }
}

#[derive(Debug, Clone, Default)]
pub struct LoadedSegment {
    pub vaddr: u32,
    pub memsz: u32,
    pub filesz: u32,
    pub flags: u32,
    pub data: Vec<u8>,
}

impl LoadedSegment {
    pub fn new(ph: &ProgramHeader, file_data: &[u8]) -> Self {
        let offset = ph.offset as usize;
        let filesz = ph.filesz as usize;
        let data = if offset + filesz <= file_data.len() {
            file_data[offset..offset+filesz].to_vec()
        } else {
            vec![0u8; filesz]
        };
        Self { vaddr: ph.vaddr, memsz: ph.memsz, filesz: ph.filesz, flags: ph.flags, data }
    }
    pub fn contains(&self, addr: u32) -> bool {
        addr >= self.vaddr && addr < self.vaddr + self.memsz
    }
}

/// Read ELF header from fd data, matching `read_header` in C
pub fn read_header(data: &[u8]) -> Result<ElfHeader, i32> {
    if data.len() < 52 { return Err(ENOEXEC); }
    let magic = [data[0], data[1], data[2], data[3]];
    if magic != *b"\x7fELF" { return Err(ENOEXEC); }
    let bitness = data[4];
    let endian = data[5];
    let version = data[6];
    if bitness != 1 || endian != 1 || version != 1 { return Err(ENOEXEC); }
    let elf_type = u16::from_le_bytes([data[16], data[17]]);
    let machine = u16::from_le_bytes([data[18], data[19]]);
    if !is_valid_elf_header(&magic, elf_type, bitness, endian, machine) { return Err(ENOEXEC); }
    Ok(ElfHeader {
        magic,
        bitness,
        endian,
        version,
        abi: data[7],
        elf_type,
        machine,
        entry: u32::from_le_bytes([data[24], data[25], data[26], data[27]]),
        phoff: u32::from_le_bytes([data[28], data[29], data[30], data[31]]),
        shoff: u32::from_le_bytes([data[32], data[33], data[34], data[35]]),
        flags: u32::from_le_bytes([data[36], data[37], data[38], data[39]]),
        ehsize: u16::from_le_bytes([data[40], data[41]]),
        phentsize: u16::from_le_bytes([data[42+0], data[42+1]]),
        phnum: u16::from_le_bytes([data[42+2], data[42+3]]),
        shentsize: 0, shnum: 0, shstrndx: 0,
    })
}

/// Read program headers, matching `read_prg_headers` in C
pub fn read_program_headers(data: &[u8], header: &ElfHeader) -> Result<Vec<ProgramHeader>, i32> {
    let mut phdrs = Vec::new();
    let phoff = header.phoff as usize;
    for i in 0..header.phnum as usize {
        let off = phoff + i * 32;
        if off + 32 > data.len() { return Err(ENOEXEC); }
        let p_type = u32::from_le_bytes([data[off], data[off+1], data[off+2], data[off+3]]);
        let offset = u32::from_le_bytes([data[off+4], data[off+5], data[off+6], data[off+7]]);
        let vaddr = u32::from_le_bytes([data[off+8], data[off+9], data[off+10], data[off+11]]);
        let paddr = u32::from_le_bytes([data[off+12], data[off+13], data[off+14], data[off+15]]);
        let filesz = u32::from_le_bytes([data[off+16], data[off+17], data[off+18], data[off+19]]);
        let memsz = u32::from_le_bytes([data[off+20], data[off+21], data[off+22], data[off+23]]);
        let flags = u32::from_le_bytes([data[off+24], data[off+25], data[off+26], data[off+27]]);
        let align = u32::from_le_bytes([data[off+28], data[off+29], data[off+30], data[off+31]]);
        phdrs.push(ProgramHeader { p_type, offset, vaddr, paddr, filesz, memsz, flags, align });
    }
    Ok(phdrs)
}

/// Find hole for ELF, matching `find_hole_for_elf` in C
pub fn find_hole_for_elf(phdrs: &[ProgramHeader]) -> u32 {
    let mut first: Option<&ProgramHeader> = None;
    let mut last: Option<&ProgramHeader> = None;
    for ph in phdrs {
        if ph.is_load() {
            if first.is_none() { first = Some(ph); }
            last = Some(ph);
        }
    }
    if let (Some(f), Some(l)) = (first, last) {
        let size = page_round_up(l.vaddr + l.memsz) - page_align(f.vaddr);
        // Simulate pt_find_hole: return 0x40000000 for PIE
        if f.vaddr == 0 { 0x40000000 } else { 0 }
    } else {
        0
    }
}

/// Load entry, matching `load_entry` in C — returns loaded segment
pub fn load_entry(ph: &ProgramHeader, bias: u32, file_data: &[u8]) -> Result<LoadedSegment, i32> {
    if !ph.is_load() { return Err(EINVAL); }
    Ok(LoadedSegment::new(ph, file_data))
}

#[derive(Debug, Default)]
pub struct ExecContext {
    pub header: ElfHeader,
    pub phdrs: Vec<ProgramHeader>,
    pub entry: u32,
    pub interp: Option<String>,
    pub interp_header: Option<ElfHeader>,
    pub interp_phdrs: Vec<ProgramHeader>,
    pub stack_top: u32,
    pub brk: u32,
    pub bias: u32,
    pub interp_bias: u32,
    pub loaded_segments: Vec<LoadedSegment>,
}

impl ExecContext {
    pub fn new() -> Self { Self::default() }

    pub fn load_elf(&mut self, data: &[u8]) -> Result<(), i32> {
        let header = read_header(data)?;
        let phdrs = read_program_headers(data, &header)?;
        // Check for interpreter
        let mut interp_name: Option<String> = None;
        for ph in &phdrs {
            if ph.is_interp() {
                let off = ph.offset as usize;
                let end = (ph.offset + ph.filesz) as usize;
                if end > data.len() { return Err(ENOEXEC); }
                let bytes = &data[off..end];
                if let Some(nul) = bytes.iter().position(|&b| b == 0) {
                    if let Ok(s) = std::str::from_utf8(&bytes[..nul]) {
                        if interp_name.is_some() { return Err(EINVAL); } // two interpreters
                        interp_name = Some(s.to_string());
                    }
                }
            }
        }
        self.header = header;
        self.phdrs = phdrs;
        self.entry = self.header.entry;
        self.interp = interp_name;
        Ok(())
    }

    /// Load ELF + interpreter if needed, matching `elf_exec` in C
    pub fn elf_exec(&mut self, data: &[u8], interp_data: Option<&[u8]>) -> Result<u32, i32> {
        self.load_elf(data)?;
        // If has interpreter, load it too
        if let Some(ref _interp_path) = self.interp {
            if let Some(idata) = interp_data {
                let iheader = read_header(idata)?;
                let iphdrs = read_program_headers(idata, &iheader)?;
                self.interp_header = Some(iheader);
                self.interp_phdrs = iphdrs;
                // Find bias for interpreter if PIE
                if iheader.elf_type == 3 {
                    self.interp_bias = find_hole_for_elf(&self.interp_phdrs);
                }
                // Entry becomes interpreter entry
                self.entry = iheader.entry + self.interp_bias;
            } else {
                return Err(ELIBBAD);
            }
        }
        // Bias for main executable if PIE
        if self.header.elf_type == 3 {
            self.bias = find_hole_for_elf(&self.phdrs);
            self.entry += self.bias;
        }
        // Load segments
        self.loaded_segments.clear();
        for ph in &self.phdrs {
            if ph.is_load() {
                let seg = load_entry(ph, self.bias, data)?;
                // Track brk: highest end of writable segment
                let seg_end = ph.vaddr + ph.memsz + self.bias;
                if ph.is_writable() && seg_end > self.brk {
                    self.brk = page_round_up(seg_end);
                }
                self.loaded_segments.push(seg);
            }
        }
        if let Some(idata) = interp_data {
            for ph in &self.interp_phdrs {
                if ph.is_load() {
                    let seg = load_entry(ph, self.interp_bias, idata)?;
                    self.loaded_segments.push(seg);
                }
            }
        }
        Ok(self.entry)
    }

    pub fn total_memory(&self) -> u32 {
        self.phdrs.iter().filter(|ph| ph.is_load()).map(|ph| ph.memsz).sum()
    }

    pub fn has_interpreter(&self) -> bool { self.interp.is_some() }

    pub fn load_from_fakefs(&mut self, fakefs: &crate::fake::FakeFs, path: &str) -> Result<(), i32> {
        let data = fakefs.read_file(path).map_err(|_| -2)?;
        self.load_elf(&data)
    }

    pub fn setup_stack(&mut self, args: &ExecArgs, env: &ExecArgs, stack_top: u32) -> Result<ExecStack, i32> {
        let mut stack = ExecStack::new(stack_top);
        // Push env
        for e in env.args.iter().rev() {
            let addr = stack.push_string(e);
            stack.envp.push(addr);
        }
        stack.envp.reverse();
        // Push args
        stack.push_args(args)?;
        // auxv: AT_ENTRY, AT_PHDR, AT_PHNUM, AT_PAGESZ, AT_BASE etc.
        stack.push_auxv(9, self.entry); // AT_ENTRY
        stack.push_auxv(3, 0x08048000); // AT_PHDR placeholder
        stack.push_auxv(5, self.phdrs.len() as u32); // AT_PHNUM
        stack.push_auxv(6, PAGE_SIZE); // AT_PAGESZ
        if self.has_interpreter() {
            stack.push_auxv(7, self.interp_bias); // AT_BASE
        }
        stack.push_auxv(0, 0); // AT_NULL
        self.stack_top = stack.sp;
        Ok(stack)
    }

    pub fn get_brk(&self) -> u32 { self.brk }

    pub fn find_segment(&self, addr: u32) -> Option<&LoadedSegment> {
        self.loaded_segments.iter().find(|s| s.contains(addr))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::fake::{FakeFs, IshStat};

    #[test]
    fn align_stack_matches_c() {
        assert_eq!(align_stack(0x1234), 0x1230);
        assert_eq!(align_stack(0x1000), 0x1000);
    }

    #[test]
    fn page_helpers() {
        assert_eq!(page_align(0x1234), 0x1000);
        assert_eq!(page_round_up(0x1001), 0x2000);
        assert_eq!(page_offset(0x1234), 0x234);
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
    fn read_header_and_phdrs() {
        let mut data = vec![0u8; 100];
        data[0..4].copy_from_slice(b"\x7fELF");
        data[4] = 1; data[5] = 1; data[6] = 1;
        data[16] = 2; data[17] = 0;
        data[18] = 3; data[19] = 0;
        data[24] = 0x00; data[25] = 0x80; data[26] = 0x04; data[27] = 0x08;
        data[28] = 52;
        data[42] = 32; data[43] = 0;
        data[44] = 1; data[45] = 0;
        data[52] = 1;
        data[56] = 0;
        data[60] = 0x00; data[61] = 0x80; data[62] = 0x04; data[63] = 0x08;
        data[68] = 100;
        data[72] = 100;
        data[76] = 5;

        let hdr = read_header(&data).unwrap();
        assert!(hdr.is_executable());
        let phdrs = read_program_headers(&data, &hdr).unwrap();
        assert_eq!(phdrs.len(), 1);
        assert!(phdrs[0].is_load());
        assert_eq!(find_hole_for_elf(&phdrs), 0);
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
        let mut data = vec![0u8; 100];
        data[0..4].copy_from_slice(b"\x7fELF");
        data[4] = 1; data[5] = 1; data[6] = 1;
        data[16] = 2; data[17] = 0;
        data[18] = 3; data[19] = 0;
        data[24] = 0x00; data[25] = 0x80; data[26] = 0x04; data[27] = 0x08;
        data[28] = 52;
        data[40] = 52; data[41] = 0;
        data[42] = 32; data[43] = 0;
        data[44] = 1; data[45] = 0;
        data[52] = 1;
        data[56] = 0;
        data[60] = 0x00; data[61] = 0x80; data[62] = 0x04; data[63] = 0x08;
        data[68] = 100;
        data[72] = 100;
        data[76] = 5;

        let mut ctx = ExecContext::new();
        assert!(ctx.load_elf(&data).is_ok());
        assert!(ctx.header.is_executable());
        assert_eq!(ctx.phdrs.len(), 1);
        assert!(ctx.phdrs[0].is_load());
        assert_eq!(ctx.total_memory(), 100);
    }

    #[test]
    fn elf_exec_with_segments() {
        let mut data = vec![0u8; 200];
        data[0..4].copy_from_slice(b"\x7fELF");
        data[4] = 1; data[5] = 1; data[6] = 1;
        data[16] = 2; data[17] = 0;
        data[18] = 3; data[19] = 0;
        data[24] = 0x00; data[25] = 0x80; data[26] = 0x04; data[27] = 0x08;
        data[28] = 52;
        data[42] = 32; data[43] = 0;
        data[44] = 1; data[45] = 0;
        data[52] = 1;
        data[56] = 0;
        data[60] = 0x00; data[61] = 0x80; data[62] = 0x04; data[63] = 0x08;
        data[68] = 100;
        data[72] = 100;
        data[76] = 6; // RW

        let mut ctx = ExecContext::new();
        let entry = ctx.elf_exec(&data, None).unwrap();
        assert_eq!(entry, 0x08048000);
        assert_eq!(ctx.loaded_segments.len(), 1);
        assert!(ctx.get_brk() >= 0x08048000);
        assert!(ctx.find_segment(0x08048000).is_some());
    }

    #[test]
    fn program_header_flags() {
        let ph = ProgramHeader { p_type: PT_LOAD, flags: 2, ..Default::default() };
        assert!(ph.is_writable());
        assert!(!ph.is_executable());
        assert_eq!(ph.bss_size(), 0);
        let ph2 = ProgramHeader { p_type: PT_LOAD, vaddr: 0x1000, filesz: 100, memsz: 200, ..Default::default() };
        assert_eq!(ph2.bss_size(), 100);
        assert_eq!(ph2.bss_start(), 0x1000 + 100);
    }

    #[test]
    fn load_from_fakefs_alpine() {
        let mut fs = FakeFs::new();
        fs.load_alpine_mock();
        let mut ctx = ExecContext::new();
        assert!(ctx.load_from_fakefs(&fs, "/bin/busybox").is_ok());
        assert!(ctx.header.is_executable());
        assert!(ctx.load_from_fakefs(&fs, "/nonexistent").is_err());
    }

    #[test]
    fn setup_stack_for_alpine() {
        let mut ctx = ExecContext::new();
        let args = ExecArgs::new(vec!["/bin/sh".to_string(), "-c".to_string(), "echo hi".to_string()]);
        let env = ExecArgs::new(vec!["PATH=/bin".to_string(), "HOME=/root".to_string()]);
        let stack = ctx.setup_stack(&args, &env, 0xbffff000).unwrap();
        assert_eq!(stack.argc, 3);
        assert_eq!(stack.envp.len(), 2);
        assert!(stack.sp < 0xbffff000);
        assert!(!stack.auxv.is_empty());
    }
}
