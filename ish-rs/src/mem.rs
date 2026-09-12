//! `fs/mem.c` — memory device (null, zero, full, random) constants and logic.

/// Mem device minor numbers (also in devices.rs, re-exported for fs/mem.h fidelity)
pub use crate::devices::{DEV_FULL_MINOR, DEV_NULL_MINOR, DEV_RANDOM_MINOR, DEV_URANDOM_MINOR, DEV_ZERO_MINOR, MEM_MAJOR};

/// Null device: reads return 0 bytes, writes succeed.
pub fn null_read(_buf: &mut [u8]) -> usize {
    0
}

pub fn null_write(buf: &[u8]) -> usize {
    buf.len()
}

/// Zero device: reads return zeroes.
pub fn zero_read(buf: &mut [u8]) -> usize {
    for b in buf.iter_mut() {
        *b = 0;
    }
    buf.len()
}

/// Full device: writes always fail with ENOSPC.
pub fn full_write(_buf: &[u8]) -> Result<usize, i32> {
    Err(-28) // ENOSPC
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn mem_device_logic() {
        let mut buf = [1u8; 4];
        assert_eq!(null_read(&mut buf), 0);
        assert_eq!(null_write(b"abc"), 3);
        assert_eq!(zero_read(&mut buf), 4);
        assert_eq!(buf, [0; 4]);
        assert_eq!(full_write(b"abc"), Err(-28));
    }
}
