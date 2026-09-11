//! Word-for-word differential test of `mmu.rs` + `tlb.rs` against the C
//! original.
//!
//! `tests/fixtures/tlb_reference.txt` is produced by `tools/tlb-dump.c`,
//! compiled against the *unmodified* iSH `emu/tlb.c` and `emu/tlb.h`. The C
//! driver runs a fixed sequence of accesses against a fake `mmu_ops` backend
//! and, after **every** step, dumps not just the bytes read but the whole
//! `struct tlb`: which of the 1024 entries are filled, what page they hold,
//! whether they are writable, what got marked dirty, where a fault was
//! recorded, and how many times `translate` had to be called.
//!
//! That last number is what makes this a real test of the *cache* rather than
//! of the page walker: a port that simply translated every access would still
//! return the right bytes, and would fail here immediately.
//!
//! Raw host pointers are never compared — they differ between processes, so
//! the fixture records only guest-observable state plus an FNV-1a hash of the
//! backing store, which catches any stray write.
//!
//! **Not covered here:** `tlb_handle_miss` translates *first* and only then
//! compares `mmu->changes`, so a backend that remaps while translating (the
//! real one in `kernel/memory.c` does exactly that when it faults a page in)
//! leaves the TLB flushed and `mem_changes` already resynced. This fake cannot
//! reach `mmu.changes` from inside `translate`, so the ordering of those two
//! steps is reproduced by inspection rather than pinned by this corpus.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_tlb_reference.sh
//! ```

use std::cell::RefCell;
use std::ptr::NonNull;

use ish_emu::cpu::AddrT;
use ish_emu::mmu::{Mmu, MmuOps, MemType, PAGE_BITS, PAGE_SIZE};
use ish_emu::tlb::{Tlb, TLB_SIZE};

const FIXTURE: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/tests/fixtures/tlb_reference.txt");

// ---- the same fake backend the C driver uses ----------------------------

const FAKE_PAGES: usize = 16;
const FAKE_BYTES: usize = FAKE_PAGES << PAGE_BITS as usize;

/// Mirrors `tools/tlb-dump.c`. The address space aliases every `FAKE_PAGES`
/// pages onto the same backing store, so guest addresses 4 MiB apart can land
/// in the same TLB slot — that is what exercises eviction.
struct Backend {
    backing: Box<[u8; FAKE_BYTES]>,
    ro_mask: u32,
    unmap_mask: u32,
    translate_calls: u32,
}

thread_local! {
    static BACKEND: RefCell<Backend> = RefCell::new(Backend {
        backing: Box::new([(0u8); FAKE_BYTES]),
        ro_mask: 0,
        unmap_mask: 0,
        translate_calls: 0,
    });
}

fn reset_backend() {
    BACKEND.with(|b| {
        let mut b = b.borrow_mut();
        for (i, byte) in b.backing.iter_mut().enumerate() {
            // the same deterministic fill as the C driver
            *byte = ((i as u32).wrapping_mul(31).wrapping_add(7) & 0xff) as u8;
        }
        b.ro_mask = 0;
        b.unmap_mask = 0;
        b.translate_calls = 0;
    });
}

fn translate_calls() -> u32 {
    BACKEND.with(|b| b.borrow().translate_calls)
}

/// FNV-1a over the backing store, so the two implementations can be compared
/// without shipping 64 KiB per step.
fn hash_backing() -> u64 {
    BACKEND.with(|b| {
        let b = b.borrow();
        let mut h: u64 = 0xcbf2_9ce4_8422_2325;
        for byte in b.backing.iter() {
            h ^= *byte as u64;
            h = h.wrapping_mul(0x0000_0100_0000_01b3);
        }
        h
    })
}

struct FakeMmu;

impl MmuOps for FakeMmu {
    fn translate(&mut self, addr: AddrT, type_: MemType) -> Option<NonNull<u8>> {
        BACKEND.with(|b| {
            let mut b = b.borrow_mut();
            b.translate_calls += 1;
            let idx = ((addr >> PAGE_BITS) & (FAKE_PAGES as u32 - 1)) as usize;
            if b.unmap_mask & (1 << idx) != 0 {
                return None;
            }
            // MEM_WRITE_PTRACE is deliberately *not* blocked, exactly as the C
            // compares `type == MEM_WRITE`.
            if type_ == MemType::Write && b.ro_mask & (1 << idx) != 0 {
                return None;
            }
            let ptr = b.backing.as_mut_ptr().wrapping_add(idx << PAGE_BITS as usize);
            // SAFETY: `backing` is a live Box for as long as the process runs
            // and `idx << PAGE_BITS` is a page-aligned offset inside it.
            Some(unsafe { NonNull::new_unchecked(ptr) })
        })
    }
}

// ---- the step table, identical to the one in tools/tlb-dump.c -----------

#[derive(Debug, PartialEq, Eq)]
enum Op {
    Read,
    Write,
    ReadPtr,
    WritePtr,
    Refresh,
    Flush,
    Bump,
    Ro,
    Unmap,
}

impl Op {
    fn name(&self) -> &'static str {
        match self {
            Op::Read => "read",
            Op::Write => "write",
            Op::ReadPtr => "read_ptr",
            Op::WritePtr => "write_ptr",
            Op::Refresh => "refresh",
            Op::Flush => "flush",
            Op::Bump => "bump",
            Op::Ro => "ro",
            Op::Unmap => "unmap",
        }
    }
}

struct Step {
    op: Op,
    addr: u32,
    size: u32,
}

const fn step(op: Op, addr: u32, size: u32) -> Step {
    Step { op, addr, size }
}

const STEPS: &[Step] = &[
    step(Op::Refresh, 0, 0),
    step(Op::Read, 0x0000, 4),
    step(Op::Read, 0x0004, 4), // hit
    step(Op::Write, 0x0010, 4),
    step(Op::Read, 0x0010, 4), // read back what was written
    step(Op::Read, 0x0ffc, 8), // cross-page 0 -> 1
    step(Op::Write, 0x0ffe, 8),
    step(Op::Read, 0x0ffe, 8),
    step(Op::ReadPtr, 0x2000, 0),
    step(Op::WritePtr, 0x2004, 0),
    step(Op::Ro, 1 << 3, 0), // page 3 becomes read-only
    step(Op::Read, 0x3000, 4),
    step(Op::Write, 0x3000, 4), // must fault
    step(Op::WritePtr, 0x3004, 0),
    step(Op::Read, 0x3008, 4),
    step(Op::Ro, 0, 0),
    step(Op::Unmap, 1 << 5, 0), // page 5 disappears
    step(Op::Read, 0x5020, 4),
    step(Op::Write, 0x5020, 4),
    step(Op::Read, 0x4ffe, 8), // cross-page into the missing page
    step(Op::Write, 0x4ffe, 8), // must not write the first half
    step(Op::Unmap, 0, 0),
    step(Op::Read, 0x1000, 4), // TLB index 1
    step(Op::Read, 0x400000, 4), // same TLB index, evicts the entry above
    step(Op::Read, 0x1000, 4), // missed again
    step(Op::Read, 0x401000, 4), // index 0, no collision with 0x1000
    step(Op::Bump, 0, 0),
    step(Op::Refresh, 0, 0), // must flush
    step(Op::Refresh, 0, 0), // must be a no-op
    step(Op::Bump, 0, 0), // flush must resync mem_changes, not just empty it
    step(Op::Flush, 0, 0),
    step(Op::Read, 0x7000, 1),
    step(Op::Write, 0x7fff, 2), // cross-page 7 -> 8
    step(Op::Read, 0x7fff, 2),
    step(Op::Read, 0xf000, 16),
    step(Op::Write, 0xfffc, 8), // last page, next page wraps to index 0
    step(Op::Read, 0xfffc, 8),
    // size > PAGE_SIZE at a non-zero offset: the unsigned wrap takes the fast
    // path and copies straight from the page base, while a saturating subtract
    // would cross pages instead
    step(Op::Read, 0x0010, 5000),
    // the same wrap on the write path: the cross-page route would retranslate
    // the next page, which the call count catches
    step(Op::Write, 0x0010, 5000),
    step(Op::Write, 0x0000, 4096), // exactly one page
    step(Op::Read, 0x0000, 4096),
    step(Op::Ro, 0xffff_ffff, 0), // everything read-only
    step(Op::Write, 0x9000, 4),
    step(Op::Ro, 0, 0),
    step(Op::Unmap, 0xffff_ffff, 0), // nothing mapped
    step(Op::Read, 0x0000, 4),
    step(Op::Refresh, 0, 0),
    step(Op::Unmap, 0, 0),
    step(Op::Bump, 0, 0),
    step(Op::Refresh, 0, 0),
    step(Op::Read, 0xb000, 4),
    step(Op::Write, 0xb004, 4),
    step(Op::Read, 0xb004, 4),
    // write *hits*: the entry is already writable, so write_ptr has to set
    // dirty_page itself - handle_miss is not involved
    step(Op::Write, 0xb008, 4), // hit on page 11
    step(Op::Write, 0x0020, 4), // hit on page 0 -> dirty_page 0
    step(Op::Write, 0xb00c, 4), // hit on page 11 again -> dirty_page 0xb000
];

// ---- replay -------------------------------------------------------------

/// One parsed fixture record.
struct Record {
    index: usize,
    op: String,
    addr: u32,
    size: u32,
    ok: bool,
    bytes: Vec<u8>,
    dirty_page: u32,
    segfault_addr: u32,
    mem_changes: u32,
    calls: u32,
    hash: u64,
    entries: Vec<(u32, u32, bool)>,
}

fn parse_fixture() -> Vec<Record> {
    let text = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|e| panic!("cannot read {}: {e}", FIXTURE));
    let mut out = Vec::new();
    for line in text.lines() {
        if !line.starts_with("T ") {
            continue;
        }
        let f: Vec<&str> = line.split(' ').collect();
        // T index op addr size ok <max(16, size) byte slots> dirty segfault
        // changes calls hash <TLB_SIZE entry triples>. The byte section is
        // variable length (a 5000-byte read prints 5000 slots), so the tail is
        // parsed from the end.
        let index: usize = f[1].parse().unwrap();
        let op = f[2].to_string();
        let addr = u32::from_str_radix(f[3], 16).unwrap();
        let size = u32::from_str_radix(f[4], 16).unwrap();
        let ok = f[5] == "1";
        let tail = TLB_SIZE + 5;
        assert!(f.len() >= 6 + tail, "record {index} is too short");
        let mut bytes = Vec::new();
        for slot in &f[6..f.len() - tail] {
            if *slot != "--" {
                bytes.push(u8::from_str_radix(slot, 16).unwrap());
            }
        }
        let s = f.len();
        let dirty_page = u32::from_str_radix(f[s - tail], 16).unwrap();
        let segfault_addr = u32::from_str_radix(f[s - tail + 1], 16).unwrap();
        let mem_changes = u32::from_str_radix(f[s - tail + 2], 16).unwrap();
        let calls = u32::from_str_radix(f[s - tail + 3], 16).unwrap();
        let hash = u64::from_str_radix(f[s - tail + 4], 16).unwrap();
        let mut entries = Vec::with_capacity(TLB_SIZE);
        for slot in &f[s - TLB_SIZE..] {
            let mut parts = slot.split(':');
            let page = u32::from_str_radix(parts.next().unwrap(), 16).unwrap();
            let pw = u32::from_str_radix(parts.next().unwrap(), 16).unwrap();
            let present = parts.next().unwrap() == "1";
            entries.push((page, pw, present));
        }
        out.push(Record {
            index,
            op,
            addr,
            size,
            ok,
            bytes,
            dirty_page,
            segfault_addr,
            mem_changes,
            calls,
            hash,
            entries,
        });
    }
    out
}

#[test]
fn tlb_matches_the_c_reference() {
    let records = parse_fixture();
    assert_eq!(records.len(), STEPS.len(), "the step table and the fixture disagree");

    reset_backend();
    let mut fake = FakeMmu;
    let mut mmu = Mmu::new(&mut fake);
    let mut tlb = Tlb::new();

    let mut compared = 0usize;
    let mut bytes_compared = 0usize;
    for (i, st) in STEPS.iter().enumerate() {
        let rec = &records[i];
        assert_eq!(rec.index, i);
        // guard against the Rust table drifting from the C one
        assert_eq!(rec.op, st.op.name(), "step {i}: op differs from the fixture");
        assert_eq!(rec.addr, st.addr, "step {i}: addr differs from the fixture");
        assert_eq!(rec.size, st.size, "step {i}: size differs from the fixture");

        let mut buf = vec![0u8; st.size as usize];
        let mut out_size = 0usize;
        let ok = match st.op {
            Op::Read => {
                out_size = st.size as usize;
                tlb.read(&mut mmu, st.addr, &mut buf)
            }
            Op::Write => {
                for (k, byte) in buf.iter_mut().enumerate() {
                    *byte = (st.addr.wrapping_add(k as u32) & 0xff) as u8;
                }
                tlb.write(&mut mmu, st.addr, &buf)
            }
            Op::ReadPtr => tlb.read_ptr(&mut mmu, st.addr).is_some(),
            Op::WritePtr => tlb.write_ptr(&mut mmu, st.addr).is_some(),
            Op::Refresh => {
                tlb.refresh(&mut mmu);
                true
            }
            Op::Flush => {
                tlb.flush(&mmu);
                true
            }
            Op::Bump => {
                mmu.changes += 1;
                true
            }
            Op::Ro => {
                BACKEND.with(|b| b.borrow_mut().ro_mask = st.addr);
                true
            }
            Op::Unmap => {
                BACKEND.with(|b| b.borrow_mut().unmap_mask = st.addr);
                true
            }
        };

        let ctx = || format!("step {i} ({} {:#x} size {:#x})", rec.op, st.addr, st.size);
        assert_eq!(ok, rec.ok, "{}: result", ctx());
        if out_size > 0 {
            assert_eq!(buf.len(), rec.bytes.len(), "{}: byte count", ctx());
            assert_eq!(buf, rec.bytes, "{}: bytes read", ctx());
            bytes_compared += buf.len();
        }
        assert_eq!(tlb.dirty_page, rec.dirty_page, "{}: dirty_page", ctx());
        assert_eq!(tlb.segfault_addr, rec.segfault_addr, "{}: segfault_addr", ctx());
        assert_eq!(tlb.mem_changes, rec.mem_changes, "{}: mem_changes", ctx());
        assert_eq!(
            translate_calls(),
            rec.calls,
            "{}: translate call count (a cache hit/miss difference)",
            ctx()
        );
        assert_eq!(hash_backing(), rec.hash, "{}: backing store hash", ctx());
        for (slot, (page, pw, present)) in tlb.entries.iter().zip(rec.entries.iter()) {
            assert_eq!(slot.page, *page, "{}: entry page", ctx());
            assert_eq!(slot.page_if_writable, *pw, "{}: entry page_if_writable", ctx());
            assert_eq!(slot.data.is_some(), *present, "{}: entry presence", ctx());
        }
        compared += 1 + 4 + TLB_SIZE * 3;
    }

    println!(
        "{compared} state words and {bytes_compared} bytes matched the C reference exactly"
    );
}

#[test]
fn the_reference_actually_exercises_the_cache() {
    // If the fixture were all misses the test above would still pass with a
    // TLB that never caches anything, so pin the interesting properties here.
    let records = parse_fixture();
    let hits = records
        .windows(2)
        .filter(|w| w[1].calls == w[0].calls && matches!(w[1].op.as_str(), "read" | "write"))
        .count();
    assert!(hits > 5, "expected repeated accesses to hit the cache, saw {hits}");

    let faults = records.iter().filter(|r| !r.ok).count();
    assert!(faults > 5, "expected faults in the corpus, saw {faults}");

    // the collision pair: 0x1000 and 0x400000 share a TLB slot, so reading the
    // second one must evict the first and force another translate
    let a = records.iter().position(|r| r.addr == 0x1000 && r.op == "read").unwrap();
    let b = records.iter().position(|r| r.addr == 0x40_0000 && r.op == "read").unwrap();
    assert!(records[b].calls > records[a].calls, "the colliding read should miss");

    // and an access to a page that is *not* aliased must not re-translate
    let first = records.iter().position(|r| r.addr == 0x0000 && r.op == "read").unwrap();
    let again = records.iter().position(|r| r.addr == 0x0004 && r.op == "read").unwrap();
    assert_eq!(records[first].calls, records[again].calls, "0x4 should hit");

    assert_eq!(PAGE_SIZE, 4096);
}
