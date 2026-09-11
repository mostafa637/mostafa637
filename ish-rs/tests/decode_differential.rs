//! Word-for-word differential test of `decode.rs` against the C original.
//!
//! `tests/fixtures/decode_reference.txt` comes from `tools/decode-dump.c`, which
//! supplies the ~150 macros `emu/decode.h` expects as *recorders* and includes
//! the header unmodified, twice, the way `asbestos/gen.c` does. So the fixture
//! is the real header's behaviour, not a model of it.
//!
//! What this test does and does not check, precisely:
//!
//! * The opcode step lists in `src/decode_table.rs` are *generated* from this
//!   same fixture, so comparing them back against it would be circular. What is
//!   not generated is everything around them: the prefix handling, the eight map
//!   transitions, the operand-size hand-off, the ModRM and immediate reads, the
//!   fault behaviour, and the choice of which class's tail to run. Those are
//!   hand-written, and every one of the 37,891 cases exercises them.
//! * The class split is the part worth stressing. The table was built from a
//!   corpus that carries one ModRM byte per class, but `tools/decode-dump.c`
//!   proves over 1,042,066 decodes that no opcode depends on anything about the
//!   ModRM byte except its class - so a ModRM byte the corpus never used still
//!   has to land on the same behaviour.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_decode_reference.sh
//! python3 ./tools/gen_decode_table.py
//! ```

use std::cell::RefCell;
use std::ptr::NonNull;

use ish_emu::cpu::AddrT;
use ish_emu::decode::{class_of, decode, Event, Size, NCLASS};
use ish_emu::mmu::{MemType, Mmu, MmuOps, PAGE_BITS};
use ish_emu::tlb::Tlb;

const FIXTURE: &str =
    concat!(env!("CARGO_MANIFEST_DIR"), "/tests/fixtures/decode_reference.txt");

struct Case {
    size: u8,
    ip: u32,
    bytes: Vec<u8>,
    events: Vec<String>,
    ret: u8,
    end_ip: u32,
}

struct Fixture {
    pages: usize,
    fill_mul: u32,
    fill_add: u32,
    unmap_page: u32,
    base_ip: u32,
    classes: usize,
    invariant: String,
    declared_cases: usize,
    cases: Vec<Case>,
}

thread_local! {
    static BACKING: RefCell<Vec<u8>> = const { RefCell::new(Vec::new()) };
    static UNMAP_PAGE: RefCell<u32> = const { RefCell::new(0) };
    static ALIAS_MASK: RefCell<u32> = const { RefCell::new(0) };
}

struct FakeMmu;

impl MmuOps for FakeMmu {
    fn translate(&mut self, addr: AddrT, _type_: MemType) -> Option<NonNull<u8>> {
        let unmap = UNMAP_PAGE.with(|u| *u.borrow());
        if addr >> PAGE_BITS == unmap {
            return None;
        }
        BACKING.with(|b| {
            let mask = ALIAS_MASK.with(|m| *m.borrow()) as usize;
            // SAFETY: filled once before any decode, never resized, and masking
            // keeps every offset inside it - the same aliasing the C driver has.
            let ptr = unsafe { b.borrow_mut().as_mut_ptr().add((addr as usize) & mask) };
            Some(unsafe { NonNull::new_unchecked(ptr) })
        })
    }
}

fn parse_fixture() -> Fixture {
    let text = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|e| panic!("cannot read {FIXTURE}: {e}"));
    let mut fx = Fixture {
        pages: 0,
        fill_mul: 0,
        fill_add: 0,
        unmap_page: 0,
        base_ip: 0,
        classes: 0,
        invariant: String::new(),
        declared_cases: 0,
        cases: Vec::new(),
    };
    let mut cur: Option<Case> = None;
    for line in text.lines() {
        let f: Vec<&str> = line.split(' ').collect();
        match f[0] {
            "#" => match f[1] {
                "pages" => {
                    fx.pages = f[2].parse().unwrap();
                    assert_eq!(f[3], "fill");
                    fx.fill_mul = f[4].parse().unwrap();
                    fx.fill_add = f[5].parse().unwrap();
                    assert_eq!(f[6], "unmap");
                    fx.unmap_page = f[7].parse().unwrap();
                    assert_eq!(f[8], "base_ip");
                    fx.base_ip = u32::from_str_radix(f[9], 16).unwrap();
                }
                "classes" => fx.classes = f[2].parse().unwrap(),
                "class_invariant" => fx.invariant = f[2..].join(" "),
                "cases" => fx.declared_cases = f[2].parse().unwrap(),
                _ => {}
            },
            "C" => {
                if let Some(c) = cur.take() {
                    panic!("case at ip {:#x} has no result line", c.ip);
                }
                cur = Some(Case {
                    size: f[1].parse().unwrap(),
                    ip: u32::from_str_radix(f[2], 16).unwrap(),
                    bytes: f[3..].iter().map(|b| u8::from_str_radix(b, 16).unwrap()).collect(),
                    events: Vec::new(),
                    ret: 0,
                    end_ip: 0,
                });
            }
            "E" => cur.as_mut().unwrap().events.push(f[1..].join(" ")),
            "R" => {
                let mut c = cur.take().unwrap();
                c.ret = f[1].parse().unwrap();
                c.end_ip = u32::from_str_radix(f[2], 16).unwrap();
                fx.cases.push(c);
            }
            _ => {}
        }
    }
    assert!(fx.pages > 0, "the fixture header was not parsed");
    assert_eq!(fx.classes, NCLASS, "the fixture and the port disagree on the class count");
    assert!(
        fx.invariant.starts_with("1042066 decodes"),
        "the fixture was not generated with the class invariant check: {:?}",
        fx.invariant
    );
    assert_eq!(fx.cases.len(), fx.declared_cases, "case count");
    fx
}

fn build_image(fx: &Fixture) {
    let bytes = fx.pages << PAGE_BITS;
    let mut backing = vec![0u8; bytes];
    for (i, slot) in backing.iter_mut().enumerate() {
        *slot = ((i as u32).wrapping_mul(fx.fill_mul).wrapping_add(fx.fill_add) & 0xff) as u8;
    }
    BACKING.with(|b| *b.borrow_mut() = backing);
    UNMAP_PAGE.with(|u| *u.borrow_mut() = fx.unmap_page);
    ALIAS_MASK.with(|m| *m.borrow_mut() = (bytes - 1) as u32);
}

fn place(ip: u32, bytes: &[u8]) {
    BACKING.with(|b| {
        let mut b = b.borrow_mut();
        b[ip as usize..ip as usize + bytes.len()].copy_from_slice(bytes);
    });
}

#[test]
fn decode_matches_the_c_reference() {
    let fx = parse_fixture();
    build_image(&fx);

    let mut backend = FakeMmu;
    let mut mmu = Mmu::new(&mut backend);
    let mut tlb = Tlb::new();
    tlb.refresh(&mut mmu);

    let mut events = 0usize;
    for c in &fx.cases {
        place(c.ip, &c.bytes);
        let size = if c.size == 16 { Size::B16 } else { Size::B32 };
        let mut ip: AddrT = c.ip;
        let mut out = Vec::new();
        let ret = decode(size, &mut ip, &mut tlb, &mut mmu, &mut out);

        let ctx = || {
            format!(
                "{}-bit case at {:#x} [{}]",
                c.size,
                c.ip,
                c.bytes.iter().map(|b| format!("{b:02x}")).collect::<Vec<_>>().join(" ")
            )
        };
        let got: Vec<String> = out.iter().map(Event::to_line).collect();
        assert_eq!(got, c.events, "{}: events", ctx());
        assert_eq!(ret.code(), c.ret, "{}: result", ctx());
        assert_eq!(ip, c.end_ip, "{}: instruction pointer", ctx());
        events += got.len();
    }

    println!(
        "{events} events over {} instruction decodings matched the C reference exactly",
        fx.cases.len()
    );
}

#[test]
fn the_corpus_reaches_every_map_and_both_sizes() {
    let fx = parse_fixture();
    let mut maps = [0usize; 8];
    let mut sizes = [0usize; 2];
    let mut classes = [0usize; NCLASS];
    let mut outcomes = [0usize; 4];
    for c in &fx.cases {
        sizes[if c.size == 16 { 1 } else { 0 }] += 1;
        outcomes[c.ret as usize] += 1;
        // classify by the prefix the bytes start with
        let mi = match c.bytes[0] {
            0x0f => 1,
            0xf0 => {
                if c.bytes.get(1) == Some(&0x0f) {
                    3
                } else {
                    2
                }
            }
            0xf2 => {
                if c.bytes.get(1) == Some(&0x0f) {
                    5
                } else {
                    4
                }
            }
            0xf3 => {
                if c.bytes.get(1) == Some(&0x0f) {
                    7
                } else {
                    6
                }
            }
            _ => 0,
        };
        maps[mi] += 1;
        // the ModRM byte sits right after the map's prefix bytes and the opcode
        let prefix_len = [0usize, 1, 1, 2, 1, 2, 1, 2][mi];
        if let Some(&m) = c.bytes.get(prefix_len + 1) {
            classes[class_of(m)] += 1;
        }
    }
    let names = ["base", "0f", "f0", "f0/0f", "f2", "f2/0f", "f3", "f3/0f"];
    for (name, n) in names.iter().zip(maps) {
        assert!(n > 500, "map {name} only reached {n} times");
    }
    assert!(sizes[0] > 18000 && sizes[1] > 18000, "sizes {:?}", sizes);
    assert!(outcomes[0] > 10000, "only {} plain decodes", outcomes[0]);
    assert!(outcomes[1] > 1000, "only {} block-ending decodes", outcomes[1]);
    assert!(outcomes[2] > 10000, "only {} undefined opcodes", outcomes[2]);
    assert_eq!(outcomes[3], 3, "the three fault cases");
    let covered = classes.iter().filter(|&&n| n > 0).count();
    assert_eq!(covered, NCLASS, "only {covered} of {NCLASS} ModRM classes appear");
}

#[test]
fn a_modrm_byte_the_corpus_never_used_still_decodes_the_same() {
    // The table was built from one ModRM byte per class. This drives the same
    // instructions with ModRM bytes the corpus does not contain - other mod
    // values, SIB forms, displacements - and requires the same operation out.
    // It is the check that the class abstraction, and not just the corpus, is
    // what the table encodes.
    let fx = parse_fixture();
    build_image(&fx);
    let mut backend = FakeMmu;
    let mut mmu = Mmu::new(&mut backend);
    let mut tlb = Tlb::new();
    tlb.refresh(&mut mmu);

    // (opcode, the ModRM byte the corpus used, a different byte, same class)
    let probes: &[(u8, u8, u8)] = &[
        (0x01, 0x00, 0b01_000_000), // add: mod=01 disp8 instead of mod=00
        (0x01, 0x00, 0b10_000_100), // add: SIB form, [eax + eax]
        (0x8b, 0x00, 0b10_000_101), // mov: disp32 with no base
        (0xc0, 0x08, 0b01_001_000), // grp2 ror with a disp8 instead of disp0
        (0xf7, 0x00, 0b01_000_110), // grp3, disp8 form
        (0x3b, 0x38, 0b10_111_100), // cmp, SIB with an index register
        // The register form cannot be probed this way: mod=11 is part of the
        // class, so each of the 64 register classes is exactly one ModRM byte.
        // Those are covered by the corpus itself, which carries all 64.
    ];
    let mut compared = 0;
    for &(op, corpus_modrm, other_modrm) in probes {
        assert_eq!(
            class_of(corpus_modrm),
            class_of(other_modrm),
            "the probe bytes are not in the same class"
        );
        let mut run = |modrm: u8| -> (Vec<String>, u8, AddrT) {
            let bytes = [op, modrm, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88];
            place(fx.base_ip, &bytes);
            let mut ip: AddrT = fx.base_ip;
            let mut out = Vec::new();
            let ret = decode(Size::B32, &mut ip, &mut tlb, &mut mmu, &mut out);
            (out.iter().map(Event::to_line).collect(), ret.code(), ip)
        };
        let (a, ra, _) = run(corpus_modrm);
        let (b, rb, _) = run(other_modrm);
        // the operations must be identical; only the bytes consumed may differ
        let ops = |v: &[String]| -> Vec<String> {
            v.iter()
                .filter(|e| *e != "READMODRM" && !e.starts_with("READIMM"))
                .cloned()
                .collect()
        };
        assert_eq!(ops(&a), ops(&b), "opcode {op:02x} changed operation across a class");
        assert_eq!(ra, rb, "opcode {op:02x} changed outcome across a class");
        compared += 1;
    }
    assert_eq!(compared, probes.len());
}
