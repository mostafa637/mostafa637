//! `fs/dev.h` — device number encoding and ops.
//!
//! The C header defines `dev_t_` as `uint32_t` with encoding mmmMMMmm.
//! This port provides `dev_make`, `dev_major`, `dev_minor` matching C exactly,
//! plus constants for block/char device types.

/// `dev_t_` — fake device number.
pub type DevT = u32;

/// `DEV_BLOCK` and `DEV_CHAR`
pub const DEV_BLOCK: u32 = 0;
pub const DEV_CHAR: u32 = 1;

/// `dev_make` — ((minor & 0xfff00) << 12) | (major << 8) | (minor & 0xff)
pub fn dev_make(major: u32, minor: u32) -> DevT {
    ((minor & 0xfff00) << 12) | (major << 8) | (minor & 0xff)
}

/// `dev_major` — (dev & 0xfff00) >> 8
pub fn dev_major(dev: DevT) -> u32 {
    (dev & 0xfff00) >> 8
}

/// `dev_minor` — ((dev & 0xfff00000) >> 12) | (dev & 0xff)
pub fn dev_minor(dev: DevT) -> u32 {
    ((dev & 0xfff00000) >> 12) | (dev & 0xff)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dev_encoding_matches_c() {
        // Test from C logic
        let dev = dev_make(1, 5);
        assert_eq!(dev_major(dev), 1);
        assert_eq!(dev_minor(dev), 5);

        let dev2 = dev_make(240, 0);
        assert_eq!(dev_major(dev2), 240);
        assert_eq!(dev_minor(dev2), 0);

        // Test with high minor bits
        let dev3 = dev_make(4, 0x123);
        assert_eq!(dev_major(dev3), 4);
        assert_eq!(dev_minor(dev3), 0x123);

        // Test encoding: minor 0xfff00 has high bits
        let dev4 = dev_make(1, 0xfff00);
        // minor & 0xfff00 = 0xfff00, <<12 = 0xfff00000, plus major<<8
        assert_eq!(dev_major(dev4), 1);
        // dev_minor should recover high bits + low 0
        assert_eq!(dev_minor(dev4), 0xfff00);
    }

    #[test]
    fn dev_roundtrip() {
        for major in [1, 4, 5, 128, 136, 240] {
            for minor in [0, 1, 3, 5, 7, 8, 9, 0x100, 0xfff] {
                let dev = dev_make(major, minor & 0xfffff);
                assert_eq!(dev_major(dev), major);
                assert_eq!(dev_minor(dev), minor & 0xfffff);
            }
        }
    }
}
