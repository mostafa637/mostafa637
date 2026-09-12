//! `kernel/eventfd.c` — eventfd constants and logic.
//!
//! The C file implements eventfd via `adhoc_fd_create` and fd ops. This
//! module ports the constants and the pure read/write/poll logic that is
//! independent of the fd table, depending only on `poll` constants and
//! `sync` abstractions already ported.

use crate::poll::{POLL_READ, POLL_WRITE};

/// Maximum value for eventfd counter (UINT64_MAX - 1 is still writable).
pub const EVENTFD_MAX: u64 = u64::MAX - 1;

/// Eventfd state, mirroring `fd->eventfd.val`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct EventFd {
    pub val: u64,
}

impl EventFd {
    pub fn new(initval: u64) -> Self {
        Self { val: initval }
    }

    /// `eventfd_read` logic — returns value and clears, or error code.
    /// Returns Ok(value) or Err(errno).
    pub fn read(&mut self, nonblock: bool) -> Result<u64, i32> {
        if self.val == 0 {
            if nonblock {
                return Err(-11); // EAGAIN
            }
            // In blocking case, C would wait; we return EAGAIN for single-threaded port
            return Err(-11);
        }
        let v = self.val;
        self.val = 0;
        Ok(v)
    }

    /// `eventfd_write` logic.
    pub fn write(&mut self, increment: u64, nonblock: bool) -> Result<(), i32> {
        if increment == u64::MAX {
            return Err(-22); // EINVAL
        }
        if self.val >= u64::MAX - increment {
            if nonblock {
                return Err(-11); // EAGAIN
            }
            return Err(-11);
        }
        self.val += increment;
        Ok(())
    }

    /// `eventfd_poll`
    pub fn poll(&self) -> u32 {
        let mut types = 0;
        if self.val > 0 {
            types |= POLL_READ;
        }
        if self.val < u64::MAX - 1 {
            types |= POLL_WRITE;
        }
        types
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn eventfd_read_write_poll_matches_c() {
        let mut efd = EventFd::new(5);
        assert_eq!(efd.poll() & POLL_READ, POLL_READ);
        assert_eq!(efd.read(false).unwrap(), 5);
        assert_eq!(efd.val, 0);
        assert_eq!(efd.poll() & POLL_READ, 0);

        assert_eq!(efd.write(3, false), Ok(()));
        assert_eq!(efd.val, 3);
        assert_eq!(efd.write(u64::MAX, false), Err(-22));
    }

    #[test]
    fn eventfd_nonblock_eagain() {
        let mut efd = EventFd::new(0);
        assert_eq!(efd.read(true), Err(-11));
        let mut efd2 = EventFd::new(u64::MAX - 1);
        assert_eq!(efd2.write(2, true), Err(-11));
    }
}
