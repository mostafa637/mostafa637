// Line-by-line translation of emu/fpu.h + emu/fpu.c
//
// The x87 coprocessor: an eight-deep register stack addressed relative to
// `top`, condition codes C0-C3 in the status word, and arithmetic that runs on
// the 80-bit `float80` type ported in float80.rs.
//
// Two deliberate signature changes from the C, neither of which changes
// behaviour:
//   * the C functions take `struct cpu_state *cpu`; here they are methods on
//     `CpuState`;
//   * memory operands are `int16_t *` / `float *` pointers in C (the decoder
//     hands over an address inside guest memory). Here they are plain values,
//     because this crate has no guest memory yet - that arrives with mmu.rs.

use crate::cpu::CpuState;
use crate::float80::{
    self as f80, Float80, RoundingMode,
};

/// `enum fpu_const`
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FpuConst {
    One = 0,
    Log2t = 1,
    Log2e = 2,
    Pi = 3,
    Log2 = 4,
    Ln2 = 5,
    Zero = 6,
}

/// `static const float80 fpu_consts[]`
pub const FPU_CONSTS: [Float80; 7] = [
    Float80::new(0x8000_0000_0000_0000, 0x3fff), // fconst_one
    Float80::new(0xd49a_784b_cd1b_8afe, 0x4000), // fconst_log2t
    Float80::new(0xb8aa_3b29_5c17_f0bc, 0x3fff), // fconst_log2e
    Float80::new(0xc90f_daa2_2168_c235, 0x4000), // fconst_pi
    Float80::new(0x9a20_9a84_fbcf_f799, 0x3ffd), // fconst_log2
    Float80::new(0xb172_17f7_d1cf_79ac, 0x3ffe), // fconst_ln2
    Float80::new(0x0000_0000_0000_0000, 0x0000), // fconst_zero
];

/// `struct fpu_env32` — the 32-bit protected-mode FPU environment.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct FpuEnv32 {
    pub control: u32,
    pub status: u32,
    pub tag: u32,
    pub ip: u32,
    pub ip_selector: u32,
    pub operand: u32,
    pub operand_selector: u32,
}

/// `struct fpu_state32` — environment plus the eight 80-bit registers.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct FpuState32 {
    pub env: FpuEnv32,
    pub regs: [[u8; 10]; 8],
}

impl CpuState {
    /// `static void fpu_push(struct cpu_state *cpu, float80 f)`
    fn fpu_push(&mut self, f: Float80) {
        // `cpu->top--` on a 3-bit bitfield wraps mod 8.
        self.set_top(self.top().wrapping_sub(1) & 7);
        self.set_st(0, f);
    }

    /// `void fpu_pop(struct cpu_state *cpu)`
    pub fn fpu_pop(&mut self) {
        self.set_top(self.top().wrapping_add(1) & 7);
    }

    /// `void fpu_xch(struct cpu_state *cpu, int i)`
    pub fn fpu_xch(&mut self, i: usize) {
        let (a, b) = (self.st_index(0), self.st_index(i));
        self.fp.swap(a, b);
    }

    /// `void fpu_incstp(struct cpu_state *cpu)`
    ///
    /// "This is different from just popping the stack, it doesn't tag the
    /// stack element as free. We don't have stack tagging yet so in practice
    /// there's no difference."
    pub fn fpu_incstp(&mut self) {
        self.set_top(self.top().wrapping_add(1) & 7);
    }

    // ---- loads ----

    /// `void fpu_ld(struct cpu_state *cpu, int i)`
    pub fn fpu_ld(&mut self, i: usize) {
        let f = self.st(i);
        self.fpu_push(f);
    }

    /// `void fpu_ldc(struct cpu_state *cpu, enum fpu_const c)`
    pub fn fpu_ldc(&mut self, c: FpuConst) {
        let f = FPU_CONSTS[c as usize];
        self.fpu_push(f);
    }

    /// `void fpu_ild16/32/64(struct cpu_state *cpu, intN_t *i)`
    pub fn fpu_ild16(&mut self, i: i16) {
        let f = Float80::from_i64(i as i64);
        self.fpu_push(f);
    }
    pub fn fpu_ild32(&mut self, i: i32) {
        let f = Float80::from_i64(i as i64);
        self.fpu_push(f);
    }
    pub fn fpu_ild64(&mut self, i: i64) {
        let f = Float80::from_i64(i);
        self.fpu_push(f);
    }

    /// `void fpu_ldm32(struct cpu_state *cpu, float32 *f)`
    ///
    /// Note the C passes a 32-bit float through `f80_from_double`, widening to
    /// f64 on the way in. Reproduced exactly.
    pub fn fpu_ldm32(&mut self, f: f32) {
        let v = Float80::from_f64(f as f64);
        self.fpu_push(v);
    }
    /// `void fpu_ldm64(struct cpu_state *cpu, float64 *f)`
    pub fn fpu_ldm64(&mut self, f: f64) {
        let v = Float80::from_f64(f);
        self.fpu_push(v);
    }
    /// `void fpu_ldm80(struct cpu_state *cpu, float80 *f)`
    pub fn fpu_ldm80(&mut self, f: Float80) {
        self.fpu_push(f);
    }

    // ---- stores ----

    /// `void fpu_st(struct cpu_state *cpu, int i)`
    pub fn fpu_st(&mut self, i: usize) {
        let v = self.st(0);
        self.set_st(i, v);
    }

    /// `void fpu_ist16(struct cpu_state *cpu, int16_t *i)`
    ///
    /// Out-of-range results become INT16_MIN, the "integer indefinite" value.
    pub fn fpu_ist16(&self) -> i16 {
        let mut res = self.st(0).to_i64();
        if res < i16::MIN as i64 || res > i16::MAX as i64 {
            res = i16::MIN as i64;
        }
        res as i16
    }
    /// `void fpu_ist32(struct cpu_state *cpu, int32_t *i)`
    pub fn fpu_ist32(&self) -> i32 {
        let mut res = self.st(0).to_i64();
        if res < i32::MIN as i64 || res > i32::MAX as i64 {
            res = i32::MIN as i64;
        }
        res as i32
    }
    /// `void fpu_ist64(struct cpu_state *cpu, int64_t *i)`
    pub fn fpu_ist64(&self) -> i64 {
        self.st(0).to_i64()
    }

    /// `void fpu_stm32(struct cpu_state *cpu, float32 *f)`
    ///
    /// Stored through f64, exactly as the C does — an 80-bit value narrowed to
    /// single precision loses more than the format strictly requires.
    pub fn fpu_stm32(&self) -> f32 {
        self.st(0).to_f64() as f32
    }
    /// `void fpu_stm64(struct cpu_state *cpu, float64 *f)`
    pub fn fpu_stm64(&self) -> f64 {
        self.st(0).to_f64()
    }
    /// `void fpu_stm80(struct cpu_state *cpu, float80 *f)`
    ///
    /// "intel guarantees this will only write 10 bytes" — a `Float80` is
    /// exactly the 64-bit significand plus the 16-bit sign/exponent word.
    pub fn fpu_stm80(&self) -> Float80 {
        self.st(0)
    }

    // ---- moves ----

    /// `FCMOVcc(b, cpu->cf)` and friends.
    ///
    /// The conditions read `cpu->cf` (the always-materialised byte copy) and
    /// `cpu->zf` / `cpu->pf` (the bits inside the packed eflags word) — *not*
    /// the lazily evaluated `ZF`/`PF` macros. Kept as-is.
    pub fn fpu_cmovb(&mut self, i: usize) {
        if self.cf != 0 {
            let v = self.st(i);
            self.set_st(0, v);
        }
    }
    pub fn fpu_cmove(&mut self, i: usize) {
        if self.zf {
            let v = self.st(i);
            self.set_st(0, v);
        }
    }
    pub fn fpu_cmovbe(&mut self, i: usize) {
        if self.cf != 0 || self.zf {
            let v = self.st(i);
            self.set_st(0, v);
        }
    }
    pub fn fpu_cmovu(&mut self, i: usize) {
        if self.pf {
            let v = self.st(i);
            self.set_st(0, v);
        }
    }
    pub fn fpu_cmovnb(&mut self, i: usize) {
        if self.cf == 0 {
            let v = self.st(i);
            self.set_st(0, v);
        }
    }
    pub fn fpu_cmovne(&mut self, i: usize) {
        if !self.zf {
            let v = self.st(i);
            self.set_st(0, v);
        }
    }
    pub fn fpu_cmovnbe(&mut self, i: usize) {
        if !(self.cf != 0 || self.zf) {
            let v = self.st(i);
            self.set_st(0, v);
        }
    }
    pub fn fpu_cmovnu(&mut self, i: usize) {
        if !self.pf {
            let v = self.st(i);
            self.set_st(0, v);
        }
    }

    // ---- math ----

    /// `void fpu_prem(struct cpu_state *cpu)`
    pub fn fpu_prem(&mut self) {
        let (a, b) = (self.st(0), self.st(1));
        self.set_st(0, f80::f80_mod(a, b));
        self.set_c2(false); // "say we finished the entire remainder"
    }

    /// `void fpu_scale(struct cpu_state *cpu)`
    pub fn fpu_scale(&mut self) {
        let old_mode = f80::rounding_mode();
        f80::set_rounding_mode(RoundingMode::RoundChop);
        // C narrows the i64 from f80_to_int to `int`; `as i32` truncates the
        // same way, including the INT64_MIN "indefinite" case becoming 0.
        let scale = self.st(1).to_i64() as i32;
        f80::set_rounding_mode(old_mode);
        let x = self.st(0);
        self.set_st(0, f80::f80_scale(x, scale));
    }

    /// `void fpu_rndint(struct cpu_state *cpu)`
    pub fn fpu_rndint(&mut self) {
        let x = self.st(0);
        if x.is_inf() || x.is_nan() {
            return;
        }
        self.set_st(0, x.round());
    }

    /// `void fpu_sqrt(struct cpu_state *cpu)`
    pub fn fpu_sqrt(&mut self) {
        let x = self.st(0);
        self.set_st(0, f80::f80_sqrt(x));
    }

    /// `void fpu_yl2x(struct cpu_state *cpu)`
    pub fn fpu_yl2x(&mut self) {
        let (a, b) = (self.st(0), self.st(1));
        let v = f80::f80_mul(b, f80::f80_log2(a));
        self.set_st(1, v);
        self.fpu_pop();
    }

    /// `void fpu_2xm1(struct cpu_state *cpu)`
    ///
    /// "an example of the ancient chinese art of chi ting" — iSH computes this
    /// one in double precision rather than 80-bit.
    pub fn fpu_2xm1(&mut self) {
        let x = self.st(0).to_f64();
        let v = Float80::from_f64(2f64.powf(x) - 1.0);
        self.set_st(0, v);
    }

    /// `static void fpu_comparei(struct cpu_state *cpu, float80 x)`
    fn fpu_comparei(&mut self, x: Float80) {
        self.set_zf_res(false);
        self.set_pf_res(false);
        self.zf = false;
        self.pf = false;
        self.cf = 0;
        let st0 = self.st(0);
        self.cf = f80::f80_lt(st0, x) as u8;
        self.zf = f80::f80_eq(st0, x);
        if f80::f80_uncomparable(st0, x) {
            self.zf = true;
            self.pf = true;
            self.cf = 1;
        }
    }

    /// `static void fpu_compare(struct cpu_state *cpu, float80 x)`
    fn fpu_compare(&mut self, x: Float80) {
        self.set_c2(false);
        self.set_c1(false);
        let st0 = self.st(0);
        self.set_c0(f80::f80_lt(st0, x));
        self.set_c3(f80::f80_eq(st0, x));
        if f80::f80_uncomparable(st0, x) {
            self.set_c0(true);
            self.set_c2(true);
            self.set_c3(true);
        }
    }

    /// `void fpu_com(struct cpu_state *cpu, int i)` (`#define fpu_ucom fpu_com`)
    pub fn fpu_com(&mut self, i: usize) {
        let x = self.st(i);
        self.fpu_compare(x);
    }
    /// `void fpu_comi(struct cpu_state *cpu, int i)` (`#define fpu_ucomi fpu_comi`)
    pub fn fpu_comi(&mut self, i: usize) {
        let x = self.st(i);
        self.fpu_comparei(x);
    }
    pub fn fpu_comm32(&mut self, f: f32) {
        let x = Float80::from_f64(f as f64);
        self.fpu_compare(x);
    }
    pub fn fpu_comm64(&mut self, f: f64) {
        let x = Float80::from_f64(f);
        self.fpu_compare(x);
    }
    pub fn fpu_icom16(&mut self, i: i16) {
        let x = Float80::from_i64(i as i64);
        self.fpu_compare(x);
    }
    pub fn fpu_icom32(&mut self, i: i32) {
        let x = Float80::from_i64(i as i64);
        self.fpu_compare(x);
    }
    /// `void fpu_tst(struct cpu_state *cpu)`
    pub fn fpu_tst(&mut self) {
        let zero = FPU_CONSTS[FpuConst::Zero as usize];
        self.fpu_compare(zero);
    }

    /// `void fpu_abs(struct cpu_state *cpu)`
    pub fn fpu_abs(&mut self) {
        let x = self.st(0);
        self.set_st(0, x.abs());
    }

    /// `void fpu_chs(struct cpu_state *cpu)`
    pub fn fpu_chs(&mut self) {
        let x = self.st(0);
        self.set_st(0, x.neg());
    }

    // ---- arithmetic between stack registers ----

    /// `void fpu_add(struct cpu_state *cpu, int srci, int dsti)`
    pub fn fpu_add(&mut self, srci: usize, dsti: usize) {
        let (s, d) = (self.st_index(srci), self.st_index(dsti));
        self.fp[d] = f80::f80_add(self.fp[d], self.fp[s]);
    }
    /// `void fpu_sub(struct cpu_state *cpu, int srci, int dsti)`
    pub fn fpu_sub(&mut self, srci: usize, dsti: usize) {
        let (s, d) = (self.st_index(srci), self.st_index(dsti));
        self.fp[d] = f80::f80_sub(self.fp[d], self.fp[s]);
    }
    /// `void fpu_subr(struct cpu_state *cpu, int srci, int dsti)`
    pub fn fpu_subr(&mut self, srci: usize, dsti: usize) {
        let (s, d) = (self.st_index(srci), self.st_index(dsti));
        self.fp[d] = f80::f80_sub(self.fp[s], self.fp[d]);
    }
    /// `void fpu_mul(struct cpu_state *cpu, int srci, int dsti)`
    pub fn fpu_mul(&mut self, srci: usize, dsti: usize) {
        let (s, d) = (self.st_index(srci), self.st_index(dsti));
        self.fp[d] = f80::f80_mul(self.fp[d], self.fp[s]);
    }
    /// `void fpu_div(struct cpu_state *cpu, int srci, int dsti)`
    pub fn fpu_div(&mut self, srci: usize, dsti: usize) {
        let (s, d) = (self.st_index(srci), self.st_index(dsti));
        self.fp[d] = f80::f80_div(self.fp[d], self.fp[s]);
    }
    /// `void fpu_divr(struct cpu_state *cpu, int srci, int dsti)`
    pub fn fpu_divr(&mut self, srci: usize, dsti: usize) {
        let (s, d) = (self.st_index(srci), self.st_index(dsti));
        self.fp[d] = f80::f80_div(self.fp[s], self.fp[d]);
    }

    // ---- arithmetic against a memory operand ----

    pub fn fpu_iadd16(&mut self, i: i16) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_add(a, b));
    }
    pub fn fpu_isub16(&mut self, i: i16) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_sub(a, b));
    }
    pub fn fpu_isubr16(&mut self, i: i16) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_sub(b, a));
    }
    pub fn fpu_imul16(&mut self, i: i16) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_mul(a, b));
    }
    pub fn fpu_idiv16(&mut self, i: i16) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_div(a, b));
    }
    pub fn fpu_idivr16(&mut self, i: i16) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_div(b, a));
    }

    pub fn fpu_iadd32(&mut self, i: i32) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_add(a, b));
    }
    pub fn fpu_isub32(&mut self, i: i32) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_sub(a, b));
    }
    pub fn fpu_isubr32(&mut self, i: i32) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_sub(b, a));
    }
    pub fn fpu_imul32(&mut self, i: i32) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_mul(a, b));
    }
    pub fn fpu_idiv32(&mut self, i: i32) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_div(a, b));
    }
    pub fn fpu_idivr32(&mut self, i: i32) {
        let (a, b) = (self.st(0), Float80::from_i64(i as i64));
        self.set_st(0, f80::f80_div(b, a));
    }

    pub fn fpu_addm32(&mut self, f: f32) {
        let (a, b) = (self.st(0), Float80::from_f64(f as f64));
        self.set_st(0, f80::f80_add(a, b));
    }
    pub fn fpu_subm32(&mut self, f: f32) {
        let (a, b) = (self.st(0), Float80::from_f64(f as f64));
        self.set_st(0, f80::f80_sub(a, b));
    }
    pub fn fpu_subrm32(&mut self, f: f32) {
        let (a, b) = (self.st(0), Float80::from_f64(f as f64));
        self.set_st(0, f80::f80_sub(b, a));
    }
    pub fn fpu_mulm32(&mut self, f: f32) {
        let (a, b) = (self.st(0), Float80::from_f64(f as f64));
        self.set_st(0, f80::f80_mul(a, b));
    }
    pub fn fpu_divm32(&mut self, f: f32) {
        let (a, b) = (self.st(0), Float80::from_f64(f as f64));
        self.set_st(0, f80::f80_div(a, b));
    }
    pub fn fpu_divrm32(&mut self, f: f32) {
        let (a, b) = (self.st(0), Float80::from_f64(f as f64));
        self.set_st(0, f80::f80_div(b, a));
    }

    pub fn fpu_addm64(&mut self, f: f64) {
        let (a, b) = (self.st(0), Float80::from_f64(f));
        self.set_st(0, f80::f80_add(a, b));
    }
    pub fn fpu_subm64(&mut self, f: f64) {
        let (a, b) = (self.st(0), Float80::from_f64(f));
        self.set_st(0, f80::f80_sub(a, b));
    }
    pub fn fpu_subrm64(&mut self, f: f64) {
        let (a, b) = (self.st(0), Float80::from_f64(f));
        self.set_st(0, f80::f80_sub(b, a));
    }
    pub fn fpu_mulm64(&mut self, f: f64) {
        let (a, b) = (self.st(0), Float80::from_f64(f));
        self.set_st(0, f80::f80_mul(a, b));
    }
    pub fn fpu_divm64(&mut self, f: f64) {
        let (a, b) = (self.st(0), Float80::from_f64(f));
        self.set_st(0, f80::f80_div(a, b));
    }
    pub fn fpu_divrm64(&mut self, f: f64) {
        let (a, b) = (self.st(0), Float80::from_f64(f));
        self.set_st(0, f80::f80_div(b, a));
    }

    /// `void fpu_patan(struct cpu_state *cpu)`
    ///
    /// "there's no native atan2 for 80-bit float yet."
    pub fn fpu_patan(&mut self) {
        let (a, b) = (self.st(0), self.st(1));
        let v = Float80::from_f64(b.to_f64().atan2(a.to_f64()));
        self.set_st(1, v);
        self.fpu_pop();
    }

    /// `void fpu_sin(struct cpu_state *cpu)`
    pub fn fpu_sin(&mut self) {
        let x = self.st(0).to_f64();
        let v = Float80::from_f64(x.sin());
        self.set_st(0, v);
    }
    /// `void fpu_cos(struct cpu_state *cpu)`
    pub fn fpu_cos(&mut self) {
        let x = self.st(0).to_f64();
        let v = Float80::from_f64(x.cos());
        self.set_st(0, v);
    }

    /// `void fpu_xtract(struct cpu_state *cpu)`
    pub fn fpu_xtract(&mut self) {
        let x = self.st(0);
        let (exp, signif) = f80::f80_xtract(x);
        self.set_st(0, Float80::from_i64(exp as i64));
        self.fpu_push(signif);
    }

    /// `void fpu_xam(struct cpu_state *cpu)`
    pub fn fpu_xam(&mut self) {
        let f = self.st(0);
        let outflags: u8 = if !f.is_supported() {
            0b000
        } else if f.is_nan() {
            0b001
        } else if f.is_inf() {
            0b011
        } else if f.is_zero() {
            0b100
        } else if f.is_denormal() {
            0b110
        } else {
            // normal.
            // todo: empty
            0b010
        };
        self.set_c1(f.sign());
        self.set_c0(outflags & 1 != 0);
        self.set_c2((outflags >> 1) & 1 != 0);
        self.set_c3((outflags >> 2) & 1 != 0);
    }

    // ---- meta ----

    /// `void fpu_stcw16(struct cpu_state *cpu, uint16_t *i)`
    pub fn fpu_stcw16(&self) -> u16 {
        self.fcw
    }
    /// `void fpu_ldcw16(struct cpu_state *cpu, uint16_t *i)`
    ///
    /// Loads the control word *and* pushes its rounding control into the
    /// `float80` rounding mode.
    pub fn fpu_ldcw16(&mut self, i: u16) {
        self.fcw = i;
        let rc = self.rc();
        f80::set_rounding_mode(match rc {
            0 => RoundingMode::RoundToNearest,
            1 => RoundingMode::RoundDown,
            2 => RoundingMode::RoundUp,
            _ => RoundingMode::RoundChop,
        });
    }

    /// `void fpu_stenv32(struct cpu_state *cpu, struct fpu_env32 *env)`
    pub fn fpu_stenv32(&self) -> FpuEnv32 {
        FpuEnv32 {
            control: self.fcw as u32,
            status: self.fsw as u32,
            // "hope nobody looks at these"
            tag: 0,
            ip: 0,
            ip_selector: 0,
            operand: 0,
            operand_selector: 0,
        }
    }
    /// `void fpu_ldenv32(struct cpu_state *cpu, struct fpu_env32 *env)`
    pub fn fpu_ldenv32(&mut self, env: &FpuEnv32) {
        self.fcw = env.control as u16;
        self.fsw = env.status as u16;
    }

    /// `void fpu_save32(struct cpu_state *cpu, struct fpu_state32 *state)`
    pub fn fpu_save32(&self) -> FpuState32 {
        let mut state = FpuState32 {
            env: self.fpu_stenv32(),
            regs: [[0u8; 10]; 8],
        };
        for i in 0..8 {
            state.regs[i] = self.st(i).to_le_bytes();
        }
        state
    }

    /// `void fpu_restore32(struct cpu_state *cpu, struct fpu_state32 *state)`
    pub fn fpu_restore32(&mut self, state: &FpuState32) {
        self.fpu_ldenv32(&state.env);
        for i in 0..8 {
            let f = Float80::from_le_bytes(state.regs[i]);
            self.set_st(i, f);
        }
    }

    /// `void fpu_clex(struct cpu_state *cpu)`
    ///
    /// Note the C clears `cpu->sf`, which is the *EFLAGS* sign flag — the FPU
    /// stack-fault bit is the separate `stf` field and is left alone. That is
    /// what the original does, so that is what this does.
    pub fn fpu_clex(&mut self) {
        let clear = crate::cpu::FSW_PE
            | crate::cpu::FSW_UE
            | crate::cpu::FSW_OE
            | crate::cpu::FSW_ZE
            | crate::cpu::FSW_DE
            | crate::cpu::FSW_IE
            | crate::cpu::FSW_ES
            | crate::cpu::FSW_B;
        self.fsw &= !clear;
        self.sf = false;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn cpu_with(values: &[f64]) -> CpuState {
        let mut cpu = CpuState::default();
        for (i, v) in values.iter().enumerate() {
            cpu.fp[i] = Float80::from_f64(*v);
        }
        cpu
    }

    #[test]
    fn constants_match_the_c_table() {
        assert_eq!(FPU_CONSTS[FpuConst::One as usize].to_f64(), 1.0);
        assert_eq!(FPU_CONSTS[FpuConst::Zero as usize].to_f64(), 0.0);
        assert!((FPU_CONSTS[FpuConst::Pi as usize].to_f64() - std::f64::consts::PI).abs() < 1e-18);
        assert!((FPU_CONSTS[FpuConst::Ln2 as usize].to_f64() - std::f64::consts::LN_2).abs() < 1e-18);
        assert!((FPU_CONSTS[FpuConst::Log2e as usize].to_f64() - std::f64::consts::LOG2_E).abs() < 1e-18);
        // fldl2t loads log2(10); fldlg2 loads log10(2)
        assert!((FPU_CONSTS[FpuConst::Log2t as usize].to_f64() - std::f64::consts::LOG2_10).abs() < 1e-15);
        assert!((FPU_CONSTS[FpuConst::Log2 as usize].to_f64() - std::f64::consts::LOG10_2).abs() < 1e-18);
    }

    #[test]
    fn push_pop_xch_incstp_move_top() {
        let mut cpu = CpuState::default();
        cpu.set_top(0);
        cpu.fpu_ldc(FpuConst::One); // top -> 7
        assert_eq!(cpu.top(), 7);
        assert_eq!(cpu.st(0).to_f64(), 1.0);
        cpu.fpu_pop();
        assert_eq!(cpu.top(), 0);
        cpu.fpu_incstp();
        assert_eq!(cpu.top(), 1);
        // wraps the 3-bit field, it does not run off the end
        cpu.set_top(7);
        cpu.fpu_incstp();
        assert_eq!(cpu.top(), 0);

        let mut cpu = cpu_with(&[10.0, 20.0]);
        cpu.fpu_xch(1);
        assert_eq!(cpu.st(0).to_f64(), 20.0);
        assert_eq!(cpu.st(1).to_f64(), 10.0);
    }

    #[test]
    fn loads_and_stores_roundtrip() {
        let mut cpu = CpuState::default();
        cpu.fpu_ild32(-12345);
        assert_eq!(cpu.fpu_ist32(), -12345);
        cpu.fpu_ild64(i64::MAX);
        assert_eq!(cpu.fpu_ist64(), i64::MAX);
        cpu.fpu_ldm64(2.5);
        assert_eq!(cpu.fpu_stm64(), 2.5);
        cpu.fpu_ldm32(0.5);
        assert_eq!(cpu.fpu_stm32(), 0.5);
        let raw = Float80::from_bits(0xd49a_784b_cd1b_8afe, 0x4000);
        cpu.fpu_ldm80(raw);
        assert_eq!(cpu.fpu_stm80(), raw);
    }

    #[test]
    fn integer_stores_saturate_to_the_indefinite_value() {
        let mut cpu = CpuState::default();
        cpu.fpu_ldm64(1e300); // way past i32
        assert_eq!(cpu.fpu_ist32(), i32::MIN);
        assert_eq!(cpu.fpu_ist16(), i16::MIN);
        cpu.fpu_ldm64(-1e300);
        assert_eq!(cpu.fpu_ist32(), i32::MIN);
        cpu.fpu_ldm64(40000.0); // fits i32, not i16
        assert_eq!(cpu.fpu_ist32(), 40000);
        assert_eq!(cpu.fpu_ist16(), i16::MIN);
    }

    #[test]
    fn compare_sets_c0_c3_and_uncomparable_sets_them_all() {
        let mut cpu = cpu_with(&[1.0, 2.0]);
        cpu.fpu_com(1); // 1 < 2
        assert!(cpu.c0() && !cpu.c3() && !cpu.c2() && !cpu.c1());
        cpu.fpu_com(0); // equal
        assert!(!cpu.c0() && cpu.c3());

        let mut cpu = CpuState::default();
        cpu.fp[0] = Float80::nan();
        cpu.fpu_com(0);
        assert!(cpu.c0() && cpu.c2() && cpu.c3(), "NaN is unordered");

        // comi routes the same result into ZF/PF/CF instead
        let mut cpu = CpuState::default();
        cpu.fp[0] = Float80::nan();
        cpu.fpu_comi(0);
        assert!(cpu.zf && cpu.pf && cpu.cf == 1);
        let mut cpu = cpu_with(&[1.0, 2.0]);
        cpu.fpu_comi(1);
        assert!(cpu.cf == 1 && !cpu.zf && !cpu.pf);
    }

    #[test]
    fn arithmetic_matches_the_operand_order() {
        // fpu_sub(src, dst) computes ST(dst) = ST(dst) - ST(src)
        let mut cpu = cpu_with(&[10.0, 3.0]);
        cpu.fpu_sub(1, 0);
        assert_eq!(cpu.st(0).to_f64(), 7.0);
        // fpu_subr reverses it
        let mut cpu = cpu_with(&[10.0, 3.0]);
        cpu.fpu_subr(1, 0);
        assert_eq!(cpu.st(0).to_f64(), -7.0);
        // div and divr likewise
        let mut cpu = cpu_with(&[10.0, 2.0]);
        cpu.fpu_div(1, 0);
        assert_eq!(cpu.st(0).to_f64(), 5.0);
        let mut cpu = cpu_with(&[10.0, 2.0]);
        cpu.fpu_divr(1, 0);
        assert_eq!(cpu.st(0).to_f64(), 0.2);
        // and a non-zero destination index
        let mut cpu = cpu_with(&[1.0, 5.0, 3.0]);
        cpu.fpu_add(2, 1);
        assert_eq!(cpu.st(1).to_f64(), 8.0);
    }

    #[test]
    fn xam_classifies() {
        let cases = [
            (Float80::from_f64(0.0), 0b100, false),
            (Float80::from_f64(-0.0), 0b100, true),
            (Float80::from_f64(1.5), 0b010, false),
            (Float80::from_f64(-1.5), 0b010, true),
            (Float80::inf(), 0b011, false),
            (Float80::nan(), 0b001, false),
            (Float80::from_bits(1, 0), 0b110, false), // denormal
            (Float80::from_bits(0x4000_0000_0000_0000, 0x3fff), 0b000, false), // unsupported
        ];
        for (f, flags, sign) in cases {
            let mut cpu = CpuState::default();
            cpu.fp[0] = f;
            cpu.fpu_xam();
            let got =
                (cpu.c0() as u8) | ((cpu.c2() as u8) << 1) | ((cpu.c3() as u8) << 2);
            assert_eq!(got, flags, "xam({f:?})");
            assert_eq!(cpu.c1(), sign, "xam({f:?}) sign");
        }
    }

    #[test]
    fn ldcw16_drives_the_rounding_mode() {
        let mut cpu = CpuState::default();
        // rc is bits 10-11 of fcw; pc is bits 8-9. 0x0fff sets both to 0b11.
        cpu.fpu_ldcw16(0x0fff);
        assert_eq!(cpu.rc(), 3);
        assert_eq!(cpu.fpu_stcw16(), 0x0fff);
        assert_eq!(f80::rounding_mode(), RoundingMode::RoundChop);
        cpu.fpu_ldcw16(0x037f); // pc = 0b11 (extended), rc = 0 -> nearest
        assert_eq!(cpu.pc(), 3);
        assert_eq!(cpu.rc(), 0);
        assert_eq!(f80::rounding_mode(), RoundingMode::RoundToNearest);
        cpu.fpu_ldcw16(0x0400); // rc = 0b01 -> round down
        assert_eq!(f80::rounding_mode(), RoundingMode::RoundDown);
        f80::set_rounding_mode(RoundingMode::RoundToNearest);
    }

    #[test]
    fn save_and_restore_roundtrip() {
        let mut cpu = cpu_with(&[1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0]);
        cpu.set_top(3);
        cpu.fcw = 0x037f;
        cpu.set_c0(true);
        let saved = cpu.fpu_save32();
        assert_eq!(saved.env.control, 0x037f);
        assert_eq!(saved.env.status, cpu.fsw as u32);

        let mut other = CpuState::default();
        other.fpu_restore32(&saved);
        assert_eq!(other.fsw, cpu.fsw);
        assert_eq!(other.fcw, cpu.fcw);
        for i in 0..8 {
            assert_eq!(other.st(i), cpu.st(i), "register {i}");
        }
    }

    #[test]
    fn clex_clears_exceptions_and_the_eflags_sign_bit() {
        let mut cpu = CpuState { fsw: 0xffff, sf: true, ..CpuState::default() };
        cpu.set_c0(true);
        cpu.fpu_clex();
        // IE, DE, ZE, OE, UE, PE, ES and B are clear (bits 0-5, 7 and 15)...
        assert_eq!(cpu.fsw & 0x80bf, 0);
        // ...but the stack-fault bit, C0-C3 and top all survive
        assert_eq!(cpu.fsw & crate::cpu::FSW_STF, crate::cpu::FSW_STF);
        assert!(cpu.c0());
        assert_eq!(cpu.fsw & crate::cpu::FSW_C1, crate::cpu::FSW_C1);
        // and EFLAGS.SF was cleared while the FPU stack-fault bit was not
        assert!(!cpu.sf);
        assert_eq!(cpu.fsw & crate::cpu::FSW_STF, crate::cpu::FSW_STF);
    }

    #[test]
    fn trig_and_transcendentals_run_in_double_precision() {
        let mut cpu = CpuState::default();
        cpu.fpu_ldm64(0.0);
        cpu.fpu_sin();
        assert_eq!(cpu.st(0).to_f64(), 0.0);
        cpu.fpu_cos();
        assert_eq!(cpu.st(0).to_f64(), 1.0);

        // 2^x - 1
        let mut cpu = CpuState::default();
        cpu.fpu_ldm64(1.0);
        cpu.fpu_2xm1();
        assert_eq!(cpu.st(0).to_f64(), 1.0);

        // patan: ST(1) = atan2(ST(1), ST(0)), then pop
        let mut cpu = CpuState::default();
        cpu.fpu_ldm64(1.0); // ST(0) after both pushes
        cpu.fpu_ldm64(1.0);
        cpu.set_st(1, Float80::from_f64(1.0));
        cpu.set_st(0, Float80::from_f64(1.0));
        cpu.fpu_patan();
        assert!((cpu.st(0).to_f64() - std::f64::consts::FRAC_PI_4).abs() < 1e-15);
    }
}
