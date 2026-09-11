// The instruction decoder: a translation of the dispatch in emu/decode.h.
//
// emu/decode.h is not a library. It is a 1,416-line template that
// asbestos/gen.c includes twice - once with OP_SIZE 32, once with 16 - and
// supplies ~150 macros that turn each decoded instruction into gadget
// emissions. So there are two things in it, and they port differently:
//
//   * the *dispatch*: prefix handling, the eight opcode maps, the ModRM and
//     immediate reads, the GRP groups, the x87 split, and where an instruction
//     ends a block. That is real logic, and it is translated here by hand.
//   * the *semantics*: what ADD or CVTSS2SD actually does. That lives in the
//     gadget backends (asbestos/gadgets-*), which are not ported. The decoder's
//     job ends at naming the operation and its operands, which is what
//     [`Op`] and the argument tokens record.
//
// The per-opcode step lists live in `decode_table.rs`, generated from a
// reference run of the unmodified header by tools/gen_decode_table.py. Hand
// transcription of 648 case labels would be slower and, worse, unverifiable;
// the generated table is checked by tools/decode-dump.c, which proves over
// 1,042,066 decodes that once the ModRM byte is consumed the dispatch depends
// on it only through [`class_of`], and by tests/decode_differential.rs, which
// replays 37,891 byte strings through the hand-written machinery below.

use crate::cpu::AddrT;
use crate::decode_table::{table, Entry};
use crate::mmu::Mmu;
use crate::modrm::{modrm_decode32, Modrm, ModrmType};
use crate::tlb::Tlb;

pub use crate::decode_table::Op;

/// The two decoder instances: `OP_SIZE` in the C.
///
/// `0x66` switches between them, and it is the only difference the two produce
/// at this level - the operand-size token the header passes to the semantic
/// macros is spelled `oz` in both, so it reads back as `oz` either way.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Size {
    B32 = 32,
    B16 = 16,
}

impl Size {
    pub fn bits(self) -> u8 {
        self as u8
    }
    pub fn other(self) -> Size {
        match self {
            Size::B32 => Size::B16,
            Size::B16 => Size::B32,
        }
    }
}

/// The eight opcode maps emu/decode.h reaches.
///
/// The base map, the two-byte map behind `0x0f`, the lock map behind `0xf0` and
/// its own two-byte map, and the two scalar maps behind `0xf2`/`0xf3` with
/// theirs. The order here is the order of `MAPS` in tools/gen_decode_table.py,
/// which is what the generated table is indexed by.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Map {
    Base,
    Of,
    Lock,
    LockOf,
    F2,
    F2Of,
    F3,
    F3Of,
}

impl Map {
    pub const COUNT: usize = 8;

    pub fn index(self) -> usize {
        self as usize
    }
}

/// One step of an opcode's decode, from the generated table.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Step {
    /// Consume a ModRM byte, and a SIB byte and displacement if it asks for them.
    ReadModrm,
    /// Consume an immediate of this many *bits* (8, 16 or 32).
    ReadImm(u8),
    /// `SEG_GS()` - the operand uses the gs segment.
    SegGs,
    /// A semantic operation, with the C's own operand tokens as arguments.
    Op(Op, &'static [&'static str]),
    /// `UNDEFINED` - raise the undefined-instruction interrupt.
    Undefined,
    /// The instruction decoded and the block can continue.
    Done,
    /// The instruction decoded and ends the block: a jump, call, return or
    /// interrupt. `asbestos/gen.c` sets `end_block` for exactly these.
    EndBlock,
}

/// What the decoder did, in the order it did it. This is what the differential
/// test compares against the C, event for event.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Event {
    ReadModrm,
    ReadImm(u8),
    SegGs,
    Op(Op, &'static [&'static str]),
    Undefined,
    /// A read touched an unmapped page.
    Segfault,
    Done,
    EndBlock,
}

impl Event {
    /// The fixture's `E ...` line for this event.
    pub fn write_to(&self, out: &mut String) {
        match self {
            Event::ReadModrm => out.push_str("READMODRM"),
            Event::ReadImm(bits) => {
                out.push_str("READIMM ");
                out.push_str(&bits.to_string());
            }
            Event::SegGs => out.push_str("SEG_GS"),
            Event::Op(op, args) => {
                out.push_str(op.as_macro());
                for a in *args {
                    out.push(' ');
                    out.push_str(a);
                }
            }
            Event::Undefined => out.push_str("UNDEFINED"),
            Event::Segfault => out.push_str("SEGFAULT"),
            Event::Done => out.push_str("DONE"),
            Event::EndBlock => out.push_str("END_BLOCK"),
        }
    }

    pub fn to_line(&self) -> String {
        let mut s = String::new();
        self.write_to(&mut s);
        s
    }
}

/// How the decode ended. The codes match the reference generator's `RET_*`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Ret {
    /// Decoded, and the block continues.
    Ok,
    /// Decoded, and this instruction ends the block.
    EndBlock,
    /// No such instruction.
    Undefined,
    /// A read faulted.
    Segfault,
}

impl Ret {
    pub fn code(self) -> u8 {
        match self {
            Ret::Ok => 0,
            Ret::EndBlock => 1,
            Ret::Undefined => 2,
            Ret::Segfault => 3,
        }
    }
}

/// How many ModRM classes there are: 8 memory + 64 register.
///
/// Almost every opcode inspects only `modrm.opcode` (the reg field, in the GRP
/// groups) and `modrm.type` (in `READMODRM_MEM`/`READMODRM_NOMEM` and the x87
/// split). The x87 register form is the exception: its fallback switch is
/// `insn << 8 | modrm.opcode << 4 | modrm.rm_opcode`, so it reads the rm field
/// too and needs 8 * 8 classes. `tools/decode-dump.c` checks that no other
/// opcode depends on anything more than this, over every ModRM byte.
pub const NCLASS: usize = 8 + 64;

/// The class of a raw ModRM byte.
pub fn class_of(modrm_byte: u8) -> usize {
    if modrm_byte >> 6 == 3 {
        8 + ((((modrm_byte >> 3) & 7) as usize) << 3) + (modrm_byte & 7) as usize
    } else {
        ((modrm_byte >> 3) & 7) as usize
    }
}

/// The class of a decoded ModRM operand - the same value, recovered from the
/// decoded fields rather than the byte. `base` and `rm_opcode` are one union in
/// the C, so in the register form `base` still holds the rm field.
pub fn class_of_modrm(m: &Modrm) -> usize {
    if m.type_ == ModrmType::Reg {
        8 + ((m.reg as usize) << 3) + (m.base as usize & 7)
    } else {
        m.reg as usize
    }
}

/// `modrm_decode32` plus the two hooks the C's macros wrap it in.
///
/// `_READIMM` advances `*ip` *before* it reads, so a faulting read still leaves
/// the instruction pointer past the bytes it consumed. `READMODRM` is
/// `if (!modrm_decode32(...)) SEGFAULT`.
fn read_bytes(ip: &mut AddrT, tlb: &mut Tlb, mmu: &mut Mmu, bytes: usize) -> Option<[u8; 4]> {
    *ip = ip.wrapping_add(bytes as AddrT);
    let mut buf = [0u8; 4];
    if !tlb.read(mmu, ip.wrapping_sub(bytes as AddrT), &mut buf[..bytes]) {
        return None;
    }
    Some(buf)
}

/// Run one step. `None` means "carry on"; `Some(ret)` means the instruction is
/// finished or faulted.
fn run_step(
    step: &Step,
    ip: &mut AddrT,
    tlb: &mut Tlb,
    mmu: &mut Mmu,
    modrm: &mut Modrm,
    have_modrm: &mut bool,
    out: &mut Vec<Event>,
) -> Option<Ret> {
    match *step {
        Step::ReadModrm => {
            out.push(Event::ReadModrm);
            if !modrm_decode32(ip, tlb, mmu, modrm) {
                out.push(Event::Segfault);
                return Some(Ret::Segfault);
            }
            *have_modrm = true;
            None
        }
        Step::ReadImm(bits) => {
            out.push(Event::ReadImm(bits));
            if read_bytes(ip, tlb, mmu, (bits / 8) as usize).is_none() {
                out.push(Event::Segfault);
                return Some(Ret::Segfault);
            }
            None
        }
        Step::SegGs => {
            out.push(Event::SegGs);
            None
        }
        Step::Op(op, args) => {
            out.push(Event::Op(op, args));
            None
        }
        Step::Undefined => {
            out.push(Event::Undefined);
            Some(Ret::Undefined)
        }
        Step::Done => {
            out.push(Event::Done);
            Some(Ret::Ok)
        }
        Step::EndBlock => {
            out.push(Event::EndBlock);
            Some(Ret::EndBlock)
        }
    }
}

fn run_steps(
    steps: &'static [Step],
    ip: &mut AddrT,
    tlb: &mut Tlb,
    mmu: &mut Mmu,
    modrm: &mut Modrm,
    have_modrm: &mut bool,
    out: &mut Vec<Event>,
) -> Option<Ret> {
    for step in steps {
        if let Some(ret) = run_step(step, ip, tlb, mmu, modrm, have_modrm, out) {
            return Some(ret);
        }
    }
    None
}

/// Decode one instruction at `*ip`.
///
/// `out` receives what happened, in order; the return value says how it ended
/// and `*ip` is left past whatever was consumed - including on a fault, which
/// is what the C's `READ` macro does.
pub fn decode(
    size: Size,
    ip: &mut AddrT,
    tlb: &mut Tlb,
    mmu: &mut Mmu,
    out: &mut Vec<Event>,
) -> Ret {
    let orig_ip = *ip;
    let mut size = size;
    let mut map = Map::Base;
    let mut modrm = Modrm::default();

    // `restart:` in the C. A prefix does not consume the instruction, it only
    // changes what the next byte means.
    loop {
        // `READINSN` is `_READIMM(insn, 8)`, so the opcode fetch is recorded
        // exactly like an immediate read - and, as with every read, it is
        // recorded before it is attempted, so a faulting fetch still shows up.
        out.push(Event::ReadImm(8));
        let insn = match read_bytes(ip, tlb, mmu, 1) {
            Some(b) => b[0],
            None => {
                out.push(Event::Segfault);
                return Ret::Segfault;
            }
        };

        // The prefix opcodes are exactly the sites the header reaches with
        // `goto restart`, `goto lockrestart` or `return glue(DECODER_NAME, ...)`,
        // plus the four map switches. tools/decode-dump.c derives the same list
        // and prints it into the fixture header.
        let restart = match map {
            Map::Base => match insn {
                0x0f => {
                    map = Map::Of;
                    true
                }
                0xf0 => {
                    map = Map::Lock;
                    true
                }
                0xf2 => {
                    map = Map::F2;
                    true
                }
                0xf3 => {
                    map = Map::F3;
                    true
                }
                0x2e | 0x3e | 0x67 => true, // segment and address-size prefixes
                0x65 => {
                    out.push(Event::SegGs);
                    true
                }
                0x66 => {
                    // `return glue(DECODER_NAME, 16)(...)`: the other instance
                    // starts over from its own `restart:`
                    size = size.other();
                    map = Map::Base;
                    true
                }
                _ => false,
            },
            Map::Lock => match insn {
                0x0f => {
                    map = Map::LockOf;
                    true
                }
                0x65 => {
                    out.push(Event::SegGs);
                    true
                }
                0x66 => {
                    if size == Size::B32 {
                        // The C comments this "I didn't think this through":
                        // `RESTORE_IP` rewinds to the start of the instruction
                        // and hands it to the 16-bit decoder, prefixes and all.
                        *ip = orig_ip;
                        size = Size::B16;
                        map = Map::Base;
                    }
                    true // 16-bit: `goto lockrestart`
                }
                _ => false,
            },
            Map::F2 => {
                if insn == 0x0f {
                    map = Map::F2Of;
                    true
                } else {
                    false
                }
            }
            Map::F3 => {
                if insn == 0x0f {
                    map = Map::F3Of;
                    true
                } else {
                    false
                }
            }
            Map::Of | Map::LockOf | Map::F2Of | Map::F3Of => false,
        };
        if restart {
            continue;
        }

        // An empty slot is unreachable: the generated table fills every opcode
        // except the prefix ones, and the match above intercepts exactly those.
        // `the_table_covers_exactly_the_opcodes_the_decoder_does_not_handle`
        // checks both halves of that, so this arm is a guard, not a behaviour.
        let entry: &Entry = match table(size.bits())[map.index() * 256 + insn as usize] {
            Some(e) => e,
            None => {
                out.push(Event::Undefined);
                return Ret::Undefined;
            }
        };

        let mut have_modrm = false;
        if let Some(ret) =
            run_steps(entry.head, ip, tlb, mmu, &mut modrm, &mut have_modrm, out)
        {
            return ret;
        }
        let tails: &'static [&'static [Step]] = if entry.tails.len() == 1 {
            entry.tails
        } else {
            // A class-split entry always reads a ModRM byte in its head;
            // tools/gen_decode_table.py refuses to emit one that does not.
            debug_assert!(have_modrm, "class-split entry with no ModRM byte");
            let cls = class_of_modrm(&modrm);
            &entry.tails[cls..cls + 1]
        };
        if let Some(ret) = run_steps(tails[0], ip, tlb, mmu, &mut modrm, &mut have_modrm, out) {
            return ret;
        }
        // A table entry that neither faults nor terminates would spin forever;
        // every one ends in Done, EndBlock or Undefined.
        out.push(Event::Done);
        return Ret::Ok;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::mmu::{MemType, MmuOps, PAGE_BITS};
    use std::cell::RefCell;
    use std::ptr::NonNull;

    const PAGE: usize = 1 << PAGE_BITS;
    thread_local! {
        static MEM: RefCell<Box<[u8; PAGE]>> = RefCell::new(Box::new([0u8; PAGE]));
    }

    struct ByteMmu;
    impl MmuOps for ByteMmu {
        fn translate(&mut self, addr: AddrT, _type_: MemType) -> Option<NonNull<u8>> {
            if addr >> PAGE_BITS != 0 {
                return None;
            }
            MEM.with(|m| {
                let ptr = m.borrow_mut().as_mut_ptr();
                // SAFETY: the Box outlives the test and every access is inside it.
                Some(unsafe { NonNull::new_unchecked(ptr) })
            })
        }
    }

    fn reg32_of(i: usize) -> crate::cpu::Reg32 {
        use crate::cpu::Reg32;
        match i {
            0 => Reg32::Eax,
            1 => Reg32::Ecx,
            2 => Reg32::Edx,
            3 => Reg32::Ebx,
            4 => Reg32::Esp,
            5 => Reg32::Ebp,
            6 => Reg32::Esi,
            _ => Reg32::Edi,
        }
    }

    fn decode_bytes(size: Size, bytes: &[u8]) -> (Ret, Vec<Event>, AddrT) {
        MEM.with(|m| {
            let mut m = m.borrow_mut();
            **m = [0u8; PAGE];
            m[..bytes.len()].copy_from_slice(bytes);
        });
        let mut backend = ByteMmu;
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);
        let mut ip = 0;
        let mut out = Vec::new();
        let ret = decode(size, &mut ip, &mut tlb, &mut mmu, &mut out);
        (ret, out, ip)
    }

    fn names(events: &[Event]) -> Vec<String> {
        events.iter().map(Event::to_line).collect()
    }

    #[test]
    fn the_class_functions_agree() {
        for byte in 0..=255u8 {
            let mut m = Modrm::default();
            // rebuild what modrm_decode32 would have stored for this byte
            let reg = ((byte >> 3) & 7) as usize;
            m.reg = reg32_of(reg);
            if byte >> 6 == 3 {
                m.type_ = ModrmType::Reg;
                m.base = reg32_of((byte & 7) as usize);
            } else {
                m.type_ = ModrmType::Mem;
            }
            assert_eq!(class_of(byte), class_of_modrm(&m), "modrm byte {byte:02x}");
        }
        assert_eq!(class_of(0b00_010_000), 2, "memory operand, reg field 2");
        assert_eq!(class_of(0b11_010_101), 8 + 2 * 8 + 5, "register operand");
        assert_eq!(NCLASS, 72);
    }

    #[test]
    fn a_register_add_is_one_modrm_read_and_one_operation() {
        // 01 c8 -> add eax, ecx
        let (ret, ev, ip) = decode_bytes(Size::B32, &[0x01, 0xc8]);
        assert_eq!(ret, Ret::Ok);
        assert_eq!(
            names(&ev),
            ["READIMM 8", "READMODRM", "ADD modrm_reg modrm_val oz", "DONE"]
        );
        assert_eq!(ip, 2, "opcode plus modrm byte");
    }

    #[test]
    fn an_immediate_is_read_at_the_operand_size_of_the_instance() {
        // 05 imm -> add eax, imm
        let (_, ev32, ip32) = decode_bytes(Size::B32, &[0x05, 1, 2, 3, 4]);
        assert_eq!(names(&ev32), ["READIMM 8", "READIMM 32", "ADD imm reg_a oz", "DONE"]);
        assert_eq!(ip32, 5);
        let (_, ev16, ip16) = decode_bytes(Size::B16, &[0x05, 1, 2, 3, 4]);
        assert_eq!(names(&ev16), ["READIMM 8", "READIMM 16", "ADD imm reg_a oz", "DONE"]);
        assert_eq!(ip16, 3, "a 16-bit immediate in the 16-bit instance");
    }

    #[test]
    fn the_operand_size_prefix_hands_over_to_the_other_instance() {
        // 66 05 imm -> the 32-bit decoder returns into the 16-bit one
        let (_, ev, ip) = decode_bytes(Size::B32, &[0x66, 0x05, 1, 2, 3, 4]);
        assert_eq!(names(&ev), ["READIMM 8", "READIMM 8", "READIMM 16", "ADD imm reg_a oz", "DONE"]);
        assert_eq!(ip, 4);
    }

    #[test]
    fn a_group_dispatches_on_the_reg_field() {
        // c0 /0 -> rol modrm8, imm8 ; c0 /1 -> ror
        let (_, rol, _) = decode_bytes(Size::B32, &[0xc0, 0b00_000_000, 0x11]);
        assert_eq!(
            names(&rol),
            ["READIMM 8", "READMODRM", "READIMM 8", "ROL imm modrm_val 8", "DONE"],
            "grp2 reads the imm8 count itself"
        );
        let (_, ror, _) = decode_bytes(Size::B32, &[0xc0, 0b00_001_000, 0x11]);
        assert_eq!(names(&ror)[3], "ROR imm modrm_val 8");
    }

    #[test]
    fn an_x87_register_form_depends_on_the_rm_field_too() {
        // d9 e0 is fchs and d9 e1 is fabs: same opcode, same reg field, only the
        // rm field differs. This is why the register form has 8 * 8 classes -
        // the C's fallback switch is `insn << 8 | modrm.opcode << 4 |
        // modrm.rm_opcode`.
        let (_, a, _) = decode_bytes(Size::B32, &[0xd9, 0b11_100_000]);
        let (_, b, _) = decode_bytes(Size::B32, &[0xd9, 0b11_100_001]);
        assert_eq!(names(&a)[2], "FCHS");
        assert_eq!(names(&b)[2], "FABS");
    }

    #[test]
    fn a_jump_ends_the_block() {
        // eb 00 -> jmp rel8
        let (ret, ev, _) = decode_bytes(Size::B32, &[0xeb, 0x00]);
        assert_eq!(ret, Ret::EndBlock);
        assert_eq!(names(&ev), ["READIMM 8", "READIMM 8", "JMP_REL imm", "END_BLOCK"]);
    }

    #[test]
    fn an_unknown_opcode_is_undefined() {
        // 0f 0b would be ud2 upstream; iSH has no case for it
        let (ret, ev, ip) = decode_bytes(Size::B32, &[0x0f, 0x0b, 0x00]);
        assert_eq!(ret, Ret::Undefined);
        assert_eq!(names(&ev), ["READIMM 8", "READIMM 8", "UNDEFINED"]);
        assert_eq!(ip, 2, "both opcode bytes were consumed");
    }

    #[test]
    fn a_fault_leaves_the_ip_past_what_it_consumed() {
        // a modrm byte asking for a disp32 at the very end of the page
        let at = (PAGE - 3) as AddrT;
        MEM.with(|m| {
            let mut m = m.borrow_mut();
            **m = [0u8; PAGE];
            m[at as usize] = 0x8b; // mov reg, modrm
            // mod=00 rm=101 is the disp32-with-no-base form. rm=110 (esi) would
            // be a plain [esi] with no displacement and would not fault here.
            m[at as usize + 1] = 0b00_000_101;
        });
        let mut backend = ByteMmu;
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);
        let mut ip = at;
        let mut out = Vec::new();
        let ret = decode(Size::B32, &mut ip, &mut tlb, &mut mmu, &mut out);
        assert_eq!(ret, Ret::Segfault);
        assert_eq!(names(&out), ["READIMM 8", "READMODRM", "SEGFAULT"]);
        // 4093 + 1 opcode + 1 modrm + 4 disp32; the ip is past the bytes even
        // though the last read faulted
        assert_eq!(ip, PAGE as AddrT + 3, "1 for the opcode, 1 for modrm, 4 for the disp32");
    }

    #[test]
    fn every_table_entry_terminates() {
        // A step list that ended without Done/EndBlock/Undefined would make the
        // decoder fall through to the next opcode byte, which is not what the C
        // does. Check all 4,096 slots rather than trusting the generator.
        for size in [Size::B32, Size::B16] {
            for slot in table(size.bits()).iter().flatten() {
                for tail in slot.tails {
                    let last = tail.last().or_else(|| slot.head.last());
                    assert!(
                        matches!(
                            last,
                            Some(Step::Done | Step::EndBlock | Step::Undefined)
                        ),
                        "entry does not terminate: head {:?} tail {:?}",
                        slot.head,
                        tail
                    );
                }
            }
        }
    }

    #[test]
    fn the_table_covers_exactly_the_opcodes_the_decoder_does_not_handle() {
        // The generated table leaves a slot empty only for the prefix opcodes,
        // and those are exactly the ones the match above intercepts before the
        // lookup. So the `None` arm of the lookup is unreachable - but only for
        // as long as those two lists agree. Check the coupling instead of
        // trusting it: a prefix that fell through would silently decode as
        // undefined, and a non-prefix with no entry would do the same.
        let prefixes: &[(Map, &[u8])] = &[
            (Map::Base, &[0x0f, 0x2e, 0x3e, 0x65, 0x66, 0x67, 0xf0, 0xf2, 0xf3]),
            (Map::Lock, &[0x0f, 0x65, 0x66]),
            (Map::F2, &[0x0f]),
            (Map::F3, &[0x0f]),
        ];
        let maps = [
            Map::Base,
            Map::Of,
            Map::Lock,
            Map::LockOf,
            Map::F2,
            Map::F2Of,
            Map::F3,
            Map::F3Of,
        ];
        for size in [Size::B32, Size::B16] {
            for map in maps {
                let except: &[u8] =
                    prefixes.iter().find(|(m, _)| *m == map).map(|(_, p)| *p).unwrap_or(&[]);
                for op in 0..=255u8 {
                    let slot = &table(size.bits())[map.index() * 256 + op as usize];
                    if except.contains(&op) {
                        assert!(slot.is_none(), "{map:?} {op:02x} should be left to the decoder");
                    } else {
                        assert!(slot.is_some(), "{map:?} {op:02x} has no table entry");
                    }
                }
            }
        }
    }
}
