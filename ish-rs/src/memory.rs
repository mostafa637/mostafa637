// Line-by-line translation of kernel/memory.c and kernel/memory.h.
//
// This is the guest address space: a two-level page table over 2^20 pages, with
// reference-counted backing objects, copy-on-write, and the grows-down rule
// that lets a stack region claim the hole below it. It is the backend that
// `mmu.rs` only declares and `tlb.rs` caches the results of, so with this the
// emulator's memory path is complete from `mem_ptr` down to a host pointer.
//
// Two things are deliberately different, both because the alternative needs
// parts of iSH that are not ported:
//
//   * The C's `struct mem` embeds `struct mmu` and `mem_mmu_ops.translate`
//     recovers it with `container_of`. Here [`Mem`] *implements* [`MmuOps`], so
//     there is nothing to recover. The change counter the C keeps in
//     `mem->mmu.changes` lives on `Mem` and is reconciled with an [`Mmu`]
//     through `Mmu::sync_changes`.
//   * Backing memory is an owned allocation rather than an `mmap` region. The
//     page-table logic is identical, and protection is still enforced by the
//     flag checks in `mem_ptr` — which is what the emulator actually consults.
//     What is lost is the host `mprotect` in `pt_set_flags`, which only ever
//     *raises* protection on the host mapping; there is no host mapping here to
//     raise it on, so that call cannot fail and the `_ENOMEM` it could return
//     is unreachable. `struct pt_entry`'s `struct list blocks[2]` is likewise
//     absent: it exists so the JIT can find compiled blocks touching a page,
//     and the JIT is not ported. `asbestos_invalidate_page` becomes a counter
//     for the same reason, which the reference generator also counts.

use std::rc::Rc;

use crate::mmu::{MemType, MmuOps, PAGE_BITS, PAGE_SIZE};
use std::ptr::NonNull;

/// `MEM_PAGES` — at least on 32-bit.
pub const MEM_PAGES: u32 = 1 << 20;
/// `MEM_PGDIR_SIZE`
pub const MEM_PGDIR_SIZE: usize = 1 << 10;
/// `BAD_PAGE`
pub const BAD_PAGE: u32 = 0x1_0000;

/// `_ENOMEM`
pub const ENOMEM: i32 = -12;
/// `SEGV_MAPERR_`
pub const SEGV_MAPERR: i32 = 1;
/// `SEGV_ACCERR_`
pub const SEGV_ACCERR: i32 = 2;

/// `P_READ`
pub const P_READ: u32 = 1 << 0;
/// `P_WRITE`
pub const P_WRITE: u32 = 1 << 1;
/// `P_EXEC`
pub const P_EXEC: u32 = 1 << 2;
/// `P_RWX`
pub const P_RWX: u32 = P_READ | P_WRITE | P_EXEC;
/// `P_GROWSDOWN`
pub const P_GROWSDOWN: u32 = 1 << 3;
/// `P_COW`
pub const P_COW: u32 = 1 << 4;
/// `P_ANONYMOUS` — set by `pt_map_nothing`
pub const P_ANONYMOUS: u32 = 1 << 6;
/// `P_SHARED` — `MAP_SHARED`, so it must not be copied on write
pub const P_SHARED: u32 = 1 << 7;

/// `P_WRITABLE(flags)` — `flags & P_WRITE && !(flags & P_COW)`
pub fn p_writable(flags: u32) -> bool {
    flags & P_WRITE != 0 && flags & P_COW == 0
}

/// `BYTES_ROUND_DOWN(bytes)`
pub const fn bytes_round_down(bytes: u32) -> u32 {
    (bytes >> PAGE_BITS) << PAGE_BITS
}
/// `BYTES_ROUND_UP(bytes)`
pub const fn bytes_round_up(bytes: u32) -> u32 {
    ((bytes + PAGE_SIZE - 1) >> PAGE_BITS) << PAGE_BITS
}

/// `struct data` — the reference-counted backing object behind a mapping.
///
/// The C holds a host pointer from `mmap` plus the size to `munmap` it with.
/// Here the bytes are owned. The C's `fd`, `file_offset` and `name` exist only
/// to render `/proc/pid/maps` and are not ported with it.
pub struct Data {
    /// The backing bytes. Immutable once created, exactly as the C's `data`.
    pub bytes: Box<[u8]>,
    /// `size` — `pages * PAGE_SIZE + offset`, kept because it is what the C
    /// would `munmap` and because it is part of the mapping's identity.
    pub size: usize,
}

/// `struct pt_entry`
pub struct PtEntry {
    /// `data`. One strong reference per mapped page, so `Rc::strong_count` is
    /// exactly the C's hand-maintained `data->refcount`.
    pub data: Rc<Data>,
    /// `offset` — where this page starts inside the backing object.
    pub offset: usize,
    /// `flags`
    pub flags: u32,
}

/// `PGDIR_TOP(page)`
const fn pgdir_top(page: u32) -> usize {
    (page >> 10) as usize
}
/// `PGDIR_BOTTOM(page)`
const fn pgdir_bottom(page: u32) -> usize {
    (page & (MEM_PGDIR_SIZE as u32 - 1)) as usize
}

/// `struct mem`
pub struct Mem {
    pgdir: Vec<Option<Box<[Option<PtEntry>]>>>,
    /// `pgdir_used`
    pub pgdir_used: i32,
    /// The C's `mem->mmu.changes`, bumped by `mem_changed`.
    changes: u64,
    /// `asbestos_invalidate_page` calls. The JIT is not ported; the count is
    /// kept so the reference generator can compare that the same invalidations
    /// happen at the same points.
    pub invalidations: u64,
}

impl Default for Mem {
    fn default() -> Self {
        Self::new()
    }
}

impl Mem {
    /// `mem_init`
    pub fn new() -> Mem {
        Mem {
            pgdir: (0..MEM_PGDIR_SIZE).map(|_| None).collect(),
            pgdir_used: 0,
            changes: 0,
            invalidations: 0,
        }
    }

    /// `mem->mmu.changes`
    pub fn changes(&self) -> u64 {
        self.changes
    }

    /// `mem_changed` — increment the change count
    fn changed(&mut self) {
        self.changes += 1;
    }

    /// `asbestos_invalidate_page`
    fn invalidate_page(&mut self, _page: u32) {
        self.invalidations += 1;
    }

    /// `mem_pt` — the page table entry for a page, or `None` if the page is not
    /// mapped. Note this is `None` for a page whose directory was never
    /// allocated *and* for one whose entry has no data.
    pub fn pt(&self, page: u32) -> Option<&PtEntry> {
        let dir = self.pgdir[pgdir_top(page)].as_ref()?;
        let entry = dir[pgdir_bottom(page)].as_ref()?;
        Some(entry)
    }

    fn pt_mut(&mut self, page: u32) -> Option<&mut PtEntry> {
        let dir = self.pgdir[pgdir_top(page)].as_mut()?;
        dir[pgdir_bottom(page)].as_mut()
    }

    /// `mem_pt_new` — like `mem_pt` but allocates the directory if needed.
    fn pt_new(&mut self, page: u32) -> &mut Option<PtEntry> {
        let top = pgdir_top(page);
        if self.pgdir[top].is_none() {
            self.pgdir[top] =
                Some((0..MEM_PGDIR_SIZE).map(|_| None).collect::<Box<[_]>>());
            self.pgdir_used += 1;
        }
        &mut self.pgdir[top].as_mut().unwrap()[pgdir_bottom(page)]
    }

    /// `mem_pt_del` — dropping the `Rc` is the C's `--data->refcount` and, when
    /// it reaches zero, its `munmap` + `free`.
    fn pt_del(&mut self, page: u32) {
        if let Some(dir) = self.pgdir[pgdir_top(page)].as_mut() {
            dir[pgdir_bottom(page)] = None;
        }
    }

    /// `mem_next_page` — increment, skipping over unallocated page directories.
    /// Intended as the increment of a loop that walks mappings.
    pub fn next_page(&self, page: &mut u32) {
        *page += 1;
        if *page >= MEM_PAGES {
            return;
        }
        while *page < MEM_PAGES && self.pgdir[pgdir_top(*page)].is_none() {
            *page = (*page - pgdir_bottom(*page) as u32) + MEM_PGDIR_SIZE as u32;
        }
    }

    /// `pt_find_hole`
    ///
    /// The C's comment on this loop is "I don't know how this works but it
    /// does", and the port keeps it exactly as written rather than tidying it:
    /// it walks downwards from `0xf7ffd` to `0x40000`, and returns the *bottom*
    /// page of the first hole it finds that is exactly `size` pages long.
    pub fn find_hole(&self, size: u32) -> u32 {
        let mut hole_end: u32 = 0; // never read before it is set; gcc worried too
        let mut in_hole = false;
        let mut page: u32 = 0xf_7ffd;
        while page > 0x4_0000 {
            if !in_hole && self.pt(page).is_none() {
                in_hole = true;
                hole_end = page + 1;
            }
            if self.pt(page).is_some() {
                in_hole = false;
            } else if hole_end - page == size {
                // `==` and `>=` are the same test here: while the scan stays in
                // a hole this expression grows by exactly one per page, so it
                // cannot step over `size` without landing on it. The C's `==`
                // is kept because it is what the C says. This is the one
                // mutation in the sweep that no test can catch.
                return page;
            }
            page -= 1;
        }
        BAD_PAGE
    }

    /// `pt_is_hole`
    pub fn is_hole(&self, start: u32, pages: u32) -> bool {
        for page in start..start + pages {
            if self.pt(page).is_some() {
                return false;
            }
        }
        true
    }

    /// `pt_map` — map `bytes` at `start`, replacing anything already there, and
    /// take ownership of the backing object.
    pub fn map(
        &mut self,
        start: u32,
        pages: u32,
        bytes: Box<[u8]>,
        offset: usize,
        flags: u32,
    ) -> i32 {
        // `data->size = pages * PAGE_SIZE + offset`
        let size = (pages as usize) << PAGE_BITS;
        let data = Rc::new(Data { bytes, size: size + offset });
        for page in start..start + pages {
            if self.pt(page).is_some() {
                self.unmap(page, 1);
            }
            *self.pt_new(page) = Some(PtEntry {
                data: Rc::clone(&data),
                offset: (((page - start) as usize) << PAGE_BITS) + offset,
                flags,
            });
        }
        0
    }

    /// `pt_map_nothing` — map freshly zeroed pages.
    pub fn map_nothing(&mut self, start: u32, pages: u32, flags: u32) -> i32 {
        if pages == 0 {
            return 0;
        }
        let bytes = vec![0u8; (pages as usize) << PAGE_BITS].into_boxed_slice();
        self.map(start, pages, bytes, 0, flags | P_ANONYMOUS)
    }

    /// `pt_unmap` — `-1` if any part of the range is not mapped, and then
    /// nothing is unmapped at all.
    pub fn unmap(&mut self, start: u32, pages: u32) -> i32 {
        for page in start..start + pages {
            if self.pt(page).is_none() {
                return -1;
            }
        }
        self.unmap_always(start, pages)
    }

    /// `pt_unmap_always` — unmap whatever is there and do not complain about
    /// the holes.
    pub fn unmap_always(&mut self, start: u32, pages: u32) -> i32 {
        let mut page = start;
        while page < start + pages {
            if self.pt(page).is_some() {
                self.invalidate_page(page);
                self.pt_del(page);
            }
            self.next_page(&mut page);
        }
        self.changed();
        0
    }

    /// `pt_set_flags`
    ///
    /// The C follows the flag change with an `mprotect` when protection is
    /// increasing, so the host mapping matches the guest's. There is no host
    /// mapping here, so that step is absent and cannot fail; the guest-side
    /// checks in [`Mem::ptr`] are what actually enforce protection.
    pub fn set_flags(&mut self, start: u32, pages: u32, flags: u32) -> i32 {
        for page in start..start + pages {
            if self.pt(page).is_none() {
                return ENOMEM;
            }
        }
        for page in start..start + pages {
            if let Some(entry) = self.pt_mut(page) {
                entry.flags = flags;
            }
        }
        self.changed();
        0
    }

    /// `pt_copy_on_write` — share `src`'s pages with `dst`, marking both
    /// copy-on-write unless they are `MAP_SHARED`.
    pub fn copy_on_write(src: &mut Mem, dst: &mut Mem, start: u32, pages: u32) -> i32 {
        let mut page = start;
        while page < start + pages {
            // everything the loop body needs from src, taken before touching dst
            let shared = match src.pt(page) {
                None => {
                    src.next_page(&mut page);
                    continue;
                }
                Some(entry) => (Rc::clone(&entry.data), entry.offset, entry.flags),
            };
            dst.unmap_always(page, 1);
            let (data, offset, mut flags) = shared;
            if flags & P_SHARED == 0 {
                flags |= P_COW;
                // the C sets P_COW on the source entry, which is the same entry
                // this loop is holding
                if let Some(entry) = src.pt_mut(page) {
                    entry.flags |= P_COW;
                }
            }
            *dst.pt_new(page) = Some(PtEntry { data, offset, flags });
            src.next_page(&mut page);
        }
        src.changed();
        dst.changed();
        0
    }

    /// `mem_ptr_nofault` — the version that returns `None` instead of making
    /// page table changes. This is what the TLB calls, so that translating an
    /// address can never deadlock against a mapping change.
    pub fn ptr_nofault(&mut self, addr: u32, type_: MemType) -> Option<NonNull<u8>> {
        let entry = self.pt(page(addr))?;
        if type_ == MemType::Write && !p_writable(entry.flags) {
            return None;
        }
        // SAFETY: the pointer is into the backing object's bytes, at an offset
        // the mapping was built to keep in range. It stays valid for as long as
        // the page stays mapped - the same contract the C gives its TLB, which
        // re-checks the change count before reusing a cached entry.
        let base = Rc::as_ptr(&entry.data);
        let ptr = unsafe { (*base).bytes.as_ptr().add(entry.offset + pgoffset(addr)) };
        NonNull::new(ptr as *mut u8)
    }

    /// `mem_ptr` — the faulting version: it will grow a region down into the
    /// hole below it and break copy-on-write, changing the page table as it goes.
    pub fn ptr(&mut self, addr: u32, type_: MemType) -> Option<NonNull<u8>> {
        // "just for an assert" in the C, and the same here
        let old_ptr = self.ptr_nofault(addr, type_).map(|p| p.as_ptr());

        let page = page(addr);
        if self.pt(page).is_none() {
            // page does not exist; look to see if the next VM region is willing
            // to grow down
            let mut p = page + 1;
            while p < MEM_PAGES && self.pt(p).is_none() {
                p += 1;
            }
            if p >= MEM_PAGES {
                return None;
            }
            if self.pt(p).map(|e| e.flags & P_GROWSDOWN == 0).unwrap_or(true) {
                return None;
            }
            // The C drops the read lock, takes the write lock, maps, and puts
            // them back. Single-threaded here, so there is nothing to drop.
            self.map_nothing(page, 1, P_WRITE | P_GROWSDOWN);
        }

        if self.pt(page).is_some()
            && (type_ == MemType::Write || type_ == MemType::WritePtrace)
        {
            let flags = self.pt(page).unwrap().flags;
            // if page is unwritable, well tough luck
            if type_ != MemType::WritePtrace && flags & P_WRITE == 0 {
                return None;
            }
            if type_ == MemType::WritePtrace {
                if let Some(entry) = self.pt_mut(page) {
                    entry.flags |= P_WRITE | P_COW;
                }
            }
            // get rid of any compiled blocks in this page
            self.invalidate_page(page);
            // if page is cow, ~~milk~~ copy it
            let flags = self.pt(page).unwrap().flags;
            if flags & P_COW != 0 {
                let entry = self.pt(page).unwrap();
                let copy = {
                    let src = &entry.data.bytes[entry.offset..entry.offset + PAGE_SIZE as usize];
                    let mut buf = vec![0u8; PAGE_SIZE as usize];
                    buf.copy_from_slice(src);
                    buf.into_boxed_slice()
                };
                // the copy is a fresh single-page object, so it is mapped at
                // offset 0 - not at the old page's offset into the shared one
                self.map(page, 1, copy, 0, flags & !P_COW);
            }
        }

        let ptr = self.ptr_nofault(addr, type_);
        debug_assert!(
            old_ptr.is_none()
                || old_ptr == ptr.map(|p| p.as_ptr())
                || type_ == MemType::WritePtrace,
            "a non-ptrace access must not move an already-translated pointer"
        );
        ptr
    }

    /// `mem_segv_reason`
    pub fn segv_reason(&self, addr: u32) -> i32 {
        if self.pt(page(addr)).is_none() {
            SEGV_MAPERR
        } else {
            SEGV_ACCERR
        }
    }
}

impl MmuOps for Mem {
    /// `mem_mmu_translate`
    fn translate(&mut self, addr: u32, type_: MemType) -> Option<NonNull<u8>> {
        self.ptr_nofault(addr, type_)
    }

    /// `mem->mmu.changes`
    fn changes(&self) -> u64 {
        self.changes
    }
}

/// `PAGE(addr)`
pub const fn page(addr: u32) -> u32 {
    addr >> PAGE_BITS
}
/// `PGOFFSET(addr)`
pub const fn pgoffset(addr: u32) -> usize {
    (addr & (PAGE_SIZE - 1)) as usize
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn the_flag_bits_are_the_c_values() {
        assert_eq!((P_READ, P_WRITE, P_EXEC), (1, 2, 4));
        assert_eq!(P_RWX, 7);
        assert_eq!(P_GROWSDOWN, 8);
        assert_eq!(P_COW, 16);
        assert_eq!(P_ANONYMOUS, 64);
        assert_eq!(P_SHARED, 128);
        // a CoW page is not writable even though the write bit is set
        assert!(p_writable(P_WRITE));
        assert!(!p_writable(P_WRITE | P_COW));
        assert!(!p_writable(P_READ));
    }

    #[test]
    fn the_directory_split_matches_the_macros() {
        assert_eq!(pgdir_top(0), 0);
        assert_eq!(pgdir_bottom(0), 0);
        assert_eq!(pgdir_top(1023), 0);
        assert_eq!(pgdir_bottom(1023), 1023);
        assert_eq!(pgdir_top(1024), 1);
        assert_eq!(pgdir_bottom(1024), 0);
        assert_eq!(MEM_PGDIR_SIZE * MEM_PGDIR_SIZE, MEM_PAGES as usize);
        assert_eq!(bytes_round_down(4097), 4096);
        assert_eq!(bytes_round_up(4097), 8192);
    }

    #[test]
    fn a_fresh_address_space_has_nothing_mapped() {
        let mem = Mem::new();
        assert_eq!(mem.pgdir_used, 0);
        assert!(mem.is_hole(0, MEM_PAGES));
        assert!(mem.pt(0).is_none());
        assert_eq!(mem.segv_reason(0x1000), SEGV_MAPERR);
        assert_eq!(mem.changes(), 0, "nothing has changed yet");
    }

    #[test]
    fn mapping_allocates_one_directory_and_shares_one_backing_object() {
        let mut mem = Mem::new();
        assert_eq!(mem.map_nothing(0x100, 4, P_RWX), 0);
        assert_eq!(mem.pgdir_used, 1);
        assert_eq!(mem.changes(), 0, "pt_map does not bump the change count");
        for i in 0..4u32 {
            let pt = mem.pt(0x100 + i).unwrap();
            assert_eq!(pt.offset, (i as usize) << PAGE_BITS);
            assert_eq!(pt.flags, P_RWX | P_ANONYMOUS);
            assert_eq!(Rc::strong_count(&pt.data), 4, "one object, four pages");
        }
    }

    #[test]
    fn an_unmap_that_is_partly_mapped_changes_nothing() {
        let mut mem = Mem::new();
        mem.map_nothing(0x100, 4, P_RWX);
        assert_eq!(mem.unmap(0x102, 4), -1);
        assert!(mem.pt(0x102).is_some(), "nothing was unmapped");
        assert_eq!(mem.unmap_always(0x102, 4), 0);
        assert!(mem.pt(0x102).is_none());
        assert!(mem.pt(0x103).is_none());
        assert_eq!(Rc::strong_count(&mem.pt(0x100).unwrap().data), 2);
    }

    #[test]
    fn next_page_skips_unallocated_directories() {
        let mut mem = Mem::new();
        mem.map_nothing(0x100, 1, P_RWX);
        mem.map_nothing(0x800, 1, P_RWX); // a different top-level directory
        // 0x101 is in the same (allocated) directory, so it is not skipped even
        // though nothing is mapped there
        let mut p = 0x100;
        mem.next_page(&mut p);
        assert_eq!(p, 0x101);
        // from the last page of directory 0 the whole empty directory 1 goes
        let mut q = 0x3ff;
        mem.next_page(&mut q);
        assert_eq!(q, 0x800, "the unallocated directory is stepped over");
        // 0x801 is in the same allocated directory, so it is returned as-is
        let mut r = 0x800;
        mem.next_page(&mut r);
        assert_eq!(r, 0x801);
        // and past the last page of the address space it stops at the end
        let mut end = MEM_PAGES - 1;
        mem.next_page(&mut end);
        assert_eq!(end, MEM_PAGES);
    }

    #[test]
    fn set_flags_refuses_an_unmapped_page_and_reports_enomem() {
        let mut mem = Mem::new();
        mem.map_nothing(0x300, 2, P_READ);
        assert_eq!(mem.set_flags(0x200, 1, P_READ), ENOMEM, "0x200 is not mapped");
        assert_eq!(mem.set_flags(0x301, 2, P_READ), ENOMEM, "0x302 is not mapped");
        assert_eq!(mem.set_flags(0x300, 1, P_READ), 0);
        assert_eq!(mem.pt(0x300).unwrap().flags, P_READ);
    }

    #[test]
    fn a_write_to_a_read_only_page_is_refused_and_says_why() {
        let mut mem = Mem::new();
        mem.map_nothing(0x300, 1, P_READ);
        let addr = 0x300 << PAGE_BITS;
        assert!(mem.ptr(addr, MemType::Write).is_none());
        assert!(mem.ptr(addr, MemType::Read).is_some());
        assert_eq!(mem.segv_reason(addr), SEGV_ACCERR, "mapped but not permitted");
        assert_eq!(mem.set_flags(0x300, 1, P_RWX), 0);
        assert!(mem.ptr(addr, MemType::Write).is_some());
    }

    #[test]
    fn copy_on_write_shares_until_somebody_writes() {
        let mut parent = Mem::new();
        let mut child = Mem::new();
        parent.map_nothing(0x300, 2, P_RWX);
        parent.ptr(0x300 << PAGE_BITS, MemType::Write);
        assert_eq!(Mem::copy_on_write(&mut parent, &mut child, 0x300, 2), 0);
        // both sides now share one object and both are marked CoW
        assert!(Rc::ptr_eq(
            &parent.pt(0x300).unwrap().data,
            &child.pt(0x300).unwrap().data
        ));
        assert_eq!(parent.pt(0x300).unwrap().flags & P_COW, P_COW);
        assert_eq!(child.pt(0x300).unwrap().flags & P_COW, P_COW);
        assert_eq!(Rc::strong_count(&parent.pt(0x300).unwrap().data), 4);

        // a write in the child breaks the sharing for that page only
        let before = Rc::as_ptr(&child.pt(0x300).unwrap().data);
        child.ptr(0x300 << PAGE_BITS, MemType::Write);
        let after = Rc::as_ptr(&child.pt(0x300).unwrap().data);
        assert_ne!(before, after, "the child got its own copy");
        assert_eq!(child.pt(0x300).unwrap().flags & P_COW, 0);
        assert!(Rc::ptr_eq(
            &parent.pt(0x300).unwrap().data,
            &child.pt(0x301).unwrap().data
        ), "the untouched page is still shared");
    }

    #[test]
    fn a_ptrace_write_bypasses_the_write_bit_and_leaves_the_page_cow() {
        let mut mem = Mem::new();
        mem.map_nothing(0x400, 1, P_READ);
        let addr = 0x400 << PAGE_BITS;
        assert!(mem.ptr(addr, MemType::WritePtrace).is_some());
        // the ptrace write sets P_WRITE|P_COW and then immediately breaks the
        // CoW it just created, so what is left is a private writable page
        let pt = mem.pt(0x400).unwrap();
        assert_eq!(pt.flags & P_WRITE, P_WRITE);
        assert_eq!(pt.flags & P_COW, 0, "the copy already happened");
        assert!(p_writable(pt.flags));
        assert_eq!(pt.offset, 0, "the copy is a fresh one-page object");
        assert_eq!(Rc::strong_count(&pt.data), 1);
    }

    #[test]
    fn a_region_grows_down_into_the_hole_below_it() {
        let mut mem = Mem::new();
        mem.map_nothing(0x500, 1, P_RWX | P_GROWSDOWN);
        let below = 0x4ff << PAGE_BITS;
        assert!(mem.pt(0x4ff).is_none());
        assert!(mem.ptr(below, MemType::Read).is_some());
        let pt = mem.pt(0x4ff).unwrap();
        assert_eq!(pt.flags, P_WRITE | P_GROWSDOWN | P_ANONYMOUS);
        // with nothing above it, a hole stays a hole
        assert!(mem.ptr(0xf_0000 << PAGE_BITS, MemType::Read).is_none());
    }

    #[test]
    fn the_backing_object_remembers_how_big_it_really_is() {
        let mut mem = Mem::new();
        let bytes = vec![0u8; (2usize << PAGE_BITS) + 0x1000].into_boxed_slice();
        mem.map(0x600, 2, bytes, 0x1000, P_RWX);
        // `data->size = pages * PAGE_SIZE + offset`, which is the length the C
        // would munmap when the last reference goes away
        assert_eq!(mem.pt(0x600).unwrap().data.size, (2usize << PAGE_BITS) + 0x1000);
        assert_eq!(mem.pt(0x601).unwrap().offset, 0x2000);
    }

    #[test]
    fn find_hole_gives_up_below_the_floor_of_its_scan() {
        // The scan runs from 0xf7ffd down to 0x40000. Filling every page it
        // looks at forces the loop all the way to the floor, and a hole placed
        // *below* the floor must not be found - that is what the floor means.
        // The buffer is 3 GB of never-touched zero pages; alloc_zeroed mmaps it
        // lazily, so this costs address space, not memory.
        const FLOOR: u32 = 0x4_0000;
        const TOP: u32 = 0xf_7ffd;
        let mut mem = Mem::new();
        // leave exactly one 4-page hole at 0x40800, just above the floor
        let hole = 0x4_0800;
        // everything from just above the hole up to the top of the scan
        mem.map_nothing(hole + 4, TOP - (hole + 4) + 1, P_READ);
        // and everything from the floor up to the hole
        mem.map_nothing(FLOOR, hole - FLOOR, P_READ);
        assert_eq!(mem.find_hole(4), hole, "the only hole in range is found");

        // now cover the hole too: nothing is left in range
        mem.map_nothing(hole, 4, P_READ);
        assert_eq!(mem.find_hole(4), BAD_PAGE, "the scan gives up at the floor");
        // the floor itself is not scanned, so a hole exactly at it is invisible
        mem.unmap_always(FLOOR, 4);
        assert_eq!(mem.find_hole(4), BAD_PAGE, "pages at the floor are never examined");
    }

    #[test]
    fn a_hole_bigger_than_the_request_gives_back_its_top_pages() {
        // `find_hole` returns as soon as the run reaches `size`, so what comes
        // back is the *top* `size` pages of the hole - the bottom stays free.
        // This is also why `== size` and `>= size` cannot differ: the run length
        // passes through every value on its way up.
        // the scan starts at the top, so take the four pages it would find
        // first and the hole below them is the one under test
        let mut mem = Mem::new();
        mem.map_nothing(0xf_7ffa, 4, P_READ);
        // everything below 0xf7ff9 is free - a hole far bigger than any request
        assert_eq!(mem.find_hole(1), 0xf_7ff9);
        assert_eq!(mem.find_hole(4), 0xf_7ff6, "the top four pages of the hole");
        assert_eq!(mem.find_hole(8), 0xf_7ff2, "and the top eight");
    }

    #[test]
    fn find_hole_returns_the_bottom_of_an_exact_fit() {
        let mem = Mem::new();
        // everything below 0x40000 is empty, so a request walks down from
        // 0xf7ffd and lands at the top of the space
        let h = mem.find_hole(4);
        assert_ne!(h, BAD_PAGE);
        assert!(mem.is_hole(h, 4));
    }

    #[test]
    fn translate_goes_through_the_nofault_path_only() {
        // The TLB must never be able to change the page table, so translate()
        // has to refuse rather than grow a region down or break CoW.
        let mut mem = Mem::new();
        mem.map_nothing(0x500, 1, P_RWX | P_GROWSDOWN);
        let below = 0x4ff << PAGE_BITS;
        assert!(MmuOps::translate(&mut mem, below, MemType::Read).is_none());
        assert!(mem.pt(0x4ff).is_none(), "translate mapped nothing");
        let at = 0x500 << PAGE_BITS;
        assert!(MmuOps::translate(&mut mem, at, MemType::Read).is_some());
    }

    #[test]
    fn unmapping_bumps_the_change_count_so_the_tlb_flushes() {
        let mut mem = Mem::new();
        mem.map_nothing(0x100, 2, P_RWX);
        let before = mem.changes();
        mem.unmap_always(0x100, 2);
        assert_eq!(mem.changes(), before + 1);
        mem.set_flags(0x200, 0, P_RWX);
        // set_flags on an empty range still reports a change, as the C does
        assert_eq!(mem.changes(), before + 2);
    }
}
