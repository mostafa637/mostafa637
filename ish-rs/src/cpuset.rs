//! `kernel/cpuset.c` — cpuset helpers.

pub const CPU_SET_SIZE: usize = 128;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CpuSet {
    pub bits: [u8; CPU_SET_SIZE],
}

impl Default for CpuSet {
    fn default() -> Self { Self { bits: [0; CPU_SET_SIZE] } }
}

impl CpuSet {
    pub fn new() -> Self { Self::default() }

    pub fn set(&mut self, cpu: usize) {
        if cpu < CPU_SET_SIZE * 8 {
            self.bits[cpu / 8] |= 1 << (cpu % 8);
        }
    }

    pub fn clear(&mut self, cpu: usize) {
        if cpu < CPU_SET_SIZE * 8 {
            self.bits[cpu / 8] &= !(1 << (cpu % 8));
        }
    }

    pub fn is_set(&self, cpu: usize) -> bool {
        if cpu >= CPU_SET_SIZE * 8 { return false; }
        (self.bits[cpu / 8] & (1 << (cpu % 8))) != 0
    }

    pub fn count(&self) -> usize {
        self.bits.iter().map(|b| b.count_ones() as usize).sum()
    }

    pub fn zero(&mut self) { self.bits = [0; CPU_SET_SIZE]; }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn cpuset_basic() {
        let mut set = CpuSet::new();
        assert_eq!(set.count(), 0);
        set.set(0);
        set.set(1);
        set.set(63);
        assert!(set.is_set(0));
        assert!(set.is_set(1));
        assert!(set.is_set(63));
        assert!(!set.is_set(2));
        assert_eq!(set.count(), 3);
        set.clear(1);
        assert!(!set.is_set(1));
        assert_eq!(set.count(), 2);
        set.zero();
        assert_eq!(set.count(), 0);
    }
}
