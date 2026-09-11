// Line-by-line translation of emu/tlb.h + emu/tlb.c
//
// The TLB caches "guest page -> host pointer" so the interpreter's memory
// accesses are one array lookup and one add. Every entry stores the page it
// covers twice: `page` for reads and `page_if_writable` for writes, so a
// read-only mapping still serves reads but misses on writes.
//
// Signature changes from the C, none of which change behaviour:
//   * the C keeps `struct mmu *mmu` inside `struct tlb` and dereferences it.
//     Here the mmu is passed to each call, and the TLB keeps only an opaque
//     identity token so `tlb_refresh` can tell "same mmu" from "different
//     one". That keeps the borrow checker happy *and* lets a caller touch the
//     mmu (bump `changes`, remap) between operations, which the tests need.
//   * `tlb_read`/`tlb_write` take slices instead of `void *` + size.
//   * `tlb_free` is Rust's `Drop`.

use std::ptr::NonNull;

use crate::cpu::AddrT;
use crate::mmu::{page, pgoffset, MemType, Mmu, PageT, PAGE_BITS, PAGE_SIZE};

/// `#define TLB_BITS 10`
pub const TLB_BITS: u32 = 10;
/// `#define TLB_SIZE (1 << TLB_BITS)`
pub const TLB_SIZE: usize = 1 << TLB_BITS;
/// `#define TLB_PAGE_EMPTY 1` — "1 is not a valid page so this won't look like
/// a hit".
pub const TLB_PAGE_EMPTY: PageT = 1;

/// `#define TLB_INDEX(addr)`
pub const fn tlb_index(addr: AddrT) -> usize {
    (((addr >> PAGE_BITS) & (TLB_SIZE as u32 - 1)) ^ (addr >> (PAGE_BITS + TLB_BITS))) as usize
}

/// `#define TLB_PAGE(addr) (addr & 0xfffff000)`
pub const fn tlb_page(addr: AddrT) -> AddrT {
    addr & 0xffff_f000
}

/// `struct tlb_entry`
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct TlbEntry {
    pub page: PageT,
    pub page_if_writable: PageT,
    /// The C field is `uintptr_t data_minus_addr`: the host pointer minus the
    /// guest page address, so the gadgets get a byte pointer with a single
    /// add. Storing the page base is the same information; [`Self::data_minus_addr`]
    /// recovers the C value.
    pub data: Option<NonNull<u8>>,
}

impl TlbEntry {
    /// The initial value `tlb_flush` writes: `{.page = 1, .page_if_writable = 1}`.
    pub const EMPTY: TlbEntry = TlbEntry {
        page: TLB_PAGE_EMPTY,
        page_if_writable: TLB_PAGE_EMPTY,
        data: None,
    };

    /// `entry.data_minus_addr` as the C computes it.
    pub fn data_minus_addr(&self) -> Option<usize> {
        self.data.map(|d| d.as_ptr() as usize - self.page as usize)
    }

    /// `entry.data_minus_addr + addr` — the host pointer for a guest address.
    fn host_ptr(&self, addr: AddrT) -> Option<NonNull<u8>> {
        let base = self.data?;
        // addr - TLB_PAGE(addr) == PGOFFSET(addr)
        let off = pgoffset(addr) as usize;
        // SAFETY: `base` came from a backend that mapped this whole page, and
        // `off` is inside it by construction.
        Some(unsafe { NonNull::new_unchecked(base.as_ptr().add(off)) })
    }
}

/// `struct tlb`
pub struct Tlb {
    /// Identity of the `Mmu` this TLB was last refreshed against. Only ever
    /// compared, never dereferenced.
    mmu_id: *const (),
    pub dirty_page: PageT,
    pub mem_changes: u32,
    /// "this is basically one of the return values of tlb_handle_miss,
    /// tlb_{read,write}, and __tlb_{read,write}_cross_page. yes, this sucks"
    pub segfault_addr: AddrT,
    pub entries: [TlbEntry; TLB_SIZE],
}

/// `tlb->mmu == mmu`
fn same_mmu(a: *const (), mmu: &Mmu) -> bool {
    std::ptr::eq(a, mmu as *const Mmu as *const ())
}

impl Tlb {
    /// An empty TLB.
    ///
    /// The C gets this from `calloc` (all zeroes) and then always calls
    /// `tlb_refresh` before the first access. A *zeroed* entry would be a
    /// hazard — `page == 0` matches `TLB_PAGE(addr)` for every address in
    /// guest page 0, so `__tlb_read_ptr` would report a hit and hand back a
    /// wild pointer. This starts in the post-flush state instead, which is
    /// what the first `tlb_refresh` writes anyway, so the difference is never
    /// observable — but it cannot produce that bad hit either.
    pub fn new() -> Self {
        Tlb {
            mmu_id: std::ptr::null(),
            dirty_page: 0,
            mem_changes: 0,
            segfault_addr: 0,
            entries: [TlbEntry::EMPTY; TLB_SIZE],
        }
    }

    /// `void tlb_refresh(struct tlb *tlb, struct mmu *mmu)`
    pub fn refresh(&mut self, mmu: &mut Mmu) {
        if same_mmu(self.mmu_id, mmu) && self.mem_changes as u64 == mmu.changes {
            return;
        }
        self.mmu_id = mmu as *const Mmu as *const ();
        self.dirty_page = TLB_PAGE_EMPTY;
        self.mem_changes = mmu.changes as u32;
        self.flush(mmu);
    }

    /// `void tlb_flush(struct tlb *tlb)`
    ///
    /// Note this reads `tlb->mmu->changes`, so unlike the C it needs the mmu.
    pub fn flush(&mut self, mmu: &Mmu) {
        self.mem_changes = mmu.changes as u32;
        for entry in self.entries.iter_mut() {
            *entry = TlbEntry::EMPTY;
        }
    }

    /// `void *tlb_handle_miss(struct tlb *tlb, addr_t addr, int type)`
    pub fn handle_miss(&mut self, mmu: &mut Mmu, addr: AddrT, type_: MemType) -> Option<NonNull<u8>> {
        // The order is the C's, and it matters: translate runs *first*, and
        // the changes check happens after it. The real backend
        // (kernel/memory.c) can install a mapping and bump `changes` from
        // inside translate when it faults a page in, so checking afterwards is
        // what leaves `mem_changes` resynced with the mapping we just cached.
        let ptr = mmu.translate(tlb_page(addr), type_);
        if mmu.changes != self.mem_changes as u64 {
            self.flush(mmu);
        }
        let ptr = match ptr {
            Some(p) => p,
            None => {
                // C: `tlb->segfault_addr = addr; return NULL;`
                self.segfault_addr = addr;
                return None;
            }
        };
        self.dirty_page = tlb_page(addr);

        let entry = &mut self.entries[tlb_index(addr)];
        entry.page = tlb_page(addr);
        entry.page_if_writable = if type_ == MemType::Write {
            entry.page
        } else {
            TLB_PAGE_EMPTY
        };
        entry.data = Some(ptr);
        entry.host_ptr(addr)
    }

    /// `void *__tlb_read_ptr(struct tlb *tlb, addr_t addr)`
    pub fn read_ptr(&mut self, mmu: &mut Mmu, addr: AddrT) -> Option<NonNull<u8>> {
        let entry = self.entries[tlb_index(addr)];
        if entry.page == tlb_page(addr) {
            let p = entry.host_ptr(addr);
            debug_assert!(p.is_some(), "posit(address != NULL)");
            return p;
        }
        self.handle_miss(mmu, addr, MemType::Read)
    }

    /// `void *__tlb_write_ptr(struct tlb *tlb, addr_t addr)`
    pub fn write_ptr(&mut self, mmu: &mut Mmu, addr: AddrT) -> Option<NonNull<u8>> {
        let entry = self.entries[tlb_index(addr)];
        if entry.page_if_writable == tlb_page(addr) {
            self.dirty_page = tlb_page(addr);
            let p = entry.host_ptr(addr);
            debug_assert!(p.is_some(), "posit(address != NULL)");
            return p;
        }
        self.handle_miss(mmu, addr, MemType::Write)
    }

    /// `bool __tlb_read_cross_page(struct tlb *tlb, addr_t addr, char *out, unsigned size)`
    pub fn read_cross_page(&mut self, mmu: &mut Mmu, addr: AddrT, out: &mut [u8]) -> bool {
        let size = out.len();
        let ptr1 = match self.read_ptr(mmu, addr) {
            Some(p) => p,
            None => return false,
        };
        let next_page = (page(addr).wrapping_add(1)) << PAGE_BITS;
        let ptr2 = match self.read_ptr(mmu, next_page) {
            Some(p) => p,
            None => return false,
        };
        let part1 = (PAGE_SIZE - pgoffset(addr)) as usize;
        debug_assert!(part1 < size);
        // SAFETY: ptr1 covers the rest of its page (part1 bytes) and ptr2
        // covers the start of the next one; the caller's slice is `size` long.
        unsafe {
            std::ptr::copy_nonoverlapping(ptr1.as_ptr(), out.as_mut_ptr(), part1);
            std::ptr::copy_nonoverlapping(ptr2.as_ptr(), out[part1..].as_mut_ptr(), size - part1);
        }
        true
    }

    /// `bool tlb_read(struct tlb *tlb, addr_t addr, void *out, unsigned size)`
    pub fn read(&mut self, mmu: &mut Mmu, addr: AddrT, out: &mut [u8]) -> bool {
        let size = out.len() as u32;
        // C: `PGOFFSET(addr) > PAGE_SIZE - size`, with unsigned arithmetic, so
        // a size larger than a page wraps and takes the fast path instead.
        if pgoffset(addr) > PAGE_SIZE.wrapping_sub(size) {
            return self.read_cross_page(mmu, addr, out);
        }
        let ptr = match self.read_ptr(mmu, addr) {
            Some(p) => p,
            None => return false,
        };
        // SAFETY: the whole range lies inside one translated page.
        unsafe {
            std::ptr::copy_nonoverlapping(ptr.as_ptr(), out.as_mut_ptr(), out.len());
        }
        true
    }

    /// `bool __tlb_write_cross_page(struct tlb *tlb, addr_t addr, const char *value, unsigned size)`
    pub fn write_cross_page(&mut self, mmu: &mut Mmu, addr: AddrT, value: &[u8]) -> bool {
        let size = value.len();
        let ptr1 = match self.write_ptr(mmu, addr) {
            Some(p) => p,
            None => return false,
        };
        let next_page = (page(addr).wrapping_add(1)) << PAGE_BITS;
        let ptr2 = match self.write_ptr(mmu, next_page) {
            Some(p) => p,
            None => return false,
        };
        let part1 = (PAGE_SIZE - pgoffset(addr)) as usize;
        debug_assert!(part1 < size);
        // SAFETY: as in read_cross_page, both halves lie in translated pages.
        unsafe {
            std::ptr::copy_nonoverlapping(value.as_ptr(), ptr1.as_ptr(), part1);
            std::ptr::copy_nonoverlapping(value[part1..].as_ptr(), ptr2.as_ptr(), size - part1);
        }
        true
    }

    /// `bool tlb_write(struct tlb *tlb, addr_t addr, const void *value, unsigned size)`
    pub fn write(&mut self, mmu: &mut Mmu, addr: AddrT, value: &[u8]) -> bool {
        let size = value.len() as u32;
        if pgoffset(addr) > PAGE_SIZE.wrapping_sub(size) {
            return self.write_cross_page(mmu, addr, value);
        }
        let ptr = match self.write_ptr(mmu, addr) {
            Some(p) => p,
            None => return false,
        };
        // SAFETY: the whole range lies inside one translated page.
        unsafe {
            std::ptr::copy_nonoverlapping(value.as_ptr(), ptr.as_ptr(), value.len());
        }
        true
    }

    /// `void tlb_free(struct tlb *tlb)` — Rust's `Drop` does this.
    pub fn is_empty_entry(&self, i: usize) -> bool {
        self.entries[i].data.is_none()
    }
}

impl Default for Tlb {
    fn default() -> Self {
        Tlb::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::mmu::MmuOps;
    use std::cell::Cell;

    thread_local! {
        static TRANSLATE_CALLS: Cell<u32> = const { Cell::new(0) };
    }

    fn calls() -> u32 {
        TRANSLATE_CALLS.with(|c| c.get())
    }

    /// A backend covering `PAGES` pages, with per-page read-only and unmapped
    /// masks, mirroring the one the C reference uses.
    struct FakeMmu {
        backing: Vec<u8>,
        read_only: u32,
        unmapped: u32,
    }

    const PAGES: usize = 16;

    impl FakeMmu {
        fn new() -> Self {
            let mut backing = vec![0u8; PAGES * PAGE_SIZE as usize];
            for (i, b) in backing.iter_mut().enumerate() {
                *b = (i as u32).wrapping_mul(31).wrapping_add(7) as u8;
            }
            TRANSLATE_CALLS.with(|c| c.set(0));
            FakeMmu { backing, read_only: 0, unmapped: 0 }
        }
    }

    impl MmuOps for FakeMmu {
        fn translate(&mut self, addr: AddrT, type_: MemType) -> Option<NonNull<u8>> {
            TRANSLATE_CALLS.with(|c| c.set(c.get() + 1));
            let idx = (addr >> PAGE_BITS) as usize;
            if idx >= PAGES {
                return None;
            }
            if self.unmapped & (1 << idx) != 0 {
                return None;
            }
            if type_ == MemType::Write && self.read_only & (1 << idx) != 0 {
                return None;
            }
            NonNull::new(self.backing.as_mut_ptr().wrapping_add(idx << PAGE_BITS))
        }
    }

    #[test]
    fn tlb_index_is_the_c_macro() {
        // ((addr >> 12) & 1023) ^ (addr >> 22)
        assert_eq!(tlb_index(0), 0);
        assert_eq!(tlb_index(0x1000), 1);
        // (0x400 & 1023) ^ (0x400000 >> 22) == 0 ^ 1
        assert_eq!(tlb_index(0x40_0000), 1);
        // (0x401 & 1023) ^ (0x401000 >> 22) == 1 ^ 1
        assert_eq!(tlb_index(0x40_1000), 0);
        // The xor with the high bits is what *reduces* aliasing: pages 1 and
        // 1025 (4 MiB + 4 KiB apart) land in different slots...
        assert_ne!(tlb_index(0x1000), tlb_index(0x40_1000));
        // ...while pages 1 and 1024 still collide, since 1 ^ 1024 == 1 ^ 0.
        assert_eq!(tlb_index(0x1000), tlb_index(0x40_0000));
        assert_eq!(TLB_SIZE, 1024);
        assert_eq!(tlb_page(0x1234_5678), 0x1234_5000);
    }

    #[test]
    fn read_hits_the_cache_on_the_second_access() {
        let mut backend = FakeMmu::new();
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);

        let mut buf = [0u8; 4];
        assert!(tlb.read(&mut mmu, 0x2000, &mut buf));
        let first = calls();
        assert!(tlb.read(&mut mmu, 0x2004, &mut buf));
        assert_eq!(calls(), first, "same page must not re-translate");
        assert_eq!(tlb.entries[tlb_index(0x2000)].page, tlb_page(0x2000));
    }

    #[test]
    fn read_only_page_serves_reads_but_faults_on_writes() {
        let mut backend = FakeMmu::new();
        backend.read_only = 1 << 3;
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);

        let addr = 3 << PAGE_BITS;
        let mut buf = [0u8; 4];
        assert!(tlb.read(&mut mmu, addr, &mut buf), "read of a read-only page works");
        assert!(!tlb.write(&mut mmu, addr, &[1, 2, 3, 4]), "write must fault");
        assert_eq!(tlb.segfault_addr, addr);
        assert_eq!(tlb.entries[tlb_index(addr)].page_if_writable, TLB_PAGE_EMPTY);
        // A ptrace write *is* allowed: the C backend compares `type == MEM_WRITE`,
        // and MEM_WRITE_PTRACE is a different value (see kernel/user.c).
        assert!(mmu.translate(addr, MemType::Write).is_none());
        assert!(mmu.translate(addr, MemType::WritePtrace).is_some());
    }

    #[test]
    fn unmapped_page_faults_and_records_the_address() {
        let mut backend = FakeMmu::new();
        backend.unmapped = 1 << 5;
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);

        let addr = (5 << PAGE_BITS) + 0x20;
        let mut buf = [0u8; 4];
        assert!(!tlb.read(&mut mmu, addr, &mut buf));
        assert_eq!(tlb.segfault_addr, addr);
        assert!(tlb.entries[tlb_index(addr)].data.is_none());
    }

    #[test]
    fn cross_page_access_spans_two_pages() {
        let mut backend = FakeMmu::new();
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);

        let addr = 0x2ffe; // two bytes from the end of page 2
        assert!(tlb.write(&mut mmu, addr, &[0xaa, 0xbb, 0xcc, 0xdd]));
        let mut buf = [0u8; 4];
        assert!(tlb.read(&mut mmu, addr, &mut buf));
        assert_eq!(buf, [0xaa, 0xbb, 0xcc, 0xdd]);
        // bytes landed on both sides of the boundary
        assert_eq!(backend.backing[0x2ffe], 0xaa);
        assert_eq!(backend.backing[0x2fff], 0xbb);
        assert_eq!(backend.backing[0x3000], 0xcc);
        assert_eq!(backend.backing[0x3001], 0xdd);
    }

    #[test]
    fn cross_page_fails_atomically_when_the_second_page_is_missing() {
        let mut backend = FakeMmu::new();
        backend.unmapped = 1 << 3;
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);

        let addr = 0x2ffe;
        let mut buf = [0u8; 4];
        assert!(!tlb.read(&mut mmu, addr, &mut buf));
        assert_eq!(tlb.segfault_addr, 0x3000, "the faulting address is the second page");
    }

    #[test]
    fn refresh_flushes_when_the_mmu_changes() {
        let mut backend = FakeMmu::new();
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);

        let mut buf = [0u8; 4];
        assert!(tlb.read(&mut mmu, 0x1000, &mut buf));
        let before = calls();
        assert!(tlb.entries[tlb_index(0x1000)].data.is_some());

        // no change counter bump -> refresh is a no-op
        tlb.refresh(&mut mmu);
        assert!(tlb.entries[tlb_index(0x1000)].data.is_some());

        // bump changes -> refresh flushes
        mmu.changes += 1;
        tlb.refresh(&mut mmu);
        assert!(tlb.entries[tlb_index(0x1000)].data.is_none());
        assert_eq!(tlb.mem_changes, mmu.changes as u32);
        assert_eq!(tlb.dirty_page, TLB_PAGE_EMPTY);

        assert!(tlb.read(&mut mmu, 0x1000, &mut buf));
        assert!(calls() > before);
    }

    #[test]
    fn dirty_page_tracks_misses_and_write_hits() {
        let mut backend = FakeMmu::new();
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);
        tlb.dirty_page = TLB_PAGE_EMPTY;

        let mut buf = [0u8; 1];
        // A read that *misses* goes through tlb_handle_miss, which sets
        // dirty_page unconditionally - that is what the C does.
        assert!(tlb.read(&mut mmu, 0x4000, &mut buf));
        assert_eq!(tlb.dirty_page, tlb_page(0x4000), "a read miss marks the page");

        // A read that *hits* leaves it alone.
        tlb.dirty_page = TLB_PAGE_EMPTY;
        assert!(tlb.read(&mut mmu, 0x4000, &mut buf));
        assert_eq!(tlb.dirty_page, TLB_PAGE_EMPTY, "a read hit is not dirty");

        // A write hit marks the page, which is what the JIT uses to know when
        // to re-translate self-modifying code.
        assert!(tlb.write_ptr(&mut mmu, 0x5000).is_some());
        assert_eq!(tlb.dirty_page, tlb_page(0x5000));
    }

    #[test]
    fn flush_resets_every_entry_to_the_empty_marker() {
        let mut backend = FakeMmu::new();
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);
        let mut buf = [0u8; 4];
        assert!(tlb.read(&mut mmu, 0x1000, &mut buf));
        tlb.flush(&mmu);
        for e in tlb.entries.iter() {
            assert_eq!(e.page, TLB_PAGE_EMPTY);
            assert_eq!(e.page_if_writable, TLB_PAGE_EMPTY);
            assert!(e.data.is_none());
        }
    }
}
