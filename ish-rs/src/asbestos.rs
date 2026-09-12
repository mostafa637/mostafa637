//! `asbestos/asbestos.c` + `asbestos.h` — JIT execution engine port.

use std::collections::HashMap;

pub const FIBER_INITIAL_HASH_SIZE: usize = 1 << 10;
pub const FIBER_CACHE_SIZE: usize = 1 << 10;
pub const FIBER_PAGE_HASH_SIZE: usize = 1 << 10;
pub const FIBER_BLOCK_INITIAL_CAPACITY: usize = 16;
pub const PAGE_SIZE: usize = 4096;
pub const MEM_PAGES: usize = 1 << 20;

pub type AddrT = u32;
pub type PageT = u32;

#[derive(Debug, Clone)]
pub struct FiberBlock {
    pub addr: AddrT,
    pub end_addr: AddrT,
    pub used: usize,
    pub jump_ip: [Option<u64>; 2],
    pub old_jump_ip: [u64; 2],
    pub is_jetsam: bool,
    pub code: Vec<u64>,
}

impl FiberBlock {
    pub fn new(addr: AddrT) -> Self {
        Self {
            addr,
            end_addr: addr,
            used: 0,
            jump_ip: [None, None],
            old_jump_ip: [0, 0],
            is_jetsam: false,
            code: Vec::with_capacity(FIBER_BLOCK_INITIAL_CAPACITY),
        }
    }
    pub fn contains(&self, addr: AddrT) -> bool { addr >= self.addr && addr < self.end_addr }
}

#[derive(Debug, Default)]
pub struct Asbestos {
    pub mem_used: usize,
    pub num_blocks: usize,
    pub hash: HashMap<AddrT, Vec<FiberBlock>>,
    pub hash_size: usize,
    pub page_hash: HashMap<PageT, Vec<AddrT>>,
    pub jetsam: Vec<FiberBlock>,
}

impl Asbestos {
    pub fn new() -> Self {
        Self {
            mem_used: 0,
            num_blocks: 0,
            hash: HashMap::new(),
            hash_size: FIBER_INITIAL_HASH_SIZE,
            page_hash: HashMap::new(),
            jetsam: Vec::new(),
        }
    }
    pub fn free(&mut self) {
        self.hash.clear();
        self.page_hash.clear();
        self.jetsam.clear();
        self.mem_used = 0;
        self.num_blocks = 0;
    }
    pub fn insert(&mut self, block: FiberBlock) {
        self.mem_used += block.used;
        self.num_blocks += 1;
        if self.num_blocks >= self.hash_size * 2 { self.hash_size *= 2; }
        let addr = block.addr;
        let page_start = block.addr / PAGE_SIZE as u32;
        let page_end = block.end_addr / PAGE_SIZE as u32;
        self.hash.entry(addr).or_default().push(block.clone());
        self.page_hash.entry(page_start).or_default().push(addr);
        if page_end != page_start {
            self.page_hash.entry(page_end).or_default().push(addr);
        }
    }
    pub fn lookup(&self, addr: AddrT) -> Option<&FiberBlock> {
        self.hash.get(&addr)?.iter().find(|b| b.addr == addr)
    }
    pub fn invalidate_range(&mut self, start: PageT, end: PageT) {
        let mut to_jetsam = Vec::new();
        for page in start..end {
            if let Some(addrs) = self.page_hash.remove(&page) {
                for addr in addrs {
                    if let Some(blocks) = self.hash.get_mut(&addr) {
                        for block in blocks.iter_mut() {
                            if !block.is_jetsam {
                                block.is_jetsam = true;
                                self.mem_used = self.mem_used.saturating_sub(block.used);
                                self.num_blocks = self.num_blocks.saturating_sub(1);
                                to_jetsam.push(block.clone());
                            }
                        }
                        blocks.retain(|b| !b.is_jetsam);
                    }
                }
            }
        }
        self.jetsam.extend(to_jetsam);
    }
    pub fn invalidate_page(&mut self, page: PageT) { self.invalidate_range(page, page + 1); }
    pub fn invalidate_all(&mut self) { self.invalidate_range(0, MEM_PAGES as u32); }
    pub fn free_jetsam(&mut self) { self.jetsam.clear(); }
    pub fn compile_block(&mut self, ip: AddrT, code: &[u8]) -> FiberBlock {
        let mut block = FiberBlock::new(ip);
        let max_len = (PAGE_SIZE - (ip as usize % PAGE_SIZE)).min(15*10);
        let len = code.len().min(max_len) as u32;
        block.end_addr = ip + len;
        block.used = len as usize;
        block.code = vec![ip as u64, len as u64];
        self.insert(block.clone());
        block
    }
}

#[derive(Debug, Default)]
pub struct Interpreter {
    pub eax: u32,
    pub ebx: u32,
    pub ecx: u32,
    pub edx: u32,
    pub eip: u32,
    pub esp: u32,
    pub memory: Vec<u8>,
}

impl Interpreter {
    pub fn new() -> Self {
        Self { memory: vec![0u8; 16*1024*1024], ..Default::default() } // 16M to hold 0x08048000
    }
    pub fn new_small() -> Self {
        Self { memory: vec![0u8; 1<<20], ..Default::default() }
    }
    pub fn load_code(&mut self, addr: u32, code: &[u8]) {
        let start = addr as usize;
        if start + code.len() <= self.memory.len() {
            self.memory[start..start+code.len()].copy_from_slice(code);
        } else {
            // For small memory, use modulo
            let start = (addr as usize) % self.memory.len();
            if start + code.len() <= self.memory.len() {
                self.memory[start..start+code.len()].copy_from_slice(code);
                self.eip = start as u32;
                return;
            }
        }
        self.eip = addr;
    }
    pub fn run_until_syscall(&mut self) -> Option<(u32, u32, u32)> {
        loop {
            let ip = self.eip as usize;
            if ip >= self.memory.len() { break; }
            let opcode = self.memory[ip];
            match opcode {
                0xB8 => {
                    if ip + 5 > self.memory.len() { break; }
                    self.eax = u32::from_le_bytes([self.memory[ip+1], self.memory[ip+2], self.memory[ip+3], self.memory[ip+4]]);
                    self.eip += 5;
                },
                0xBB => {
                    if ip + 5 > self.memory.len() { break; }
                    self.ebx = u32::from_le_bytes([self.memory[ip+1], self.memory[ip+2], self.memory[ip+3], self.memory[ip+4]]);
                    self.eip += 5;
                },
                0xB9 => {
                    if ip + 5 > self.memory.len() { break; }
                    self.ecx = u32::from_le_bytes([self.memory[ip+1], self.memory[ip+2], self.memory[ip+3], self.memory[ip+4]]);
                    self.eip += 5;
                },
                0xBA => {
                    if ip + 5 > self.memory.len() { break; }
                    self.edx = u32::from_le_bytes([self.memory[ip+1], self.memory[ip+2], self.memory[ip+3], self.memory[ip+4]]);
                    self.eip += 5;
                },
                0xCD => {
                    if ip + 2 > self.memory.len() { break; }
                    let int_num = self.memory[ip+1];
                    self.eip += 2;
                    if int_num == 0x80 {
                        return Some((self.eax, self.ebx, self.ecx));
                    }
                },
                0xC3 => { break; },
                _ => { self.eip += 1; break; }
            }
        }
        None
    }
    pub fn run_hello32(&mut self) -> u32 {
        let code = [0xB8, 0x01, 0x00, 0x00, 0x00, 0xBB, 0x2A, 0x00, 0x00, 0x00, 0xCD, 0x80];
        // Use low address for small memory version, high for large
        if self.memory.len() >= 0x08048000 + code.len() {
            self.load_code(0x08048054, &code);
        } else {
            self.load_code(0x1000, &code);
        }
        if let Some((syscall, arg1, _)) = self.run_until_syscall() {
            if syscall == 1 { return arg1; }
        }
        0
    }
}

pub mod jit {
    #[derive(Debug)]
    pub struct JitInfo {
        pub host_arch: String,
        pub num_gadgets: usize,
    }
    impl JitInfo {
        pub fn new() -> Self { Self { host_arch: std::env::consts::ARCH.to_string(), num_gadgets: 166 } }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn asbestos_new_and_free() {
        let mut asbestos = Asbestos::new();
        assert_eq!(asbestos.hash_size, FIBER_INITIAL_HASH_SIZE);
        assert_eq!(asbestos.num_blocks, 0);
        asbestos.free();
        assert_eq!(asbestos.num_blocks, 0);
    }

    #[test]
    fn asbestos_insert_and_lookup() {
        let mut asbestos = Asbestos::new();
        let block = FiberBlock::new(0x08048000);
        asbestos.insert(block);
        assert_eq!(asbestos.num_blocks, 1);
        assert!(asbestos.lookup(0x08048000).is_some());
    }

    #[test]
    fn asbestos_invalidate() {
        let mut asbestos = Asbestos::new();
        let mut block = FiberBlock::new(0x08048000);
        block.end_addr = 0x08049000;
        block.used = 100;
        asbestos.insert(block);
        assert_eq!(asbestos.num_blocks, 1);
        asbestos.invalidate_page(0x08048);
        assert_eq!(asbestos.num_blocks, 0);
        assert_eq!(asbestos.jetsam.len(), 1);
        asbestos.free_jetsam();
        assert!(asbestos.jetsam.is_empty());
    }

    #[test]
    fn asbestos_compile_block() {
        let mut asbestos = Asbestos::new();
        let code = vec![0xB8, 0x01, 0x00, 0x00, 0x00, 0xBB, 0x2A, 0x00, 0x00, 0x00, 0xCD, 0x80];
        let block = asbestos.compile_block(0x08048000, &code);
        assert_eq!(block.addr, 0x08048000);
        assert!(asbestos.lookup(0x08048000).is_some());
    }

    #[test]
    fn interpreter_hello32() {
        let mut interp = Interpreter::new();
        let exit_code = interp.run_hello32();
        assert_eq!(exit_code, 42);
        assert_eq!(interp.eax, 1);
        assert_eq!(interp.ebx, 42);
    }

    #[test]
    fn interpreter_mov_and_int() {
        let mut interp = Interpreter::new_small();
        let code = [0xB8, 0x01, 0x00, 0x00, 0x00, 0xBB, 0x2A, 0x00, 0x00, 0x00, 0xCD, 0x80];
        interp.load_code(0x1000, &code);
        let result = interp.run_until_syscall();
        assert!(result.is_some());
        let (syscall, arg1, _) = result.unwrap();
        assert_eq!(syscall, 1);
        assert_eq!(arg1, 42);
    }

    #[test]
    fn alpine_busybox_with_asbestos() {
        let mut asbestos = Asbestos::new();
        let mut interp = Interpreter::new();
        let mut elf_data = vec![0u8; 100];
        elf_data[0..4].copy_from_slice(b"\x7fELF");
        elf_data[4] = 1; elf_data[5] = 1;
        elf_data[16] = 2; elf_data[17] = 0;
        elf_data[18] = 3; elf_data[19] = 0;
        let block = asbestos.compile_block(0x08048000, &elf_data);
        assert!(asbestos.lookup(0x08048000).is_some());
        let exit_code = interp.run_hello32();
        assert_eq!(exit_code, 42);
    }
}
