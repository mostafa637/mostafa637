//! Word-for-word differential test of `memory.rs` against the C original.
//!
//! `tests/fixtures/memory_reference.txt` is produced by `tools/memory-dump.c`,
//! which links the **unmodified** `kernel/memory.c` and drives its page table
//! through a scripted sequence of operations. The script is printed into the
//! fixture as it runs, so this test replays the same operations rather than
//! duplicating the generator's tables.
//!
//! Host pointers never appear in the fixture. What is compared per mapped page
//! is its flags, its offset into the backing object, its reference count, and a
//! small identity for the backing object assigned in first-appearance order —
//! so "these two pages share one object" survives the trip and the addresses do
//! not.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_memory_reference.sh
//! ```

use std::collections::HashMap;
use std::rc::Rc;

use ish_emu::memory::{Mem, PtEntry, MEM_PAGES};
use ish_emu::mmu::{MemType, PAGE_BITS};

const FIXTURE: &str =
    concat!(env!("CARGO_MANIFEST_DIR"), "/tests/fixtures/memory_reference.txt");

enum Line {
    Op(Vec<String>),
    Ret(String),
    State { tag: String, pgdir_used: i32, changes: u64 },
    Page { tag: String, page: u32, flags: u32, offset: usize, data_id: i32, refcount: usize },
    Header(String, String),
    Counters { invalidations: u64, fd_closes: u64, signals: u64 },
}

fn parse() -> Vec<Line> {
    let text = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|e| panic!("cannot read {FIXTURE}: {e}"));
    let mut out = Vec::new();
    for line in text.lines() {
        let f: Vec<&str> = line.split(' ').collect();
        match f[0] {
            "#" if f[1] == "real_page_size" => {
                out.push(Line::Header("real_page_size".into(), f[2].into()))
            }
            "#" if f[1] == "asbestos_invalidations" => out.push(Line::Counters {
                invalidations: f[2].parse().unwrap(),
                fd_closes: f[4].parse().unwrap(),
                signals: f[6].parse().unwrap(),
            }),
            "#" => {}
            "O" => out.push(Line::Op(f[1..].iter().map(|s| s.to_string()).collect())),
            "R" => out.push(Line::Ret(f[1].to_string())),
            "S" => out.push(Line::State {
                tag: f[1].to_string(),
                pgdir_used: f[3].parse().unwrap(),
                changes: f[5].parse().unwrap(),
            }),
            "P" => out.push(Line::Page {
                tag: f[1].to_string(),
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

fn mem_type(n: u32) -> MemType {
    match n {
        0 => MemType::Read,
        1 => MemType::Write,
        2 => MemType::WritePtrace,
        _ => panic!("bad mem type {n}"),
    }
}

/// The same identity the generator computes: the index of the first page, in
/// page order, that references the same backing object.
fn data_ids(mem: &Mem) -> HashMap<u32, (i32, usize)> {
    let mut ids: Vec<(usize, i32)> = Vec::new();
    let mut next = 0;
    let mut out = HashMap::new();
    for page in 0..MEM_PAGES {
        let pt: &PtEntry = match mem.pt(page) {
            Some(p) => p,
            None => continue,
        };
        let key = Rc::as_ptr(&pt.data) as usize;
        let id = match ids.iter().find(|(k, _)| *k == key) {
            Some((_, id)) => *id,
            None => {
                ids.push((key, next));
                next += 1;
                next - 1
            }
        };
        out.insert(page, (id, Rc::strong_count(&pt.data)));
    }
    out
}

#[test]
fn memory_matches_the_c_reference() {
    let lines = parse();
    let mut main = Mem::new();
    let mut child = Mem::new();
    let mut ops = 0usize;
    let mut pages_compared = 0usize;
    let mut pending: Option<(Vec<String>, usize)> = None;
    let mut last_ret: Option<String> = None;

    for line in &lines {
        match line {
            Line::Header(k, v) => {
                assert_eq!(k, "real_page_size");
                assert_eq!(v.parse::<usize>().unwrap(), 4096, "this corpus assumes 4K pages");
            }
            Line::Op(f) => {
                let ret = match f[0].as_str() {
                    "MN" => main
                        .map_nothing(hp(f, 1), h(f, 2), h(f, 3))
                        .to_string(),
                    "MP" => {
                        let (start, pages, offset) = (hp(f, 1), h(f, 2), hp(f, 3) as usize);
                        let bytes =
                            vec![0u8; ((pages as usize) << PAGE_BITS) + offset].into_boxed_slice();
                        main.map(start, pages, bytes, offset, h(f, 4)).to_string()
                    }
                    "UN" => main.unmap(hp(f, 1), h(f, 2)).to_string(),
                    "UA" => main.unmap_always(hp(f, 1), h(f, 2)).to_string(),
                    "SF" => main.set_flags(hp(f, 1), h(f, 2), h(f, 3)).to_string(),
                    "IH" => (main.is_hole(hp(f, 1), h(f, 2)) as i32).to_string(),
                    "FH" => format!("{:x}", main.find_hole(h(f, 1))),
                    "CW" => {
                        Mem::copy_on_write(&mut main, &mut child, hp(f, 1), h(f, 2)).to_string()
                    }
                    "NP" => {
                        let mut p = hp(f, 1);
                        main.next_page(&mut p);
                        format!("{p:x}")
                    }
                    "PT" => {
                        let target = if f[3] == "child" { &mut child } else { &mut main };
                        let got = target.ptr(hp(f, 1), mem_type(h(f, 2)));
                        (got.is_some() as i32).to_string()
                    }
                    // mem_segv_reason takes no access type: mapped -> ACCERR,
                    // unmapped -> MAPERR
                    "SG" => main.segv_reason(hp(f, 1)).to_string(),
                    other => panic!("unknown operation {other}"),
                };
                pending = Some((f.clone(), ops));
                ops += 1;
                // the return value follows on the next line
                last_ret = Some(ret);
            }
            Line::Ret(want) => {
                let (f, n) = pending.take().expect("R without an O");
                let got = last_ret.take().expect("R without a computed result");
                assert_eq!(&got, want, "operation #{} `{}`", n, f.join(" "));
            }
            Line::State { tag, pgdir_used, changes } => {
                let m = if tag == "child" { &child } else { &main };
                assert_eq!(m.pgdir_used, *pgdir_used, "{tag}: pgdir_used after op #{ops}");
                assert_eq!(m.changes(), *changes, "{tag}: change count after op #{ops}");
            }
            Line::Page { tag, page, flags, offset, data_id, refcount } => {
                let m = if tag == "child" { &child } else { &main };
                let pt = m.pt(*page).unwrap_or_else(|| {
                    panic!("{tag}: page {page:x} is not mapped after op #{ops}")
                });
                let ctx = || format!("{tag} page {page:x} after op #{ops}");
                assert_eq!(pt.flags, *flags, "{}: flags", ctx());
                assert_eq!(pt.offset, *offset, "{}: offset", ctx());
                let ids = data_ids(m);
                let (got_id, got_refs) = ids[page];
                assert_eq!(got_id, *data_id, "{}: backing object", ctx());
                assert_eq!(got_refs, *refcount, "{}: refcount", ctx());
                pages_compared += 1;
            }
            Line::Counters { invalidations, fd_closes, signals } => {
                assert_eq!(main.invalidations + child.invalidations, *invalidations,
                    "page invalidations");
                assert_eq!(*fd_closes, 0, "no fd should ever be closed");
                assert_eq!(*signals, 0, "no signal should ever be sent");
            }
        }
    }
    println!("{ops} operations and {pages_compared} page records matched the C exactly");
}

/// counts, flags and mem types are printed in decimal
fn h(f: &[String], i: usize) -> u32 {
    f[i].parse().unwrap()
}

/// page numbers, addresses and offsets are printed in hex
fn hp(f: &[String], i: usize) -> u32 {
    u32::from_str_radix(&f[i], 16).unwrap()
}

#[test]
fn every_mapped_page_in_the_corpus_is_reachable_both_ways() {
    // The C reference walks its page table with mem_pt in a plain loop. Walk it
    // the way the C's own traversal helper does - mem_next_page, which skips
    // unallocated directories - and require the same set of pages, so the
    // skipping logic is checked against the mapping it produced.
    let lines = parse();
    let mut main = Mem::new();
    let mut child = Mem::new();
    let mut seen: Vec<u32> = Vec::new();
    let mut last_state: Option<(String, Vec<u32>)> = None;

    for line in &lines {
        match line {
            Line::Op(f) => match f[0].as_str() {
                "MN" => {
                    main.map_nothing(hp(f, 1), h(f, 2), h(f, 3));
                }
                "MP" => {
                    let (start, pages, offset) = (hp(f, 1), h(f, 2), hp(f, 3) as usize);
                    let bytes =
                        vec![0u8; ((pages as usize) << PAGE_BITS) + offset].into_boxed_slice();
                    main.map(start, pages, bytes, offset, h(f, 4));
                }
                "UN" => {
                    main.unmap(hp(f, 1), h(f, 2));
                }
                "UA" => {
                    main.unmap_always(hp(f, 1), h(f, 2));
                }
                "SF" => {
                    main.set_flags(hp(f, 1), h(f, 2), h(f, 3));
                }
                "CW" => {
                    Mem::copy_on_write(&mut main, &mut child, hp(f, 1), h(f, 2));
                }
                "PT" => {
                    let target = if f[3] == "child" { &mut child } else { &mut main };
                    target.ptr(hp(f, 1), mem_type(h(f, 2)));
                }
                _ => {}
            },
            Line::State { tag, .. } => {
                let m = if tag == "child" { &child } else { &main };
                // walk with next_page
                let mut walked = Vec::new();
                let mut page = 0u32;
                loop {
                    if m.pt(page).is_some() {
                        walked.push(page);
                    }
                    let before = page;
                    m.next_page(&mut page);
                    if page <= before || page >= MEM_PAGES {
                        break;
                    }
                }
                last_state = Some((tag.clone(), walked));
            }
            Line::Page { tag, page, .. } => {
                if let Some((t, walked)) = &last_state {
                    if t == tag {
                        assert!(walked.contains(page), "{tag} page {page:x} missed by next_page");
                        seen.push(*page);
                    }
                }
            }
            _ => {}
        }
    }
    assert!(seen.len() > 100, "only {} page records cross-checked", seen.len());
}
