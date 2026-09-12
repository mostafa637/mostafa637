//! `util/bits.h` — bitset helpers.
//!
//! The C header defines `bits_t` as `void` and three inline helpers that
//! operate on a byte array. This port provides the same operations over
//! `&[u8]` / `&mut [u8]` and a safe `Bits` wrapper used by higher-level
//! kernel code (e.g. CPU affinity bitmaps in `resource.rs`).

/// Number of bytes required to store `bits` bits, i.e. `BITS_SIZE(bits)`.
#[inline]
pub fn bits_size(bits: usize) -> usize {
    if bits == 0 {
        0
    } else {
        (bits - 1) / 8 + 1
    }
}

/// `bit_test(i, data)` — return whether bit `i` is set.
#[inline]
pub fn bit_test(i: usize, data: &[u8]) -> bool {
    let byte = i >> 3;
    let bit = i & 7;
    if byte >= data.len() {
        false
    } else {
        (data[byte] & (1 << bit)) != 0
    }
}

/// `bit_set(i, data)` — set bit `i`.
#[inline]
pub fn bit_set(i: usize, data: &mut [u8]) {
    let byte = i >> 3;
    let bit = i & 7;
    if byte < data.len() {
        data[byte] |= 1 << bit;
    }
}

/// `bit_clear(i, data)` — clear bit `i`.
#[inline]
pub fn bit_clear(i: usize, data: &mut [u8]) {
    let byte = i >> 3;
    let bit = i & 7;
    if byte < data.len() {
        data[byte] &= !(1 << bit);
    }
}

/// Owned bitset, convenience wrapper around `Vec<u8>` sized by `bits_size`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Bits {
    data: Vec<u8>,
    bits: usize,
}

impl Bits {
    /// Create a zeroed bitset capable of holding `bits` bits.
    pub fn new(bits: usize) -> Self {
        Self {
            data: vec![0u8; bits_size(bits)],
            bits,
        }
    }

    /// Raw byte view, matching C's `bits_t *`.
    pub fn as_bytes(&self) -> &[u8] {
        &self.data
    }

    /// Mutable raw byte view.
    pub fn as_bytes_mut(&mut self) -> &mut [u8] {
        &mut self.data
    }

    /// Test bit `i`.
    pub fn test(&self, i: usize) -> bool {
        assert!(i < self.bits);
        bit_test(i, &self.data)
    }

    /// Set bit `i`.
    pub fn set(&mut self, i: usize) {
        assert!(i < self.bits);
        bit_set(i, &mut self.data);
    }

    /// Clear bit `i`.
    pub fn clear(&mut self, i: usize) {
        assert!(i < self.bits);
        bit_clear(i, &mut self.data);
    }

    /// Number of bits this set was created for.
    pub fn len(&self) -> usize {
        self.bits
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bits_size_matches_c_macro() {
        assert_eq!(bits_size(0), 0);
        assert_eq!(bits_size(1), 1);
        assert_eq!(bits_size(8), 1);
        assert_eq!(bits_size(9), 2);
        assert_eq!(bits_size(16), 2);
    }

    #[test]
    fn set_test_clear_roundtrip() {
        let mut bits = Bits::new(20);
        assert!(!bits.test(0));
        bits.set(0);
        assert!(bits.test(0));
        bits.set(9);
        assert!(bits.test(9));
        bits.clear(0);
        assert!(!bits.test(0));
        assert!(bits.test(9));
        bits.set(8);
        assert!(bits.test(8));
        bits.clear(8);
        assert!(!bits.test(8));
    }

    #[test]
    fn bit_helpers_work_on_raw_slices() {
        let mut data = [0u8; 2];
        bit_set(0, &mut data);
        bit_set(9, &mut data);
        assert!(bit_test(0, &data));
        assert!(bit_test(9, &data));
        assert!(!bit_test(1, &data));
        bit_clear(0, &mut data);
        assert!(!bit_test(0, &data));
    }
}
