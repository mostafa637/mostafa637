//! `fs/mem.c` + `kernel/mem.c` — memory device and mem constants.

pub use crate::devices::{DEV_FULL_MINOR, DEV_NULL_MINOR, DEV_RANDOM_MINOR, DEV_URANDOM_MINOR, DEV_ZERO_MINOR, MEM_MAJOR};

pub fn null_read(_buf: &mut [u8]) -> usize { 0 }
pub fn null_write(buf: &[u8]) -> usize { buf.len() }
pub fn zero_read(buf: &mut [u8]) -> usize {
    for b in buf.iter_mut() { *b = 0; }
    buf.len()
}
pub fn full_write(_buf: &[u8]) -> Result<usize, i32> { Err(-28) }

pub fn full_read(buf: &mut [u8]) -> usize {
    for b in buf.iter_mut() { *b = 0; }
    buf.len()
}

/// Random device, simplified
pub fn random_read(buf: &mut [u8]) -> usize {
    // In C, reads from host random; here fill with pseudo-random
    for (i, b) in buf.iter_mut().enumerate() {
        *b = (i as u8).wrapping_mul(37);
    }
    buf.len()
}

/// Memory info, matching C's meminfo
#[derive(Debug, Default, Clone)]
pub struct MemInfo {
    pub total: u64,
    pub free: u64,
    pub available: u64,
    pub buffers: u64,
    pub cached: u64,
}

impl MemInfo {
    pub fn new(total: u64) -> Self { Self { total, free: total, available: total, ..Default::default() } }
    pub fn allocate(&mut self, size: u64) -> Result<(), i32> {
        if size > self.free { return Err(-12); } // ENOMEM
        self.free -= size;
        Ok(())
    }
    pub fn free_mem(&mut self, size: u64) { self.free += size; if self.free > self.total { self.free = self.total; } }
    pub fn used(&self) -> u64 { self.total - self.free }
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
        assert_eq!(full_read(&mut buf), 4);
        assert_eq!(buf, [0; 4]);
    }

    #[test]
    fn random_read_test() {
        let mut buf = [0u8; 10];
        assert_eq!(random_read(&mut buf), 10);
    }

    #[test]
    fn meminfo_alloc_free() {
        let mut info = MemInfo::new(1024);
        assert_eq!(info.used(), 0);
        assert!(info.allocate(512).is_ok());
        assert_eq!(info.used(), 512);
        assert!(info.allocate(600).is_err());
        info.free_mem(512);
        assert_eq!(info.used(), 0);
    }
}
