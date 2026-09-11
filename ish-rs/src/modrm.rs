// Line-by-line translation of emu/modrm.h
//
// The ModRM (and SIB) decoder: it turns the bytes at the instruction pointer
// into an operand description, reading them through the TLB. This is the first
// module that actually *uses* `tlb.rs` rather than just describing it.
//
// One subtlety drives the whole shape of the C and is easy to port wrong:
// `struct modrm` puts `base` and `rm_opcode` in a union, so they are the *same*
// storage. The branch the C annotates "// wtf intel" reads `rm_opcode` right
// after writing `base = RM(sib_byte)`, so it tests the SIB base register that
// was just stored — not the original `rm` field of the ModRM byte. That is what
// makes the branch live, and what makes `[ebp]` with no displacement mean
// disp32. Reading it as a separate field would silently drop the case.

use crate::cpu::{AddrT, Reg32};
use crate::mmu::Mmu;
use crate::tlb::Tlb;

/// `static const unsigned rm_sib = reg_esp;`
pub const RM_SIB: u32 = Reg32::Esp as u32;
/// `static const unsigned rm_none = reg_esp;` — an SIB index of 4 means "no
/// index register".
pub const RM_NONE: u32 = Reg32::Esp as u32;
/// `static const unsigned rm_disp32 = reg_ebp;`
pub const RM_DISP32: u32 = Reg32::Ebp as u32;

/// `#define MOD(byte) ((byte & 0b11000000) >> 6)`
pub const fn mod_field(byte: u8) -> u32 {
    (byte & 0b1100_0000) as u32 >> 6
}
/// `#define REG(byte) ((byte & 0b00111000) >> 3)`
pub const fn reg_field(byte: u8) -> u32 {
    (byte & 0b0011_1000) as u32 >> 3
}
/// `#define RM(byte) ((byte & 0b00000111) >> 0)`
pub const fn rm_field(byte: u8) -> u32 {
    (byte & 0b0000_0111) as u32
}

/// The `mode` values the C names in a local enum inside `modrm_decode32`.
const MODE_DISP0: u32 = 0;
const MODE_DISP8: u32 = 1;
const MODE_DISP32: u32 = 2;
const MODE_REG: u32 = 3;

/// `enum { modrm_reg, modrm_mem, modrm_mem_si }`
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ModrmType {
    Reg,
    Mem,
    /// A memory operand with a SIB index register, i.e. `[base + index<<shift + disp]`.
    MemSi,
}

/// `enum { times_1 = 0, times_2 = 1, times_4 = 2 }`
///
/// The C names only three values but stores `MOD(sib_byte)` straight into the
/// field, so the SIB scale of 8 arrives as the unnamed `3`. It is not dead:
/// `asbestos/gen.c:166` indexes a gadget table with `modrm->index * 4 +
/// modrm->shift`, and that table is generated over `.irp times, 1,2,4,8`
/// (`asbestos/gadgets-x86_64/memory.S:47-50`). Folding 3 back to 0 would decode
/// `[eax + ecx*8]` as `[eax + ecx]`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum SibShift {
    #[default]
    Times1 = 0,
    Times2 = 1,
    Times4 = 2,
    /// The C has no name for this one; it is the fourth gadget table entry.
    Times8 = 3,
}

/// `struct modrm`.
///
/// The C's two unions (`reg`/`opcode` and `base`/`rm_opcode`) are the same
/// storage under two names; `opcode()` and `rm_opcode()` recover the unsigned
/// view. Both `REG()` and `RM()` produce 0..7, so they always fit a `Reg32`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Modrm {
    pub reg: Reg32,
    pub type_: ModrmType,
    pub base: Reg32,
    pub offset: i32,
    pub index: Reg32,
    pub shift: SibShift,
}

impl Default for Modrm {
    /// The C leaves `index` and `shift` untouched when there is no SIB byte, so
    /// they hold whatever the caller's stack had. Callers never read them
    /// unless `type_ == MemSi`; the port starts them at zero so the value is at
    /// least defined, and `tools/modrm-dump.c` zeroes the struct before each
    /// decode so the two sides agree on what "untouched" means.
    fn default() -> Self {
        Modrm {
            reg: Reg32::Eax,
            type_: ModrmType::Reg,
            base: Reg32::Eax,
            offset: 0,
            index: Reg32::Eax,
            shift: SibShift::Times1,
        }
    }
}

impl Modrm {
    /// The `unsigned opcode` arm of the union.
    pub fn opcode(&self) -> u32 {
        self.reg as u32
    }
    /// The `unsigned rm_opcode` arm of the union — the same storage as `base`.
    pub fn rm_opcode(&self) -> u32 {
        self.base as u32
    }
}

/// The C's `modrm->shift = MOD(sib_byte)` store, keeping all four encodings.
const fn sib_shift_of(field: u32) -> SibShift {
    match field {
        1 => SibShift::Times2,
        2 => SibShift::Times4,
        3 => SibShift::Times8,
        _ => SibShift::Times1,
    }
}

fn reg32_of(field: u32) -> Reg32 {
    match field {
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

/// `modrm_decode32` — read the ModRM byte and maybe the SIB byte plus a
/// displacement, advancing `*ip` past them. Returns false if any of the reads
/// faulted, which is how the C reports a segfault to its caller.
pub fn modrm_decode32(ip: &mut AddrT, tlb: &mut Tlb, mmu: &mut Mmu, modrm: &mut Modrm) -> bool {
    // #define READ(thing) *ip += sizeof(thing); \
    //     if (!tlb_read(tlb, *ip - sizeof(thing), &(thing), sizeof(thing))) return false
    macro_rules! read {
        ($size:expr) => {{
            *ip = ip.wrapping_add($size);
            let mut buf = [0u8; $size];
            if !tlb.read(mmu, ip.wrapping_sub($size), &mut buf) {
                return false;
            }
            buf
        }};
    }

    let modrm_byte = read!(1)[0];

    let mut mode = mod_field(modrm_byte);
    modrm.type_ = ModrmType::Mem;
    modrm.reg = reg32_of(reg_field(modrm_byte));
    // `modrm->rm_opcode = RM(modrm_byte)` — the same storage as `base`
    modrm.base = reg32_of(rm_field(modrm_byte));

    if mode == MODE_REG {
        modrm.type_ = ModrmType::Reg;
    } else if modrm.rm_opcode() == RM_DISP32 && mode == MODE_DISP0 {
        modrm.base = Reg32::None;
        mode = MODE_DISP32;
    // `&& mode != MODE_REG` is in the C but cannot matter: this arm is only
    // reached after the `mode == MODE_REG` arm above fell through. Kept for
    // line-by-line correspondence; dropping it is the one mutation of this
    // decoder the test suite does not catch, and provably so.
    } else if modrm.rm_opcode() == RM_SIB && mode != MODE_REG {
        let sib_byte = read!(1)[0];
        modrm.base = reg32_of(rm_field(sib_byte));
        // "wtf intel" — `rm_opcode` *is* `base`, so this tests the SIB base
        // register just written, not the ModRM byte's rm field.
        if modrm.rm_opcode() == RM_DISP32 {
            if mode == MODE_DISP0 {
                modrm.base = Reg32::None;
                mode = MODE_DISP32;
            } else {
                modrm.base = Reg32::Ebp;
            }
        }
        modrm.index = reg32_of(reg_field(sib_byte));
        // `modrm->shift = MOD(sib_byte);` — a direct store, including the
        // unnamed 3 (scale 8)
        modrm.shift = sib_shift_of(mod_field(sib_byte));
        if modrm.index as u32 != RM_NONE {
            modrm.type_ = ModrmType::MemSi;
        }
    }

    if mode == MODE_DISP0 {
        modrm.offset = 0;
    } else if mode == MODE_DISP8 {
        modrm.offset = read!(1)[0] as i8 as i32;
    } else if mode == MODE_DISP32 {
        let b = read!(4);
        modrm.offset = i32::from_le_bytes(b);
    }

    true
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::mmu::{MemType, MmuOps, PAGE_BITS};
    use std::cell::RefCell;
    use std::ptr::NonNull;

    // A one-page backing store the decoder reads its instruction bytes from.
    const PAGE: usize = 1 << PAGE_BITS;
    thread_local! {
        static MEM: RefCell<Box<[u8; PAGE]>> = RefCell::new(Box::new([0u8; PAGE]));
    }

    /// Maps guest page 0 onto the backing store and nothing else, so an access
    /// that runs off the end of the page faults the way a real one would.
    struct ByteMmu;
    impl MmuOps for ByteMmu {
        fn translate(&mut self, addr: AddrT, _type_: MemType) -> Option<NonNull<u8>> {
            if addr >> PAGE_BITS != 0 {
                return None;
            }
            MEM.with(|m| {
                let ptr = m.borrow_mut().as_mut_ptr();
                // SAFETY: the Box lives for the whole test and every access the
                // decoder makes is inside this page.
                Some(unsafe { NonNull::new_unchecked(ptr) })
            })
        }
    }

    /// Replace the whole page, so tests cannot see each other's leftovers.
    fn set_bytes(bytes: &[u8]) {
        MEM.with(|m| {
            let mut m = m.borrow_mut();
            **m = [0u8; PAGE];
            m[..bytes.len()].copy_from_slice(bytes);
        });
    }

    fn decode(bytes: &[u8]) -> (bool, Modrm, AddrT) {
        set_bytes(bytes);
        let mut backend = ByteMmu;
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);
        let mut ip = 0;
        let mut m = Modrm::default();
        let ok = modrm_decode32(&mut ip, &mut tlb, &mut mmu, &mut m);
        (ok, m, ip)
    }

    #[test]
    fn field_macros_match_the_bit_layout() {
        assert_eq!(mod_field(0b11_010_101), 3);
        assert_eq!(reg_field(0b11_010_101), 2);
        assert_eq!(rm_field(0b11_010_101), 5);
        assert_eq!((RM_SIB, RM_NONE, RM_DISP32), (4, 4, 5));
    }

    #[test]
    fn mod_11_is_a_register_operand() {
        // mod=11 reg=eax rm=ecx
        let (ok, m, ip) = decode(&[0b11_000_001]);
        assert!(ok);
        assert_eq!(m.type_, ModrmType::Reg);
        assert_eq!(m.reg, Reg32::Eax);
        assert_eq!(m.base, Reg32::Ecx);
        assert_eq!(m.offset, 0);
        assert_eq!(ip, 1, "only the modrm byte is consumed");
    }

    #[test]
    fn ebp_with_no_displacement_means_disp32() {
        // mod=00 rm=101 (ebp) is the special "no base, disp32" form
        let (ok, m, ip) = decode(&[0b00_000_101, 0x78, 0x56, 0x34, 0x12]);
        assert!(ok);
        assert_eq!(m.type_, ModrmType::Mem);
        assert_eq!(m.base, Reg32::None);
        assert_eq!(m.offset, 0x1234_5678);
        assert_eq!(ip, 5);
    }

    #[test]
    fn sib_with_ebp_base_and_no_displacement_also_means_disp32() {
        // mod=00 rm=100 (SIB), SIB base=101 (ebp) -> the "wtf intel" branch
        let (ok, m, ip) = decode(&[0b00_000_100, 0b00_100_101, 0xff, 0xff, 0xff, 0xff]);
        assert!(ok);
        assert_eq!(m.base, Reg32::None);
        assert_eq!(m.offset, -1);
        assert_eq!(m.index, Reg32::Esp, "index 100 means none");
        assert_eq!(m.type_, ModrmType::Mem, "index 100 is not a real index");
        assert_eq!(ip, 6);
    }

    #[test]
    fn sib_with_an_index_register_becomes_mem_si() {
        // mod=00 rm=100 (SIB), SIB: scale=10 index=110(esi) base=000(eax)
        let (ok, m, ip) = decode(&[0b00_000_100, 0b10_110_000]);
        assert!(ok);
        assert_eq!(m.type_, ModrmType::MemSi);
        assert_eq!(m.base, Reg32::Eax);
        assert_eq!(m.index, Reg32::Esi);
        assert_eq!(m.shift, SibShift::Times4);
        assert_eq!(m.offset, 0);
        assert_eq!(ip, 2);
    }

    #[test]
    fn a_sib_scale_of_eight_keeps_its_own_encoding() {
        // mod=00 rm=100 (SIB), SIB: scale=11 index=001(ecx) base=000(eax)
        // -> [eax + ecx*8]. The C's enum has no name for 3, but gen.c indexes a
        // gadget table with `index * 4 + shift` and that table is generated over
        // `.irp times, 1,2,4,8`, so 3 is a live entry rather than garbage.
        let (ok, m, ip) = decode(&[0b00_000_100, 0b11_001_000]);
        assert!(ok);
        assert_eq!(m.type_, ModrmType::MemSi);
        assert_eq!(m.index, Reg32::Ecx);
        assert_eq!(m.shift, SibShift::Times8);
        assert_eq!(m.shift as u32, 3);
        assert_eq!(ip, 2);
    }

    #[test]
    fn an_index_of_esp_means_no_index_register() {
        // mod=00 rm=100 (SIB), SIB: scale=00 index=100(esp) base=001(ecx)
        // -> [ecx]; `rm_none` is esp, so index and shift stay written but the
        // operand is plain memory, not modrm_mem_si.
        let (ok, m, ip) = decode(&[0b00_000_100, 0b00_100_001]);
        assert!(ok);
        assert_eq!(m.type_, ModrmType::Mem);
        assert_eq!(m.base, Reg32::Ecx);
        assert_eq!(m.index, Reg32::Esp);
        assert_eq!(ip, 2);
    }

    #[test]
    fn disp8_is_sign_extended() {
        let (ok, m, ip) = decode(&[0b01_000_011, 0x80]); // [ebx - 128]
        assert!(ok);
        assert_eq!(m.type_, ModrmType::Mem);
        assert_eq!(m.base, Reg32::Ebx);
        assert_eq!(m.offset, -128);
        assert_eq!(ip, 2);
    }

    #[test]
    fn the_ip_advances_past_every_byte_read() {
        // mod=10 (disp32) rm=110 (esi), reg=111 (edi)
        let (ok, m, ip) = decode(&[0b10_111_110, 0x01, 0x02, 0x03, 0x04]);
        assert!(ok);
        assert_eq!(m.base, Reg32::Esi);
        assert_eq!(m.reg, Reg32::Edi);
        assert_eq!(m.offset, 0x0403_0201);
        assert_eq!(ip, 5);
    }

    #[test]
    fn a_displacement_running_off_the_page_faults() {
        // mod=10 rm=esi -> a 4-byte displacement, placed so it straddles the
        // end of the only mapped page
        let mut bytes = vec![0u8; PAGE];
        bytes[PAGE - 2] = 0b10_000_110;
        set_bytes(&bytes);
        let mut backend = ByteMmu;
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);
        let mut ip: AddrT = (PAGE - 2) as AddrT;
        let mut m = Modrm::default();
        assert!(
            !modrm_decode32(&mut ip, &mut tlb, &mut mmu, &mut m),
            "the cross-page read must fail when the next page is unmapped"
        );
        assert_eq!(tlb.segfault_addr, 0x1000, "the fault is on the second page");
    }

    #[test]
    fn a_fault_leaves_the_ip_past_what_it_consumed() {
        // the C advances *ip before reading, so a failed read still moves it
        let mut bytes = vec![0u8; PAGE];
        bytes[PAGE - 2] = 0b10_000_110;
        set_bytes(&bytes);
        let mut backend = ByteMmu;
        let mut mmu = Mmu::new(&mut backend);
        let mut tlb = Tlb::new();
        tlb.refresh(&mut mmu);
        let mut ip: AddrT = (PAGE - 2) as AddrT;
        let mut m = Modrm::default();
        modrm_decode32(&mut ip, &mut tlb, &mut mmu, &mut m);
        assert_eq!(ip, PAGE as AddrT + 3, "1 for the modrm byte, 4 for the disp32");
    }
}
