// Line-by-line translation of emu/cpuid.h
//
// iSH deliberately reports a very small feature set: an emulated CPU that
// claims more than the emulator implements is a bug waiting to happen.

use crate::cpu::DwordT;

/// `do_cpuid` — leaf 0 is the vendor string, everything else falls through to
/// leaf 1. The C's `default:` arm shares leaf 1's body, so an unsupported leaf
/// returns the same thing as leaf 1 rather than an error.
pub fn do_cpuid(eax: &mut DwordT, ebx: &mut DwordT, ecx: &mut DwordT, edx: &mut DwordT) {
    let leaf = *eax;
    match leaf {
        0 => {
            *eax = 0x01; // "we support barely anything"
            *ebx = 0x756e_6547; // Genu
            *edx = 0x4965_6e69; // ineI
            *ecx = 0x6c65_746e; // ntel
        }
        _ => {
            *eax = 0x0; // say nothing about cpu model number
            *ebx = 0x0; // processor number 0, flushes 0 bytes on clflush
            *ecx = 0; // we support none of the features in ecx
            *edx = (1 << 0)  // fpu
                | (1 << 15) // cmov
                | (1 << 23) // mmx
                | (1 << 26); // sse2
        }
    }
}

/// The `edx` feature bits leaf 1 reports, named for the tests.
pub const CPUID_EDX_FPU: DwordT = 1 << 0;
pub const CPUID_EDX_CMOV: DwordT = 1 << 15;
pub const CPUID_EDX_MMX: DwordT = 1 << 23;
pub const CPUID_EDX_SSE2: DwordT = 1 << 26;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn leaf_zero_is_the_intel_vendor_string() {
        let (mut a, mut b, mut c, mut d) = (0, 0, 0, 0);
        do_cpuid(&mut a, &mut b, &mut c, &mut d);
        assert_eq!(a, 1, "the highest supported leaf");
        let vendor: [u8; 12] = {
            let mut v = [0u8; 12];
            v[0..4].copy_from_slice(&b.to_le_bytes());
            v[4..8].copy_from_slice(&d.to_le_bytes());
            v[8..12].copy_from_slice(&c.to_le_bytes());
            v
        };
        assert_eq!(&vendor, b"GenuineIntel");
    }

    #[test]
    fn leaf_one_reports_only_what_is_implemented() {
        let (mut a, mut b, mut c, mut d) = (1, 0, 0, 0);
        do_cpuid(&mut a, &mut b, &mut c, &mut d);
        assert_eq!(a, 0);
        assert_eq!(b, 0);
        assert_eq!(c, 0, "none of the ecx features");
        assert_eq!(d, CPUID_EDX_FPU | CPUID_EDX_CMOV | CPUID_EDX_MMX | CPUID_EDX_SSE2);
        assert_eq!(d & (1 << 25), 0, "sse (bit 25) is not claimed, only sse2");
    }

    #[test]
    fn an_unsupported_leaf_falls_through_to_leaf_one() {
        let (mut a, mut b, mut c, mut d) = (0x8000_0000, 0, 0, 0);
        do_cpuid(&mut a, &mut b, &mut c, &mut d);
        let (mut a1, mut b1, mut c1, mut d1) = (1, 0, 0, 0);
        do_cpuid(&mut a1, &mut b1, &mut c1, &mut d1);
        assert_eq!((a, b, c, d), (a1, b1, c1, d1));
    }
}
