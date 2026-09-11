// Line-by-line translation of emu/cpu.h
//
// The C original packs the general-purpose registers and the flag words into
// unions of overlapping bitfields so the assembly gadgets can reach them at
// fixed offsets. Rust has no bitfields, so each union becomes a raw integer
// plus named accessors, and the bit layout is spelled out in the constants
// below. `tests/cpu_differential.rs` checks the packing against the C struct.

use crate::float80::Float80;

pub type ByteT = u8;
pub type WordT = u16;
pub type DwordT = u32;
pub type QwordT = u64;
pub type AddrT = u32;

// eflags bit positions, in the order the C bitfields are declared:
//   cf_bit:1 pad1_1:1 pf:1 pad2_0:1 af:1 pad3_0:1 zf:1 sf:1 tf:1 if_:1 df:1
//   of_bit:1 iopl:2
// (cpu.h only names PF/AF/ZF/SF/DF; the rest follow from the same layout.)
pub const CF_FLAG: u32 = 1 << 0;
pub const PF_FLAG: u32 = 1 << 2; // #define PF_FLAG in cpu.h
pub const AF_FLAG: u32 = 1 << 4; // #define AF_FLAG in cpu.h
pub const ZF_FLAG: u32 = 1 << 6; // #define ZF_FLAG in cpu.h
pub const SF_FLAG: u32 = 1 << 7; // #define SF_FLAG in cpu.h
pub const TF_FLAG: u32 = 1 << 8;
pub const IF_FLAG: u32 = 1 << 9;
pub const DF_FLAG: u32 = 1 << 10; // #define DF_FLAG in cpu.h
pub const OF_FLAG: u32 = 1 << 11;
pub const IOPL_SHIFT: u32 = 12;

// flags_res bit positions (cpu.h: PF_RES/ZF_RES/SF_RES/AF_OPS)
pub const PF_RES: u8 = 1 << 0;
pub const ZF_RES: u8 = 1 << 1;
pub const SF_RES: u8 = 1 << 2;
pub const AF_OPS: u8 = 1 << 3;

// fsw bit positions, in the order the C bitfields are declared:
//   ie:1 de:1 ze:1 oe:1 ue:1 pe:1 stf:1 es:1 c0:1 c1:1 c2:1 top:3 c3:1 b:1
pub const FSW_IE: u16 = 1 << 0;
pub const FSW_DE: u16 = 1 << 1;
pub const FSW_ZE: u16 = 1 << 2;
pub const FSW_OE: u16 = 1 << 3;
pub const FSW_UE: u16 = 1 << 4;
pub const FSW_PE: u16 = 1 << 5;
pub const FSW_STF: u16 = 1 << 6;
pub const FSW_ES: u16 = 1 << 7;
pub const FSW_C0: u16 = 1 << 8;
pub const FSW_C1: u16 = 1 << 9;
pub const FSW_C2: u16 = 1 << 10;
pub const FSW_TOP_SHIFT: u16 = 11;
pub const FSW_TOP_MASK: u16 = 0b111 << FSW_TOP_SHIFT;
pub const FSW_C3: u16 = 1 << 14;
pub const FSW_B: u16 = 1 << 15;

// fcw bit positions: im:1 dm:1 zm:1 om:1 um:1 pm:1 pad4:2 pc:2 rc:2 y:1
pub const FCW_PC_SHIFT: u16 = 8;
pub const FCW_RC_SHIFT: u16 = 10;
pub const FCW_RC_MASK: u16 = 0b11 << FCW_RC_SHIFT;

/// `enum reg32` from cpu.h.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Reg32 {
    Eax = 0,
    Ecx = 1,
    Edx = 2,
    Ebx = 3,
    Esp = 4,
    Ebp = 5,
    Esi = 6,
    Edi = 7,
    /// `reg_none = reg_count`. Not a register: `modrm_decode32` uses it to say
    /// "no base register", and indexing `cpu->regs` with it is out of bounds in
    /// the C too, so never pass it to [`CpuState::reg`].
    None = 8,
}

pub const REG_COUNT: usize = 8;

/// `reg32_name` from cpu.h.
pub fn reg32_name(reg: Reg32) -> &'static str {
    match reg {
        Reg32::Eax => "eax",
        Reg32::Ecx => "ecx",
        Reg32::Edx => "edx",
        Reg32::Ebx => "ebx",
        Reg32::Esp => "esp",
        Reg32::Ebp => "ebp",
        Reg32::Esi => "esi",
        Reg32::Edi => "edi",
        // the C's `default:` arm
        Reg32::None => "?",
    }
}

/// `union mm_reg` — an MMX register.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct MmReg(pub QwordT);

impl MmReg {
    pub const ZERO: MmReg = MmReg(0);

    pub fn qw(&self) -> QwordT {
        self.0
    }
    pub fn set_qw(&mut self, v: QwordT) {
        self.0 = v;
    }
    pub fn dw(&self, i: usize) -> DwordT {
        ((self.0 >> (32 * i)) & 0xffff_ffff) as DwordT
    }
    pub fn set_dw(&mut self, i: usize, v: DwordT) {
        self.0 = (self.0 & !(0xffff_ffffu64 << (32 * i))) | ((v as u64) << (32 * i));
    }
    pub fn bytes(&self) -> [u8; 8] {
        self.0.to_le_bytes()
    }
    pub fn from_bytes(b: [u8; 8]) -> Self {
        MmReg(u64::from_le_bytes(b))
    }
    pub fn u32(&self, i: usize) -> u32 {
        self.dw(i)
    }
    pub fn u16(&self, i: usize) -> u16 {
        u16::from_le_bytes(self.bytes()[2 * i..2 * i + 2].try_into().unwrap())
    }
    pub fn set_u16(&mut self, i: usize, v: u16) {
        let mut b = self.bytes();
        b[2 * i..2 * i + 2].copy_from_slice(&v.to_le_bytes());
        *self = Self::from_bytes(b);
    }
    pub fn u8(&self, i: usize) -> u8 {
        self.bytes()[i]
    }
}

/// `union xmm_reg` — an SSE register.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct XmmReg(pub u128);

impl XmmReg {
    pub const ZERO: XmmReg = XmmReg(0);

    /// The register as bytes, low byte first. Every element accessor below is
    /// defined in terms of this, so "little endian" (which `emu/cpu.h` simply
    /// assumes: "as does literally everything") is stated once, here.
    pub fn bytes(&self) -> [u8; 16] {
        self.0.to_le_bytes()
    }
    pub fn from_bytes(b: [u8; 16]) -> Self {
        XmmReg(u128::from_le_bytes(b))
    }

    pub fn qw(&self, i: usize) -> QwordT {
        u64::from_le_bytes(self.bytes()[8 * i..8 * i + 8].try_into().unwrap())
    }
    pub fn set_qw(&mut self, i: usize, v: QwordT) {
        let mut b = self.bytes();
        b[8 * i..8 * i + 8].copy_from_slice(&v.to_le_bytes());
        *self = Self::from_bytes(b);
    }
    pub fn u32(&self, i: usize) -> u32 {
        u32::from_le_bytes(self.bytes()[4 * i..4 * i + 4].try_into().unwrap())
    }
    pub fn set_u32(&mut self, i: usize, v: u32) {
        let mut b = self.bytes();
        b[4 * i..4 * i + 4].copy_from_slice(&v.to_le_bytes());
        *self = Self::from_bytes(b);
    }
    pub fn u16(&self, i: usize) -> u16 {
        u16::from_le_bytes(self.bytes()[2 * i..2 * i + 2].try_into().unwrap())
    }
    pub fn set_u16(&mut self, i: usize, v: u16) {
        let mut b = self.bytes();
        b[2 * i..2 * i + 2].copy_from_slice(&v.to_le_bytes());
        *self = Self::from_bytes(b);
    }
    pub fn u8(&self, i: usize) -> u8 {
        self.bytes()[i]
    }
    pub fn set_u8(&mut self, i: usize, v: u8) {
        let mut b = self.bytes();
        b[i] = v;
        *self = Self::from_bytes(b);
    }
    pub fn f32(&self, i: usize) -> f32 {
        f32::from_bits(self.u32(i))
    }
    pub fn set_f32(&mut self, i: usize, v: f32) {
        self.set_u32(i, v.to_bits());
    }
    pub fn f64(&self, i: usize) -> f64 {
        f64::from_bits(self.qw(i))
    }
    pub fn set_f64(&mut self, i: usize, v: f64) {
        self.set_qw(i, v.to_bits());
    }
}

/// `struct cpu_state`.
///
/// Field names follow the C original. The `*_bit` names are the flag bits as
/// they live inside the packed `eflags` word; the bare `cf`/`of` bytes are the
/// separate, always-materialised copies the interpreter keeps ("for maximum
/// efficiency these are stored in bytes").
#[derive(Debug, Clone, PartialEq, Default)]
pub struct CpuState {
    /// `dword_t regs[8]` — eax, ecx, edx, ebx, esp, ebp, esi, edi.
    pub regs: [DwordT; REG_COUNT],
    pub eip: DwordT,

    // ---- eflags union ----
    pub cf_bit: bool,
    pub pad1_1: bool,
    pub pf: bool,
    pub pad2_0: bool,
    pub af: bool,
    pub pad3_0: bool,
    pub zf: bool,
    pub sf: bool,
    pub tf: bool,
    pub if_: bool,
    pub df: bool,
    pub of_bit: bool,
    pub iopl: u8,
    /// Bits 14-31 of the `dword_t eflags` union. The C bitfields stop at
    /// `iopl`, but the union member spans the whole dword, so whatever sits
    /// above bit 13 survives a write to `cpu->eflags` and must survive here.
    pub eflags_high: u32,

    /// "please pretend this doesn't exist"
    pub df_offset: DwordT,
    pub cf: ByteT,
    pub of: ByteT,

    /// Stored result and operands, for the lazily-computed flags.
    pub res: DwordT,
    pub op1: DwordT,
    pub op2: DwordT,
    /// `flags_res` union: pf_res/zf_res/sf_res/af_ops packed into a byte.
    pub flags_res: ByteT,

    pub mm: [MmReg; 8],
    pub xmm: [XmmReg; 8],

    // ---- fpu ----
    pub fp: [Float80; 8],
    pub fsw: WordT,
    pub fcw: WordT,

    /// TLS bullshit
    pub gs: WordT,
    pub tls_ptr: AddrT,

    /// for the page fault handler
    pub segfault_addr: AddrT,
    pub segfault_was_write: bool,

    pub trapno: DwordT,
    pub _poked: bool,
}

// The C `struct cpu_state` also holds `struct mmu *mmu`, `bool *poked_ptr` and
// `long cycle`. Pointers have no meaning outside a running emulator and `cycle`
// is bookkeeping for the execution engine, so neither is modelled here yet;
// they arrive with mmu.rs.

impl CpuState {
    // ---- general registers ----

    /// `cpu->e##n`
    pub fn reg(&self, r: Reg32) -> DwordT {
        self.regs[r as usize]
    }
    pub fn set_reg(&mut self, r: Reg32, v: DwordT) {
        self.regs[r as usize] = v;
    }
    /// `cpu->n##x` — the 16-bit half.
    pub fn reg16(&self, r: Reg32) -> WordT {
        self.regs[r as usize] as WordT
    }
    pub fn set_reg16(&mut self, r: Reg32, v: WordT) {
        let full = &mut self.regs[r as usize];
        *full = (*full & 0xffff_0000) | v as DwordT;
    }
    /// `cpu->n##l` — the low byte (eax/ecx/edx/ebx only, as in the C union).
    pub fn reg8_low(&self, r: Reg32) -> ByteT {
        self.regs[r as usize] as ByteT
    }
    pub fn set_reg8_low(&mut self, r: Reg32, v: ByteT) {
        let full = &mut self.regs[r as usize];
        *full = (*full & 0xffff_ff00) | v as DwordT;
    }
    /// `cpu->n##h` — the high byte of the low half (eax/ecx/edx/ebx only).
    pub fn reg8_high(&self, r: Reg32) -> ByteT {
        (self.regs[r as usize] >> 8) as ByteT
    }
    pub fn set_reg8_high(&mut self, r: Reg32, v: ByteT) {
        let full = &mut self.regs[r as usize];
        *full = (*full & 0xffff_00ff) | ((v as DwordT) << 8);
    }

    // ---- flags ----

    pub fn pf_res(&self) -> bool {
        self.flags_res & PF_RES != 0
    }
    pub fn zf_res(&self) -> bool {
        self.flags_res & ZF_RES != 0
    }
    pub fn sf_res(&self) -> bool {
        self.flags_res & SF_RES != 0
    }
    pub fn af_ops(&self) -> bool {
        self.flags_res & AF_OPS != 0
    }
    pub fn set_pf_res(&mut self, v: bool) {
        self.flags_res = (self.flags_res & !PF_RES) | if v { PF_RES } else { 0 };
    }
    pub fn set_zf_res(&mut self, v: bool) {
        self.flags_res = (self.flags_res & !ZF_RES) | if v { ZF_RES } else { 0 };
    }
    pub fn set_sf_res(&mut self, v: bool) {
        self.flags_res = (self.flags_res & !SF_RES) | if v { SF_RES } else { 0 };
    }
    pub fn set_af_ops(&mut self, v: bool) {
        self.flags_res = (self.flags_res & !AF_OPS) | if v { AF_OPS } else { 0 };
    }

    /// `#define ZF (cpu->zf_res ? cpu->res == 0 : cpu->zf)`
    pub fn zf_eval(&self) -> bool {
        if self.zf_res() {
            self.res == 0
        } else {
            self.zf
        }
    }
    /// `#define SF (cpu->sf_res ? (int32_t) cpu->res < 0 : cpu->sf)`
    pub fn sf_eval(&self) -> bool {
        if self.sf_res() {
            (self.res as i32) < 0
        } else {
            self.sf
        }
    }
    /// `#define CF (cpu->cf)`
    pub fn cf_eval(&self) -> bool {
        self.cf != 0
    }
    /// `#define OF (cpu->of)`
    pub fn of_eval(&self) -> bool {
        self.of != 0
    }
    /// `#define PF (cpu->pf_res ? !__builtin_parity(cpu->res & 0xff) : cpu->pf)`
    ///
    /// PF is set when the low byte holds an *even* number of one bits.
    pub fn pf_eval(&self) -> bool {
        if self.pf_res() {
            (self.res & 0xff).count_ones().is_multiple_of(2)
        } else {
            self.pf
        }
    }
    /// `#define AF (cpu->af_ops ? ((cpu->op1 ^ cpu->op2 ^ cpu->res) >> 4) & 1 : cpu->af)`
    pub fn af_eval(&self) -> bool {
        if self.af_ops() {
            ((self.op1 ^ self.op2 ^ self.res) >> 4) & 1 != 0
        } else {
            self.af
        }
    }

    /// `collapse_flags` — materialise every lazy flag into its bit.
    pub fn collapse_flags(&mut self) {
        self.zf = self.zf_eval();
        self.sf = self.sf_eval();
        self.pf = self.pf_eval();
        self.set_zf_res(false);
        self.set_sf_res(false);
        self.set_pf_res(false);
        self.of_bit = self.of != 0;
        self.cf_bit = self.cf != 0;
        self.af = self.af_eval();
        self.set_af_ops(false);
        self.pad1_1 = true;
        self.pad2_0 = false;
        self.pad3_0 = false;
        self.if_ = true;
    }

    /// `expand_flags` — split the packed bits back into the byte copies.
    pub fn expand_flags(&mut self) {
        self.of = self.of_bit as ByteT;
        self.cf = self.cf_bit as ByteT;
        self.set_zf_res(false);
        self.set_sf_res(false);
        self.set_pf_res(false);
        self.set_af_ops(false);
    }

    /// Read the whole `dword_t eflags` union.
    pub fn eflags(&self) -> DwordT {
        let mut f = 0u32;
        f |= self.cf_bit as u32; // bit 0
        f |= (self.pad1_1 as u32) << 1;
        f |= (self.pf as u32) << 2;
        f |= (self.pad2_0 as u32) << 3;
        f |= (self.af as u32) << 4;
        f |= (self.pad3_0 as u32) << 5;
        f |= (self.zf as u32) << 6;
        f |= (self.sf as u32) << 7;
        f |= (self.tf as u32) << 8;
        f |= (self.if_ as u32) << 9;
        f |= (self.df as u32) << 10;
        f |= (self.of_bit as u32) << 11;
        f |= ((self.iopl as u32) & 0b11) << IOPL_SHIFT;
        f |= self.eflags_high << 14;
        f
    }

    /// Write the whole `dword_t eflags` union.
    pub fn set_eflags(&mut self, f: DwordT) {
        self.cf_bit = f & CF_FLAG != 0;
        self.pad1_1 = f & (1 << 1) != 0;
        self.pf = f & PF_FLAG != 0;
        self.pad2_0 = f & (1 << 3) != 0;
        self.af = f & AF_FLAG != 0;
        self.pad3_0 = f & (1 << 5) != 0;
        self.zf = f & ZF_FLAG != 0;
        self.sf = f & SF_FLAG != 0;
        self.tf = f & TF_FLAG != 0;
        self.if_ = f & IF_FLAG != 0;
        self.df = f & DF_FLAG != 0;
        self.of_bit = f & OF_FLAG != 0;
        self.iopl = ((f >> IOPL_SHIFT) & 0b11) as u8;
        self.eflags_high = f >> 14;
    }

    // ---- fsw / fcw ----

    /// `cpu->top` — the 3-bit field inside `fsw`.
    pub fn top(&self) -> u8 {
        ((self.fsw & FSW_TOP_MASK) >> FSW_TOP_SHIFT) as u8
    }
    pub fn set_top(&mut self, top: u8) {
        self.fsw = (self.fsw & !FSW_TOP_MASK) | (((top as u16) & 0b111) << FSW_TOP_SHIFT);
    }

    fn fsw_bit(&self, mask: u16) -> bool {
        self.fsw & mask != 0
    }
    fn set_fsw_bit(&mut self, mask: u16, v: bool) {
        self.fsw = (self.fsw & !mask) | if v { mask } else { 0 };
    }

    pub fn c0(&self) -> bool {
        self.fsw_bit(FSW_C0)
    }
    pub fn c1(&self) -> bool {
        self.fsw_bit(FSW_C1)
    }
    pub fn c2(&self) -> bool {
        self.fsw_bit(FSW_C2)
    }
    pub fn c3(&self) -> bool {
        self.fsw_bit(FSW_C3)
    }
    pub fn set_c0(&mut self, v: bool) {
        self.set_fsw_bit(FSW_C0, v)
    }
    pub fn set_c1(&mut self, v: bool) {
        self.set_fsw_bit(FSW_C1, v)
    }
    pub fn set_c2(&mut self, v: bool) {
        self.set_fsw_bit(FSW_C2, v)
    }
    pub fn set_c3(&mut self, v: bool) {
        self.set_fsw_bit(FSW_C3, v)
    }

    /// `cpu->rc` — the 2-bit rounding control inside `fcw`. Its encoding
    /// matches `enum f80_rounding_mode`, which is why `fpu_ldcw16` can assign
    /// one to the other.
    pub fn rc(&self) -> u8 {
        ((self.fcw & FCW_RC_MASK) >> FCW_RC_SHIFT) as u8
    }
    pub fn set_rc(&mut self, rc: u8) {
        self.fcw = (self.fcw & !FCW_RC_MASK) | (((rc as u16) & 0b11) << FCW_RC_SHIFT);
    }
    /// `cpu->pc` — the 2-bit precision control inside `fcw`.
    pub fn pc(&self) -> u8 {
        ((self.fcw >> FCW_PC_SHIFT) & 0b11) as u8
    }
    pub fn set_pc(&mut self, pc: u8) {
        let mask = 0b11 << FCW_PC_SHIFT;
        self.fcw = (self.fcw & !mask) | (((pc as u16) & 0b11) << FCW_PC_SHIFT);
    }

    /// `ST(i)` — `cpu->fp[(cpu->top + i) % 8]`, as an index.
    pub fn st_index(&self, i: usize) -> usize {
        (self.top() as usize + i) % 8
    }
    pub fn st(&self, i: usize) -> Float80 {
        self.fp[self.st_index(i)]
    }
    pub fn set_st(&mut self, i: usize, f: Float80) {
        let idx = self.st_index(i);
        self.fp[idx] = f;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn eflags_roundtrip_matches_bit_layout() {
        // CF|PF|AF|ZF|SF|DF with iopl 0: bits 0, 2, 4, 6, 7 and 10.
        let all = CF_FLAG | PF_FLAG | AF_FLAG | ZF_FLAG | SF_FLAG | DF_FLAG;
        assert_eq!(all, 0x4d5);
        let mut cpu = CpuState::default();
        cpu.set_eflags(all);
        assert!(cpu.cf_bit && cpu.pf && cpu.af && cpu.zf && cpu.sf && cpu.df);
        assert!(!cpu.of_bit && !cpu.tf && !cpu.if_);
        assert_eq!(cpu.eflags(), all);
        assert_eq!(cpu.iopl, 0);

        // the dword union is wider than the bitfields: bits 14-31 round trip
        cpu.set_eflags(0xffff_ffff);
        assert_eq!(cpu.eflags(), 0xffff_ffff);
        assert_eq!(cpu.iopl, 3);
        assert_eq!(cpu.eflags_high, 0xffff_ffff >> 14);
        cpu.set_eflags(0);
        assert_eq!(cpu.eflags_high, 0);
    }

    #[test]
    fn lazy_flags_follow_the_macros() {
        let mut cpu = CpuState::default();

        // zf_res set -> ZF comes from res, not from the zf bit
        cpu.set_zf_res(true);
        cpu.zf = false;
        cpu.res = 0;
        assert!(cpu.zf_eval());
        cpu.res = 1;
        assert!(!cpu.zf_eval());
        cpu.set_zf_res(false);
        assert!(!cpu.zf_eval(), "zf_res clear -> use the zf bit");
        cpu.zf = true;
        assert!(cpu.zf_eval());

        // sf_res set -> SF is the sign of res as a signed 32-bit int
        cpu.set_sf_res(true);
        cpu.sf = false;
        cpu.res = 0x8000_0000;
        assert!(cpu.sf_eval());
        cpu.res = 0x7fff_ffff;
        assert!(!cpu.sf_eval());

        // pf_res set -> even parity of the low byte
        cpu.set_pf_res(true);
        cpu.pf = false;
        cpu.res = 0x0000_000f; // 4 bits -> even
        assert!(cpu.pf_eval());
        cpu.res = 0x0000_0107; // low byte 0x07 -> 3 bits -> odd
        assert!(!cpu.pf_eval());
        cpu.res = 0x0000_0f00; // low byte 0x00, high bits ignored
        assert!(cpu.pf_eval());

        // af_ops set -> AF is bit 4 of op1 ^ op2 ^ res
        cpu.set_af_ops(true);
        cpu.af = false;
        cpu.op1 = 0x0f;
        cpu.op2 = 0x01;
        cpu.res = 0x10;
        assert!(cpu.af_eval(), "(0x0f ^ 0x01 ^ 0x10) >> 4 & 1 == 1");
        cpu.res = 0x00;
        assert!(!cpu.af_eval());
    }

    #[test]
    fn collapse_and_expand_are_inverses_for_the_bits_they_touch() {
        let mut cpu = CpuState::default();
        cpu.set_zf_res(true);
        cpu.set_sf_res(true);
        cpu.set_pf_res(true);
        cpu.set_af_ops(true);
        cpu.res = 0x8000_0081; // zf=0, sf=1, pf(low byte 0x81 -> 2 bits)=1, af from xor
        cpu.op1 = 0xff;
        cpu.op2 = 0xf0;
        cpu.cf = 1;
        cpu.of = 1;

        cpu.collapse_flags();
        assert!(!cpu.zf && cpu.sf && cpu.pf);
        assert!(cpu.cf_bit && cpu.of_bit);
        assert_eq!(cpu.flags_res, 0, "collapse clears the lazy markers");
        assert!(
            cpu.pad1_1 && cpu.if_,
            "collapse forces the reserved/IF bits"
        );
        assert!(!cpu.pad2_0 && !cpu.pad3_0);

        cpu.expand_flags();
        assert_eq!(cpu.cf, 1);
        assert_eq!(cpu.of, 1);
    }

    #[test]
    fn top_wraps_and_st_index_follows_it() {
        let mut cpu = CpuState::default();
        assert_eq!(cpu.top(), 0);
        cpu.set_top(7);
        assert_eq!(cpu.st_index(0), 7);
        assert_eq!(cpu.st_index(1), 0, "wraps mod 8 like the C macro");
        assert_eq!(cpu.st_index(7), 6);
        // only the 3-bit field moves, the rest of fsw is untouched
        cpu.fsw = 0xffff;
        cpu.set_top(2);
        assert_eq!(cpu.fsw, !FSW_TOP_MASK | (2 << FSW_TOP_SHIFT));
    }

    #[test]
    fn register_halves_and_bytes() {
        let mut cpu = CpuState::default();
        cpu.set_reg(Reg32::Eax, 0x1234_5678);
        assert_eq!(cpu.reg16(Reg32::Eax), 0x5678);
        assert_eq!(cpu.reg8_low(Reg32::Eax), 0x78);
        assert_eq!(cpu.reg8_high(Reg32::Eax), 0x56);
        cpu.set_reg8_low(Reg32::Eax, 0xab);
        assert_eq!(cpu.reg(Reg32::Eax), 0x1234_56ab);
        cpu.set_reg8_high(Reg32::Eax, 0xcd);
        assert_eq!(cpu.reg(Reg32::Eax), 0x1234_cdab);
        cpu.set_reg16(Reg32::Eax, 0x0001);
        assert_eq!(cpu.reg(Reg32::Eax), 0x1234_0001);
        assert_eq!(reg32_name(Reg32::Esp), "esp");
    }

    #[test]
    fn rc_and_pc_live_in_fcw() {
        let mut cpu = CpuState::default();
        cpu.set_rc(3);
        assert_eq!(cpu.fcw, 3 << FCW_RC_SHIFT);
        cpu.set_pc(2);
        assert_eq!(cpu.fcw, (3 << FCW_RC_SHIFT) | (2 << FCW_PC_SHIFT));
        assert_eq!(cpu.rc(), 3);
        assert_eq!(cpu.pc(), 2);
    }
}
