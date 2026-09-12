//! `util/refcount.h` — reference counting helpers.
//!
//! The C header provides `refcount_init`, `refcount_get`, and the
//! `DEFINE_REFCOUNT` macro that generates retain/release functions. In Rust
//! we use `Arc` for shared ownership, but we also provide a `RefCount` type
//! that mirrors the C's explicit atomic counter for cases where the C code
//! manages lifetime manually (e.g. `struct mm`, `fs_info`, `sighand`).

use std::sync::atomic::{AtomicUsize, Ordering};

/// Explicit refcount, mirroring `struct refcount`.
#[derive(Debug)]
pub struct RefCount {
    rc: AtomicUsize,
}

impl RefCount {
    /// `refcount_init` — start at 1.
    pub fn new() -> Self {
        Self {
            rc: AtomicUsize::new(1),
        }
    }

    /// `refcount_get`
    pub fn get(&self) -> usize {
        self.rc.load(Ordering::Relaxed)
    }

    /// Retain — increment.
    pub fn retain(&self) -> usize {
        self.rc.fetch_add(1, Ordering::Relaxed) + 1
    }

    /// Release — decrement, returns true if it reached zero.
    pub fn release(&self) -> bool {
        let prev = self.rc.fetch_sub(1, Ordering::AcqRel);
        prev == 1
    }
}

impl Default for RefCount {
    fn default() -> Self {
        Self::new()
    }
}

impl Clone for RefCount {
    fn clone(&self) -> Self {
        // Cloning a RefCount does not clone the count; it creates a new
        // counter at 1, matching C's shallow copy + refcount_init pattern.
        Self::new()
    }
}

/// Helper trait for types that have an embedded `RefCount`, mirroring
/// `DECLARE_REFCOUNT` / `DEFINE_REFCOUNT`.
pub trait RefCounted {
    fn refcount(&self) -> &RefCount;
    fn retain(&self) -> usize {
        self.refcount().retain()
    }
    fn release(&self) -> bool {
        self.refcount().release()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn refcount_init_get_retain_release() {
        let rc = RefCount::new();
        assert_eq!(rc.get(), 1);
        assert_eq!(rc.retain(), 2);
        assert_eq!(rc.get(), 2);
        assert!(!rc.release());
        assert_eq!(rc.get(), 1);
        assert!(rc.release());
        assert_eq!(rc.get(), 0);
    }

    #[test]
    fn refcount_clone_starts_at_one() {
        let rc = RefCount::new();
        rc.retain();
        let rc2 = rc.clone();
        assert_eq!(rc2.get(), 1);
    }
}
