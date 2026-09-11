//! Word-for-word differential test of `modrm.rs` against the C original.
//!
//! `tests/fixtures/modrm_reference.txt` is produced by `tools/modrm-dump.c`,
//! which includes the **unmodified** `emu/modrm.h` — `modrm_decode32` is
//! `static inline`, so there is no separate object to link against; the header
//! *is* the implementation.
//!
//! The corpus is every one of the 256 ModRM bytes, every one of the 256 SIB
//! bytes under each of the four `mod` values (including `mod=11`, where
//! `rm=100` is *not* a SIB escape), and three accesses placed against an
//! unmapped page so the fault path and the instruction pointer's behaviour on a
//! fault are compared too. 1,283 cases in all.
//!
//! The corpus is described in the fixture header — the backing-store fill, the
//! unmapped page, every byte the C driver wrote and each starting ip — so this
//! test rebuilds the identical memory image instead of duplicating the
//! generator's tables.
//!
//! Regenerate with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_modrm_reference.sh
//! ```

use std::cell::RefCell;
use std::ptr::NonNull;

use ish_emu::cpu::{AddrT, Reg32};
use ish_emu::mmu::{MemType, Mmu, MmuOps, PAGE_BITS};
use ish_emu::modrm::{modrm_decode32, Modrm, ModrmType};
use ish_emu::tlb::Tlb;

const FIXTURE: &str =
    concat!(env!("CARGO_MANIFEST_DIR"), "/tests/fixtures/modrm_reference.txt");

/// One decoded record from the C.
#[derive(Debug)]
struct Record {
    ip: u32,
    ok: bool,
    end_ip: u32,
    type_: u32,
    reg: u32,
    base: u32,
    offset: i32,
    index: u32,
    shift: u32,
}

/// The fixture: the memory image to rebuild plus what the C decoded from it.
struct Fixture {
    pages: usize,
    fill_mul: u32,
    fill_add: u32,
    unmap_page: u32,
    writes: Vec<(u32, Vec<u8>)>,
    records: Vec<Record>,
    declared_cases: usize,
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
            // SAFETY: the Vec is filled once before any decode and never
            // resized afterwards, and masking keeps the offset inside it.
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
        writes: Vec::new(),
        records: Vec::new(),
        declared_cases: 0,
    };
    for line in text.lines() {
        let f: Vec<&str> = line.split(' ').collect();
        match f[0] {
            "#" if f[1] == "pages" => {
                fx.pages = f[2].parse().unwrap();
                assert_eq!(f[3], "fill");
                fx.fill_mul = f[4].parse().unwrap();
                fx.fill_add = f[5].parse().unwrap();
                assert_eq!(f[6], "unmap");
                fx.unmap_page = f[7].parse().unwrap();
            }
            "#" if f[1] == "W" => {
                let addr = u32::from_str_radix(f[2], 16).unwrap();
                let bytes: Vec<u8> =
                    f[3..].iter().map(|b| u8::from_str_radix(b, 16).unwrap()).collect();
                fx.writes.push((addr, bytes));
            }
            "#" if f[1] == "cases" => fx.declared_cases = f[2].parse().unwrap(),
            "M" => {
                assert_eq!(f.len(), 11, "malformed record: {line}");
                fx.records.push(Record {
                    ip: u32::from_str_radix(f[2], 16).unwrap(),
                    ok: f[3] == "1",
                    end_ip: u32::from_str_radix(f[4], 16).unwrap(),
                    type_: f[5].parse().unwrap(),
                    reg: f[6].parse().unwrap(),
                    base: f[7].parse().unwrap(),
                    offset: u32::from_str_radix(f[8], 16).unwrap() as i32,
                    index: f[9].parse().unwrap(),
                    shift: f[10].parse().unwrap(),
                });
            }
            _ => {}
        }
    }
    assert!(fx.pages > 0, "the fixture header was not parsed");
    assert_eq!(fx.records.len(), fx.declared_cases, "case count");
    fx
}

fn build_image(fx: &Fixture) {
    let bytes = fx.pages << PAGE_BITS;
    let mut backing = vec![0u8; bytes];
    for (i, slot) in backing.iter_mut().enumerate() {
        *slot = ((i as u32).wrapping_mul(fx.fill_mul).wrapping_add(fx.fill_add) & 0xff) as u8;
    }
    for (addr, data) in &fx.writes {
        backing[*addr as usize..*addr as usize + data.len()].copy_from_slice(data);
    }
    BACKING.with(|b| *b.borrow_mut() = backing);
    UNMAP_PAGE.with(|u| *u.borrow_mut() = fx.unmap_page);
    ALIAS_MASK.with(|m| *m.borrow_mut() = (bytes - 1) as u32);
}

#[test]
fn modrm_matches_the_c_reference() {
    let fx = parse_fixture();
    build_image(&fx);

    let mut backend = FakeMmu;
    let mut mmu = Mmu::new(&mut backend);
    let mut tlb = Tlb::new();
    tlb.refresh(&mut mmu);

    let mut compared = 0usize;
    for r in &fx.records {
        let mut ip: AddrT = r.ip;
        let mut m = Modrm::default();
        let ok = modrm_decode32(&mut ip, &mut tlb, &mut mmu, &mut m);

        let ctx = || format!("case {} (ip {:#x})", r.ip, r.ip);
        assert_eq!(ok, r.ok, "{}: result", ctx());
        assert_eq!(ip, r.end_ip, "{}: instruction pointer", ctx());
        let got_type = match m.type_ {
            ModrmType::Reg => 0,
            ModrmType::Mem => 1,
            ModrmType::MemSi => 2,
        };
        assert_eq!(got_type, r.type_, "{}: type", ctx());
        assert_eq!(m.reg as u32, r.reg, "{}: reg", ctx());
        assert_eq!(m.base as u32, r.base, "{}: base/rm", ctx());
        assert_eq!(m.offset, r.offset, "{}: offset", ctx());
        assert_eq!(m.index as u32, r.index, "{}: index", ctx());
        let got_shift = m.shift as u32;
        assert_eq!(got_shift, r.shift, "{}: shift", ctx());
        compared += 8;
    }

    println!(
        "{compared} decode fields over {} cases matched the C reference exactly",
        fx.records.len()
    );
}

#[test]
fn the_corpus_covers_the_interesting_forms() {
    let fx = parse_fixture();
    let (mut reg_form, mut mem_si, mut faults, mut no_base, mut ebp, mut times8) =
        (0usize, 0, 0, 0, 0, 0);
    for r in &fx.records {
        if !r.ok {
            faults += 1;
            continue;
        }
        match r.type_ {
            0 => reg_form += 1,
            2 => mem_si += 1,
            _ => {}
        }
        // base 8 is reg_none: the disp32-with-no-base forms. Only mod=00 gets
        // there — with mod=01 or mod=10 an rm (or SIB base) of 101 stays ebp
        // and carries an 8- or 32-bit displacement.
        if r.base == 8 {
            no_base += 1;
        }
        if r.base == 5 {
            ebp += 1;
        }
        // scale 8 has no name in the C's enum but is a live gadget index
        if r.shift == 3 {
            times8 += 1;
        }
    }
    // 64 plain mod=11 ModRM bytes, plus the 256-byte SIB sweep run under
    // mod=11 as well — there `rm=100` is an ordinary register, not a SIB escape
    assert_eq!(reg_form, 320, "the mod=11 register forms");
    // 64 of the 256 SIB bytes have scale 8, and only three of the four mod
    // rows actually read a SIB byte (mod=11 has none), so 64 * 3
    assert_eq!(times8, 192, "the scale-8 SIB index sweep");
    assert!(mem_si > 500, "only {mem_si} indexed-memory operands");
    assert_eq!(faults, 3, "expected exactly the three fault cases");
    // 8 plain rm=101 with mod=00, plus 32 SIB bytes whose base is 101
    assert_eq!(no_base, 40, "disp32-with-no-base operands");
    // 8 rm=101 with mod=01, 8 with mod=10, 32 SIB base=101 for each of those
    // two modes, and 8 mod=11 rm=101 register operands
    assert_eq!(ebp, 88, "operands based on ebp");
}

#[test]
fn specific_encodings_decode_the_way_intel_documents() {
    // Checked against the architecture rather than against the C, so a mistake
    // shared by both cannot hide here.
    let fx = parse_fixture();
    build_image(&fx);
    let decode_at = |ip: u32| -> (bool, Modrm, AddrT) {
        let mut backend = FakeMmu;
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);
        let mut ip = ip;
        let mut m = Modrm::default();
        let ok = modrm_decode32(&mut ip, &mut tlb, &mut mmu, &mut m);
        (ok, m, ip)
    };

    // case 0 is ModRM 0x00 at ip 0x400: mod=00 reg=eax rm=eax -> [eax]
    let (ok, m, ip) = decode_at(0x400);
    assert!(ok);
    assert_eq!(m.type_, ModrmType::Mem);
    assert_eq!(m.reg, Reg32::Eax);
    assert_eq!(m.base, Reg32::Eax);
    assert_eq!(m.offset, 0);
    assert_eq!(ip, 0x401, "one byte consumed");

    // case 5 is ModRM 0x05 at ip 0x428: mod=00 rm=101 -> disp32, no base
    let (ok, m, ip) = decode_at(0x428);
    assert!(ok);
    assert_eq!(m.base, Reg32::None);
    assert_eq!(m.offset, 0x4433_2211, "the four filler bytes, little endian");
    assert_eq!(ip, 0x42d, "modrm byte plus a dword");

    // case 0xc0 is ModRM 0xc0 at ip 0xa00: mod=11 -> register to register
    let (ok, m, ip) = decode_at(0xa00);
    assert!(ok);
    assert_eq!(m.type_, ModrmType::Reg);
    assert_eq!(m.reg, Reg32::Eax);
    assert_eq!(m.base, Reg32::Eax);
    assert_eq!(ip, 0xa01, "no displacement for a register operand");
}
