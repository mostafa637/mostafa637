//! `kernel/mmap.c` — the address-space syscalls.
//!
//! This is the layer the guest actually calls: `mmap2`, the old six-argument
//! `mmap`, `munmap`, `mremap`, `mprotect`, `brk`, and the four that Linux lets
//! get away with doing nothing (`madvise`, `mbind`, `mlock`, `msync`). All of it
//! sits directly on [`Mem`].
//!
//! Three things here are behaviour rather than tidiness, and all three are
//! pinned by the differential test:
//!
//! * **A non-`MAP_FIXED` hint that is not free is silently ignored — and the
//!   mapping lands on top of what was there anyway.** When the hint overlaps an
//!   existing mapping, the C sets `addr = 0` and then never recomputes `page`,
//!   which it had already assigned from the hint. So the "fall back to finding a
//!   hole" path does not exist: the hint is used whether or not it is free. The
//!   port keeps this, with the dead assignment reproduced as a comment.
//! * **`mremap` uses `PAGE(len)`, not `PAGE_ROUND_UP(len)`.** A length under one
//!   page rounds *down* to zero pages, so `mremap(a, 0x100, 0x200)` shrinks
//!   nothing and grows nothing, and returns the address unchanged.
//! * **`brk` reports the old value when it cannot honour the request** rather
//!   than failing. That is the interface: the caller compares what came back
//!   with what it asked for.
//!
//! Two things are not ported, both said so at the point they would matter:
//!
//! * **File-backed mappings.** `do_mmap`'s other branch needs `f_get` and
//!   `fd->ops->mmap`, and there is no fd table yet. With no fd table every
//!   descriptor is invalid, so a non-`MAP_ANONYMOUS` mapping returns `EBADF` —
//!   which is what the C returns for an invalid descriptor, so the port is
//!   faithful for every input it can currently be given.
//! * **`struct mm`'s procfs fields and `exefile`.** `argv_start`, `env_end`,
//!   `auxv_*`, `stack_start` and `vdso` exist to render `/proc/pid/maps` and to
//!   hold the executable's fd; nothing ported reads them.
//!
//! The C's `write_wrlock`/`write_wrunlock` around each entry point have no
//! counterpart: there is one thread.

use crate::memory::{Mem, P_ANONYMOUS, P_RWX, P_SHARED, P_WRITE, ENOMEM};
use crate::mmu::{page, page_round_up, pgoffset, BAD_PAGE, MEM_PAGES, PAGE_BITS};
use crate::user::User;

/// `MMAP_SHARED`
pub const MMAP_SHARED: u32 = 0x1;
/// `MMAP_PRIVATE`
pub const MMAP_PRIVATE: u32 = 0x2;
/// `MMAP_FIXED`
pub const MMAP_FIXED: u32 = 0x10;
/// `MMAP_ANONYMOUS`
pub const MMAP_ANONYMOUS: u32 = 0x20;
/// `MREMAP_MAYMOVE_`
pub const MREMAP_MAYMOVE: u32 = 1;
/// `MREMAP_FIXED_`
pub const MREMAP_FIXED: u32 = 2;

// guest (i386 Linux) errno numbers, as kernel/errno.h defines them
const EFAULT: i32 = -14;
const EBADF: i32 = -9;
const EINVAL: i32 = -22;

/// `struct mm`, minus the procfs fields and `exefile` (see the module docs).
pub struct Mm {
    /// `mem`
    pub mem: Mem,
    /// `start_brk` — immutable after exec; `brk` refuses to go below it
    pub start_brk: u32,
    /// `brk`
    pub brk: u32,
    /// `refcount`. Rust would drop the space for us, but the C's counting is
    /// part of how fork and exec share an address space, so it is kept and
    /// [`Mm::release`] reports whether the count hit zero.
    pub refcount: usize,
}

impl Mm {
    /// `mm_new`
    pub fn new() -> Mm {
        Mm {
            mem: Mem::new(),
            // "should get overwritten by exec"
            start_brk: 0,
            brk: 0,
            refcount: 1,
        }
    }

    /// `mm_copy` — share everything copy-on-write, the way `fork` does.
    pub fn copy(&mut self) -> Mm {
        let mut new_mm = Mm {
            mem: Mem::new(),
            start_brk: self.start_brk,
            brk: self.brk,
            refcount: 1,
        };
        Mem::copy_on_write(&mut self.mem, &mut new_mm.mem, 0, MEM_PAGES);
        new_mm
    }

    /// `mm_retain`
    pub fn retain(&mut self) {
        self.refcount += 1;
    }

    /// `mm_release` — true when the count reached zero and the C would have
    /// destroyed the space.
    pub fn release(&mut self) -> bool {
        self.refcount -= 1;
        self.refcount == 0
    }

    /// `do_mmap`
    fn do_mmap(
        &mut self,
        addr: u32,
        len: u32,
        prot: u32,
        flags: u32,
        offset: u32,
    ) -> Result<u32, i32> {
        let pages = page_round_up(len);
        if pages == 0 {
            return Err(EINVAL);
        }
        let page: u32;
        if addr != 0 {
            if pgoffset(addr) != 0 {
                return Err(EINVAL);
            }
            // the local `page` shadows the imported function of the same name,
            // exactly as the C's local `page` shadows its macro
            page = crate::mmu::page(addr);
            if flags & MMAP_FIXED == 0 && !self.mem.is_hole(page, pages) {
                // The C does `addr = 0;` here, which reads as "the hint is taken,
                // go find a hole instead" - but `page` was already assigned from
                // the hint and is never recomputed, and `addr` is not read again.
                // The assignment is dead: the mapping lands on the hint whether
                // or not it was free. Reproduced, not fixed.
            }
        } else {
            page = self.mem.find_hole(pages);
            if page == BAD_PAGE {
                return Err(ENOMEM);
            }
        }

        let mut prot = prot;
        if flags & MMAP_SHARED != 0 {
            prot |= P_SHARED;
        }

        if flags & MMAP_ANONYMOUS != 0 {
            let err = self.mem.map_nothing(page, pages, prot);
            if err < 0 {
                return Err(err);
            }
        } else {
            // `f_get(fd_no)`: there is no fd table in the port yet, so no
            // descriptor is valid and this is the C's answer for an invalid one.
            //
            // `offset` is therefore unused, which has a consequence worth
            // stating: mmap2's page-to-byte conversion (`offset << PAGE_BITS`)
            // and the old mmap's choice of the offset field over the fd field
            // are both untested, because nothing reads the value. They become
            // testable the day the fd table lands.
            let _ = offset;
            return Err(EBADF);
        }
        Ok(page << PAGE_BITS)
    }

    /// `mmap_common`
    fn mmap_common(
        &mut self,
        addr: u32,
        len: u32,
        prot: u32,
        flags: u32,
        offset: u32,
    ) -> Result<u32, i32> {
        if len == 0 {
            // redundant with do_mmap's `pages == 0` check, because
            // page_round_up(0) is 0. Kept because the C has it here, and because
            // the two checks would diverge the moment the rounding changed.
            return Err(EINVAL);
        }
        if prot & !P_RWX != 0 {
            return Err(EINVAL);
        }
        if flags & MMAP_PRIVATE != 0 && flags & MMAP_SHARED != 0 {
            return Err(EINVAL);
        }
        self.do_mmap(addr, len, prot, flags, offset)
    }

    /// `sys_mmap2` — `offset` is in pages, as the `2` in the name says.
    pub fn mmap2(
        &mut self,
        addr: u32,
        len: u32,
        prot: u32,
        flags: u32,
        offset: u32,
    ) -> Result<u32, i32> {
        self.mmap_common(addr, len, prot, flags, offset << PAGE_BITS)
    }

    /// `sys_mmap` — the old form, whose six arguments arrive as a struct in
    /// guest memory. `offset` there is in bytes, not pages.
    ///
    /// The read and the mapping are sequenced rather than overlapped: the C
    /// holds the address space's write lock across both, but a single thread
    /// does not need to, and Rust will not let one `&mut Mem` serve both.
    pub fn mmap(&mut self, args_addr: u32) -> Result<u32, i32> {
        let raw = {
            let mut raw = [0u8; 24];
            if User::new(&mut self.mem).read(args_addr, &mut raw).is_err() {
                return Err(EFAULT);
            }
            raw
        };
        let w = |i: usize| u32::from_le_bytes(raw[i * 4..i * 4 + 4].try_into().unwrap());
        self.mmap_common(w(0), w(1), w(2), w(3), w(5))
    }

    /// `sys_munmap`
    pub fn munmap(&mut self, addr: u32, len: u32) -> Result<(), i32> {
        if pgoffset(addr) != 0 {
            return Err(EINVAL);
        }
        if len == 0 {
            return Err(EINVAL);
        }
        let err = self.mem.unmap_always(page(addr), page_round_up(len));
        if err < 0 {
            return Err(EINVAL);
        }
        Ok(())
    }

    /// `sys_mremap`
    ///
    /// The C's grow path checks its pages with
    /// `if (entry == NULL && entry->flags != pt_flags)`, which dereferences
    /// `entry` in the branch that has just established it is `NULL`. Any sparse
    /// range therefore crashes the C rather than returning an error, so the
    /// reference generator cannot contain that case. The port implements what
    /// the check was plainly meant to be — unmapped *or* differently-flagged —
    /// and says so here because it is the one place this file departs from the
    /// letter of the original.
    pub fn mremap(
        &mut self,
        addr: u32,
        old_len: u32,
        new_len: u32,
        flags: u32,
    ) -> Result<u32, i32> {
        if pgoffset(addr) != 0 {
            return Err(EINVAL);
        }
        if flags & !(MREMAP_MAYMOVE | MREMAP_FIXED) != 0 {
            return Err(EINVAL);
        }
        if flags & MREMAP_FIXED != 0 {
            // FIXME("missing MREMAP_FIXED")
            return Err(EINVAL);
        }
        // note: PAGE, not PAGE_ROUND_UP - a sub-page length rounds down to zero
        let old_pages = page(old_len);
        let new_pages = page(new_len);

        // shrinking always works
        if new_pages <= old_pages {
            let err = self.mem.unmap(page(addr) + new_pages, old_pages - new_pages);
            if err < 0 {
                return Err(EFAULT);
            }
            return Ok(addr);
        }

        let pt_flags = match self.mem.pt(page(addr)) {
            Some(entry) => entry.flags,
            None => return Err(EFAULT),
        };
        for p in page(addr)..page(addr) + old_pages {
            match self.mem.pt(p) {
                None => return Err(EFAULT),
                Some(entry) if entry.flags != pt_flags => return Err(EFAULT),
                Some(_) => {}
            }
        }
        if pt_flags & P_ANONYMOUS == 0 {
            // FIXME("mremap grow on file mappings")
            return Err(EFAULT);
        }
        let extra_start = page(addr) + old_pages;
        let extra_pages = new_pages - old_pages;
        if !self.mem.is_hole(extra_start, extra_pages) {
            return Err(ENOMEM);
        }
        let err = self.mem.map_nothing(extra_start, extra_pages, pt_flags);
        if err < 0 {
            return Err(err);
        }
        Ok(addr)
    }

    /// `sys_mprotect`
    pub fn mprotect(&mut self, addr: u32, len: u32, prot: u32) -> Result<(), i32> {
        if pgoffset(addr) != 0 {
            return Err(EINVAL);
        }
        if prot & !P_RWX != 0 {
            return Err(EINVAL);
        }
        let pages = page_round_up(len);
        let err = self.mem.set_flags(page(addr), pages, prot);
        if err < 0 {
            return Err(err);
        }
        Ok(())
    }

    /// `sys_madvise` — "portable applications should not rely on linux's
    /// destructive semantics for MADV_DONTNEED", so this does nothing.
    pub fn madvise(&mut self, _addr: u32, _len: u32, _advice: u32) -> u32 {
        0
    }

    /// `sys_mbind`
    pub fn mbind(
        &mut self,
        _addr: u32,
        _len: u32,
        _mode: i32,
        _nodemask: u32,
        _maxnode: u32,
        _flags: u32,
    ) -> u32 {
        0
    }

    /// `sys_mlock`
    pub fn mlock(&mut self, _addr: u32, _len: u32) -> i32 {
        0
    }

    /// `sys_msync`
    pub fn msync(&mut self, _addr: u32, _len: u32, _flags: u32) -> i32 {
        0
    }

    /// `sys_brk` — returns the brk in force after the call, which is the old one
    /// whenever the request could not be honoured.
    pub fn brk(&mut self, new_brk: u32) -> u32 {
        if new_brk < self.start_brk {
            return self.brk;
        }
        let old_brk = self.brk;

        if new_brk > old_brk {
            // round up, because brk is "the first location after the end of the
            // uninitialized data segment": at 0x2000 page 0x2000 must stay
            // unmapped, and at 0x2001 it must not
            let start = page_round_up(old_brk);
            let size = page_round_up(new_brk) - page_round_up(old_brk);
            if !self.mem.is_hole(start, size) {
                return self.brk;
            }
            let err = self.mem.map_nothing(start, size, P_WRITE);
            if err < 0 {
                return self.brk;
            }
        } else if new_brk < old_brk {
            self.mem.unmap_always(page(new_brk), page(old_brk) - page(new_brk));
        }

        self.brk = new_brk;
        self.brk
    }
}

impl Default for Mm {
    fn default() -> Self {
        Mm::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::P_READ;

    fn with_heap() -> Mm {
        let mut mm = Mm::new();
        mm.start_brk = 0x1000 << PAGE_BITS;
        mm.brk = mm.start_brk;
        mm
    }

    #[test]
    fn an_anonymous_mapping_with_no_hint_finds_a_hole() {
        let mut mm = Mm::new();
        let at = mm.mmap2(0, 4096, P_RWX, MMAP_PRIVATE | MMAP_ANONYMOUS, 0).unwrap();
        assert_ne!(at, 0);
        assert_eq!(pgoffset(at), 0, "mmap returns a page-aligned address");
        assert!(mm.mem.pt(page(at)).is_some());
    }

    #[test]
    fn a_hint_that_is_already_taken_is_used_anyway() {
        // the C's `addr = 0` fallback is dead code: `page` was already taken from
        // the hint, so a non-FIXED mapping over an existing one replaces it
        let mut mm = Mm::new();
        let first = mm.mmap2(0x400 << PAGE_BITS, 4096, P_RWX, MMAP_ANONYMOUS, 0).unwrap();
        assert_eq!(first, 0x400 << PAGE_BITS);
        let second = mm.mmap2(0x400 << PAGE_BITS, 4096, P_READ, MMAP_ANONYMOUS, 0).unwrap();
        assert_eq!(second, first, "the hint is honoured even though it is not free");
        assert_eq!(mm.mem.pt(0x400).unwrap().flags & P_RWX, P_READ);
    }

    #[test]
    fn zero_length_and_unaligned_addresses_are_refused() {
        let mut mm = Mm::new();
        assert_eq!(mm.mmap2(0, 0, P_RWX, MMAP_ANONYMOUS, 0), Err(EINVAL));
        assert_eq!(mm.mmap2(0x400 << PAGE_BITS | 1, 4096, P_RWX, MMAP_ANONYMOUS, 0), Err(EINVAL));
        // a length under one page still rounds up to one page here
        assert!(mm.mmap2(0, 1, P_RWX, MMAP_ANONYMOUS, 0).is_ok());
    }

    #[test]
    fn protection_outside_rwx_and_both_share_flags_are_refused() {
        let mut mm = Mm::new();
        assert_eq!(mm.mmap2(0, 4096, P_RWX | 0x100, MMAP_ANONYMOUS, 0), Err(EINVAL));
        assert_eq!(
            mm.mmap2(0, 4096, P_RWX, MMAP_ANONYMOUS | MMAP_PRIVATE | MMAP_SHARED, 0),
            Err(EINVAL)
        );
    }

    #[test]
    fn a_file_backed_mapping_needs_an_fd_table_there_isnt_one_yet() {
        let mut mm = Mm::new();
        // no MMAP_ANONYMOUS bit, so this goes looking for a descriptor
        assert_eq!(mm.mmap2(0, 4096, P_READ, MMAP_PRIVATE, 0), Err(EBADF));
    }

    #[test]
    fn shared_mappings_carry_the_shared_flag() {
        let mut mm = Mm::new();
        let at = mm.mmap2(0, 4096, P_RWX, MMAP_SHARED | MMAP_ANONYMOUS, 0).unwrap();
        assert_eq!(mm.mem.pt(page(at)).unwrap().flags & P_SHARED, P_SHARED);
    }

    #[test]
    fn munmap_refuses_unaligned_and_empty_and_accepts_the_rest() {
        let mut mm = Mm::new();
        mm.mmap2(0x400 << PAGE_BITS, 8192, P_RWX, MMAP_ANONYMOUS, 0).unwrap();
        assert_eq!(mm.munmap(0x400 << PAGE_BITS | 8, 4096), Err(EINVAL));
        assert_eq!(mm.munmap(0x400 << PAGE_BITS, 0), Err(EINVAL));
        assert_eq!(mm.munmap(0x400 << PAGE_BITS, 8192), Ok(()));
        assert!(mm.mem.pt(0x400).is_none());
        // unmap_always does not mind holes
        assert_eq!(mm.munmap(0x400 << PAGE_BITS, 4096), Ok(()));
    }

    #[test]
    fn mremap_uses_page_not_page_round_up() {
        let mut mm = with_heap();
        mm.mmap2(0x400 << PAGE_BITS, 4096, P_RWX, MMAP_ANONYMOUS, 0).unwrap();
        // both lengths are under a page, so both round down to zero pages: this
        // is the shrink branch (new_pages <= old_pages), unmapping nothing
        assert_eq!(mm.mremap(0x400 << PAGE_BITS, 0x100, 0x200, 0), Ok(0x400 << PAGE_BITS));
        assert!(mm.mem.pt(0x400).is_some(), "nothing was unmapped");
    }

    #[test]
    fn mremap_grows_an_anonymous_mapping_in_place() {
        let mut mm = with_heap();
        mm.mmap2(0x400 << PAGE_BITS, 4096, P_RWX, MMAP_ANONYMOUS, 0).unwrap();
        assert_eq!(mm.mremap(0x400 << PAGE_BITS, 4096, 3 * 4096, 0), Ok(0x400 << PAGE_BITS));
        assert!(mm.mem.pt(0x401).is_some());
        assert!(mm.mem.pt(0x402).is_some());
        assert!(mm.mem.pt(0x403).is_none());
    }

    #[test]
    fn mremap_refuses_to_grow_where_something_already_is() {
        let mut mm = with_heap();
        mm.mmap2(0x400 << PAGE_BITS, 4096, P_RWX, MMAP_ANONYMOUS, 0).unwrap();
        mm.mmap2(0x401 << PAGE_BITS, 4096, P_RWX, MMAP_ANONYMOUS, 0).unwrap();
        assert_eq!(mm.mremap(0x400 << PAGE_BITS, 4096, 2 * 4096, 0), Err(ENOMEM));
    }

    #[test]
    fn mremap_rejects_fixed_and_unknown_flags() {
        let mut mm = with_heap();
        assert_eq!(mm.mremap(0x400 << PAGE_BITS, 4096, 8192, MREMAP_FIXED), Err(EINVAL));
        assert_eq!(mm.mremap(0x400 << PAGE_BITS, 4096, 8192, 4), Err(EINVAL));
        assert_eq!(mm.mremap(0x400 << PAGE_BITS | 1, 4096, 8192, 0), Err(EINVAL));
    }

    #[test]
    fn mprotect_changes_protection_and_reports_holes() {
        let mut mm = with_heap();
        mm.mmap2(0x400 << PAGE_BITS, 4096, P_RWX, MMAP_ANONYMOUS, 0).unwrap();
        assert_eq!(mm.mprotect(0x400 << PAGE_BITS, 4096, P_READ), Ok(()));
        assert_eq!(mm.mem.pt(0x400).unwrap().flags, P_READ);
        assert_eq!(mm.mprotect(0x500 << PAGE_BITS, 4096, P_READ), Err(ENOMEM));
        assert_eq!(mm.mprotect(0x400 << PAGE_BITS, 4096, P_RWX | 0x100), Err(EINVAL));
    }

    #[test]
    fn brk_grows_the_heap_and_shrinks_it_back() {
        let mut mm = with_heap();
        assert_eq!(mm.brk(mm.start_brk + 0x2000), mm.start_brk + 0x2000);
        // "if the brk is 0x2000, page 0x2000 shouldn't be mapped, but it should
        // be if the brk is 0x2001"
        assert!(mm.mem.pt(page(mm.start_brk)).is_some());
        assert!(mm.mem.pt(page(mm.start_brk) + 1).is_some());
        assert_eq!(mm.brk(mm.start_brk), mm.start_brk);
        assert!(mm.mem.pt(page(mm.start_brk)).is_none(), "shrinking unmapped it");
    }

    #[test]
    fn brk_reports_the_old_value_when_it_cannot_help() {
        let mut mm = with_heap();
        mm.brk(mm.start_brk + 0x2000);
        // below start_brk
        assert_eq!(mm.brk(mm.start_brk - 1), mm.start_brk + 0x2000);
        assert_eq!(mm.brk, mm.start_brk + 0x2000, "and the brk did not move");
    }

    #[test]
    fn brk_will_not_grow_into_something_else() {
        let mut mm = with_heap();
        mm.mmap2(page_round_up(mm.start_brk + 0x1000) << PAGE_BITS, 4096, P_RWX, MMAP_ANONYMOUS, 0)
            .unwrap();
        let before = mm.brk;
        assert_eq!(mm.brk(mm.start_brk + 0x4000), before, "the hole is not free");
    }

    #[test]
    fn mmap_with_no_hint_fails_when_the_scan_finds_no_hole() {
        // pt_find_hole scans 0xf7ffd down to 0x40000. Filling every page it looks
        // at is the only way to reach its BAD_PAGE answer, and with it the
        // ENOMEM branch in do_mmap. The buffer is 3 GB of never-touched zero
        // pages: alloc_zeroed mmaps it lazily, so this costs address space.
        let mut mm = Mm::new();
        let _ = mm.mem.map_nothing(0x4_0001, 0xf_7ffd - 0x4_0001 + 1, P_RWX);
        assert_eq!(mm.mmap2(0, 4096, P_RWX, MMAP_ANONYMOUS, 0), Err(ENOMEM));
        // with an explicit hint there is no scan, so it still works
        assert!(mm.mmap2(0x100 << PAGE_BITS, 4096, P_RWX, MMAP_ANONYMOUS, 0).is_ok());
    }

    #[test]
    fn the_no_op_syscalls_succeed() {
        let mut mm = Mm::new();
        assert_eq!(mm.madvise(0x400 << PAGE_BITS, 4096, 4), 0);
        assert_eq!(mm.mbind(0, 0, 0, 0, 0, 0), 0);
        assert_eq!(mm.mlock(0, 0), 0);
        assert_eq!(mm.msync(0, 0, 0), 0);
    }

    #[test]
    fn copying_an_address_space_shares_it_copy_on_write() {
        let mut mm = with_heap();
        mm.mmap2(0x400 << PAGE_BITS, 4096, P_RWX, MMAP_ANONYMOUS, 0).unwrap();
        assert_eq!(mm.refcount, 1);
        // mm_copy does not retain the source; the caller does, as fork does
        mm.retain();
        assert_eq!(mm.refcount, 2);
        let child = mm.copy();
        assert!(child.mem.pt(0x400).is_some());
        assert_eq!(child.brk, mm.brk);
        assert_eq!(child.refcount, 1, "the copy starts at one");
        assert!(!mm.release(), "one reference is still held");
        assert_eq!(mm.refcount, 1);
        assert!(mm.release(), "and now the C would destroy the space");
    }

    #[test]
    fn the_old_mmap_reads_its_arguments_out_of_guest_memory() {
        let mut mm = Mm::new();
        // a page to hold the argument struct
        mm.mem.map_nothing(0x100, 1, P_RWX);
        let args: [u32; 6] = [0, 4096, P_RWX, MMAP_PRIVATE | MMAP_ANONYMOUS, 0, 0];
        let raw: Vec<u8> = args.iter().flat_map(|w| w.to_le_bytes()).collect();
        User::new(&mut mm.mem).write(0x100 << PAGE_BITS, &raw).unwrap();
        let at = mm.mmap(0x100 << PAGE_BITS).unwrap();
        assert!(mm.mem.pt(page(at)).is_some());
        // and a fault reading the arguments is EFAULT
        assert_eq!(mm.mmap(0x900 << PAGE_BITS), Err(EFAULT));
    }
}
