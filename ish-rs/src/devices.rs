//! `fs/devices.h` — device major/minor constants.
//!
//! Leaf header, no dependencies.

pub const MEM_MAJOR: u32 = 1;
pub const DEV_NULL_MINOR: u32 = 3;
pub const DEV_ZERO_MINOR: u32 = 5;
pub const DEV_FULL_MINOR: u32 = 7;
pub const DEV_RANDOM_MINOR: u32 = 8;
pub const DEV_URANDOM_MINOR: u32 = 9;

pub const TTY_CONSOLE_MAJOR: u32 = 4;

pub const TTY_ALTERNATE_MAJOR: u32 = 5;
pub const DEV_TTY_MINOR: u32 = 0;
pub const DEV_CONSOLE_MINOR: u32 = 1;
pub const DEV_PTMX_MINOR: u32 = 2;

pub const TTY_PSEUDO_MASTER_MAJOR: u32 = 128;
pub const TTY_PSEUDO_SLAVE_MAJOR: u32 = 136;

pub const DYN_DEV_MAJOR: u32 = 240;
pub const DEV_CLIPBOARD_MINOR: u32 = 0;
pub const DEV_LOCATION_MINOR: u32 = 1;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn constants_match_c() {
        assert_eq!(MEM_MAJOR, 1);
        assert_eq!(DEV_NULL_MINOR, 3);
        assert_eq!(DEV_ZERO_MINOR, 5);
        assert_eq!(TTY_CONSOLE_MAJOR, 4);
        assert_eq!(DYN_DEV_MAJOR, 240);
    }
}
