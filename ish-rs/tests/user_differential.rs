//! Byte-for-byte differential test of `user.rs` against the C original.
//!
//! `tests/fixtures/user_reference.txt` is produced by `tools/user-dump.c`, which
//! links the **unmodified** `kernel/user.c` (on top of `kernel/memory.c`) and
//! drives `user_read`, `user_write`, `user_write_task_ptrace`,
//! `user_read_string` and `user_write_string` over an address space built to
//! make their corner cases reachable.
//!
//! What this compares that the other fixtures do not is *bytes on both sides*:
//! the host buffer after each call (`B`) and the guest pages the call could have
//! reached (`G`). Both matter because these functions do not roll back — a fault
//! on page *N* leaves pages *0..N-1* written — so a return-code-only comparison
//! would pass a port that wrote to the wrong page and faulted at the right one.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_user_reference.sh
//! ```

use std::ffi::CStr;

use ish_emu::memory::{Mem, PtEntry, MEM_PAGES};
use ish_emu::mmu::PAGE_BITS;
use ish_emu::user::{Fault, User};

const FIXTURE: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/tests/fixtures/user_reference.txt");

/// the generator's two patterns, so the port can rebuild any buffer from a seed
fn pat(j: usize, seed: u32) -> u8 {
    (j as u32 * 31 + seed) as u8
}
fn wpat(j: usize, seed: u32) -> u8 {
    (j as u32 * 7 + seed) as u8
}

enum Line {
    Op(Vec<String>),
    Ret(i32),
    Buf(Vec<u8>),
    Guest { tag: String, page: u32, flags: Option<u32>, bytes: Option<Vec<u8>> },
    State { tag: String, pgdir_used: i32, changes: u64 },
    Page { tag: String, page: u32, flags: u32, offset: usize, data_id: i32, refcount: usize },
    Counters { invalidations: u64, fd_closes: u64, signals: u64 },
    Header,
}

fn hex(f: &[String], from: usize) -> Vec<u8> {
    f[from..].iter().map(|s| u8::from_str_radix(s, 16).unwrap()).collect()
}

fn parse() -> Vec<Line> {
    let text = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|e| panic!("cannot read {FIXTURE}: {e}"));
    let mut out = Vec::new();
    for line in text.lines() {
        let f: Vec<&str> = line.split(' ').collect();
        let s: Vec<String> = f.iter().map(|s| s.to_string()).collect();
        match f[0] {
            "#" => {
                if f[1] == "asbestos_invalidations" {
                    out.push(Line::Counters {
                        invalidations: f[2].parse().unwrap(),
                        fd_closes: f[4].parse().unwrap(),
                        signals: f[6].parse().unwrap(),
                    });
                } else {
                    out.push(Line::Header);
                }
            }
            "O" => out.push(Line::Op(s[1..].to_vec())),
            "R" => out.push(Line::Ret(f[1].parse().unwrap())),
            "B" => out.push(Line::Buf(hex(&s, 1))),
            "G" => {
                if f[3] == "hole" {
                    out.push(Line::Guest {
                        tag: f[1].to_string(),
                        page: u32::from_str_radix(f[2], 16).unwrap(),
                        flags: None,
                        bytes: None,
                    });
                } else {
                    out.push(Line::Guest {
                        tag: f[1].to_string(),
                        page: u32::from_str_radix(f[2], 16).unwrap(),
                        flags: Some(u32::from_str_radix(f[3], 16).unwrap()),
                        bytes: Some(hex(&s, 4)),
                    });
                }
            }
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

/// the same identity the generator computes: first-appearance order of the
/// backing object, within one address space
fn data_ids(mem: &Mem) -> std::collections::HashMap<u32, (i32, usize)> {
    let mut seen: Vec<usize> = Vec::new();
    let mut out = std::collections::HashMap::new();
    for page in 0..MEM_PAGES {
        let pt: &PtEntry = match mem.pt(page) {
            Some(p) => p,
            None => continue,
        };
        let key = std::rc::Rc::as_ptr(&pt.data) as usize;
        let id = match seen.iter().position(|k| *k == key) {
            Some(i) => i as i32,
            None => {
                seen.push(key);
                seen.len() as i32 - 1
            }
        };
        out.insert(page, (id, std::rc::Rc::strong_count(&pt.data)));
    }
    out
}

fn ret(r: Result<(), Fault>) -> i32 {
    match r {
        Ok(()) => 0,
        Err(Fault) => 1,
    }
}

/// page bytes, or None for a hole
fn page_bytes(mem: &Mem, page: u32) -> Option<(u32, Vec<u8>)> {
    let pt = mem.pt(page)?;
    Some((pt.flags, pt.data.bytes[pt.offset..pt.offset + (1 << PAGE_BITS)].to_vec()))
}

#[test]
fn user_memory_matches_the_c_reference() {
    let lines = parse();
    let mut main = Mem::new();
    let mut child = Mem::new();
    let mut current_is_child = false;
    let mut ops = 0usize;
    let mut bytes_compared = 0usize;
    let mut pages_compared = 0usize;

    // the result of the operation whose R/B/G/S/P lines are still to come
    let mut pending: Option<(String, usize)> = None;
    let mut got_ret: Option<i32> = None;
    let mut got_buf: Option<Vec<u8>> = None;

    for line in &lines {
        match line {
            Line::Header => {}
            Line::Op(f) => {
                let mut buf_out: Option<Vec<u8>> = None;
                let r = match f[0].as_str() {
                    // ---- building the address space ----
                    "MF" => {
                        let (start, pages, flags, seed) =
                            (hp(f, 1), d(f, 2), d(f, 3), d(f, 4));
                        let bytes = (0..(pages as usize) << PAGE_BITS)
                            .map(|j| pat(j, seed))
                            .collect::<Vec<u8>>()
                            .into_boxed_slice();
                        main.map(start, pages, bytes, 0, flags)
                    }
                    "MN" => main.map_nothing(hp(f, 1), d(f, 2), d(f, 3)),
                    "CW" => Mem::copy_on_write(&mut main, &mut child, hp(f, 1), d(f, 2)),
                    // ---- the reads ----
                    "RD" | "RT" => {
                        let target = if f[1] == "child" { &mut child } else { &mut main };
                        if f[0] == "RD" {
                            assert_eq!(f[1] == "child", current_is_child,
                                "user_read follows `current`, the fixture must agree");
                        }
                        let mut buf = vec![0xaau8; d(f, 3) as usize];
                        let r = ret(User::new(target).read(hp(f, 2), &mut buf));
                        buf_out = Some(buf);
                        r
                    }
                    // ---- the writes ----
                    "WR" | "WT" | "WP" => {
                        let target = if f[1] == "child" { &mut child } else { &mut main };
                        if f[0] == "WR" {
                            assert_eq!(f[1] == "child", current_is_child,
                                "user_write follows `current`, the fixture must agree");
                        }
                        let (addr, count, seed) = (hp(f, 2), d(f, 3) as usize, d(f, 4));
                        let buf: Vec<u8> = (0..count).map(|j| wpat(j, seed)).collect();
                        let mut u = User::new(target);
                        ret(if f[0] == "WP" {
                            u.write_ptrace(addr, &buf)
                        } else {
                            u.write(addr, &buf)
                        })
                    }
                    // ---- the strings ----
                    "RS" => {
                        let target = if current_is_child { &mut child } else { &mut main };
                        let mut buf = vec![0xaau8; d(f, 2) as usize];
                        let r = ret(User::new(target).read_string(hp(f, 1), &mut buf));
                        buf_out = Some(buf);
                        r
                    }
                    "WS" => {
                        let target = if current_is_child { &mut child } else { &mut main };
                        let bytes = hex(f, 2);
                        assert_eq!(*bytes.last().unwrap(), 0, "a C string ends in a NUL");
                        let s = CStr::from_bytes_with_nul(&bytes).unwrap();
                        ret(User::new(target).write_string(hp(f, 1), s))
                    }
                    "CUR" => {
                        current_is_child = f[1] == "child";
                        0
                    }
                    other => panic!("unknown operation {other}"),
                };
                pending = Some((f.join(" "), ops));
                ops += 1;
                got_ret = Some(r);
                got_buf = buf_out;
            }
            Line::Ret(want) => {
                let (op, n) = pending.as_ref().expect("R without an O");
                assert_eq!(got_ret.unwrap(), *want, "operation #{n} `{op}`");
            }
            Line::Buf(want) => {
                let (op, n) = pending.as_ref().expect("B without an O");
                let got = got_buf.as_ref().unwrap_or_else(|| panic!("op #{n} produced no buffer"));
                assert_eq!(got.len(), want.len(), "operation #{n} `{op}`: buffer length");
                assert_eq!(got, want, "operation #{n} `{op}`: host buffer");
                bytes_compared += want.len();
            }
            Line::Guest { tag, page, flags, bytes } => {
                let m = if tag == "child" { &child } else { &main };
                let (op, n) = pending.as_ref().unwrap();
                match (page_bytes(m, *page), bytes) {
                    (None, None) => pages_compared += 1,
                    (None, Some(_)) => {
                        panic!("op #{n} `{op}`: {tag} page {page:x} is a hole in the port")
                    }
                    (Some(_), None) => {
                        panic!("op #{n} `{op}`: {tag} page {page:x} is mapped in the port")
                    }
                    (Some((got_flags, got)), Some(want)) => {
                        assert_eq!(got_flags, flags.unwrap(), "op #{n} `{op}`: {tag} {page:x} flags");
                        assert_eq!(got, *want, "op #{n} `{op}`: {tag} page {page:x} bytes");
                        assert_eq!(got.len(), 1 << PAGE_BITS);
                        pages_compared += 1;
                        bytes_compared += want.len();
                    }
                }
            }
            Line::State { tag, pgdir_used, changes } => {
                let m = if tag == "child" { &child } else { &main };
                assert_eq!(m.pgdir_used, *pgdir_used, "{tag}: pgdir_used");
                assert_eq!(m.changes(), *changes, "{tag}: change count");
            }
            Line::Page { tag, page, flags, offset, data_id, refcount } => {
                let m = if tag == "child" { &child } else { &main };
                let pt = m.pt(*page).unwrap_or_else(|| panic!("{tag} page {page:x} not mapped"));
                assert_eq!(pt.flags, *flags, "{tag} page {page:x}: flags");
                assert_eq!(pt.offset, *offset, "{tag} page {page:x}: offset");
                let ids = data_ids(m);
                let (got_id, got_refs) = ids[page];
                assert_eq!(got_id, *data_id, "{tag} page {page:x}: backing object");
                assert_eq!(got_refs, *refcount, "{tag} page {page:x}: refcount");
            }
            Line::Counters { invalidations, fd_closes, signals } => {
                assert_eq!(main.invalidations + child.invalidations, *invalidations,
                    "page invalidations");
                assert_eq!(*fd_closes, 0, "no fd should ever be closed");
                assert_eq!(*signals, 0, "no signal should ever be sent");
            }
        }
    }
    println!(
        "{ops} operations, {pages_compared} guest pages and {bytes_compared} bytes \
         matched the C exactly"
    );
}

/// decimal field
fn d(f: &[String], i: usize) -> u32 {
    f[i].parse().unwrap()
}

/// hex field
fn hp(f: &[String], i: usize) -> u32 {
    u32::from_str_radix(&f[i], 16).unwrap()
}
