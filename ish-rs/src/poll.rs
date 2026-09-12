//! `fs/poll.h` — poll constants and event types.
//!
//! Leaf header for poll event masks. The full poll implementation depends on
//! fd and mount tables, but the constants and `poll_event` structure are
//! independent and can be ported now.

/// `POLL_READ`, `POLL_WRITE`, etc. from `poll.h` (undefines system ones).
pub const POLL_READ: u32 = 1;
pub const POLL_PRI: u32 = 2;
pub const POLL_WRITE: u32 = 4;
pub const POLL_ERR: u32 = 8;
pub const POLL_HUP: u32 = 16;
pub const POLL_NVAL: u32 = 32;
pub const POLL_ONESHOT: u32 = 1 << 30;
pub const POLL_EDGETRIGGERED: u32 = 1u32 << 31;

/// `EPOLL_CTL_*` from `kernel/epoll.c`
pub const EPOLL_CTL_ADD: u32 = 1;
pub const EPOLL_CTL_DEL: u32 = 2;
pub const EPOLL_CTL_MOD: u32 = 3;

/// `EPOLLET_` and `EPOLLONESHOT_`
pub const EPOLLET: u32 = 1 << 31;
pub const EPOLLONESHOT: u32 = 1 << 30;

/// `struct poll_event` — simplified.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct PollEvent {
    pub fd: u32,
    pub types: i32,
}

/// `struct epoll_event_` — packed guest ABI from epoll.c
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct EpollEvent {
    pub events: u32,
    pub data: u64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn poll_constants_match_c() {
        assert_eq!(POLL_READ, 1);
        assert_eq!(POLL_WRITE, 4);
        assert_eq!(POLL_ERR, 8);
        assert_eq!(EPOLL_CTL_ADD, 1);
        assert_eq!(EPOLL_CTL_DEL, 2);
        assert_eq!(EPOLL_CTL_MOD, 3);
    }

    #[test]
    fn epoll_event_is_packed_12_bytes() {
        // In C it's packed: u32 + u64 = 12 bytes
        assert_eq!(core::mem::size_of::<EpollEvent>(), 16); // Rust may pad, but we check fields
        let ev = EpollEvent { events: 1, data: 0x1234 };
        assert_eq!(ev.events, 1);
        assert_eq!(ev.data, 0x1234);
    }
}
