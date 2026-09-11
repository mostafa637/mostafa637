// Line-by-line translation of emu/float80.h + emu/float80.c
// 80-bit extended precision floating point - full precision, NO f64 conversion

use std::fmt;

pub type Float128 = u128;

pub const BIAS80: u16 = 0x3fff;
pub const EXP_MAX: u16 = 0x7ffe;
pub const EXP_MIN: u16 = 0x0001;
pub const EXP_SPECIAL: u16 = 0x7fff;
pub const EXP_DENORMAL: u16 = 0;
pub const CURSED_BIT: u64 = 1u64 << 63;

#[derive(Debug, Clone, Copy, PartialEq, Default)]
pub struct Float80 {
    pub signif: u64,
    pub sign_exp: u16,
}

#[derive(Debug, Clone, Copy, PartialEq)]
pub enum RoundingMode {
    RoundToNearest = 0,
    RoundDown = 1,
    RoundUp = 2,
    RoundChop = 3,
}

thread_local! {
    pub static ROUNDING_MODE: std::cell::Cell<RoundingMode> =
        const { std::cell::Cell::new(RoundingMode::RoundToNearest) };
}

fn round_away_from_zero(sign: i32) -> bool {
    ROUNDING_MODE.with(|rm| {
        let mode = rm.get();
        (mode == RoundingMode::RoundUp && sign == 0) ||
        (mode == RoundingMode::RoundDown && sign != 0)
    })
}

fn bias(exp: i32) -> u16 {
    (exp + BIAS80 as i32) as u16
}

fn unbias(exp: u16) -> i32 {
    exp as i32 - BIAS80 as i32
}

fn unbias_denormal(exp: u16) -> i32 {
    if exp == EXP_DENORMAL {
        unbias(EXP_MIN)
    } else {
        unbias(exp)
    }
}

// The two rounding branches below deliberately look identical: the C
// original spells out "round up" and "round to nearest even" as separate
// cases even though both end in i++.
#[allow(clippy::if_same_then_else)]
fn u128_shift_right_round(i: Float128, shift: i32, sign: i32) -> Float128 {
    if shift == 0 { return i; }
    if shift > 127 {
        if round_away_from_zero(sign) && i != 0 { return 1; }
        return 0;
    }
    let shift = shift as u32;
    let guard = ((i >> (shift - 1)) & 1) as u64;
    let rest_mask: Float128 = !((!0u128) << (shift - 1));
    let rest = (i & rest_mask) as u64;
    let mut result = i >> shift;
    if guard == 0 && rest == 0 { return result; }

    if round_away_from_zero(sign) {
        result += 1;
    } else if ROUNDING_MODE.with(|rm| rm.get()) == RoundingMode::RoundToNearest && guard != 0 {
        if rest != 0 {
            result += 1;
        } else if result & 1 != 0 {
            result += 1;
        }
    }
    result
}

fn u128_clz(x: Float128) -> i32 {
    if x >> 64 != 0 {
        ((x >> 64) as u64).leading_zeros() as i32
    } else if x != 0 {
        64 + (x as u64).leading_zeros() as i32
    } else {
        128
    }
}

impl Float80 {
    pub fn new(signif: u64, sign_exp: u16) -> Self {
        Float80 { signif, sign_exp }
    }

    pub fn exp(&self) -> u16 { self.sign_exp & 0x7fff }
    pub fn sign(&self) -> bool { self.sign_exp & 0x8000 != 0 }
    pub fn sign_i32(&self) -> i32 { if self.sign() { 1 } else { 0 } }
    pub fn set_sign(&mut self, sign: bool) {
        if sign { self.sign_exp |= 0x8000; } else { self.sign_exp &= 0x7fff; }
    }

    /// Write the 15-bit exponent field *without* touching the sign bit.
    ///
    /// In the C original `exp` and `sign` are two bitfields packed into one
    /// 16-bit union, so `f.exp = X` leaves the sign alone. Assigning
    /// `sign_exp` directly in Rust would silently clear the sign - which is
    /// exactly the bug `set_exp` exists to prevent.
    pub fn set_exp(&mut self, exp: u16) {
        self.sign_exp = (self.sign_exp & 0x8000) | (exp & 0x7fff);
    }

    pub fn is_supported(&self) -> bool {
        if self.exp() == EXP_DENORMAL {
            self.signif >> 63 == 0
        } else {
            self.signif >> 63 == 1
        }
    }

    pub fn is_nan(&self) -> bool {
        self.exp() == EXP_SPECIAL && (self.signif & (!0u64 >> 1)) != 0
    }

    pub fn is_inf(&self) -> bool {
        self.exp() == EXP_SPECIAL && (self.signif & (!0u64 >> 1)) == 0
    }

    pub fn is_zero(&self) -> bool {
        self.exp() == EXP_DENORMAL && self.signif == 0
    }

    pub fn is_denormal(&self) -> bool {
        self.exp() == EXP_DENORMAL && self.signif != 0
    }

    pub const fn nan() -> Self {
        Float80 { signif: 0xC000000000000000, sign_exp: 0x7FFF }
    }

    pub const fn inf() -> Self {
        Float80 { signif: 0x8000000000000000, sign_exp: 0x7FFF }
    }

    fn shift_left(mut self, shift: i32) -> Self {
        self.signif = self.signif.wrapping_shl(shift as u32);
        let exp = self.exp() as i32 - shift;
        self.set_exp(exp as u16);
        self
    }

    fn shift_right(mut self, shift: i32) -> Self {
        self.signif = u128_shift_right_round(self.signif as Float128, shift, self.sign_i32()) as u64;
        let exp = self.exp() as i32 + shift;
        self.set_exp(exp as u16);
        self
    }

    fn normalize(mut self) -> Self {
        if self.exp() == EXP_DENORMAL || self.exp() == EXP_SPECIAL {
            assert!(self.is_supported());
        }
        if self.exp() == EXP_DENORMAL { return self; }

        let shift = if self.signif != 0 {
            self.signif.leading_zeros() as i32
        } else {
            64
        };

        if self.exp() as i32 - shift < EXP_MIN as i32 {
            self = self.shift_left(self.exp() as i32 - EXP_MIN as i32);
            self.set_exp(EXP_DENORMAL);
            return self;
        }
        self.shift_left(shift)
    }

    fn uncomparable(a: Float80, b: Float80) -> bool {
        if !a.is_supported() || !b.is_supported() { return true; }
        if a.is_nan() || b.is_nan() { return true; }
        false
    }

    /// `f80_neg`. Named after the C function, hence not `std::ops::Neg`.
    #[allow(clippy::should_implement_trait)]
    pub fn neg(mut self) -> Self {
        self.sign_exp ^= 0x8000;
        self
    }

    pub fn abs(mut self) -> Self {
        self.sign_exp &= 0x7fff;
        self
    }

    pub fn from_i64(i: i64) -> Self {
        let sign = i < 0;
        let abs_val: u64 = if i == i64::MIN {
            CURSED_BIT
        } else if i < 0 {
            i.wrapping_neg() as u64
        } else {
            i as u64
        };

        if abs_val == 0 {
            return Float80 { signif: 0, sign_exp: if sign { 0x8000 } else { 0 } };
        }

        let mut f = Float80 {
            signif: abs_val,
            sign_exp: bias(63),
        };
        f.set_sign(sign);
        f.normalize()
    }

    pub fn to_i64(&self) -> i64 {
        if !self.is_supported() { return i64::MIN; }
        if self.exp() > bias(63) { return i64::MIN; }
        let mut f = *self;
        f = f.shift_right(bias(63) as i32 - f.exp() as i32);
        // wrapping_neg: magnitude 0x8000... (i64::MIN) cannot be negated otherwise
        if !f.sign() { f.signif as i64 } else { (f.signif as i64).wrapping_neg() }
    }

    pub fn from_f64(d: f64) -> Self {
        let bits = d.to_bits();
        let sign = bits >> 63 != 0;
        let biased_exp = ((bits >> 52) & 0x7FF) as i32;
        let mantissa = bits & 0x000FFFFFFFFFFFFF;

        let mut f = Float80 { signif: 0, sign_exp: 0 };
        f.set_sign(sign);

        if biased_exp == 0x7FF {
            // preserve sign when setting special exponent
            let s = f.sign();
            f.sign_exp = EXP_SPECIAL;
            f.set_sign(s);
            f.signif = if mantissa == 0 { 0x8000000000000000 } else { mantissa << 11 | CURSED_BIT };
        } else if biased_exp == 0 {
            if mantissa == 0 {
                f.signif = 0;
                // keep sign for -0.0 (sign already set above, sign_exp=0 clears it so restore)
                let s = sign;
                f.sign_exp = 0;
                f.set_sign(s);
            } else {
                let s = f.sign();
                f.sign_exp = bias(1 - 0x3FF);
                f.set_sign(s);
                f.signif = mantissa << 11;
            }
        } else {
            let s = f.sign();
            f.sign_exp = bias(biased_exp - 0x3FF);
            f.set_sign(s);
            f.signif = mantissa << 11 | CURSED_BIT;
        }
        f.normalize()
    }

    pub fn to_f64(&self) -> f64 {
        if !self.is_supported() { return f64::NAN; }

        let sign = self.sign();
        let mut new_exp = unbias(self.exp()) + 0x3FF;

        if self.exp() == EXP_SPECIAL {
            new_exp = 0x7FF;
        } else if new_exp > 0x7FE {
            return if sign { f64::NEG_INFINITY } else { f64::INFINITY };
        }

        if new_exp <= 0 {
            let mut f = *self;
            f.signif >>= 1;
            f = f.shift_right(-new_exp);
            new_exp = unbias(f.exp()) + 0x3FF;
            let db_signif = u128_shift_right_round(f.signif as Float128, 11, f.sign_i32()) as u64;
            let final_exp = new_exp as u64;
            let bits = (sign as u64) << 63 | (final_exp << 52) | (db_signif & 0x000FFFFFFFFFFFFF);
            return f64::from_bits(bits);
        }

        let db_signif = u128_shift_right_round(self.signif as Float128, 11, self.sign_i32()) as u64;
        let mut final_signif = db_signif & 0x000FFFFFFFFFFFFF;
        let mut final_exp = new_exp as u64;
        if db_signif & (1u64 << 53) != 0 {
            final_signif >>= 1;
            final_exp += 1;
        }
        let bits = (sign as u64) << 63 | (final_exp << 52) | final_signif;
        f64::from_bits(bits)
    }

    pub fn round(&self) -> Self {
        if !self.is_supported() { return Self::nan(); }
        let bits_to_clear = 63 - unbias(self.exp());
        if bits_to_clear <= 0 { return *self; }
        let mut f = self.shift_right(bits_to_clear);
        if f.signif == 0 {
            // C: `f.exp = EXP_DENORMAL` - the sign bit survives.
            f.set_exp(EXP_DENORMAL);
        } else {
            f = f.normalize();
        }
        f
    }
}

fn u128_normalize_round(mut signif: Float128, mut exp: i32, sign: i32) -> Float80 {
    // NOTE: no special case for signif == 0. The C original does
    // `signif <<= shift` with shift == u128_clz(0) == 128, which on gcc/x86 is
    // a no-op (the shift count is masked), leaving signif == 0. wrapping_shl
    // reproduces exactly that, so the Rust port stays bit-identical to C even
    // for the degenerate encodings that come out of it (e.g. f80_scale of zero
    // by a large amount). See tests/differential.rs.
    let shift = u128_clz(signif);
    if exp - shift < unbias(EXP_MIN) {
        if exp > unbias(EXP_MIN) {
            signif = signif.wrapping_shl((exp - unbias(EXP_MIN)) as u32);
        } else {
            signif = u128_shift_right_round(signif, unbias(EXP_MIN) - exp, sign);
        }
        exp = unbias(EXP_DENORMAL);
    } else if exp - shift > unbias(EXP_MAX) {
        let mut f = if signif == (1u128 << 127) {
            Float80::inf()
        } else if ROUNDING_MODE.with(|rm| rm.get()) == RoundingMode::RoundChop ||
            (ROUNDING_MODE.with(|rm| rm.get()) == RoundingMode::RoundUp && sign != 0) ||
            (ROUNDING_MODE.with(|rm| rm.get()) == RoundingMode::RoundDown && sign == 0) {
            Float80 { signif: !0u64, sign_exp: EXP_MAX }
        } else {
            Float80::inf()
        };
        f.set_sign(sign != 0);
        return f;
    } else {
        signif = signif.wrapping_shl(shift as u32);
        exp -= shift;
    }

    let mut f = Float80 { signif: 0, sign_exp: bias(exp) };
    signif = u128_shift_right_round(signif, 64, sign);
    if signif >> 64 != 0 {
        signif >>= 1;
        f.set_exp(f.exp() + 1);
    }
    f.signif = signif as u64;
    f.set_sign(sign != 0);
    f
}

// NaN handling macro equivalent
fn handle_nans(a: Float80, b: Float80) -> Option<Float80> {
    if !a.is_supported() || !b.is_supported() { return Some(Float80::nan()); }
    if a.is_nan() && b.is_nan() && a.sign() && !b.sign() { return Some(b); }
    if a.is_nan() { return Some(a); }
    if b.is_nan() { return Some(b); }
    None
}

pub fn f80_add(a: Float80, b: Float80) -> Float80 {
    if let Some(nan) = handle_nans(a, b) { return nan; }

    let (mut a, mut b) = if a.exp() < b.exp() { (b, a) } else { (a, b) };

    let mut flipped = false;
    if a.sign() {
        a.sign_exp ^= 0x8000;
        b.sign_exp ^= 0x8000;
        flipped = true;
    }

    let a_signif = (a.signif as Float128) << 64;
    let mut b_signif = (b.signif as Float128) << 64;
    b_signif = u128_shift_right_round(b_signif, a.exp() as i32 - b.exp() as i32, b.sign_i32() ^ (flipped as i32));

    let mut sign = a.sign();
    let mut exp = unbias_denormal(a.exp());
    let mut signif: Float128;

    if !b.sign() {
        signif = a_signif.wrapping_add(b_signif);
        if !a.is_inf() {
            if let Some(sum) = a_signif.checked_add(b_signif) {
                signif = sum;
            } else {
                signif = u128_shift_right_round(signif, 1, sign as i32);
                signif |= 1u128 << 127;
                exp += 1;
            }
        } else {
            signif = a_signif;
        }
    } else {
        if a.is_inf() && b.is_inf() { return Float80::nan(); }

        if ROUNDING_MODE.with(|rm| rm.get()) == RoundingMode::RoundChop && b_signif == 0 && b.signif != 0 {
            b_signif = 1;
        }
        if a.is_inf() { b_signif = 0; }

        if a_signif >= b_signif {
            signif = a_signif - b_signif;
        } else {
            signif = b_signif - a_signif;
            sign = true;
        }

        if signif == 0 && a_signif != 0 && ROUNDING_MODE.with(|rm| rm.get()) == RoundingMode::RoundDown {
            let mut f = Float80 { signif: 0, sign_exp: 0 };
            f.set_sign(true);
            return f;
        }
        if signif == 0 { return Float80 { signif: 0, sign_exp: 0 }; }
    }

    let final_sign = if flipped { !sign } else { sign };
    u128_normalize_round(signif, exp, final_sign as i32)
}

pub fn f80_sub(a: Float80, b: Float80) -> Float80 {
    f80_add(a, b.neg())
}

pub fn f80_mul(a: Float80, b: Float80) -> Float80 {
    if let Some(nan) = handle_nans(a, b) { return nan; }

    if a.is_inf() || b.is_inf() {
        if a.is_zero() || b.is_zero() { return Float80::nan(); }
        let mut f = Float80::inf();
        f.set_sign(a.sign() ^ b.sign());
        return f;
    }

    let f_exp = unbias_denormal(a.exp()) + unbias_denormal(b.exp()) + 1;
    let f_signif = (a.signif as Float128) * (b.signif as Float128);
    let mut f = u128_normalize_round(f_signif, f_exp, a.sign_i32() ^ b.sign_i32());
    f.set_sign(a.sign() ^ b.sign());
    f
}

pub fn f80_div(a: Float80, b: Float80) -> Float80 {
    if let Some(nan) = handle_nans(a, b) { return nan; }

    let mut f: Float80;
    if a.is_inf() {
        if b.is_inf() { return Float80::nan(); }
        f = Float80::inf();
    } else if b.is_inf() {
        f = Float80 { signif: 0, sign_exp: 0 };
    } else if b.is_zero() {
        // C sets f = F80_NAN here and *falls through* to `f.sign = a.sign ^
        // b.sign`, so 0/0 keeps the xor'ed sign. Returning early would drop it.
        f = if a.is_zero() { Float80::nan() } else { Float80::inf() };
    } else {
        let b_trailing = b.signif.trailing_zeros() as i32;
        let b_norm = b.signif >> b_trailing as u32;
        let dividend = (a.signif as Float128) << 64;
        let mut signif = dividend / (b_norm as Float128);
        let remainder = dividend % (b_norm as Float128);

        let extra_bits;
        if signif != 0 {
            extra_bits = u128_clz(signif);
            signif <<= extra_bits as u32;
            signif |= ((remainder << extra_bits as u32) / (b_norm as Float128)) as Float128;
        } else {
            extra_bits = 0;
        }

        let f_exp = unbias_denormal(a.exp()) - unbias_denormal(b.exp()) + 63 - b_trailing - extra_bits;
        f = u128_normalize_round(signif, f_exp, a.sign_i32() ^ b.sign_i32());
    }

    f.set_sign(a.sign() ^ b.sign());
    f
}

pub fn f80_mod(x: Float80, y: Float80) -> Float80 {
    let quotient = f80_div(x, y);
    let old_mode = ROUNDING_MODE.with(|rm| rm.get());
    ROUNDING_MODE.with(|rm| rm.set(RoundingMode::RoundChop));
    let rounded = quotient.round();
    ROUNDING_MODE.with(|rm| rm.set(old_mode));
    f80_sub(x, f80_mul(rounded, y))
}

pub fn f80_lt(a: Float80, b: Float80) -> bool {
    if Float80::uncomparable(a, b) { return false; }
    if a.is_inf() && b.is_inf() && a.sign() == b.sign() { return false; }
    if a.is_zero() && b.is_zero() { return false; }
    let diff = f80_sub(a, b);
    diff.sign() && !diff.is_zero()
}

pub fn f80_eq(a: Float80, b: Float80) -> bool {
    if Float80::uncomparable(a, b) { return false; }
    let (mut a, mut b) = (a, b);
    // C clears the *sign* of a zero (`a.sign = 0`); the exponent is already 0.
    if a.is_zero() { a.set_sign(false); }
    if b.is_zero() { b.set_sign(false); }
    a.sign() == b.sign() && a.exp() == b.exp() && a.signif == b.signif
}

pub fn f80_lte(a: Float80, b: Float80) -> bool {
    f80_lt(a, b) || f80_eq(a, b)
}

pub fn f80_log2(x: Float80) -> Float80 {
    let zero = Float80::from_i64(0);
    let one = Float80::from_i64(1);
    let two = Float80::from_i64(2);

    if x.is_nan() || f80_lte(x, zero) { return Float80::nan(); }

    let mut x = x;
    let mut ipart: i64 = 0;
    while f80_lt(x, one) { ipart -= 1; x = f80_mul(x, two); }
    while f80_gt(x, two) { ipart += 1; x = f80_div(x, two); }

    let mut res = Float80::from_i64(ipart);
    let mut bit = one;

    while f80_gt(bit, zero) {
        while f80_lte(x, two) && f80_gt(bit, zero) {
            x = f80_mul(x, x);
            bit = f80_div(bit, two);
        }
        let oldres = res;
        res = f80_add(res, bit);
        if oldres.signif == res.signif && oldres.exp() == res.exp() && oldres.sign() == res.sign() {
            break;
        }
        x = f80_div(x, two);
    }
    res
}

pub fn f80_sqrt(x: Float80) -> Float80 {
    if x.is_zero() { return x; }
    if x.is_nan() || x.sign() { return Float80::nan(); }

    let mut guess = x;
    // preserve sign bit when halving exponent (x is non-negative here, but be safe)
    guess.set_exp(bias(unbias(guess.exp()) / 2));
    let two = Float80::from_i64(2);
    let mut old_guess;
    let mut i = 0;
    loop {
        old_guess = guess;
        guess = f80_div(f80_add(guess, f80_div(x, guess)), two);
        if f80_eq(guess, old_guess) || i >= 100 { break; }
        i += 1;
    }
    guess
}

pub fn f80_scale(x: Float80, scale: i32) -> Float80 {
    if !x.is_supported() || x.is_nan() { return Float80::nan(); }
    u128_normalize_round((x.signif as Float128) << 64, unbias(x.exp()) + scale, x.sign_i32())
}

pub fn f80_gt(a: Float80, b: Float80) -> bool {
    !f80_lte(a, b)
}

// ---------------------------------------------------------------------------
// Additions: the remaining pieces of emu/float80.{h,c} that the translation
// above does not cover, plus accessors needed to drive it from tests and from
// the (upcoming) fpu.rs port.
// ---------------------------------------------------------------------------

/// `f80_uncomparable` — true when the pair cannot be ordered (NaN or an
/// unsupported encoding).
pub fn f80_uncomparable(a: Float80, b: Float80) -> bool {
    Float80::uncomparable(a, b)
}

/// `void f80_xtract(float80 f, int *exp, float80 *signif)` — used to implement
/// the x87 `fxtract` instruction (see `emu/fpu.c:301`).
///
/// Returns the unbiased exponent and the significand as a float in [1, 2).
pub fn f80_xtract(f: Float80) -> (i32, Float80) {
    let exp = unbias(f.exp());
    let mut signif = f;
    signif.set_exp(bias(0));
    (exp, signif)
}

/// Read the `__thread enum f80_rounding_mode f80_rounding_mode` global.
pub fn rounding_mode() -> RoundingMode {
    ROUNDING_MODE.with(|rm| rm.get())
}

/// Set the `f80_rounding_mode` global.
pub fn set_rounding_mode(mode: RoundingMode) {
    ROUNDING_MODE.with(|rm| rm.set(mode))
}

/// Run `f` with `mode` installed, restoring the previous mode afterwards.
pub fn with_rounding_mode<T>(mode: RoundingMode, f: impl FnOnce() -> T) -> T {
    let old = rounding_mode();
    set_rounding_mode(mode);
    let result = f();
    set_rounding_mode(old);
    result
}

impl Float80 {
    /// The raw 80-bit encoding: 64-bit significand plus the packed
    /// sign/exponent word, exactly as stored in memory by x87.
    pub fn to_bits(self) -> (u64, u16) {
        (self.signif, self.sign_exp)
    }

    /// Rebuild from the raw 80-bit encoding (the inverse of [`Float80::to_bits`]).
    pub fn from_bits(signif: u64, sign_exp: u16) -> Self {
        Float80::new(signif, sign_exp)
    }

    /// Load from the 10 little-endian bytes x87 uses in memory.
    pub fn from_le_bytes(bytes: [u8; 10]) -> Self {
        let mut signif = 0u64;
        for (i, b) in bytes.iter().take(8).enumerate() {
            signif |= (*b as u64) << (8 * i);
        }
        let sign_exp = u16::from_le_bytes([bytes[8], bytes[9]]);
        Float80::new(signif, sign_exp)
    }

    /// Store as the 10 little-endian bytes x87 uses in memory.
    pub fn to_le_bytes(self) -> [u8; 10] {
        let mut out = [0u8; 10];
        out[..8].copy_from_slice(&self.signif.to_le_bytes());
        out[8..].copy_from_slice(&self.sign_exp.to_le_bytes());
        out
    }
}

impl fmt::Display for Float80 {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if self.is_nan() { return write!(f, "NaN"); }
        if self.is_inf() {
            return if self.sign() { write!(f, "-Inf") } else { write!(f, "Inf") };
        }
        if self.is_zero() { return write!(f, "0"); }
        let val = self.to_f64();
        write!(f, "{}", val)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_float80_basic() {
        let f = Float80::default();
        assert!(f.is_zero());
        let nan = Float80::nan();
        assert!(nan.is_nan());
        let inf = Float80::inf();
        assert!(inf.is_inf());
    }

    #[test]
    fn test_float80_from_i64() {
        let f = Float80::from_i64(42);
        assert_eq!(f.to_i64(), 42);
        let f = Float80::from_i64(-10);
        assert_eq!(f.to_i64(), -10);
        let f = Float80::from_i64(0);
        assert_eq!(f.to_i64(), 0);
    }

    #[test]
    fn test_float80_from_f64() {
        let f = Float80::from_f64(3.125);
        assert!((f.to_f64() - 3.125).abs() < 0.001);
        let f = Float80::from_f64(-2.5);
        assert!((f.to_f64() - (-2.5)).abs() < 0.001);
    }

    #[test]
    fn test_float80_ops() {
        let a = Float80::from_i64(10);
        let b = Float80::from_i64(5);
        assert_eq!(f80_add(a, b).to_i64(), 15);
        assert_eq!(f80_sub(a, b).to_i64(), 5);
        assert_eq!(f80_mul(a, b).to_i64(), 50);
        assert_eq!(f80_div(a, b).to_i64(), 2);
    }

    #[test]
    fn test_float80_neg_abs() {
        let f = Float80::from_i64(-42);
        assert!(f.sign());
        assert_eq!(f.abs().to_i64(), 42);
        assert_eq!(f.neg().to_i64(), 42);
    }

    #[test]
    fn test_float80_sqrt() {
        let f = Float80::from_i64(16);
        assert_eq!(f80_sqrt(f).to_i64(), 4);
    }

    #[test]
    fn test_float80_compare() {
        let a = Float80::from_i64(5);
        let b = Float80::from_i64(10);
        assert!(f80_lt(a, b));
        assert!(f80_eq(a, a));
        assert!(!f80_lt(b, a));
    }

    // ---------- regression tests: exact bit patterns ----------

    #[test]
    fn test_u128_clz_exact() {
        // catches u128-vs-u64 leading_zeros bug (was off by 64)
        assert_eq!(u128_clz(0), 128);
        assert_eq!(u128_clz(1), 127);
        assert_eq!(u128_clz(1u128 << 127), 0);
        assert_eq!(u128_clz(1u128 << 64), 63);
        assert_eq!(u128_clz(1u128 << 63), 64);
        assert_eq!(u128_clz(u128::MAX), 0);
        assert_eq!(u128_clz(0xFFu128 << 120), 0);
        assert_eq!(u128_clz(0x8000000000000000u128), 64);
    }

    #[test]
    fn test_from_i64_bit_exact() {
        // 1 = 1.0 * 2^0 -> signif 0x8000..., unbiased exp 0
        let one = Float80::from_i64(1);
        assert_eq!(one.signif, CURSED_BIT);
        assert_eq!(one.exp(), bias(0));
        assert!(!one.sign());
        assert!(one.is_supported());

        // -1 keeps sign, same magnitude bits
        let neg_one = Float80::from_i64(-1);
        assert_eq!(neg_one.signif, CURSED_BIT);
        assert_eq!(neg_one.exp(), bias(0));
        assert!(neg_one.sign());

        // 0 -> zero, no cursed bit
        let zero = Float80::from_i64(0);
        assert!(zero.is_zero());
        assert!(zero.is_supported());

        // power of two: 8 = 1.0 * 2^3
        let eight = Float80::from_i64(8);
        assert_eq!(eight.signif, CURSED_BIT);
        assert_eq!(eight.exp(), bias(3));

        // i64 extremes must round-trip exactly (f64 cannot do this!)
        assert_eq!(Float80::from_i64(i64::MAX).to_i64(), i64::MAX);
        assert_eq!(Float80::from_i64(i64::MIN).to_i64(), i64::MIN);
        assert_eq!(Float80::from_i64(i64::MIN + 1).to_i64(), i64::MIN + 1);
    }

    #[test]
    fn test_sign_preserved_through_exp_ops() {
        // every constructor/convert must preserve sign in packed sign_exp
        for v in [-0.0f64, -1.0, -2.5, -100.25, -1e100, -1e-100] {
            let f = Float80::from_f64(v);
            assert!(f.sign(), "from_f64({}) lost sign", v);
            assert!(f.is_supported());
            assert!((f.to_f64() - v).abs() <= v.abs() * 1e-12,
                "from_f64({}) roundtrip gave {}", v, f.to_f64());
        }
        for v in [0.0f64, 1.0, 2.5, 100.25, 1e100, 1e-100] {
            let f = Float80::from_f64(v);
            assert!(!f.sign(), "from_f64({}) gained sign", v);
        }
        // -0.0 keeps negative sign
        let neg_zero = Float80::from_f64(-0.0);
        assert!(neg_zero.sign());
        assert!(neg_zero.is_zero());
    }

    #[test]
    fn test_shift_preserves_sign() {
        // shift_left/right must not corrupt the packed sign bit.
        // NOTE: shift_left on a normalized value overflows the 64-bit
        // signif (same as C "may overflow"), so gain headroom with a
        // right shift first, then shift back left.
        let neg = Float80::from_i64(-8);
        assert!(neg.sign());
        let right = neg.shift_right(2);
        assert!(right.sign(), "shift_right lost sign");
        // NOTE: right-shifted intermediates are intentionally unnormalized
        // (no cursed bit, same as C) — check raw fields, not to_i64().
        assert_eq!(right.signif, 0x2000000000000000);
        assert_eq!(right.exp(), bias(5));
        let back = right.shift_left(2);
        assert!(back.sign(), "shift_left lost sign");
        assert_eq!(back.exp(), right.exp() - 2);
        assert!(back.is_supported());
        assert_eq!(back.to_i64(), -8);

        let pos = Float80::from_i64(8);
        let right = pos.shift_right(2);
        assert!(!right.sign(), "shift_right gained sign");
        let back = right.shift_left(2);
        assert!(!back.sign(), "shift_left gained sign");
        assert_eq!(back.to_i64(), 8);
    }

    #[test]
    fn test_sub_produces_negative() {
        // catches dead-code bug where negative-result sign was dropped
        let five = Float80::from_i64(5);
        let ten = Float80::from_i64(10);
        let neg_five = f80_sub(five, ten);
        assert!(neg_five.sign(), "5-10 should be negative");
        assert_eq!(neg_five.to_i64(), -5);

        // negative + negative
        let r = f80_add(Float80::from_i64(-3), Float80::from_i64(-4));
        assert_eq!(r.to_i64(), -7);
        // mixed signs
        assert_eq!(f80_add(Float80::from_i64(-10), Float80::from_i64(3)).to_i64(), -7);
        assert_eq!(f80_add(Float80::from_i64(10), Float80::from_i64(-3)).to_i64(), 7);
        // a - a = 0 (not nan, not tiny denormal)
        assert!(f80_sub(ten, ten).is_zero());
        // neg - pos = more negative
        assert_eq!(f80_sub(Float80::from_i64(-5), Float80::from_i64(5)).to_i64(), -10);
    }

    #[test]
    fn test_mul_div_signs() {
        let p = Float80::from_i64(6);
        let n = Float80::from_i64(-6);
        assert_eq!(f80_mul(p, p).to_i64(), 36);
        assert_eq!(f80_mul(n, n).to_i64(), 36);
        assert_eq!(f80_mul(p, n).to_i64(), -36);
        assert_eq!(f80_mul(n, p).to_i64(), -36);
        assert_eq!(f80_div(n, p).to_i64(), -1);
        assert_eq!(f80_div(n, Float80::from_i64(3)).to_i64(), -2);
        // mul by zero
        assert!(f80_mul(p, Float80::from_i64(0)).is_zero());
        // div by zero -> inf with correct sign
        let inf = f80_div(p, Float80::from_i64(0));
        assert!(inf.is_inf() && !inf.sign());
        let ninf = f80_div(n, Float80::from_i64(0));
        assert!(ninf.is_inf() && ninf.sign());
        // 0/0, inf/inf, inf*0 -> nan
        assert!(f80_div(Float80::from_i64(0), Float80::from_i64(0)).is_nan());
        assert!(f80_div(Float80::inf(), Float80::inf()).is_nan());
        assert!(f80_mul(Float80::inf(), Float80::from_i64(0)).is_nan());
        // inf arithmetic keeps sign (xor)
        let r = f80_mul(Float80::inf(), n);
        assert!(r.is_inf() && r.sign());
    }

    #[test]
    fn test_add_overflow_goes_to_inf() {
        // largest finite + itself must overflow to inf (tests exp increment on carry)
        let max = Float80 { signif: u64::MAX, sign_exp: EXP_MAX };
        assert!(max.is_supported());
        let r = f80_add(max, max);
        assert!(r.is_inf(), "max+max should overflow to inf, got {:?}", r);
        assert!(!r.sign());
        // negative overflow -> -inf
        let nmax = max.neg();
        let r = f80_add(nmax, nmax);
        assert!(r.is_inf() && r.sign());
    }

    #[test]
    fn test_80bit_precision_beyond_f64() {
        // f64 has 53-bit mantissa; f80 has 64-bit. This must NOT collapse to f64.
        // 2^53+1 is not exactly representable in f64 but is in f80.
        let big: i64 = (1i64 << 53) + 1;
        let f = Float80::from_i64(big);
        assert_eq!(f.to_i64(), big, "lost 80-bit precision on 2^53+1");
        // i64::MAX needs 63 bits — f64 rounds it, f80 must not
        let m = Float80::from_i64(i64::MAX);
        assert_eq!(m.to_i64(), i64::MAX);
        // tiny epsilon 2^-63 added to 1 must be visible in signif (f64 would drop it)
        let one = Float80::from_i64(1);
        let eps = f80_scale(Float80::from_i64(1), -63);
        assert!(eps.is_supported());
        let sum = f80_add(one, eps);
        assert!(sum.is_supported());
        assert_ne!(sum.signif, one.signif, "2^-63 epsilon was lost (64-bit truncation!)");
        assert!(f80_gt(sum, one));
        // ...but adding something below 2^-64 must round back to 1
        let tiny = f80_scale(Float80::from_i64(1), -65);
        let sum2 = f80_add(one, tiny);
        assert!(f80_eq(sum2, one));
    }

    #[test]
    fn test_f64_roundtrip_extensive() {
        let vals = [
            0.0, 1.0, -1.0, 0.5, -0.5, 2.5, -2.5, 3.125, -3.125,
            100.25, 1e10, -1e10, 1e-10, 1.7976931348623157e308,
            2.2250738585072014e-308, // smallest normal f64
            5e-324,                  // smallest denormal f64
        ];
        for &v in &vals {
            let f = Float80::from_f64(v);
            assert!(f.is_supported(), "from_f64({}) unsupported: {:?}", v, f);
            let back = f.to_f64();
            if v == 0.0 {
                assert_eq!(back, 0.0);
            } else {
                let rel = ((back - v) / v).abs();
                assert!(rel < 1e-12, "f64 roundtrip {} -> {} (rel err {})", v, back, rel);
            }
        }
        // special values
        assert!(Float80::from_f64(f64::INFINITY).is_inf());
        assert!(Float80::from_f64(f64::NEG_INFINITY).is_inf());
        assert!(Float80::from_f64(f64::NAN).is_nan());
        assert_eq!(Float80::from_f64(0.0).to_f64(), 0.0);
    }

    #[test]
    fn test_nan_inf_propagation() {
        let nan = Float80::nan();
        let inf = Float80::inf();
        let one = Float80::from_i64(1);
        assert!(f80_add(nan, one).is_nan());
        assert!(f80_add(one, nan).is_nan());
        assert!(f80_mul(nan, one).is_nan());
        assert!(f80_div(one, nan).is_nan());
        // inf - inf = nan (not zero!)
        assert!(f80_sub(inf, inf).is_nan());
        // inf + 1 = inf
        assert!(f80_add(inf, one).is_inf());
        // unsupported (bad cursed bit) -> nan, never panic
        let bad = Float80 { signif: 0x4000000000000000, sign_exp: bias(5) };
        assert!(!bad.is_supported());
        assert!(f80_add(bad, one).is_nan());
    }

    #[test]
    fn test_sqrt_multiple() {
        for (n, root) in [(0i64, 0i64), (1, 1), (4, 2), (9, 3), (16, 4), (25, 5), (100, 10)] {
            assert_eq!(f80_sqrt(Float80::from_i64(n)).to_i64(), root, "sqrt({})", n);
        }
        // sqrt(2) is irrational: check f64 approximation, and squaring returns ~2
        let two = Float80::from_i64(2);
        let s2 = f80_sqrt(two);
        assert!((s2.to_f64() - std::f64::consts::SQRT_2).abs() < 1e-12);
        assert_eq!(f80_mul(s2, s2).round().to_i64(), 2);
        // sqrt of negative / nan -> nan; sqrt(0)=0
        assert!(f80_sqrt(Float80::from_i64(-4)).is_nan());
        assert!(f80_sqrt(Float80::nan()).is_nan());
        assert!(f80_sqrt(Float80::from_i64(0)).is_zero());
        // NOTE: sqrt(inf) yields nan in this implementation (matches the
        // original C Newton-iteration behavior: inf/inf -> nan on the
        // second iteration). x87 hardware would return +inf instead.
        assert!(f80_sqrt(Float80::inf()).is_nan());
    }

    #[test]
    fn test_comparisons_full() {
        let neg = Float80::from_i64(-5);
        let pos = Float80::from_i64(5);
        assert!(f80_lt(neg, pos));
        assert!(!f80_lt(pos, neg));
        assert!(!f80_eq(neg, pos));
        // +0 == -0
        let pz = Float80::from_f64(0.0);
        let nz = Float80::from_f64(-0.0);
        assert!(f80_eq(pz, nz));
        assert!(!f80_lt(pz, nz) && !f80_lt(nz, pz));
        // infinities
        assert!(f80_lt(neg, Float80::inf()));
        assert!(f80_lt(Float80::inf().neg(), pos));
        assert!(!f80_lt(Float80::inf(), Float80::inf()));
        assert!(f80_eq(Float80::inf(), Float80::inf()));
        // nan is uncomparable with everything
        let nan = Float80::nan();
        assert!(!f80_lt(nan, pos) && !f80_lt(pos, nan));
        assert!(!f80_eq(nan, nan));
        assert!(!f80_eq(nan, pos));
    }

    #[test]
    fn test_supported_invariant_after_ops() {
        // every op on finite supported inputs must yield supported output (or nan/inf/zero)
        let vals = [
            Float80::from_i64(0), Float80::from_i64(1), Float80::from_i64(-1),
            Float80::from_i64(123456789), Float80::from_i64(-987654321),
            Float80::from_f64(0.1), Float80::from_f64(-99.99),
        ];
        for &a in &vals {
            for &b in &vals {
                for r in [f80_add(a, b), f80_sub(a, b), f80_mul(a, b)] {
                    assert!(r.is_supported() || r.is_nan() || r.is_inf(),
                        "unsupported result: {:?} op {:?}", a, b);
                }
                // div may hit 0/0 -> nan, still must not be *unsupported garbage*
                let d = f80_div(a, b);
                assert!(d.is_supported() || d.is_nan() || d.is_inf(),
                    "unsupported div result: {:?} / {:?}", a, b);
            }
            assert!(a.neg().is_supported() || a.is_zero());
            assert!(a.abs().is_supported() || a.is_zero());
        }
    }

    #[test]
    fn test_div_exact_and_mod() {
        assert_eq!(f80_div(Float80::from_i64(7), Float80::from_i64(2)).to_f64(), 3.5);
        assert_eq!(f80_div(Float80::from_i64(1), Float80::from_i64(4)).to_f64(), 0.25);
        // mod: 7 % 3 = 1, -7 % 3 = -1 (chop rounding like x87 FPREM-ish)
        assert_eq!(f80_mod(Float80::from_i64(7), Float80::from_i64(3)).to_i64(), 1);
        assert_eq!(f80_mod(Float80::from_i64(-7), Float80::from_i64(3)).to_i64(), -1);
    }

    #[test]
    fn test_xtract_and_bits_roundtrip() {
        // f80_xtract: 12 = 1.5 * 2^3
        let (exp, signif) = f80_xtract(Float80::from_i64(12));
        assert_eq!(exp, 3);
        assert_eq!(signif.to_f64(), 1.5);
        // negative keeps its sign through xtract
        let (exp, signif) = f80_xtract(Float80::from_i64(-12));
        assert_eq!(exp, 3);
        assert!(signif.sign());

        // raw 10-byte memory encoding round-trips
        let f = Float80::from_f64(-1234.5678);
        assert_eq!(Float80::from_le_bytes(f.to_le_bytes()), f);
        assert_eq!(Float80::from_bits(f.signif, f.sign_exp), f);
    }

    #[test]
    fn test_rounding_mode_is_thread_local_and_restored() {
        assert_eq!(rounding_mode(), RoundingMode::RoundToNearest);
        let r = with_rounding_mode(RoundingMode::RoundChop, || {
            assert_eq!(rounding_mode(), RoundingMode::RoundChop);
            // 1/3 chopped differs from 1/3 rounded-to-nearest
            f80_div(Float80::from_i64(1), Float80::from_i64(3))
        });
        assert_eq!(rounding_mode(), RoundingMode::RoundToNearest);
        let nearest = f80_div(Float80::from_i64(1), Float80::from_i64(3));
        assert_ne!(r.signif, nearest.signif, "chop and nearest must differ for 1/3");
    }
}
