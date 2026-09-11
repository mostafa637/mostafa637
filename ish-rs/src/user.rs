//! `kernel/user.c` — the bridge between kernel code and guest memory.
//!
//! Every syscall argument that lives in the guest goes through here. The whole
//! file is a page-at-a-time walk:
//!
//! ```text
//! while (p < addr + count) {
//!     chunk_end = end of p's page, capped at addr + count
//!     ptr = mem_ptr(mem, p, type)     // may fault, may grow, may break CoW
//!     memcpy(...)
//!     p = chunk_end
//! }
//! ```
//!
//! Two properties of that loop are load-bearing and are reproduced exactly:
//!
//! * **It does not roll back.** Each page is copied as the walk reaches it, so a
//!   fault on page *N* leaves pages *0..N-1* already written. Callers in iSH
//!   depend on this being harmless rather than on it being undone.
//! * **The arithmetic is 32-bit and wraps.** `chunk_end` is an `addr_t`, so at
//!   the top of the address space `(PAGE(p) + 1) << PAGE_BITS` becomes 0, while
//!   the loop bound `addr + count` is computed in `size_t` and does not. The
//!   comparison between the two is therefore a comparison between a wrapped and
//!   an unwrapped value. That is kept, wrap included.
//!
//! The C's `read_wrlock`/`read_wrunlock` around each entry point have no
//! counterpart: there is one thread.

use std::ffi::CStr;

use crate::memory::Mem;
use crate::mmu::{MemType, PAGE_BITS};

/// A guest access could not be completed.
///
/// The C returns `1` for this and `0` for success; [`Fault`] is the same
/// information in a type that cannot be ignored by accident (the C declarations
/// carry `must_check` for the same reason).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Fault;

impl core::fmt::Display for Fault {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.write_str("guest memory access faulted")
    }
}

impl std::error::Error for Fault {}

/// The guest address space a set of user-memory operations runs against.
///
/// This is the C's `struct task *`, narrowed to the one field these functions
/// touch: `task->mem`. The functions the C spells without a `_task` suffix
/// (`user_read`, `user_write`, `user_read_string`, `user_write_string`) read the
/// thread-local `current`; here the address space is explicit instead, which is
/// what makes the port testable without a task table.
pub struct User<'m> {
    mem: &'m mut Mem,
}

/// `PAGE(addr)`
const fn page(addr: u32) -> u32 {
    addr >> PAGE_BITS
}

impl<'m> User<'m> {
    pub fn new(mem: &'m mut Mem) -> Self {
        User { mem }
    }

    /// `__user_read_task` — the walk both readers share.
    fn read_into(&mut self, addr: u32, buf: &mut [u8]) -> Result<(), Fault> {
        // `addr + count` in the C is addr_t + size_t, i.e. 64-bit: it does not
        // wrap where `chunk_end` does
        let end = addr as u64 + buf.len() as u64;
        let mut p = addr;
        while (p as u64) < end {
            let mut chunk_end = (page(p) + 1) << PAGE_BITS;
            if chunk_end as u64 > end {
                chunk_end = end as u32;
            }
            let src = self.guest_ptr(p, MemType::Read)?;
            let n = chunk_end.wrapping_sub(p) as usize;
            // SAFETY: see `guest_slice`
            let page_bytes = unsafe { guest_slice(src, n) };
            let at = p.wrapping_sub(addr) as usize;
            buf[at..at + n].copy_from_slice(page_bytes);
            // `p = chunk_end` and `p += 1` are observably the same thing on the
            // read path, and only here: the copies overlap, but a read never
            // changes a guest byte (grow-down maps zeroes, and zeroes read back
            // as zeroes), it faults on the same first unmapped page, and it
            // bumps no counter. On the write path the same change is *not*
            // equivalent - every extra `mem_ptr` invalidates the page's compiled
            // blocks - and the corpus catches it there.
            p = chunk_end;
        }
        Ok(())
    }

    /// `__user_write_task` — `ptrace` picks `MEM_WRITE_PTRACE`, which bypasses
    /// write protection the way a debugger has to.
    fn write_from(&mut self, addr: u32, buf: &[u8], ptrace: bool) -> Result<(), Fault> {
        let type_ = if ptrace { MemType::WritePtrace } else { MemType::Write };
        let end = addr as u64 + buf.len() as u64;
        let mut p = addr;
        while (p as u64) < end {
            let mut chunk_end = (page(p) + 1) << PAGE_BITS;
            if chunk_end as u64 > end {
                chunk_end = end as u32;
            }
            let dst = self.guest_ptr(p, type_)?;
            let n = chunk_end.wrapping_sub(p) as usize;
            // SAFETY: see `guest_slice`
            let page_bytes = unsafe { guest_slice_mut(dst, n) };
            let at = p.wrapping_sub(addr) as usize;
            page_bytes.copy_from_slice(&buf[at..at + n]);
            p = chunk_end;
        }
        Ok(())
    }

    /// `mem_ptr`, with the fault turned into [`Fault`].
    fn guest_ptr(&mut self, addr: u32, type_: MemType) -> Result<std::ptr::NonNull<u8>, Fault> {
        self.mem.ptr(addr, type_).ok_or(Fault)
    }

    /// `user_read_task` — copy `buf.len()` bytes out of the guest.
    ///
    /// On a fault, `buf` holds whatever was read before it, exactly as the C
    /// leaves it.
    pub fn read(&mut self, addr: u32, buf: &mut [u8]) -> Result<(), Fault> {
        self.read_into(addr, buf)
    }

    /// `user_write_task` — copy `buf` into the guest.
    ///
    /// On a fault, the pages reached before it stay written.
    pub fn write(&mut self, addr: u32, buf: &[u8]) -> Result<(), Fault> {
        self.write_from(addr, buf, false)
    }

    /// `user_write_task_ptrace` — as [`User::write`], but write protection does
    /// not apply.
    pub fn write_ptrace(&mut self, addr: u32, buf: &[u8]) -> Result<(), Fault> {
        self.write_from(addr, buf, true)
    }

    /// `user_read_string` — read up to `buf.len()` bytes, stopping after a NUL.
    ///
    /// The C returns success even when it fills the buffer without finding a
    /// terminator, leaving the caller's buffer unterminated; so does this. The
    /// caller has to know how long it asked for.
    pub fn read_string(&mut self, addr: u32, buf: &mut [u8]) -> Result<(), Fault> {
        if addr == 0 {
            return Err(Fault);
        }
        for i in 0..buf.len() {
            // the C reads one byte at a time through the same walk, so a
            // grow-down region grows one byte at a time too
            self.read_into(addr.wrapping_add(i as u32), &mut buf[i..i + 1])?;
            if buf[i] == 0 {
                break;
            }
        }
        Ok(())
    }

    /// `user_write_string` — write `s` into the guest, terminator included.
    ///
    /// Taking a [`CStr`] is the port's one deliberate strengthening: the C walks
    /// `buf` until it finds a NUL and has undefined behaviour if there is none.
    pub fn write_string(&mut self, addr: u32, s: &CStr) -> Result<(), Fault> {
        if addr == 0 {
            return Err(Fault);
        }
        // The C writes one byte at a time through the same walk, and that is
        // observable: every byte runs `mem_ptr`, and every `mem_ptr` on a
        // writable page invalidates the page's compiled blocks. Chunking the
        // string into pages would reach the same bytes and the same faults but
        // invalidate once per page instead of once per byte.
        let bytes = s.to_bytes_with_nul();
        for i in 0..bytes.len() {
            self.write_from(addr.wrapping_add(i as u32), &bytes[i..i + 1], false)?;
        }
        Ok(())
    }
}

/// The `n` bytes of guest memory starting at `ptr`.
///
/// # Safety
///
/// `ptr` must come from [`Mem::ptr`] and `n` must not run past the end of the
/// page it points into.
///
/// Both hold at the only two call sites: `Mem::ptr` returns a pointer at
/// `pgoffset(addr)` inside the backing object of `PAGE(addr)`, and `n` is
/// `chunk_end - p`, where `chunk_end` is capped at the end of `p`'s page. So the
/// range stays inside the same page's bytes, which the mapping owns for as long
/// as it is mapped — the same window the C's `memcpy` relies on.
unsafe fn guest_slice<'a>(ptr: std::ptr::NonNull<u8>, n: usize) -> &'a [u8] {
    std::slice::from_raw_parts(ptr.as_ptr(), n)
}

/// The writable form of [`guest_slice`]; same safety requirements.
unsafe fn guest_slice_mut<'a>(ptr: std::ptr::NonNull<u8>, n: usize) -> &'a mut [u8] {
    std::slice::from_raw_parts_mut(ptr.as_ptr(), n)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::{P_COW, P_GROWSDOWN, P_READ, P_RWX};
    use std::rc::Rc;

    /// an address space with a writable page at 0x100, a read-only page at
    /// 0x200, a grow-down page at 0x500 and a hole everywhere else
    fn space() -> Mem {
        let mut mem = Mem::new();
        let bytes = (0..4096u32).map(|i| (i.wrapping_mul(31).wrapping_add(1) & 0xff) as u8)
            .collect::<Vec<u8>>()
            .into_boxed_slice();
        mem.map(0x100, 1, bytes, 0, P_RWX);
        let ro = (0..4096u32).map(|i| (i.wrapping_mul(31).wrapping_add(2) & 0xff) as u8)
            .collect::<Vec<u8>>()
            .into_boxed_slice();
        mem.map(0x200, 1, ro, 0, P_READ);
        mem.map_nothing(0x500, 1, P_RWX | P_GROWSDOWN);
        mem
    }

    fn bytes_of(mem: &Mem, page: u32) -> Vec<u8> {
        let pt = mem.pt(page).unwrap();
        pt.data.bytes[pt.offset..pt.offset + 4096].to_vec()
    }

    #[test]
    fn a_read_inside_one_page_copies_the_guest_bytes() {
        let mut mem = space();
        let mut buf = [0u8; 16];
        assert_eq!(User::new(&mut mem).read(0x100 << PAGE_BITS, &mut buf), Ok(()));
        assert_eq!(buf[0], 1, "the mapping's own pattern");
        assert_eq!(buf[1], 32, "31*1+1");
    }

    #[test]
    fn a_read_across_a_page_boundary_stitches_two_pages() {
        let mut mem = space();
        mem.map_nothing(0x101, 1, P_RWX);
        let mut buf = [0u8; 16];
        assert_eq!(User::new(&mut mem).read((0x100 << PAGE_BITS) + 4090, &mut buf), Ok(()));
        // six bytes from the end of 0x100, ten from the start of 0x101
        assert_eq!(buf[..6], bytes_of(&mem, 0x100)[4090..]);
        assert_eq!(&buf[6..], &bytes_of(&mem, 0x101)[..10]);
    }

    #[test]
    fn a_fault_leaves_what_was_read_so_far_in_the_buffer() {
        let mut mem = space();
        let mut buf = [0xaau8; 16];
        // 0x101 is a hole, so this faults after the first six bytes
        assert_eq!(User::new(&mut mem).read((0x100 << PAGE_BITS) + 4090, &mut buf), Err(Fault));
        assert_eq!(buf[..6], bytes_of(&mem, 0x100)[4090..]);
        assert_eq!(&buf[6..], &[0xaa; 10], "the rest is untouched");
    }

    #[test]
    fn a_write_that_faults_halfway_keeps_the_first_page() {
        let mut mem = space();
        let before = bytes_of(&mem, 0x100);
        let buf: Vec<u8> = (0..16u8).map(|j| j.wrapping_mul(7).wrapping_add(13)).collect();
        // spans 0x100 (writable) then 0x101 (a hole)
        assert_eq!(User::new(&mut mem).write((0x100 << PAGE_BITS) + 4090, &buf), Err(Fault));
        let after = bytes_of(&mem, 0x100);
        assert_eq!(&after[4090..], &buf[..6], "the reachable part is written");
        assert_eq!(&after[..4090], &before[..4090], "and nothing else moved");
    }

    #[test]
    fn writing_to_a_read_only_page_faults_and_writes_nothing() {
        let mut mem = space();
        let before = bytes_of(&mem, 0x200);
        assert_eq!(User::new(&mut mem).write(0x200 << PAGE_BITS, &[1, 2, 3]), Err(Fault));
        assert_eq!(bytes_of(&mem, 0x200), before);
    }

    #[test]
    fn a_ptrace_write_goes_through_where_a_plain_write_does_not() {
        let mut mem = space();
        let buf = [9u8, 8, 7];
        assert_eq!(User::new(&mut mem).write(0x200 << PAGE_BITS, &buf), Err(Fault));
        assert_eq!(User::new(&mut mem).write_ptrace(0x200 << PAGE_BITS, &buf), Ok(()));
        assert_eq!(&bytes_of(&mem, 0x200)[..3], &buf);
        // the ptrace write broke the CoW it created, so the page stays writable
        assert_eq!(mem.pt(0x200).unwrap().flags & P_COW, 0);
        assert_eq!(User::new(&mut mem).write((0x200 << PAGE_BITS) + 8, &buf), Ok(()));
    }

    #[test]
    fn a_zero_length_access_touches_nothing_and_succeeds() {
        let mut mem = space();
        let mut buf = [];
        assert_eq!(User::new(&mut mem).read(0x100 << PAGE_BITS, &mut buf), Ok(()));
        assert_eq!(User::new(&mut mem).write(0x100 << PAGE_BITS, &[]), Ok(()));
        // even at an address that is not mapped: the walk never runs
        let mut b2 = [];
        assert_eq!(User::new(&mut mem).read(0x400 << PAGE_BITS, &mut b2), Ok(()));
    }

    #[test]
    fn reading_a_grow_down_region_grows_it_in() {
        let mut mem = space();
        assert!(mem.pt(0x4ff).is_none());
        let mut buf = [0u8; 4];
        assert_eq!(User::new(&mut mem).read((0x4ff << PAGE_BITS) + 4090, &mut buf), Ok(()));
        assert!(mem.pt(0x4ff).is_some(), "the read grew the region down");
        assert_eq!(buf, [0; 4], "a freshly grown page reads as zeroes");
    }

    #[test]
    fn write_string_writes_the_terminator_and_nothing_more() {
        let mut mem = space();
        mem.map_nothing(0x600, 1, P_RWX);
        let s = CStr::from_bytes_with_nul(b"hi\0").unwrap();
        assert_eq!(User::new(&mut mem).write_string(0x600 << PAGE_BITS, s), Ok(()));
        assert_eq!(&bytes_of(&mem, 0x600)[..4], b"hi\0\0");
    }

    #[test]
    fn read_string_stops_after_the_terminator() {
        let mut mem = space();
        mem.map_nothing(0x600, 1, P_RWX);
        User::new(&mut mem).write_string(0x600 << PAGE_BITS, CStr::from_bytes_with_nul(b"abc\0").unwrap())
            .unwrap();
        let mut buf = [0xaau8; 8];
        assert_eq!(User::new(&mut mem).read_string(0x600 << PAGE_BITS, &mut buf), Ok(()));
        assert_eq!(&buf[..4], b"abc\0");
        assert_eq!(&buf[4..], &[0xaa; 4], "reading stops at the NUL");
    }

    #[test]
    fn read_string_with_a_short_max_returns_success_unterminated() {
        // the C does the same: the loop ends on `i < max`, not on a terminator
        let mut mem = space();
        mem.map_nothing(0x600, 1, P_RWX);
        User::new(&mut mem).write_string(0x600 << PAGE_BITS, CStr::from_bytes_with_nul(b"abc\0").unwrap())
            .unwrap();
        let mut buf = [0u8; 2];
        assert_eq!(User::new(&mut mem).read_string(0x600 << PAGE_BITS, &mut buf), Ok(()));
        assert_eq!(&buf, b"ab", "no terminator was stored");
    }

    #[test]
    fn a_null_address_is_refused_before_anything_is_touched() {
        let mut mem = space();
        let mut buf = [0u8; 4];
        assert_eq!(User::new(&mut mem).read_string(0, &mut buf), Err(Fault));
        assert_eq!(
            User::new(&mut mem).write_string(0, CStr::from_bytes_with_nul(b"x\0").unwrap()),
            Err(Fault)
        );
    }

    #[test]
    fn an_empty_string_is_still_one_byte() {
        let mut mem = space();
        mem.map_nothing(0x600, 1, P_RWX);
        assert_eq!(
            User::new(&mut mem).write_string(0x600 << PAGE_BITS, CStr::from_bytes_with_nul(b"\0").unwrap()),
            Ok(())
        );
        assert_eq!(bytes_of(&mem, 0x600)[0], 0);
    }

    #[test]
    fn write_string_invalidates_once_per_byte_not_once_per_page() {
        // each invalidation throws away the JIT's compiled blocks for the page,
        // so how many happen is part of the behaviour, not an implementation
        // detail
        let mut mem = space();
        mem.map_nothing(0x600, 1, P_RWX);
        let before = mem.invalidations;
        let s = CStr::from_bytes_with_nul(b"hello\0").unwrap();
        {
            let mut u = User::new(&mut mem);
            assert_eq!(u.write_string(0x600 << PAGE_BITS, s), Ok(()));
        }
        assert_eq!(mem.invalidations - before, 6, "six bytes, six invalidations");
        // a plain write of the same length is one page-sized chunk
        let before = mem.invalidations;
        assert_eq!(User::new(&mut mem).write(0x600 << PAGE_BITS, b"hello\0"), Ok(()));
        assert_eq!(mem.invalidations - before, 1);
    }

    #[test]
    fn a_string_write_that_faults_stops_at_the_first_bad_byte() {
        let mut mem = space();
        mem.map_nothing(0x600, 1, P_RWX);
        // only six bytes left in the page, and 0x601 is a hole
        let s = CStr::from_bytes_with_nul(b"overflow\0").unwrap();
        assert_eq!(
            User::new(&mut mem).write_string((0x600 << PAGE_BITS) + 4090, s),
            Err(Fault)
        );
        assert_eq!(&bytes_of(&mem, 0x600)[4090..], b"overfl", "what fitted is written");
    }

    #[test]
    fn a_write_breaks_copy_on_write_for_the_writer_only() {
        let mut parent = space();
        let mut child = Mem::new();
        Mem::copy_on_write(&mut parent, &mut child, 0x100, 1);
        assert!(Rc::ptr_eq(
            &parent.pt(0x100).unwrap().data,
            &child.pt(0x100).unwrap().data
        ));
        let shared = bytes_of(&parent, 0x100);
        assert_eq!(User::new(&mut child).write(0x100 << PAGE_BITS, &[0xff; 4]), Ok(()));
        assert_eq!(&bytes_of(&child, 0x100)[..4], &[0xff; 4]);
        assert_eq!(bytes_of(&parent, 0x100), shared, "the parent is untouched");
    }

    #[test]
    fn the_top_of_the_address_space_wraps_the_way_the_c_does() {
        // `chunk_end` is an addr_t, so (PAGE(0xfffff)+1)<<12 is 0, while the loop
        // bound addr+count is 64-bit. The first page is unmapped, so this faults
        // on the first chunk rather than looping on the wrapped one.
        let mut mem = space();
        let buf = [0u8; 8192];
        assert_eq!(User::new(&mut mem).write(0xffff_f000, &buf), Err(Fault));
        assert!(mem.pt(0xfffff).is_none());
    }
}
