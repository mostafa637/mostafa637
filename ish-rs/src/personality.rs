//! `kernel/personality.h` — execution domain and personality flags.
//!
//! iSH only exposes `ADDR_NO_RANDOMIZE`. The original header is 6 lines; the
//! Rust port keeps the same constant and the same numeric value so that
//! `sys_personality` and `task.rs` agree on the ABI.

/// `ADDR_NO_RANDOMIZE_` from `kernel/personality.h`.
///
/// The guest ABI value is `0x0040000`. `task.rs` historically redefined this
/// same value; this module is now the single source of truth.
pub const ADDR_NO_RANDOMIZE: u32 = 0x0004_0000;

/// `PER_LINUX` and friends are defined by Linux's personality(2) but iSH's
/// `getset.c` only checks for `ADDR_NO_RANDOMIZE` and `-1` (query). They are
/// listed here for completeness and future use.
pub const PER_LINUX: u32 = 0;
pub const PER_LINUX_32BIT: u32 = 0x0008;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn constant_matches_c_header() {
        assert_eq!(ADDR_NO_RANDOMIZE, 0x0040000);
    }
}
