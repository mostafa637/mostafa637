//! `kernel/mm.h` — memory descriptor.
//!
//! `struct mm` is the container for `struct mem` plus brk, vdso, and procfs
//! info. The address-space syscalls in `mmap.rs` already implemented a
//! simplified `Mm` with `mem`, `start_brk`, `brk`, and `refcount`. This module
//! extends it with the remaining fields and provides the full `mm_new`,
//! `mm_copy`, `mm_retain`, `mm_release` API, depending only on `memory`
//! (already ported).

use crate::memory::Mem;
use crate::refcount::RefCount;

/// `struct mm` — full version with procfs fields.
pub struct MmFull {
    /// `refcount`
    pub refcount: RefCount,
    /// `mem`
    pub mem: Mem,
    /// `vdso` — immutable after exec
    pub vdso: u32,
    /// `start_brk` — immutable
    pub start_brk: u32,
    /// `brk`
    pub brk: u32,
    /// `argv_start`, `argv_end`, `env_start`, `env_end`, `auxv_start`,
    /// `auxv_end`, `stack_start` — for procfs
    pub argv_start: u32,
    pub argv_end: u32,
    pub env_start: u32,
    pub env_end: u32,
    pub auxv_start: u32,
    pub auxv_end: u32,
    pub stack_start: u32,
}

impl MmFull {
    /// `mm_new`
    pub fn new() -> Self {
        Self {
            refcount: RefCount::new(),
            mem: Mem::new(),
            vdso: 0,
            start_brk: 0,
            brk: 0,
            argv_start: 0,
            argv_end: 0,
            env_start: 0,
            env_end: 0,
            auxv_start: 0,
            auxv_end: 0,
            stack_start: 0,
        }
    }

    /// `mm_copy` — COW copy.
    pub fn copy(&mut self) -> Self {
        let mut new_mm = Self {
            refcount: RefCount::new(),
            mem: Mem::new(),
            vdso: self.vdso,
            start_brk: self.start_brk,
            brk: self.brk,
            argv_start: self.argv_start,
            argv_end: self.argv_end,
            env_start: self.env_start,
            env_end: self.env_end,
            auxv_start: self.auxv_start,
            auxv_end: self.auxv_end,
            stack_start: self.stack_start,
        };
        Mem::copy_on_write(&mut self.mem, &mut new_mm.mem, 0, crate::mmu::MEM_PAGES);
        new_mm
    }

    /// `mm_retain`
    pub fn retain(&self) -> usize {
        self.refcount.retain()
    }

    /// `mm_release` — returns true if refcount hit zero.
    pub fn release(&self) -> bool {
        self.refcount.release()
    }
}

impl Default for MmFull {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn mm_new_has_refcount_one() {
        let mm = MmFull::new();
        assert_eq!(mm.refcount.get(), 1);
        assert_eq!(mm.brk, 0);
    }

    #[test]
    fn mm_copy_preserves_brk_and_vdso() {
        let mut mm = MmFull::new();
        mm.start_brk = 0x1000;
        mm.brk = 0x2000;
        mm.vdso = 0x3000;
        let copy = mm.copy();
        assert_eq!(copy.start_brk, 0x1000);
        assert_eq!(copy.brk, 0x2000);
        assert_eq!(copy.vdso, 0x3000);
        assert_eq!(copy.refcount.get(), 1);
    }

    #[test]
    fn mm_retain_release() {
        let mm = MmFull::new();
        assert_eq!(mm.retain(), 2);
        assert!(!mm.release());
        assert!(mm.release());
    }
}
