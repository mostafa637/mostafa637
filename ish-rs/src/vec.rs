// Line-by-line translation of emu/vec.c (SSE) + emu/mmx.c (MMX)
//
// 166 functions, exactly the set `nm` reports as defined in the compiled C:
// `emu/vec.h` declares 166 and the two .c files define 166, with no orphans on
// either side. `tools/vec-dump.c` calls all of them and
// `tests/vec_differential.rs` replays the results.
//
// The C generates most of these from macros (`VEC_SSE_OP`, `VEC_MMX_SHIFT`,
// `_VEC_SSE_CMP`, ...). The port spells the macro bodies out once as helper
// functions and passes the element width in, because Rust cannot build
// identifiers the way `##` does.
//
// The C works through `union vec`/`union xmm_reg`, i.e. by reinterpreting the
// same 16 bytes at different element widths on a little-endian host. Here the
// bytes are the storage (`XmmReg::bytes`) and every accessor is defined in
// terms of them, so the endianness assumption is stated in one place instead of
// assumed everywhere.

use crate::cpu::{CpuState, MmReg, XmmReg};

// ---- element access on a byte slice -------------------------------------
//
// Shared by the 16-byte and 8-byte paths, so the two cannot drift apart.

fn g16(b: &[u8], i: usize) -> u16 {
    u16::from_le_bytes([b[2 * i], b[2 * i + 1]])
}
fn s16(b: &mut [u8], i: usize, v: u16) {
    b[2 * i..2 * i + 2].copy_from_slice(&v.to_le_bytes());
}
fn g32(b: &[u8], i: usize) -> u32 {
    u32::from_le_bytes([b[4 * i], b[4 * i + 1], b[4 * i + 2], b[4 * i + 3]])
}
fn s32(b: &mut [u8], i: usize, v: u32) {
    b[4 * i..4 * i + 4].copy_from_slice(&v.to_le_bytes());
}
fn g64(b: &[u8], i: usize) -> u64 {
    let mut t = [0u8; 8];
    t.copy_from_slice(&b[8 * i..8 * i + 8]);
    u64::from_le_bytes(t)
}
fn s64(b: &mut [u8], i: usize, v: u64) {
    b[8 * i..8 * i + 8].copy_from_slice(&v.to_le_bytes());
}

// ---- the saturating helpers ---------------------------------------------

/// `static inline int32_t satsw(int32_t dw)`
///
/// The first branch looks wrong but is not: for the inputs this ever sees (a
/// `uint16_t` promoted to `int`) everything above 0xff80 is a negative byte
/// that already fits, so masking to the low byte is a no-op on the value.
fn satsw(dw: i32) -> i32 {
    if dw > 0xff80 {
        dw & 0xff
    } else if dw > 0x7fff {
        0x80
    } else if dw > 0x7f {
        0x7f
    } else {
        dw
    }
}

/// `static inline uint32_t satud(uint32_t dw)`
fn satud(dw: u32) -> u32 {
    if dw > 0xffff_8000 {
        dw & 0xffff
    } else if dw > 0x7fff_ffff {
        0x8000
    } else if dw > 0x7fff {
        0x7fff
    } else {
        dw
    }
}

/// `static inline uint32_t satub(uint32_t dw)`
fn satub(dw: u32) -> u32 {
    if dw >= 0x8000 {
        0
    } else if dw > 0xff {
        0xff
    } else {
        dw
    }
}

/// `static inline uint32_t satsb(uint32_t dw)`
fn satsb(dw: u32) -> u32 {
    if dw > 0xffff_ff80 {
        dw & 0xff
    } else if dw > 0x7fff_ffff {
        0x80
    } else if dw > 0x7f {
        0x7f
    } else {
        dw
    }
}

// ---- zero / copy / merge ------------------------------------------------

/// `#define VEC_ZERO_COPY(zero, copy)`
fn zero_copy<const ZERO: usize, const COPY: usize>(src: &[u8], dst: &mut [u8]) {
    debug_assert!(dst.len() >= ZERO / 8, "dst must hold zero/8 bytes");
    dst[..COPY / 8].copy_from_slice(&src[..COPY / 8]);
    for byte in dst[COPY / 8..ZERO / 8].iter_mut() {
        *byte = 0;
    }
}

pub fn vec_zero128_copy128(src: &[u8], dst: &mut [u8]) {
    zero_copy::<128, 128>(src, dst)
}
pub fn vec_zero128_copy64(src: &[u8], dst: &mut [u8]) {
    zero_copy::<128, 64>(src, dst)
}
pub fn vec_zero128_copy32(src: &[u8], dst: &mut [u8]) {
    zero_copy::<128, 32>(src, dst)
}
pub fn vec_zero64_copy64(src: &[u8], dst: &mut [u8]) {
    zero_copy::<64, 64>(src, dst)
}
pub fn vec_zero64_copy32(src: &[u8], dst: &mut [u8]) {
    zero_copy::<64, 32>(src, dst)
}
pub fn vec_zero32_copy32(src: &[u8], dst: &mut [u8]) {
    zero_copy::<32, 32>(src, dst)
}

/// "merge" means don't zero the register before writing to it
pub fn vec_merge32(src: &[u8], dst: &mut [u8]) {
    dst[..4].copy_from_slice(&src[..4]);
}
pub fn vec_merge64(src: &[u8], dst: &mut [u8]) {
    dst[..8].copy_from_slice(&src[..8]);
}
pub fn vec_merge128(src: &[u8], dst: &mut [u8]) {
    dst[..16].copy_from_slice(&src[..16]);
}

// ---- shifts -------------------------------------------------------------

#[derive(Clone, Copy, PartialEq, Eq)]
enum Dir {
    Left,
    Right,
}

/// `#define _SHIFT(op, size)` for the 128-bit registers.
///
/// `amount > size - 1` zeroes the *whole* register; note the C compares
/// against the element width, so a 64-bit shift by 63 is still a shift, while
/// a 16-bit shift by 16 is not.
fn shift128(amount: u32, dst: &mut XmmReg, dir: Dir, width: u32, n: usize) {
    if amount > width - 1 {
        *dst = XmmReg::ZERO;
        return;
    }
    let mut b = dst.bytes();
    for i in 0..n {
        match width {
            16 => {
                let v = g16(&b, i);
                s16(&mut b, i, shift(v as u64, amount, dir) as u16)
            }
            32 => {
                let v = g32(&b, i);
                s32(&mut b, i, shift(v as u64, amount, dir) as u32)
            }
            _ => {
                let v = g64(&b, i);
                s64(&mut b, i, shift(v, amount, dir))
            }
        }
    }
    *dst = XmmReg::from_bytes(b);
}

/// `#define _SHIFT(op, size)` for the 64-bit MMX registers.
fn shift64(amount: u32, dst: &mut MmReg, dir: Dir, width: u32, n: usize) {
    if amount > width - 1 {
        *dst = MmReg::ZERO;
        return;
    }
    let mut b = dst.bytes();
    for i in 0..n {
        match width {
            16 => {
                let v = g16(&b, i);
                s16(&mut b, i, shift(v as u64, amount, dir) as u16)
            }
            32 => {
                let v = g32(&b, i);
                s32(&mut b, i, shift(v as u64, amount, dir) as u32)
            }
            _ => {
                let v = g64(&b, i);
                s64(&mut b, i, shift(v, amount, dir))
            }
        }
    }
    *dst = MmReg::from_bytes(b);
}

fn shift(v: u64, amount: u32, dir: Dir) -> u64 {
    match dir {
        Dir::Left => v << amount,
        Dir::Right => v >> amount,
    }
}

/// `#define VEC_SSE_SHIFT(dir, suffix, op, size)`
macro_rules! sse_shift {
    ($reg:ident, $imm:ident, $dir:expr, $width:expr, $n:expr) => {
        pub fn $reg(src: &XmmReg, dst: &mut XmmReg) {
            let amount = src.u8(0);
            shift128(amount as u32, dst, $dir, $width, $n);
        }
        pub fn $imm(amount: u8, dst: &mut XmmReg) {
            shift128(amount as u32, dst, $dir, $width, $n);
        }
    };
}

/// `#define VEC_MMX_SHIFT(dir, suffix, op, size)`
///
/// Note the C takes the shift count from `src->qw` assigned to a `uint8_t`, so
/// only the low byte of the MM register survives.
macro_rules! mmx_shift {
    ($reg:ident, $imm:ident, $dir:expr, $width:expr, $n:expr) => {
        pub fn $reg(src: &MmReg, dst: &mut MmReg) {
            let amount = src.qw() as u8;
            shift64(amount as u32, dst, $dir, $width, $n);
        }
        pub fn $imm(amount: u8, dst: &mut MmReg) {
            shift64(amount as u32, dst, $dir, $width, $n);
        }
    };
}

sse_shift!(vec_shiftr_w128, vec_imm_shiftr_w128, Dir::Right, 16, 8);
sse_shift!(vec_shiftr_d128, vec_imm_shiftr_d128, Dir::Right, 32, 4);
sse_shift!(vec_shiftr_q128, vec_imm_shiftr_q128, Dir::Right, 64, 2);
sse_shift!(vec_shiftl_w128, vec_imm_shiftl_w128, Dir::Left, 16, 8);
sse_shift!(vec_shiftl_d128, vec_imm_shiftl_d128, Dir::Left, 32, 4);
sse_shift!(vec_shiftl_q128, vec_imm_shiftl_q128, Dir::Left, 64, 2);

mmx_shift!(vec_shiftr_w64, vec_imm_shiftr_w64, Dir::Right, 16, 4);
mmx_shift!(vec_shiftr_d64, vec_imm_shiftr_d64, Dir::Right, 32, 2);
mmx_shift!(vec_shiftr_q64, vec_imm_shiftr_q64, Dir::Right, 64, 1);
mmx_shift!(vec_shiftl_w64, vec_imm_shiftl_w64, Dir::Left, 16, 4);
mmx_shift!(vec_shiftl_d64, vec_imm_shiftl_d64, Dir::Left, 32, 2);
mmx_shift!(vec_shiftl_q64, vec_imm_shiftl_q64, Dir::Left, 64, 1);

pub fn vec_imm_shiftl_dq128(amount: u8, dst: &mut XmmReg) {
    if amount >= 16 {
        *dst = XmmReg::ZERO;
    } else {
        *dst = XmmReg(dst.0 << (amount as u32 * 8));
    }
}
pub fn vec_imm_shiftr_dq128(amount: u8, dst: &mut XmmReg) {
    if amount >= 16 {
        *dst = XmmReg::ZERO;
    } else {
        *dst = XmmReg(dst.0 >> (amount as u32 * 8));
    }
}

/// `vec_shiftrs_w128` / `vec_imm_shiftrs_w128` / `vec_shiftrs_w64` / ...
///
/// An arithmetic right shift; past the element width the C fills with the sign
/// bit rather than leaving the shift undefined.
fn shiftrs128_w(amount: u32, dst: &mut XmmReg) {
    let mut b = dst.bytes();
    for i in 0..8 {
        let v = g16(&b, i);
        let r = if amount > 15 {
            if (v >> 15) & 1 == 1 {
                0xffff
            } else {
                0
            }
        } else {
            ((v as i16) >> amount) as u16
        };
        s16(&mut b, i, r);
    }
    *dst = XmmReg::from_bytes(b);
}
fn shiftrs128_d(amount: u32, dst: &mut XmmReg) {
    let mut b = dst.bytes();
    for i in 0..4 {
        let v = g32(&b, i);
        let r = if amount > 31 {
            if (v >> 31) & 1 == 1 {
                0xffff_ffff
            } else {
                0
            }
        } else {
            ((v as i32) >> amount) as u32
        };
        s32(&mut b, i, r);
    }
    *dst = XmmReg::from_bytes(b);
}
fn shiftrs64_w(amount: u32, dst: &mut MmReg) {
    let mut b = dst.bytes();
    for i in 0..4 {
        let v = g16(&b, i);
        let r = if amount > 15 {
            if (v >> 15) & 1 == 1 {
                0xffff
            } else {
                0
            }
        } else {
            ((v as i16) >> amount) as u16
        };
        s16(&mut b, i, r);
    }
    *dst = MmReg::from_bytes(b);
}
fn shiftrs64_d(amount: u32, dst: &mut MmReg) {
    let mut b = dst.bytes();
    for i in 0..2 {
        let v = g32(&b, i);
        let r = if amount > 31 {
            if (v >> 31) & 1 == 1 {
                0xffff_ffff
            } else {
                0
            }
        } else {
            ((v as i32) >> amount) as u32
        };
        s32(&mut b, i, r);
    }
    *dst = MmReg::from_bytes(b);
}

pub fn vec_shiftrs_w128(src: &XmmReg, dst: &mut XmmReg) {
    shiftrs128_w(src.u8(0) as u32, dst)
}
pub fn vec_imm_shiftrs_w128(amount: u8, dst: &mut XmmReg) {
    shiftrs128_w(amount as u32, dst)
}
pub fn vec_shiftrs_d128(src: &XmmReg, dst: &mut XmmReg) {
    shiftrs128_d(src.u8(0) as u32, dst)
}
pub fn vec_imm_shiftrs_d128(amount: u8, dst: &mut XmmReg) {
    shiftrs128_d(amount as u32, dst)
}
pub fn vec_shiftrs_w64(src: &MmReg, dst: &mut MmReg) {
    shiftrs64_w(src.qw() as u8 as u32, dst)
}
pub fn vec_imm_shiftrs_w64(amount: u8, dst: &mut MmReg) {
    shiftrs64_w(amount as u32, dst)
}
pub fn vec_shiftrs_d64(src: &MmReg, dst: &mut MmReg) {
    shiftrs64_d(src.qw() as u8 as u32, dst)
}
pub fn vec_imm_shiftrs_d64(amount: u8, dst: &mut MmReg) {
    shiftrs64_d(amount as u32, dst)
}

// ---- compares -----------------------------------------------------------

/// `_VEC_SSE_CMP` / `_VEC_MMX_CMP`. Each element becomes all-ones or all-zeroes.
/// The equality compares are the macro's unsigned form; the `>` compares are
/// the signed form (`VEC_SSE_CMPS` passes no `u` prefix).
fn cmp_elems(d: &[u8], s: &[u8], width: u32, gt: bool) -> bool {
    match width {
        8 if gt => (d[0] as i8) > (s[0] as i8),
        16 if gt => (g16(d, 0) as i16) > (g16(s, 0) as i16),
        32 if gt => (g32(d, 0) as i32) > (g32(s, 0) as i32),
        8 => d[0] == s[0],
        16 => g16(d, 0) == g16(s, 0),
        _ => g32(d, 0) == g32(s, 0),
    }
}

fn cmp128(src: &XmmReg, dst: &mut XmmReg, width: u32, n: usize, gt: bool) {
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in 0..n {
        let (de, se): (&[u8], &[u8]) = match width {
            8 => (&d[i..i + 1], &s[i..i + 1]),
            16 => (&d[2 * i..2 * i + 2], &s[2 * i..2 * i + 2]),
            _ => (&d[4 * i..4 * i + 4], &s[4 * i..4 * i + 4]),
        };
        let hit = cmp_elems(de, se, width, gt);
        match width {
            8 => d[i] = if hit { 0xff } else { 0 },
            16 => s16(&mut d, i, if hit { 0xffff } else { 0 }),
            _ => s32(&mut d, i, if hit { 0xffff_ffff } else { 0 }),
        }
    }
    *dst = XmmReg::from_bytes(d);
}

fn cmp64(src: &MmReg, dst: &mut MmReg, width: u32, n: usize, gt: bool) {
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in 0..n {
        let (de, se): (&[u8], &[u8]) = match width {
            8 => (&d[i..i + 1], &s[i..i + 1]),
            16 => (&d[2 * i..2 * i + 2], &s[2 * i..2 * i + 2]),
            _ => (&d[4 * i..4 * i + 4], &s[4 * i..4 * i + 4]),
        };
        let hit = cmp_elems(de, se, width, gt);
        match width {
            8 => d[i] = if hit { 0xff } else { 0 },
            16 => s16(&mut d, i, if hit { 0xffff } else { 0 }),
            _ => s32(&mut d, i, if hit { 0xffff_ffff } else { 0 }),
        }
    }
    *dst = MmReg::from_bytes(d);
}

macro_rules! sse_cmp {
    ($name:ident, $width:expr, $n:expr, $gt:expr) => {
        pub fn $name(src: &XmmReg, dst: &mut XmmReg) {
            cmp128(src, dst, $width, $n, $gt)
        }
    };
}
macro_rules! mmx_cmp {
    ($name:ident, $width:expr, $n:expr, $gt:expr) => {
        pub fn $name(src: &MmReg, dst: &mut MmReg) {
            cmp64(src, dst, $width, $n, $gt)
        }
    };
}

sse_cmp!(vec_compare_eqb128, 8, 16, false);
sse_cmp!(vec_compare_eqw128, 16, 8, false);
sse_cmp!(vec_compare_eqd128, 32, 4, false);
sse_cmp!(vec_compares_gtb128, 8, 16, true);
sse_cmp!(vec_compares_gtw128, 16, 8, true);
sse_cmp!(vec_compares_gtd128, 32, 4, true);

mmx_cmp!(vec_compare_eqb64, 8, 8, false);
mmx_cmp!(vec_compare_eqw64, 16, 4, false);
mmx_cmp!(vec_compare_eqd64, 32, 2, false);
mmx_cmp!(vec_compares_gtb64, 8, 8, true);
mmx_cmp!(vec_compares_gtw64, 16, 4, true);
mmx_cmp!(vec_compares_gtd64, 32, 2, true);

// ---- integer add / sub / logic ------------------------------------------

/// `VEC_SSE_OP(name, suffix, op, size)`
///
/// The C works on `union vec`, whose widest member is `__uint128_t dqw`; the
/// `dq` operations are a single element spanning the whole register. The
/// operator is therefore written over `u128` and narrowed back to the element
/// width, which is exactly what the C's per-width loop does.
fn arith128(src: &XmmReg, dst: &mut XmmReg, op: fn(u128, u128) -> u128, width: u32, n: usize) {
    if width == 128 {
        *dst = XmmReg(op(dst.0, src.0));
        return;
    }
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in 0..n {
        match width {
            8 => d[i] = op(d[i] as u128, s[i] as u128) as u8,
            16 => {
                let v = op(g16(&d, i) as u128, g16(&s, i) as u128) as u16;
                s16(&mut d, i, v)
            }
            32 => {
                let v = op(g32(&d, i) as u128, g32(&s, i) as u128) as u32;
                s32(&mut d, i, v)
            }
            _ => {
                let v = op(g64(&d, i) as u128, g64(&s, i) as u128) as u64;
                s64(&mut d, i, v)
            }
        }
    }
    *dst = XmmReg::from_bytes(d);
}

/// `VEC_MMX_OP(name, suffix, op, size)`
fn arith64(src: &MmReg, dst: &mut MmReg, op: fn(u64, u64) -> u64, width: u32, n: usize) {
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in 0..n {
        match width {
            8 => d[i] = op(d[i] as u64, s[i] as u64) as u8,
            16 => {
                let v = op(g16(&d, i) as u64, g16(&s, i) as u64) as u16;
                s16(&mut d, i, v)
            }
            32 => {
                let v = op(g32(&d, i) as u64, g32(&s, i) as u64) as u32;
                s32(&mut d, i, v)
            }
            _ => {
                let v = op(g64(&d, i), g64(&s, i));
                s64(&mut d, i, v)
            }
        }
    }
    *dst = MmReg::from_bytes(d);
}

macro_rules! sse_op {
    ($name:ident, $op:expr, $width:expr, $n:expr) => {
        pub fn $name(src: &XmmReg, dst: &mut XmmReg) {
            arith128(src, dst, $op, $width, $n)
        }
    };
}
macro_rules! mmx_op {
    ($name:ident, $op:expr, $width:expr, $n:expr) => {
        pub fn $name(src: &MmReg, dst: &mut MmReg) {
            arith64(src, dst, $op, $width, $n)
        }
    };
}

sse_op!(vec_add_b128, |a, b| a.wrapping_add(b), 8, 16);
sse_op!(vec_add_w128, |a, b| a.wrapping_add(b), 16, 8);
sse_op!(vec_add_d128, |a, b| a.wrapping_add(b), 32, 4);
sse_op!(vec_add_q128, |a, b| a.wrapping_add(b), 64, 2);
sse_op!(vec_sub_b128, |a, b| a.wrapping_sub(b), 8, 16);
sse_op!(vec_sub_w128, |a, b| a.wrapping_sub(b), 16, 8);
sse_op!(vec_sub_d128, |a, b| a.wrapping_sub(b), 32, 4);
sse_op!(vec_sub_q128, |a, b| a.wrapping_sub(b), 64, 2);
sse_op!(vec_and_dq128, |a, b| a & b, 128, 1);
sse_op!(vec_or_dq128, |a, b| a | b, 128, 1);
sse_op!(vec_xor_dq128, |a, b| a ^ b, 128, 1);

mmx_op!(vec_add_b64, |a, b| a.wrapping_add(b), 8, 8);
mmx_op!(vec_add_w64, |a, b| a.wrapping_add(b), 16, 4);
mmx_op!(vec_add_d64, |a, b| a.wrapping_add(b), 32, 2);
mmx_op!(vec_add_q64, |a, b| a.wrapping_add(b), 64, 1);
mmx_op!(vec_sub_b64, |a, b| a.wrapping_sub(b), 8, 8);
mmx_op!(vec_sub_w64, |a, b| a.wrapping_sub(b), 16, 4);
mmx_op!(vec_sub_d64, |a, b| a.wrapping_sub(b), 32, 2);
mmx_op!(vec_sub_q64, |a, b| a.wrapping_sub(b), 64, 1);
mmx_op!(vec_and_q64, |a, b| a & b, 64, 1);
mmx_op!(vec_or_q64, |a, b| a | b, 64, 1);
mmx_op!(vec_xor_q64, |a, b| a ^ b, 64, 1);

pub fn vec_andn128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    let (d0, d1) = (g64(&d, 0), g64(&d, 1));
    s64(&mut d, 0, !d0 & g64(&s, 0));
    s64(&mut d, 1, !d1 & g64(&s, 1));
    *dst = XmmReg::from_bytes(d);
}

// ---- saturating add / sub -----------------------------------------------

pub fn vec_addus_b128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..16 {
        let sb = d[i] as i32 + s[i] as i32;
        d[i] = if sb > 0xff { 0xff } else { sb as u8 };
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_addus_w128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..8 {
        let sw = g16(&d, i) as i32 + g16(&s, i) as i32;
        s16(&mut d, i, if sw > 0xffff { 0xffff } else { sw as u16 });
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_addss_b128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..16 {
        let sum = (d[i] as i8) as i32 + (s[i] as i8) as i32;
        d[i] = satsb(sum as u32) as u8;
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_addss_w128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..8 {
        let sum = (g16(&d, i) as i16) as i32 + (g16(&s, i) as i16) as i32;
        s16(&mut d, i, satud(sum as u32) as u16);
    }
    *dst = XmmReg::from_bytes(d);
}

pub fn vec_subus_b128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..16 {
        let sb = d[i] as i32 - s[i] as i32;
        d[i] = if sb < 0 { 0 } else { sb as u8 };
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_subus_w128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..8 {
        let sw = g16(&d, i) as i32 - g16(&s, i) as i32;
        s16(&mut d, i, if sw < 0 { 0 } else { sw as u16 });
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_subss_b128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..16 {
        let diff = (d[i] as i8) as i32 - (s[i] as i8) as i32;
        d[i] = satsb(diff as u32) as u8;
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_subss_w128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..8 {
        let diff = (g16(&d, i) as i16) as i32 - (g16(&s, i) as i16) as i32;
        s16(&mut d, i, satud(diff as u32) as u16);
    }
    *dst = XmmReg::from_bytes(d);
}

// ---- multiply -----------------------------------------------------------

pub fn vec_madd_d128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..4 {
        let a = (g16(&d, 2 * i) as i16) as i32 * (g16(&s, 2 * i) as i16) as i32;
        let b = (g16(&d, 2 * i + 1) as i16) as i32 * (g16(&s, 2 * i + 1) as i16) as i32;
        s32(&mut d, i, a.wrapping_add(b) as u32);
    }
    *dst = XmmReg::from_bytes(d);
}

pub fn vec_sumabs_w128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    let mut sum = [0u32, 0];
    for i in 0..8 {
        let difflo = d[i] as i32 - s[i] as i32;
        let diffhi = d[i + 8] as i32 - s[i + 8] as i32;
        sum[0] = sum[0].wrapping_add(if difflo < 0 {
            (difflo as u32).wrapping_neg()
        } else {
            difflo as u32
        });
        sum[1] = sum[1].wrapping_add(if diffhi < 0 {
            (diffhi as u32).wrapping_neg()
        } else {
            diffhi as u32
        });
    }
    s32(&mut d, 0, sum[0]);
    s32(&mut d, 2, sum[1]);
    s32(&mut d, 1, 0);
    s32(&mut d, 3, 0);
    *dst = XmmReg::from_bytes(d);
}

pub fn vec_mulu_dq128(src: &mut XmmReg, dst: &mut XmmReg) {
    // PMULUDQ: the *even* dwords of each operand, which is what the C reads.
    let (s, mut d) = (src.bytes(), dst.bytes());
    let lo = g32(&s, 0) as u64 * g32(&d, 0) as u64;
    let hi = g32(&s, 2) as u64 * g32(&d, 2) as u64;
    s64(&mut d, 0, lo);
    s64(&mut d, 1, hi);
    *dst = XmmReg::from_bytes(d);
}

pub fn vec_mulu_dq64(src: &mut MmReg, dst: &mut MmReg) {
    *dst = MmReg(src.dw(0) as u64 * dst.dw(0) as u64);
}

pub fn vec_mulu64(src: &MmReg, dst: &mut MmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..4 {
        let res = ((g16(&d, i) as i16) as i32 * (g16(&s, i) as i16) as i32) as u32;
        s16(&mut d, i, ((res >> 16) & 0xffff) as u16);
    }
    *dst = MmReg::from_bytes(d);
}
pub fn vec_mull64(src: &MmReg, dst: &mut MmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..4 {
        let v = g16(&d, i).wrapping_mul(g16(&s, i));
        s16(&mut d, i, v);
    }
    *dst = MmReg::from_bytes(d);
}

pub fn vec_mulu128(src: &XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..8 {
        let res = ((g16(&d, i) as i16) as i32 * (g16(&s, i) as i16) as i32) as u32;
        s16(&mut d, i, ((res >> 16) & 0xffff) as u16);
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_muluu128(src: &XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..8 {
        let res = g16(&d, i) as u32 * g16(&s, i) as u32;
        s16(&mut d, i, ((res >> 16) & 0xffff) as u16);
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_mull128(src: &XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..8 {
        let v = g16(&d, i).wrapping_mul(g16(&s, i));
        s16(&mut d, i, v);
    }
    *dst = XmmReg::from_bytes(d);
}

// ---- packed float arithmetic --------------------------------------------

/// `VEC_PACKED_OP(name, op, field, size, n)`
macro_rules! packed_op {
    ($name:ident, $op:tt, f64) => {
        pub fn $name(src: &mut XmmReg, dst: &mut XmmReg) {
            let mut d = dst.bytes();
            let s = src.bytes();
            for i in 0..2 {
                let r = f64::from_bits(g64(&d, i)) $op f64::from_bits(g64(&s, i));
                s64(&mut d, i, r.to_bits());
            }
            *dst = XmmReg::from_bytes(d);
        }
    };
    ($name:ident, $op:tt, f32) => {
        pub fn $name(src: &mut XmmReg, dst: &mut XmmReg) {
            let mut d = dst.bytes();
            let s = src.bytes();
            for i in 0..4 {
                let r = f32::from_bits(g32(&d, i)) $op f32::from_bits(g32(&s, i));
                s32(&mut d, i, r.to_bits());
            }
            *dst = XmmReg::from_bytes(d);
        }
    };
}

packed_op!(vec_add_p64, +, f64);
packed_op!(vec_add_p32, +, f32);
packed_op!(vec_sub_p64, -, f64);
packed_op!(vec_sub_p32, -, f32);
packed_op!(vec_mul_p64, *, f64);
packed_op!(vec_mul_p32, *, f32);

// ---- min / max / avg ----------------------------------------------------

pub fn vec_min_ub128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..16 {
        if s[i] < d[i] {
            d[i] = s[i];
        }
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_max_ub128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..16 {
        if s[i] > d[i] {
            d[i] = s[i];
        }
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_mins_w128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..8 {
        let (dv, sv) = (g16(&d, i), g16(&s, i));
        s16(&mut d, i, if (dv as i16) < (sv as i16) { dv } else { sv });
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_maxs_w128(src: &mut XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..8 {
        let (dv, sv) = (g16(&d, i), g16(&s, i));
        s16(&mut d, i, if (dv as i16) > (sv as i16) { dv } else { sv });
    }
    *dst = XmmReg::from_bytes(d);
}

pub fn vec_avg_b128(src: &XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..16 {
        d[i] = ((1 + d[i] as u32 + s[i] as u32) >> 1) as u8;
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_avg_w128(src: &XmmReg, dst: &mut XmmReg) {
    let (s, mut d) = (src.bytes(), dst.bytes());
    for i in 0..8 {
        let r = (1 + g16(&d, i) as u32 + g16(&s, i) as u32) >> 1;
        s16(&mut d, i, r as u16);
    }
    *dst = XmmReg::from_bytes(d);
}

// ---- scalar float -------------------------------------------------------

// `#[allow(clippy::assign_op_pattern)]` is deliberate, not laziness: the
// spelling changes the result for NaN operands.
//
// When both operands are NaN, x86 `ADDSS`/`ADDSD`/`MULSS`/`MULDD` return the
// payload of the operand that was sitting in the destination register,
// quieted. Which one that is depends on the order the compiler picks, and both
// gcc and LLVM treat floating-point add and multiply as commutative, so either
// order is legal to them. Measured with gcc 12.2 (`tools/vec-dump.c`) and
// rustc 1.97, both operands NaN:
//
//   f32: `*dst += *src`          -> movss (dst), %xmm0; addss (src), %xmm0
//   f32: `*dst = *dst + *src`    -> movss (dst), %xmm0; addss (src), %xmm0
//   f32: `*dst += *src` in Rust  -> movss (src), %xmm0; addss (dst), %xmm0
//
// The first two agree and return the *destination* payload; the third returns
// the source's, so the compound spelling is not used here. The same holds for
// f64. Subtraction and division are not commutative, so their operand order is
// fixed and no such note applies to them.
//
// This is a codegen choice rather than a language guarantee, so
// `tests/vec_differential.rs` carries cases whose f32 *and* f64 views are both
// NaN: a compiler that starts picking differently fails that test instead of
// diverging silently.
macro_rules! scalar_f64 {
    ($name:ident, $op:tt) => {
        #[allow(clippy::assign_op_pattern)]
        pub fn $name(src: &f64, dst: &mut f64) {
            *dst = *dst $op *src;
        }
    };
}
macro_rules! scalar_f32 {
    ($name:ident, $op:tt) => {
        #[allow(clippy::assign_op_pattern)]
        pub fn $name(src: &f32, dst: &mut f32) {
            *dst = *dst $op *src;
        }
    };
}

scalar_f64!(vec_single_fadd64, +);
scalar_f32!(vec_single_fadd32, +);
scalar_f64!(vec_single_fmul64, *);
scalar_f32!(vec_single_fmul32, *);
scalar_f64!(vec_single_fsub64, -);
scalar_f32!(vec_single_fsub32, -);
scalar_f64!(vec_single_fdiv64, /);
scalar_f32!(vec_single_fdiv32, /);

pub fn vec_single_fsqrt64(src: &f64, dst: &mut f64) {
    *dst = src.sqrt();
}
pub fn vec_single_fsqrt32(src: &f32, dst: &mut f32) {
    *dst = src.sqrt();
}

pub fn vec_single_fmax64(src: &f64, dst: &mut f64) {
    if *src > *dst || src.is_nan() || dst.is_nan() {
        *dst = *src;
    }
}
pub fn vec_single_fmin64(src: &f64, dst: &mut f64) {
    if *src < *dst || src.is_nan() || dst.is_nan() {
        *dst = *src;
    }
}
pub fn vec_single_fmax32(src: &f32, dst: &mut f32) {
    if *src > *dst || src.is_nan() || dst.is_nan() {
        *dst = *src;
    }
}
pub fn vec_single_fmin32(src: &f32, dst: &mut f32) {
    if *src < *dst || src.is_nan() || dst.is_nan() {
        *dst = *src;
    }
}

/// `static bool cmpd(double a, double b, int type)`
fn cmpd(a: f64, b: f64, type_: u8) -> bool {
    let mut res = match type_ % 4 {
        0 => a == b,
        1 => a < b,
        2 => a <= b,
        _ => a.is_nan() || b.is_nan(),
    };
    if type_ >= 4 {
        res = !res;
    }
    res
}
/// `static bool cmps(float a, float b, int type)`
fn cmps(a: f32, b: f32, type_: u8) -> bool {
    let mut res = match type_ % 4 {
        0 => a == b,
        1 => a < b,
        2 => a <= b,
        _ => a.is_nan() || b.is_nan(),
    };
    if type_ >= 4 {
        res = !res;
    }
    res
}

pub fn vec_single_fcmp64(src: &f64, dst: &mut XmmReg, type_: u8) {
    let d = dst.f64(0);
    dst.set_qw(0, if cmpd(d, *src, type_) { u64::MAX } else { 0 });
}
pub fn vec_single_fcmp32(src: &f32, dst: &mut XmmReg, type_: u8) {
    let d = dst.f32(0);
    dst.set_u32(0, if cmps(d, *src, type_) { u32::MAX } else { 0 });
}
pub fn vec_fcmp_p64(src: &XmmReg, dst: &mut XmmReg, type_: u8) {
    for i in 0..2 {
        let r = cmpd(dst.f64(i), src.f64(i), type_);
        dst.set_qw(i, if r { u64::MAX } else { 0 });
    }
}

/// `vec_single_ucomi32` — one of only two functions here that touch the CPU.
pub fn vec_single_ucomi32(cpu: &mut CpuState, src: &f32, dst: &f32) {
    ucomi(cpu, *src as f64, *dst as f64, src.is_nan() || dst.is_nan())
}
/// `vec_single_ucomi64`
pub fn vec_single_ucomi64(cpu: &mut CpuState, src: &f64, dst: &f64) {
    ucomi(cpu, *src, *dst, src.is_nan() || dst.is_nan())
}

fn ucomi(cpu: &mut CpuState, src: f64, dst: f64, nan: bool) {
    cpu.set_zf_res(false);
    cpu.set_pf_res(false);
    cpu.zf = src == dst;
    cpu.cf = (src > dst) as u8;
    cpu.pf = false;
    if nan {
        cpu.zf = true;
        cpu.cf = 1;
        cpu.pf = true;
    }
    cpu.of = 0;
    cpu.sf = false;
    cpu.af = false;
    cpu.set_sf_res(false);
}

// ---- conversions --------------------------------------------------------

/// x86 `CVTTSD2SI`/`CVTTSS2SI` semantics: truncate toward zero, and anything
/// that does not fit — including NaN and both infinities — becomes INT32_MIN.
/// Rust's `as i32` *saturates* instead, which would silently differ here.
fn cvtt_to_i32(v: f64) -> i32 {
    let t = v.trunc();
    if t.is_nan() || !(-2_147_483_648.0..2_147_483_648.0).contains(&t) {
        i32::MIN
    } else {
        t as i32
    }
}

/// `#define _VEC_CVT(src, dst, src_t, dst_t, n)` — the NaN case is the C's,
/// the out-of-range case is what the conversion instruction the compiler emits
/// does.
pub fn vec_cvtsi2sd32(src: &i32, dst: &mut f64) {
    *dst = *src as f64;
}
pub fn vec_cvttsd2si64(src: &f64, dst: &mut i32) {
    *dst = if src.is_nan() { i32::MIN } else { cvtt_to_i32(*src) };
}
/// Note the NaN branch: `_VEC_CVT` stores `INT32_MIN` *in the destination
/// type*, so a NaN double becomes the float -2^31 (0xcf000000), not a float
/// NaN. Same for `vec_cvtss2sd32` below.
pub fn vec_cvtsd2ss64(src: &f64, dst: &mut f32) {
    *dst = if src.is_nan() { i32::MIN as f32 } else { *src as f32 };
}
pub fn vec_cvtsi2ss32(src: &i32, dst: &mut f32) {
    *dst = *src as f32;
}
pub fn vec_cvttss2si32(src: &f32, dst: &mut i32) {
    *dst = if src.is_nan() { i32::MIN } else { cvtt_to_i32(*src as f64) };
}
pub fn vec_cvtss2sd32(src: &f32, dst: &mut f64) {
    *dst = if src.is_nan() { i32::MIN as f64 } else { *src as f64 };
}

/// `PACKED_VEC_CVT` — convert, then zero the rest of the register. The C notes
/// that the memset must come second because src and dst may alias.
pub fn vec_cvttpd2dq64(src: &XmmReg, dst: &mut XmmReg) {
    let mut out = [0u8; 16];
    for i in 0..2 {
        let v = src.f64(i);
        let r = if v.is_nan() { i32::MIN } else { cvtt_to_i32(v) };
        s32(&mut out, i, r as u32);
    }
    *dst = XmmReg::from_bytes(out);
}
pub fn vec_cvttps2dq32(src: &XmmReg, dst: &mut XmmReg) {
    let mut out = [0u8; 16];
    for i in 0..4 {
        let v = src.f32(i);
        let r = if v.is_nan() { i32::MIN } else { cvtt_to_i32(v as f64) };
        s32(&mut out, i, r as u32);
    }
    *dst = XmmReg::from_bytes(out);
}

// ---- unpack / pack ------------------------------------------------------

pub fn vec_unpackl_bw128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in (0..8).rev() {
        let tmp = d[i];
        d[i * 2 + 1] = s[i];
        d[i * 2] = tmp;
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackl_w128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in (0..4).rev() {
        let tmp = g16(&d, i);
        s16(&mut d, i * 2 + 1, g16(&s, i));
        s16(&mut d, i * 2, tmp);
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackl_dq128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    s32(&mut d, 3, g32(&s, 1));
    let t = g32(&d, 1);
    s32(&mut d, 2, t);
    s32(&mut d, 1, g32(&s, 0));
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackl_qdq128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    s64(&mut d, 1, g64(&s, 0));
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackl_ps128(src: &XmmReg, dst: &mut XmmReg) {
    // The C's write order differs from vec_unpackl_dq128; keep it verbatim.
    let s = src.bytes();
    let mut d = dst.bytes();
    let t = g32(&d, 1);
    s32(&mut d, 2, t);
    s32(&mut d, 1, g32(&s, 0));
    s32(&mut d, 3, g32(&s, 1));
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackl_pd128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    s64(&mut d, 1, g64(&s, 0));
    *dst = XmmReg::from_bytes(d);
}

pub fn vec_unpackh_bw128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in 0..8 {
        d[2 * i] = d[i + 8];
        d[2 * i + 1] = s[i + 8];
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackh_w128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in 0..4 {
        let t = g16(&d, i + 4);
        s16(&mut d, 2 * i, t);
        s16(&mut d, 2 * i + 1, g16(&s, i + 4));
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackh_d128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    let t0 = g32(&d, 2);
    let t2 = g32(&d, 3);
    s32(&mut d, 0, t0);
    s32(&mut d, 1, g32(&s, 2));
    s32(&mut d, 2, t2);
    s32(&mut d, 3, g32(&s, 3));
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackh_dq128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    let t = g64(&d, 1);
    s64(&mut d, 0, t);
    s64(&mut d, 1, g64(&s, 1));
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackh_ps128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    let t0 = g32(&d, 2);
    let t2 = g32(&d, 3);
    s32(&mut d, 0, t0);
    s32(&mut d, 1, g32(&s, 2));
    s32(&mut d, 2, t2);
    s32(&mut d, 3, g32(&s, 3));
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackh_pd128(src: &XmmReg, dst: &mut XmmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    let t = g64(&d, 1);
    s64(&mut d, 0, t);
    s64(&mut d, 1, g64(&s, 1));
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_unpackl_dq64(src: &MmReg, dst: &mut MmReg) {
    let s = src.bytes();
    let mut d = dst.bytes();
    s32(&mut d, 1, g32(&s, 0));
    *dst = MmReg::from_bytes(d);
}

/// `vec_packss_w128` — eight signed words to eight signed bytes, `dst` first
/// then `src`. The two operands may be the same register, so both are read
/// before anything is written.
pub fn vec_packss_w128(src: &XmmReg, dst: &mut XmmReg) {
    let (s, d) = (src.bytes(), dst.bytes());
    let from = |b: &[u8; 16], start: usize| -> u32 {
        let mut acc = 0u32;
        for j in 0..4 {
            acc |= ((satsw(g16(b, start + j) as i32) as u32) & 0xff) << (8 * j);
        }
        acc
    };
    let mut out = [0u8; 16];
    s32(&mut out, 0, from(&d, 0));
    s32(&mut out, 1, from(&d, 4));
    s32(&mut out, 2, from(&s, 0));
    s32(&mut out, 3, from(&s, 4));
    *dst = XmmReg::from_bytes(out);
}

pub fn vec_packss_d128(src: &XmmReg, dst: &mut XmmReg) {
    let (s, d) = (src.bytes(), dst.bytes());
    let mut out = [0u8; 16];
    for half in 0..2 {
        let base: &[u8; 16] = if half == 0 { &d } else { &s };
        for q in 0..2 {
            let v = satud(g32(base, q * 2)) | (satud(g32(base, q * 2 + 1)) << 16);
            s32(&mut out, half * 2 + q, v);
        }
    }
    *dst = XmmReg::from_bytes(out);
}

pub fn vec_packsu_w128(src: &XmmReg, dst: &mut XmmReg) {
    let (s, d) = (src.bytes(), dst.bytes());
    let mut out = [0u8; 16];
    for half in 0..2 {
        let base: &[u8; 16] = if half == 0 { &d } else { &s };
        for q in 0..2 {
            let mut acc = 0u32;
            for j in 0..4 {
                acc |= satub(g16(base, q * 4 + j) as u32) << (8 * j);
            }
            s32(&mut out, half * 2 + q, acc);
        }
    }
    *dst = XmmReg::from_bytes(out);
}

// ---- shuffles -----------------------------------------------------------

pub fn vec_shuffle_lw128(src: &XmmReg, dst: &mut XmmReg, encoding: u8) {
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in 0..4 {
        let t = g16(&s, ((encoding >> (i * 2)) % 4) as usize);
        s16(&mut d, i, t);
    }
    let hi = g64(&s, 1);
    s64(&mut d, 1, hi);
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_shuffle_hw128(src: &XmmReg, dst: &mut XmmReg, encoding: u8) {
    let s = src.bytes();
    let mut d = dst.bytes();
    let lo = g64(&s, 0);
    s64(&mut d, 0, lo);
    // the C writes `encoding >> 0 & 3` for symmetry with the >> 2/4/6 cases
    let a = g16(&s, ((encoding & 3) | 4) as usize) as u32;
    let b = g16(&s, ((encoding >> 2 & 3) | 4) as usize) as u32;
    s32(&mut d, 2, a | (b << 16));
    let c = g16(&s, ((encoding >> 4 & 3) | 4) as usize) as u32;
    let e = g16(&s, ((encoding >> 6 & 3) | 4) as usize) as u32;
    s32(&mut d, 3, c | (e << 16));
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_shuffle_d128(src: &XmmReg, dst: &mut XmmReg, encoding: u8) {
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in 0..4 {
        let t = g32(&s, ((encoding >> (i * 2)) % 4) as usize);
        s32(&mut d, i, t);
    }
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_shuffle_ps128(src: &XmmReg, dst: &mut XmmReg, encoding: u8) {
    let s = src.bytes();
    let mut d = dst.bytes();
    // reads dst while writing it, in the C's order
    let t0 = g32(&d, (encoding & 3) as usize);
    let t1 = g32(&d, ((encoding >> 2) & 3) as usize);
    s32(&mut d, 0, t0);
    s32(&mut d, 1, t1);
    s32(&mut d, 2, g32(&s, ((encoding >> 4) & 3) as usize));
    s32(&mut d, 3, g32(&s, ((encoding >> 6) & 3) as usize));
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_shuffle_pd128(src: &XmmReg, dst: &mut XmmReg, encoding: u8) {
    let s = src.bytes();
    let mut d = dst.bytes();
    let t = g64(&d, (encoding & 1) as usize);
    s64(&mut d, 0, t);
    s64(&mut d, 1, g64(&s, ((encoding >> 1) & 1) as usize));
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_shuffle_w64(src: &MmReg, dst: &mut MmReg, encoding: u8) {
    let s = src.bytes();
    let mut d = dst.bytes();
    for i in 0..4 {
        let t = g16(&s, ((encoding >> (2 * i)) % 4) as usize);
        s16(&mut d, i, t);
    }
    *dst = MmReg::from_bytes(d);
}

// ---- moves and masks ----------------------------------------------------

pub fn vec_movl_p64(src: &u64, dst: &mut XmmReg) {
    let mut d = dst.bytes();
    s64(&mut d, 0, *src);
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_movl_pm64(src: &XmmReg, dst: &mut u64) {
    *dst = src.qw(0);
}
pub fn vec_movh_p64(src: &u64, dst: &mut XmmReg) {
    let mut d = dst.bytes();
    s64(&mut d, 1, *src);
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_movh_pm64(src: &XmmReg, dst: &mut u64) {
    *dst = src.qw(1);
}

pub fn vec_movmask_b64(src: &MmReg, dst: &mut u32) {
    let mut m = 0u32;
    for (i, byte) in src.bytes().iter().enumerate() {
        if byte & (1 << 7) != 0 {
            m |= 1 << i;
        }
    }
    *dst = m;
}
pub fn vec_movmask_b128(src: &XmmReg, dst: &mut u32) {
    let mut m = 0u32;
    for (i, byte) in src.bytes().iter().enumerate() {
        if byte & (1 << 7) != 0 {
            m |= 1 << i;
        }
    }
    *dst = m;
}
pub fn vec_fmovmask_d128(src: &XmmReg, dst: &mut u32) {
    let mut m = 0u32;
    for i in 0..2 {
        // `signbit`, so -0.0 counts as negative
        if src.f64(i).is_sign_negative() {
            m |= 1 << i;
        }
    }
    *dst = m;
}

pub fn vec_insert_w64(src: &u32, dst: &mut MmReg, index: u8) {
    let mut d = dst.bytes();
    s16(&mut d, (index % 4) as usize, *src as u16);
    *dst = MmReg::from_bytes(d);
}
pub fn vec_insert_w128(src: &u32, dst: &mut XmmReg, index: u8) {
    let mut d = dst.bytes();
    s16(&mut d, (index % 8) as usize, *src as u16);
    *dst = XmmReg::from_bytes(d);
}
pub fn vec_extract_w128(src: &XmmReg, dst: &mut u32, index: u8) {
    *dst = src.u16((index % 8) as usize) as u32;
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn saturating_helpers_match_the_c() {
        // the "impossible looking" first branch of satsw: negative bytes that
        // already fit come back unchanged
        assert_eq!(satsw(0xff81), 0x81);
        assert_eq!(satsw(0xffff), 0xff);
        assert_eq!(satsw(0xff80), 0x80);
        assert_eq!(satsw(0x8000), 0x80);
        assert_eq!(satsw(0x7fff), 0x7f);
        assert_eq!(satsw(0x0080), 0x7f);
        assert_eq!(satsw(0x007f), 0x7f);
        assert_eq!(satsw(0x0040), 0x40);

        assert_eq!(satud(0xffff_8001), 0x8001);
        assert_eq!(satud(0xffff_ffff), 0xffff);
        assert_eq!(satud(0xffff_8000), 0x8000);
        assert_eq!(satud(0x8000_0000), 0x8000);
        assert_eq!(satud(0x0000_8000), 0x7fff);
        assert_eq!(satud(0x0000_7fff), 0x7fff);

        assert_eq!(satub(0x8000), 0);
        // 0x7fff is below the 0x8000 cutoff, so it clamps to 0xff, not to 0
        assert_eq!(satub(0x7fff), 0xff);
        assert_eq!(satub(0x0100), 0xff);
        assert_eq!(satub(0x00ff), 0xff);

        assert_eq!(satsb(0xffff_ff81), 0x81);
        assert_eq!(satsb(0xffff_ffff), 0xff);
        assert_eq!(satsb(0xffff_ff80), 0x80);
        assert_eq!(satsb(0x0000_0080), 0x7f);
        assert_eq!(satsb(0x0000_007f), 0x7f);
    }

    #[test]
    fn xmm_element_views_round_trip() {
        let mut r = XmmReg::ZERO;
        r.set_u16(0, 0x1234);
        r.set_u32(1, 0xdead_beef);
        r.set_qw(1, 0x0123_4567_89ab_cdef);
        assert_eq!(r.u16(0), 0x1234);
        assert_eq!(r.u32(1), 0xdead_beef);
        assert_eq!(r.qw(1), 0x0123_4567_89ab_cdef);
        // little endian: the low byte of qw(1) is byte 8
        assert_eq!(r.u8(8), 0xef);
        // a byte write touches only that byte of the qword above it
        r.set_u8(15, 0xab);
        assert_eq!(r.u8(15), 0xab);
        assert_eq!(r.qw(1), 0xab23_4567_89ab_cdef);

        let mut f = XmmReg::ZERO;
        f.set_f32(0, 1.5);
        f.set_f64(1, -2.25);
        assert_eq!(f.f32(0), 1.5);
        assert_eq!(f.f64(1), -2.25);
        assert_eq!(f.u32(0), 0x3fc0_0000);
    }

    #[test]
    fn shifts_zero_the_whole_register_past_the_element_width() {
        let mut r = XmmReg(0x0101_0101_0101_0101_0101_0101_0101_0101);
        vec_imm_shiftl_w128(15, &mut r);
        assert_ne!(r, XmmReg::ZERO);
        vec_imm_shiftl_w128(16, &mut r);
        assert_eq!(r, XmmReg::ZERO, "amount > size-1 zeroes, it does not wrap");

        let mut r = XmmReg(0xffff_ffff_ffff_ffff_ffff_ffff_ffff_ffff);
        vec_imm_shiftr_q128(63, &mut r);
        assert_eq!(r.qw(0), 1);
        vec_imm_shiftr_q128(64, &mut r);
        assert_eq!(r, XmmReg::ZERO);
    }

    #[test]
    fn dq_shifts_count_bytes_not_bits() {
        let mut r = XmmReg(0x0000_0000_0000_0000_0000_0000_0000_0001);
        vec_imm_shiftl_dq128(15, &mut r);
        assert_eq!(r.qw(1), 1 << 56);
        vec_imm_shiftl_dq128(16, &mut r);
        assert_eq!(r, XmmReg::ZERO);
    }

    #[test]
    fn arithmetic_shifts_fill_with_the_sign_bit() {
        let mut r = XmmReg::ZERO;
        r.set_u16(0, 0x8000); // -32768
        vec_imm_shiftrs_w128(20, &mut r);
        assert_eq!(r.u16(0), 0xffff);
        r.set_u16(0, 0x7fff);
        vec_imm_shiftrs_w128(20, &mut r);
        assert_eq!(r.u16(0), 0);
        r.set_u16(0, 0x8000);
        vec_imm_shiftrs_w128(15, &mut r);
        assert_eq!(r.u16(0), 0xffff);
    }

    #[test]
    fn mmx_shift_amount_is_truncated_to_a_byte() {
        // the C does `const uint8_t amount = src->qw;`
        let src = MmReg(0x100); // low byte 0 -> shift by 0
        let mut dst = MmReg(0xffff_ffff_ffff_ffff);
        vec_shiftr_q64(&src, &mut dst);
        assert_eq!(dst.qw(), u64::MAX);

        let src = MmReg(0x140); // low byte 0x40 = 64 -> past the width -> zero
        let mut dst = MmReg(0xffff_ffff_ffff_ffff);
        vec_shiftr_q64(&src, &mut dst);
        assert_eq!(dst.qw(), 0);
    }

    #[test]
    fn cvtt_uses_the_instruction_semantics_not_saturation() {
        assert_eq!(cvtt_to_i32(1e30), i32::MIN);
        assert_eq!(cvtt_to_i32(-1e30), i32::MIN);
        assert_eq!(cvtt_to_i32(f64::INFINITY), i32::MIN);
        assert_eq!(cvtt_to_i32(f64::NAN), i32::MIN);
        assert_eq!(cvtt_to_i32(2_147_483_647.9), i32::MAX);
        assert_eq!(cvtt_to_i32(-2_147_483_648.0), i32::MIN);
        assert_eq!(cvtt_to_i32(-0.9), 0);
        // Rust's `as i32` would have saturated these to i32::MAX/MIN
        assert_eq!(1e30f64 as i32, i32::MAX);
    }

    #[test]
    fn float_conversions_store_int32_min_for_nan() {
        // _VEC_CVT writes INT32_MIN in the *destination* type, so a NaN does
        // not stay a NaN
        let mut f = 0f32;
        vec_cvtsd2ss64(&f64::NAN, &mut f);
        assert_eq!(f.to_bits(), 0xcf00_0000, "the float -2^31");
        let mut d = 0f64;
        vec_cvtss2sd32(&f32::NAN, &mut d);
        assert_eq!(d, -2_147_483_648.0);
        // while a normal value converts normally
        vec_cvtsd2ss64(&1.5, &mut f);
        assert_eq!(f, 1.5);
    }

    #[test]
    fn nan_plus_nan_payload_matches_the_compiled_c() {
        // both operands NaN: the *destination* payload survives, quieted.
        // Verified against the compiled C, not assumed - see the note above
        // vec_single_fadd32.
        let src = f32::from_bits(0x7fc0_0000); // an already-quiet NaN
        let mut dst = f32::from_bits(0xff81_ff80); // a NaN with the sign bit set
        vec_single_fadd32(&src, &mut dst);
        assert_eq!(dst.to_bits(), 0xffc1_ff80, "the destination NaN, quieted");

        // note that 0xff81ff80_00000000 is *not* a NaN (exponent 0x7f8), so the
        // f64 case needs its own pair of real NaNs
        let mut d64 = f64::from_bits(0xfff1_ff80_0000_0000);
        assert!(d64.is_nan());
        vec_single_fadd64(&f64::from_bits(0x7ffc_0000_0000_0000), &mut d64);
        assert_eq!(d64.to_bits(), 0xfff9_ff80_0000_0000, "bit 51 set");

        let mut m = f64::from_bits(0xfff1_ff80_0000_0000);
        vec_single_fmul64(&f64::from_bits(0x7ffc_0000_0000_0000), &mut m);
        assert_eq!(m.to_bits(), 0xfff9_ff80_0000_0000, "multiply behaves the same");

        // a non-NaN destination with a NaN source still yields a NaN
        let mut d = 1.0f32;
        vec_single_fadd32(&src, &mut d);
        assert!(d.is_nan());
    }

    #[test]
    fn pmuludq_reads_the_even_dwords() {
        let mut src = XmmReg::ZERO;
        let mut dst = XmmReg::ZERO;
        src.set_u32(0, 3);
        src.set_u32(1, 1000); // ignored
        src.set_u32(2, 5);
        src.set_u32(3, 1000); // ignored
        dst.set_u32(0, 7);
        dst.set_u32(2, 11);
        vec_mulu_dq128(&mut src, &mut dst);
        assert_eq!(dst.qw(0), 21);
        assert_eq!(dst.qw(1), 55);
    }

    #[test]
    fn fmovmask_uses_the_sign_bit() {
        let mut r = XmmReg::ZERO;
        r.set_f64(0, -0.0);
        r.set_f64(1, 1.0);
        let mut m = 0u32;
        vec_fmovmask_d128(&r, &mut m);
        assert_eq!(m, 1, "-0.0 is negative");
    }

    #[test]
    fn ucomi_sets_the_flags_the_c_sets() {
        let mut cpu = CpuState::default();
        cpu.set_zf_res(true);
        cpu.set_pf_res(true);
        cpu.of = 9;
        // the C is `cpu->cf = *src > *dst`, on the *source* operand
        vec_single_ucomi64(&mut cpu, &1.0, &2.0);
        assert!(!cpu.zf);
        assert_eq!(cpu.cf, 0, "1.0 > 2.0 is false");
        assert!(!cpu.pf);
        assert_eq!(cpu.of, 0);
        assert!(!cpu.zf_res());
        assert!(!cpu.pf_res());

        vec_single_ucomi64(&mut cpu, &2.0, &1.0);
        assert!(!cpu.zf);
        assert_eq!(cpu.cf, 1, "2.0 > 1.0 is true");

        vec_single_ucomi64(&mut cpu, &1.0, &1.0);
        assert!(cpu.zf);
        assert_eq!(cpu.cf, 0);

        vec_single_ucomi64(&mut cpu, &f64::NAN, &2.0);
        assert!(cpu.zf && cpu.cf == 1 && cpu.pf, "unordered sets all three");
    }

    #[test]
    fn cmp_type_wraps_modulo_four_and_negates_above_three() {
        assert!(cmpd(1.0, 1.0, 0));
        assert!(!cmpd(1.0, 1.0, 4), "type >= 4 inverts");
        // 5 % 4 == 1 is the `<` test, then inverted
        assert!(cmpd(1.0, 2.0, 1));
        assert!(!cmpd(1.0, 2.0, 5));
        assert!(cmpd(f64::NAN, 1.0, 3));
        // 9 % 4 == 1 is `<` again, and >= 4 still inverts it
        assert!(!cmpd(1.0, 2.0, 9));
        assert!(cmpd(2.0, 1.0, 8), "8 % 4 == 0 is `==`, inverted");
    }
}
