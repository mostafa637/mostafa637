// Line-by-line translation of emu/mmu.h
//
// The MMU is deliberately just an interface: a `translate` callback that turns
// a guest page address into a host pointer, plus a `changes` counter the TLB
// uses to notice when mappings moved. The real page table lives in
// kernel/memory.c and is not ported yet, so this module provides the interface
// and the page arithmetic, and tests supply their own backend.

use std::ptr::NonNull;

use crate::cpu::{AddrT, DwordT};

/// `typedef dword_t page_t` — the top 20 bits of an address, i.e. `addr >> 12`.
pub type PageT = DwordT;

/// `#define BAD_PAGE 0x10000`
pub const BAD_PAGE: PageT = 0x10000;

/// `#define PAGE_BITS 12`
pub const PAGE_BITS: u32 = 12;
/// `#define PAGE_SIZE (1 << PAGE_BITS)`
pub const PAGE_SIZE: u32 = 1 << PAGE_BITS;
/// `#define MEM_PAGES (1 << 20)` — "at least on 32-bit"
pub const MEM_PAGES: u32 = 1 << 20;

/// `#define PAGE(addr) ((addr) >> PAGE_BITS)`
pub const fn page(addr: AddrT) -> PageT {
    addr >> PAGE_BITS
}

/// `#define PGOFFSET(addr) ((addr) & (PAGE_SIZE - 1))`
pub const fn pgoffset(addr: AddrT) -> AddrT {
    addr & (PAGE_SIZE - 1)
}

/// `#define PAGE_ROUND_UP(bytes) (PAGE((bytes) + PAGE_SIZE - 1))`
///
/// The C comment is load bearing: "bytes MUST be unsigned if you would like
/// this to overflow to zero". `wrapping_add` is what makes `PAGE_ROUND_UP(0)`
/// come out as 0 rather than 1, exactly as the unsigned C arithmetic does.
pub const fn page_round_up(bytes: AddrT) -> PageT {
    page(bytes.wrapping_add(PAGE_SIZE - 1))
}

/// `MEM_READ` / `MEM_WRITE` / `MEM_WRITE_PTRACE`.
///
/// The discriminants matter: the backends compare against `MEM_WRITE` exactly,
/// so a ptrace write is *not* blocked by a read-only mapping.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MemType {
    Read = 0,
    Write = 1,
    WritePtrace = 2,
}

/// `struct mmu_ops` — `void *(*translate)(struct mmu *mmu, addr_t addr, int type)`.
///
/// One deliberate difference: the C callback is handed the `struct mmu *` so a
/// backend can recover its container with `container_of`. A Rust impl just
/// holds whatever state it needs, so there is nothing to recover and the
/// parameter is dropped.
pub trait MmuOps {
    /// Map the guest page containing `addr`, returning a host pointer to its
    /// start, or `None` if the access should fault.
    fn translate(&mut self, addr: AddrT, type_: MemType) -> Option<NonNull<u8>>;
}

/// `struct mmu`.
///
/// The C struct also carries `struct asbestos *asbestos` — the JIT's
/// back-pointer to this address space. The engine is not ported yet, so the
/// field is absent rather than a dangling stub.
pub struct Mmu<'a> {
    pub ops: &'a mut dyn MmuOps,
    /// Bumped whenever mappings change; the TLB compares it against its own
    /// cached copy to know when to flush.
    pub changes: u64,
}

impl<'a> Mmu<'a> {
    pub fn new(ops: &'a mut dyn MmuOps) -> Self {
        Mmu { ops, changes: 0 }
    }

    /// `mmu_translate`
    pub fn translate(&mut self, addr: AddrT, type_: MemType) -> Option<NonNull<u8>> {
        self.ops.translate(addr, type_)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn page_arithmetic_matches_the_macros() {
        assert_eq!(PAGE_SIZE, 4096);
        assert_eq!(page(0), 0);
        assert_eq!(page(4095), 0);
        assert_eq!(page(4096), 1);
        assert_eq!(page(0x1234_5678), 0x1_2345);

        assert_eq!(pgoffset(0), 0);
        assert_eq!(pgoffset(4095), 4095);
        assert_eq!(pgoffset(4096), 0);
        assert_eq!(pgoffset(0x1234_5678), 0x678);

        assert_eq!(page_round_up(0), 0, "unsigned overflow to zero, per the C comment");
        assert_eq!(page_round_up(1), 1);
        assert_eq!(page_round_up(4096), 1);
        assert_eq!(page_round_up(4097), 2);
        assert_eq!(BAD_PAGE, 0x1_0000);
        assert_eq!(MEM_PAGES, 0x10_0000);
    }

    #[test]
    fn mem_type_discriminants_are_the_c_values() {
        assert_eq!(MemType::Read as u32, 0);
        assert_eq!(MemType::Write as u32, 1);
        assert_eq!(MemType::WritePtrace as u32, 2);
        // a ptrace write must not compare equal to a plain write
        assert_ne!(MemType::WritePtrace, MemType::Write);
    }
}
