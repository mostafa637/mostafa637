//! `kernel/futex.h` + `kernel/futex.c` — futex constants and queue types.

/// `FUTEX_WAIT_`
pub const FUTEX_WAIT: u32 = 0;
/// `FUTEX_WAKE_`
pub const FUTEX_WAKE: u32 = 1;
/// `FUTEX_REQUEUE_`
pub const FUTEX_REQUEUE: u32 = 3;
/// `FUTEX_PRIVATE_FLAG_`
pub const FUTEX_PRIVATE_FLAG: u32 = 128;
/// `FUTEX_CMD_MASK_`
pub const FUTEX_CMD_MASK: u32 = !FUTEX_PRIVATE_FLAG;

pub const FUTEX_HASH_BITS: usize = 12;
pub const FUTEX_HASH_SIZE: usize = 1 << FUTEX_HASH_BITS;

#[derive(Debug)]
pub struct FutexWait {
    pub uaddr: u32,
    pub val: u32,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FutexOp {
    Wait,
    Wake,
    Requeue,
    Unknown(u32),
}

impl FutexOp {
    pub fn from_op(op: u32) -> Self {
        match op & FUTEX_CMD_MASK {
            FUTEX_WAIT => Self::Wait,
            FUTEX_WAKE => Self::Wake,
            FUTEX_REQUEUE => Self::Requeue,
            other => Self::Unknown(other),
        }
    }
    pub fn is_private(op: u32) -> bool {
        op & FUTEX_PRIVATE_FLAG != 0
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct RobustListHead {
    pub list: u32,
    pub offset: u32,
    pub list_op_pending: u32,
}

/// Futex table entry, matching C's `struct futex`
#[derive(Debug)]
pub struct Futex {
    pub addr: u32,
    pub refcount: usize,
    pub queue: Vec<FutexWait>,
}

impl Futex {
    pub fn new(addr: u32) -> Self {
        Self {
            addr,
            refcount: 1,
            queue: Vec::new(),
        }
    }

    pub fn retain(&mut self) {
        self.refcount += 1;
    }

    pub fn release(&mut self) -> bool {
        if self.refcount == 0 {
            return true;
        }
        self.refcount -= 1;
        self.refcount == 0
    }

    pub fn wait(&mut self, val: u32) -> FutexWait {
        let w = FutexWait { uaddr: self.addr, val };
        self.queue.push(FutexWait { uaddr: self.addr, val });
        w
    }

    pub fn wake(&mut self, max: usize) -> usize {
        let mut woken = 0;
        while woken < max && !self.queue.is_empty() {
            self.queue.remove(0);
            woken += 1;
        }
        woken
    }

    pub fn requeue(&mut self, other: &mut Futex, max: usize) -> usize {
        let mut requeued = 0;
        while requeued < max && !self.queue.is_empty() {
            let wait = self.queue.remove(0);
            other.queue.push(wait);
            requeued += 1;
        }
        requeued
    }
}

/// Futex hash table, matching C's `futex_hash`
#[derive(Debug, Default)]
pub struct FutexTable {
    buckets: Vec<Vec<Futex>>,
}

impl FutexTable {
    pub fn new() -> Self {
        Self {
            buckets: (0..FUTEX_HASH_SIZE).map(|_| Vec::new()).collect(),
        }
    }

    fn hash(addr: u32) -> usize {
        (addr as usize) % FUTEX_HASH_SIZE
    }

    pub fn get_or_create(&mut self, addr: u32) -> &mut Futex {
        let h = Self::hash(addr);
        if let Some(pos) = self.buckets[h].iter().position(|f| f.addr == addr) {
            self.buckets[h][pos].retain();
            &mut self.buckets[h][pos]
        } else {
            self.buckets[h].push(Futex::new(addr));
            let last = self.buckets[h].len() - 1;
            &mut self.buckets[h][last]
        }
    }

    pub fn get(&mut self, addr: u32) -> Option<&mut Futex> {
        let h = Self::hash(addr);
        self.buckets[h].iter_mut().find(|f| f.addr == addr)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn futex_op_decoding_matches_c() {
        assert_eq!(FutexOp::from_op(FUTEX_WAIT), FutexOp::Wait);
        assert_eq!(FutexOp::from_op(FUTEX_WAKE), FutexOp::Wake);
        assert_eq!(FutexOp::from_op(FUTEX_REQUEUE), FutexOp::Requeue);
        assert_eq!(FutexOp::from_op(99), FutexOp::Unknown(99));
        assert!(FutexOp::is_private(FUTEX_WAIT | FUTEX_PRIVATE_FLAG));
        assert!(!FutexOp::is_private(FUTEX_WAIT));
    }

    #[test]
    fn futex_wait_wake_requeue() {
        let mut f1 = Futex::new(0x1000);
        f1.wait(1);
        f1.wait(2);
        assert_eq!(f1.queue.len(), 2);
        assert_eq!(f1.wake(1), 1);
        assert_eq!(f1.queue.len(), 1);

        let mut f2 = Futex::new(0x2000);
        assert_eq!(f1.requeue(&mut f2, 1), 1);
        assert_eq!(f1.queue.len(), 0);
        assert_eq!(f2.queue.len(), 1);
    }

    #[test]
    fn futex_table_hash_and_refcount() {
        let mut table = FutexTable::new();
        {
            let f = table.get_or_create(0x1000);
            assert_eq!(f.addr, 0x1000);
            assert_eq!(f.refcount, 1);
        }
        {
            let f = table.get_or_create(0x1000);
            assert_eq!(f.refcount, 2);
        }
        assert!(table.get(0x1000).is_some());
        assert!(table.get(0x2000).is_none());
    }
}
