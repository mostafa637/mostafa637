//! Differential test of `mmap.rs` against the C original.
//!
//! `tests/fixtures/mmap_reference.txt` is produced by `tools/mmap-dump.c`, which
//! links the **unmodified** `kernel/mmap.c` on top of `kernel/memory.c` and
//! `kernel/user.c` and drives `sys_mmap2`, `sys_mmap`, `sys_mprotect`,
//! `sys_mremap`, `sys_munmap`, `sys_brk` and the three no-op syscalls.
//!
//! Returns are compared as raw 32-bit words, because that is what these syscalls
//! return: an address on success and a negated errno reinterpreted as `addr_t` on
//! failure. Printing them in decimal would hide which half of that union a caller
//! is looking at.
//!
//! After every operation the whole page table is compared — flags, offset,
//! backing-object identity and reference count — plus `pgdir_used`, the change
//! count and the brk.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_mmap_reference.sh
//! ```

use std::rc::Rc;

use ish_emu::memory::{Mem, PtEntry, MEM_PAGES};
use ish_emu::mmap::Mm;
use ish_emu::mmu::PAGE_BITS;
use ish_emu::user::User;

const FIXTURE: &str =
    concat!(env!("CARGO_MANIFEST_DIR"), "/tests/fixtures/mmap_reference.txt");

enum Line {
    Op(Vec<String>),
    Ret(u32),
    State { pgdir_used: i32, changes: u64, brk: u32, start_brk: u32, refcount: usize },
    Page { page: u32, flags: u32, offset: usize, data_id: i32, refcount: usize },
    Counters { invalidations: u64, fd_closes: u64, fd_lookups: u64, signals: u64 },
    Header,
}

fn parse() -> Vec<Line> {
    let text = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|e| panic!("cannot read {FIXTURE}: {e}"));
    let mut out = Vec::new();
    for line in text.lines() {
        let f: Vec<&str> = line.split(' ').collect();
        let s: Vec<String> = f.iter().map(|x| x.to_string()).collect();
        match f[0] {
            "#" if f[1] == "asbestos_invalidations" => out.push(Line::Counters {
                invalidations: f[2].parse().unwrap(),
                fd_closes: f[4].parse().unwrap(),
                fd_lookups: f[6].parse().unwrap(),
                signals: f[8].parse().unwrap(),
            }),
            "#" => out.push(Line::Header),
            "O" => out.push(Line::Op(s[1..].to_vec())),
            "R" => out.push(Line::Ret(u32::from_str_radix(f[1], 16).unwrap())),
            "S" => out.push(Line::State {
                pgdir_used: f[3].parse().unwrap(),
                changes: f[5].parse().unwrap(),
                brk: u32::from_str_radix(f[7], 16).unwrap(),
                start_brk: u32::from_str_radix(f[9], 16).unwrap(),
                refcount: f[11].parse().unwrap(),
            }),
            "P" => out.push(Line::Page {
                page: u32::from_str_radix(f[2], 16).unwrap(),
                flags: u32::from_str_radix(f[3], 16).unwrap(),
                offset: usize::from_str_radix(f[4], 16).unwrap(),
                data_id: f[5].parse().unwrap(),
                refcount: f[6].parse().unwrap(),
            }),
            _ => panic!("unparsed fixture line: {line}"),
        }
    }
    out
}

fn hp(f: &[String], i: usize) -> u32 {
    u32::from_str_radix(&f[i], 16).unwrap()
}
fn d(f: &[String], i: usize) -> u32 {
    f[i].parse().unwrap()
}

/// the C returns `addr_t`: an address, or a negated errno reinterpreted
fn raw(r: Result<u32, i32>) -> u32 {
    match r {
        Ok(v) => v,
        Err(e) => e as u32,
    }
}
fn raw_unit(r: Result<(), i32>) -> u32 {
    match r {
        Ok(()) => 0,
        Err(e) => e as u32,
    }
}

fn data_ids(mem: &Mem) -> std::collections::HashMap<u32, (i32, usize)> {
    let mut seen: Vec<usize> = Vec::new();
    let mut out = std::collections::HashMap::new();
    for page in 0..MEM_PAGES {
        let pt: &PtEntry = match mem.pt(page) {
            Some(p) => p,
            None => continue,
        };
        let key = Rc::as_ptr(&pt.data) as usize;
        let id = match seen.iter().position(|k| *k == key) {
            Some(i) => i as i32,
            None => {
                seen.push(key);
                seen.len() as i32 - 1
            }
        };
        out.insert(page, (id, Rc::strong_count(&pt.data)));
    }
    out
}

#[test]
fn mmap_matches_the_c_reference() {
    let lines = parse();
    let mut mm = Mm::new();
    // the way the generator sets the heap up, before the first operation
    mm.start_brk = 0x1000 << PAGE_BITS;
    mm.brk = mm.start_brk;

    let mut ops = 0usize;
    let mut pending: Option<(String, usize)> = None;
    let mut got: Option<u32> = None;

    for line in &lines {
        match line {
            Line::Header => {}
            Line::Op(f) => {
                let r = match f[0].as_str() {
                    "M2" => raw(mm.mmap2(hp(f, 1), hp(f, 2), d(f, 3), d(f, 4), hp(f, 5))),
                    "MM" => {
                        // the generator wrote the argument struct into guest
                        // memory first; when that write faults there is nothing
                        // to read and the syscall has to say so
                        let raw_args: Vec<u8> = (2..8)
                            .flat_map(|i| hp(f, i).to_le_bytes())
                            .collect();
                        let _ = User::new(&mut mm.mem).write(hp(f, 1), &raw_args);
                        raw(mm.mmap(hp(f, 1)))
                    }
                    "MN" => mm.mem.map_nothing(hp(f, 1), d(f, 2), d(f, 3)) as u32,
                    "MP" => raw_unit(mm.mprotect(hp(f, 1), hp(f, 2), d(f, 3))),
                    "MR" => raw(mm.mremap(hp(f, 1), hp(f, 2), hp(f, 3), d(f, 4))),
                    "MU" => raw_unit(mm.munmap(hp(f, 1), hp(f, 2))),
                    "BR" => mm.brk(hp(f, 1)),
                    "MA" => mm.madvise(hp(f, 1), hp(f, 2), d(f, 3)),
                    "ML" => mm.mlock(hp(f, 1), hp(f, 2)) as u32,
                    "MS" => mm.msync(hp(f, 1), hp(f, 2), d(f, 3)) as u32,
                    "RT" => {
                        mm.retain();
                        mm.refcount as u32
                    }
                    "RL" => {
                        mm.release();
                        mm.refcount as u32
                    }
                    other => panic!("unknown operation {other}"),
                };
                pending = Some((f.join(" "), ops));
                ops += 1;
                got = Some(r);
            }
            Line::Ret(want) => {
                let (op, n) = pending.as_ref().expect("R without an O");
                assert_eq!(got.unwrap(), *want, "operation #{n} `{op}`");
            }
            Line::State { pgdir_used, changes, brk, start_brk, refcount } => {
                let (op, n) = pending.as_ref().unwrap();
                let ctx = || format!("op #{n} `{op}`");
                assert_eq!(mm.mem.pgdir_used, *pgdir_used, "{}: pgdir_used", ctx());
                assert_eq!(mm.mem.changes(), *changes, "{}: change count", ctx());
                assert_eq!(mm.brk, *brk, "{}: brk", ctx());
                assert_eq!(mm.start_brk, *start_brk, "{}: start_brk", ctx());
                assert_eq!(mm.refcount, *refcount, "{}: refcount", ctx());
            }
            Line::Page { page, flags, offset, data_id, refcount } => {
                let (op, n) = pending.as_ref().unwrap();
                let pt = mm
                    .mem
                    .pt(*page)
                    .unwrap_or_else(|| panic!("op #{n} `{op}`: page {page:x} is not mapped"));
                let ctx = || format!("op #{n} `{op}`: page {page:x}");
                assert_eq!(pt.flags, *flags, "{}: flags", ctx());
                assert_eq!(pt.offset, *offset, "{}: offset", ctx());
                let ids = data_ids(&mm.mem);
                let (got_id, got_refs) = ids[page];
                assert_eq!(got_id, *data_id, "{}: backing object", ctx());
                assert_eq!(got_refs, *refcount, "{}: refcount", ctx());
            }
            Line::Counters { invalidations, fd_closes, fd_lookups, signals } => {
                assert_eq!(mm.mem.invalidations, *invalidations, "page invalidations");
                assert_eq!(*fd_closes, 0, "no fd should ever be closed");
                assert_eq!(*fd_lookups, 1, "exactly one file-backed mmap was attempted");
                assert_eq!(*signals, 0, "no signal should ever be sent");
            }
        }
    }
    println!("{ops} operations and the full page table after each matched the C exactly");
}

#[test]
fn every_mapped_page_survives_a_full_walk() {
    // mem_next_page skips unallocated directories; walking with it has to reach
    // exactly the pages the fixture lists, so the skipping is checked against
    // the mappings the syscalls actually produced
    let lines = parse();
    let mut mm = Mm::new();
    mm.start_brk = 0x1000 << PAGE_BITS;
    mm.brk = mm.start_brk;
    let mut walked_at_state: Option<Vec<u32>> = None;
    let mut checked = 0usize;

    for line in &lines {
        match line {
            // The return values are dropped here on purpose: this test only
            // asks which pages exist afterwards, and the test above compares
            // every return value against the C.
            Line::Op(f) => match f[0].as_str() {
                "M2" => {
                    let _ = mm.mmap2(hp(f, 1), hp(f, 2), d(f, 3), d(f, 4), hp(f, 5));
                }
                "MM" => {
                    let raw_args: Vec<u8> =
                        (2..8).flat_map(|i| hp(f, i).to_le_bytes()).collect();
                    let _ = User::new(&mut mm.mem).write(hp(f, 1), &raw_args);
                    let _ = mm.mmap(hp(f, 1));
                }
                "MN" => {
                    let _ = mm.mem.map_nothing(hp(f, 1), d(f, 2), d(f, 3));
                }
                "MP" => {
                    let _ = mm.mprotect(hp(f, 1), hp(f, 2), d(f, 3));
                }
                "MR" => {
                    let _ = mm.mremap(hp(f, 1), hp(f, 2), hp(f, 3), d(f, 4));
                }
                "MU" => {
                    let _ = mm.munmap(hp(f, 1), hp(f, 2));
                }
                "BR" => {
                    mm.brk(hp(f, 1));
                }
                _ => {}
            },
            Line::State { .. } => {
                let mut walked = Vec::new();
                let mut page = 0u32;
                loop {
                    if mm.mem.pt(page).is_some() {
                        walked.push(page);
                    }
                    let before = page;
                    mm.mem.next_page(&mut page);
                    if page <= before || page >= MEM_PAGES {
                        break;
                    }
                }
                walked_at_state = Some(walked);
            }
            Line::Page { page, .. } => {
                if let Some(walked) = &walked_at_state {
                    if !walked.contains(page) {
                        let mut mapped = Vec::new();
                        for q in 0..MEM_PAGES {
                            if mm.mem.pt(q).is_some() { mapped.push(q); }
                        }
                        panic!("page {page:x} missed by next_page; mapped={mapped:x?} walked={walked:x?}");
                    }
                    checked += 1;
                }
            }
            _ => {}
        }
    }
    assert!(checked > 100, "only {checked} page records cross-checked");
}
