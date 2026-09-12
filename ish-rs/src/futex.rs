//! `kernel/futex.h` + `kernel/futex.c` — futex constants and hash table.

pub const FUTEX_WAIT: u32 = 0;
pub const FUTEX_WAKE: u32 = 1;
pub const FUTEX_FD: u32 = 2;
pub const FUTEX_REQUEUE: u32 = 3;
pub const FUTEX_CMP_REQUEUE: u32 = 4;
pub const FUTEX_WAKE_OP: u32 = 5;
pub const FUTEX_LOCK_PI: u32 = 6;
pub const FUTEX_UNLOCK_PI: u32 = 7;
pub const FUTEX_TRYLOCK_PI: u32 = 8;
pub const FUTEX_WAIT_BITSET: u32 = 9;
pub const FUTEX_WAKE_BITSET: u32 = 10;
pub const FUTEX_WAIT_REQUEUE_PI: u32 = 11;
pub const FUTEX_CMP_REQUEUE_PI: u32 = 12;

pub const FUTEX_PRIVATE_FLAG: u32 = 128;
pub const FUTEX_CLOCK_REALTIME: u32 = 256;

pub const FUTEX_TID_MASK: u32 = 0x3fffffff;
pub const FUTEX_WAITERS: u32 = 0x80000000;
pub const FUTEX_OWNER_DIED: u32 = 0x40000000;

pub const FUTEX_OP_SET: u32 = 0;
pub const FUTEX_OP_ADD: u32 = 1;
pub const FUTEX_OP_OR: u32 = 2;
pub const FUTEX_OP_ANDN: u32 = 3;
pub const FUTEX_OP_XOR: u32 = 4;

pub const FUTEX_OP_CMP_EQ: u32 = 0;
pub const FUTEX_OP_CMP_NE: u32 = 1;
pub const FUTEX_OP_CMP_LT: u32 = 2;
pub const FUTEX_OP_CMP_LE: u32 = 3;
pub const FUTEX_OP_CMP_GT: u32 = 4;
pub const FUTEX_OP_CMP_GE: u32 = 5;

/// Futex hash table size, matching C's FUTEX_BUCKETS
pub const FUTEX_BUCKETS: usize = 256;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct FutexKey {
    pub addr: u32,
    pub is_private: bool,
    pub pid: u32,
}

impl FutexKey {
    pub fn new(addr: u32, is_private: bool, pid: u32) -> Self {
        Self { addr, is_private, pid }
    }
    pub fn hash(&self) -> usize {
        let mut h = self.addr as usize;
        if !self.is_private {
            h ^= self.pid as usize;
        }
        h % FUTEX_BUCKETS
    }
}

pub fn futex_op(op: u32) -> u32 { op & 0xf }
pub fn futex_cmp(op: u32) -> u32 { (op >> 4) & 0xf }
pub fn futex_op_arg(op: u32) -> u32 { (op >> 8) & 0xfff }
pub fn futex_cmp_arg(op: u32) -> u32 { (op >> 20) & 0xfff }

pub fn futex_op_apply(op: u32, oparg: u32, oldval: u32) -> u32 {
    match op {
        FUTEX_OP_SET => oparg,
        FUTEX_OP_ADD => oldval.wrapping_add(oparg),
        FUTEX_OP_OR => oldval | oparg,
        FUTEX_OP_ANDN => oldval & !oparg,
        FUTEX_OP_XOR => oldval ^ oparg,
        _ => oldval,
    }
}

pub fn futex_cmp_check(cmp: u32, cmparg: u32, uval: u32) -> bool {
    match cmp {
        FUTEX_OP_CMP_EQ => uval == cmparg,
        FUTEX_OP_CMP_NE => uval != cmparg,
        FUTEX_OP_CMP_LT => (uval as i32) < (cmparg as i32),
        FUTEX_OP_CMP_LE => (uval as i32) <= (cmparg as i32),
        FUTEX_OP_CMP_GT => (uval as i32) > (cmparg as i32),
        FUTEX_OP_CMP_GE => (uval as i32) >= (cmparg as i32),
        _ => false,
    }
}

#[derive(Debug)]
pub struct FutexWaiter {
    pub key: FutexKey,
    pub bitset: u32,
}

impl FutexWaiter {
    pub fn new(key: FutexKey, bitset: u32) -> Self { Self { key, bitset } }
    pub fn matches(&self, key: &FutexKey, bitset: u32) -> bool {
        self.key == *key && (self.bitset & bitset) != 0
    }
}

#[derive(Debug, Default)]
pub struct FutexTable {
    pub waiters: Vec<FutexWaiter>,
}

impl FutexTable {
    pub fn new() -> Self { Self::default() }
    pub fn add_waiter(&mut self, waiter: FutexWaiter) { self.waiters.push(waiter); }
    pub fn wake(&mut self, key: &FutexKey, bitset: u32, count: usize) -> usize {
        let mut woken = 0;
        self.waiters.retain(|w| {
            if woken < count && w.matches(key, bitset) {
                woken += 1;
                false
            } else {
                true
            }
        });
        woken
    }
    pub fn requeue(&mut self, from: &FutexKey, to: &FutexKey, count: usize) -> usize {
        let mut requeued = 0;
        for waiter in self.waiters.iter_mut() {
            if waiter.key == *from && requeued < count {
                waiter.key = *to;
                requeued += 1;
            }
        }
        requeued
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn futex_constants_match_c() {
        assert_eq!(FUTEX_WAIT, 0);
        assert_eq!(FUTEX_WAKE, 1);
        assert_eq!(FUTEX_PRIVATE_FLAG, 128);
        assert_eq!(FUTEX_BUCKETS, 256);
    }

    #[test]
    fn futex_key_hash() {
        let k1 = FutexKey::new(0x1000, true, 1);
        let k1b = FutexKey::new(0x1000, true, 2);
        assert_eq!(k1.hash(), k1b.hash(), "private ignores pid");
        let k2 = FutexKey::new(0x1000, false, 1);
        let k3 = FutexKey::new(0x1000, false, 2);
        assert_ne!(k2.hash(), k3.hash(), "non-private differs by pid");
        assert!(k1.hash() < FUTEX_BUCKETS);
    }

    #[test]
    fn futex_op_decode() {
        let op = FUTEX_OP_ADD | (FUTEX_OP_CMP_EQ << 4) | (10 << 8) | (20 << 20);
        assert_eq!(futex_op(op), FUTEX_OP_ADD);
        assert_eq!(futex_cmp(op), FUTEX_OP_CMP_EQ);
        assert_eq!(futex_op_arg(op), 10);
        assert_eq!(futex_cmp_arg(op), 20);
    }

    #[test]
    fn futex_op_apply_test() {
        assert_eq!(futex_op_apply(FUTEX_OP_SET, 5, 10), 5);
        assert_eq!(futex_op_apply(FUTEX_OP_ADD, 5, 10), 15);
        assert_eq!(futex_op_apply(FUTEX_OP_OR, 0b01, 0b10), 0b11);
        assert_eq!(futex_op_apply(FUTEX_OP_ANDN, 0b01, 0b11), 0b10);
        assert_eq!(futex_op_apply(FUTEX_OP_XOR, 0b11, 0b01), 0b10);
    }

    #[test]
    fn futex_cmp_check_test() {
        assert!(futex_cmp_check(FUTEX_OP_CMP_EQ, 5, 5));
        assert!(!futex_cmp_check(FUTEX_OP_CMP_EQ, 5, 6));
        assert!(futex_cmp_check(FUTEX_OP_CMP_LT, 10, 5));
        assert!(futex_cmp_check(FUTEX_OP_CMP_GT, 5, 10));
    }

    #[test]
    fn futex_table_wake_and_requeue() {
        let mut table = FutexTable::new();
        let k1 = FutexKey::new(0x1000, true, 1);
        let k2 = FutexKey::new(0x2000, true, 1);
        table.add_waiter(FutexWaiter::new(k1, 0xffffffff));
        table.add_waiter(FutexWaiter::new(k1, 0xffffffff));
        table.add_waiter(FutexWaiter::new(k2, 0xffffffff));
        assert_eq!(table.waiters.len(), 3);
        assert_eq!(table.wake(&k1, 0xffffffff, 1), 1);
        assert_eq!(table.waiters.len(), 2);
        assert_eq!(table.requeue(&k1, &k2, 1), 1);
        assert_eq!(table.waiters.iter().filter(|w| w.key == k2).count(), 2);
    }
}
