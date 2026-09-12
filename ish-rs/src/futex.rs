//! `kernel/futex.h` + `kernel/futex.c` — futex constants and queue types.
//!
//! The full futex implementation depends on `task`, `mem`, and host
//! synchronization. This leaf module ports the constants, the `FUTEX_*`
//! command masks, and the pure data-structure definitions that are
//! independent of the host threading model. The blocking/waking logic will
//! be added after `sync` and `task` host-thread support.

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

/// Hash bits for the futex table.
pub const FUTEX_HASH_BITS: usize = 12;
pub const FUTEX_HASH_SIZE: usize = 1 << FUTEX_HASH_BITS;

/// A futex wait queue entry, analogous to `struct futex_wait` in C.
/// In Rust this is a safe handle; the actual condvar lives in the host
/// sync layer.
#[derive(Debug)]
pub struct FutexWait {
    /// Guest address being waited on.
    pub uaddr: u32,
    /// Expected value (for WAIT).
    pub val: u32,
}

/// Futex operation decoded from `op`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FutexOp {
    Wait,
    Wake,
    Requeue,
    Unknown(u32),
}

impl FutexOp {
    /// Decode `op & FUTEX_CMD_MASK`.
    pub fn from_op(op: u32) -> Self {
        match op & FUTEX_CMD_MASK {
            FUTEX_WAIT => Self::Wait,
            FUTEX_WAKE => Self::Wake,
            FUTEX_REQUEUE => Self::Requeue,
            other => Self::Unknown(other),
        }
    }

    /// Whether `FUTEX_PRIVATE_FLAG` was set.
    pub fn is_private(op: u32) -> bool {
        op & FUTEX_PRIVATE_FLAG != 0
    }
}

/// `struct robust_list_head_` from `futex.c`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct RobustListHead {
    pub list: u32,
    pub offset: u32,
    pub list_op_pending: u32,
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
    fn hash_size_matches_c() {
        assert_eq!(FUTEX_HASH_SIZE, 4096);
    }
}
