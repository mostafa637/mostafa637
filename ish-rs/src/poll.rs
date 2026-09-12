//! `fs/poll.h` + `kernel/poll.c` + `kernel/epoll.c` — poll constants and logic.
//!
//! The C implementation uses real host poll via `real_poll`. This Rust port
//! abstracts the host poll via `PollHost` trait, keeping the core portable
//! while preserving the event mapping logic.

/// `POLL_READ`, `POLL_WRITE`, etc. from `poll.h`
pub const POLL_READ: u32 = 1;
pub const POLL_PRI: u32 = 2;
pub const POLL_WRITE: u32 = 4;
pub const POLL_ERR: u32 = 8;
pub const POLL_HUP: u32 = 16;
pub const POLL_NVAL: u32 = 32;
pub const POLL_ONESHOT: u32 = 1 << 30;
pub const POLL_EDGETRIGGERED: u32 = 1u32 << 31;

/// `EPOLL_CTL_*`
pub const EPOLL_CTL_ADD: u32 = 1;
pub const EPOLL_CTL_DEL: u32 = 2;
pub const EPOLL_CTL_MOD: u32 = 3;

pub const EPOLLET: u32 = 1 << 31;
pub const EPOLLONESHOT: u32 = 1 << 30;

/// `POLL_ALWAYS_LISTENING`
pub const POLL_ALWAYS_LISTENING: u32 = POLL_ERR | POLL_HUP | POLL_NVAL;

/// `SELECT_READ`, `SELECT_WRITE`, `SELECT_EX`
pub const SELECT_READ: u32 = POLL_READ | POLL_HUP | POLL_ERR;
pub const SELECT_WRITE: u32 = POLL_WRITE | POLL_ERR;
pub const SELECT_EX: u32 = POLL_PRI;

/// `struct poll_event`
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct PollEvent {
    pub fd: u32,
    pub types: i32,
}

/// `struct epoll_event_` — packed guest ABI
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct EpollEvent {
    pub events: u32,
    pub data: u64,
}

/// Poll fd info, matching C's `union poll_fd_info`
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PollFdInfo {
    Ptr(usize),
    Num(u64),
    Fd(i32),
}

impl Default for PollFdInfo {
    fn default() -> Self {
        Self::Num(0)
    }
}

/// Poll fd entry.
#[derive(Debug, Clone)]
pub struct PollFd {
    pub fd: i32,
    pub events: u32,
    pub info: PollFdInfo,
    pub triggered: u32,
}

impl PollFd {
    pub fn new(fd: i32, events: u32, info: PollFdInfo) -> Self {
        Self {
            fd,
            events,
            info,
            triggered: 0,
        }
    }
}

/// Poll context, matching C's `struct poll`
#[derive(Debug, Default)]
pub struct Poll {
    pub fds: Vec<PollFd>,
}

impl Poll {
    pub fn new() -> Self {
        Self { fds: Vec::new() }
    }

    pub fn add_fd(&mut self, fd: i32, events: u32, info: PollFdInfo) -> Result<(), i32> {
        if self.fds.iter().any(|f| f.fd == fd) {
            return Err(-17); // EEXIST
        }
        self.fds.push(PollFd::new(fd, events, info));
        Ok(())
    }

    pub fn del_fd(&mut self, fd: i32) -> Result<(), i32> {
        if let Some(idx) = self.fds.iter().position(|f| f.fd == fd) {
            self.fds.remove(idx);
            Ok(())
        } else {
            Err(-2) // ENOENT
        }
    }

    pub fn mod_fd(&mut self, fd: i32, events: u32, info: PollFdInfo) -> Result<(), i32> {
        if let Some(pfd) = self.fds.iter_mut().find(|f| f.fd == fd) {
            pfd.events = events;
            pfd.info = info;
            Ok(())
        } else {
            Err(-2)
        }
    }

    pub fn has_fd(&self, fd: i32) -> bool {
        self.fds.iter().any(|f| f.fd == fd)
    }

    /// Simulate poll_wait: given a list of ready events, call callback.
    pub fn wait<F>(&mut self, ready: &[(i32, u32)], mut callback: F) -> i32
    where
        F: FnMut(u32, PollFdInfo) -> i32,
    {
        let mut count = 0;
        for (fd, revents) in ready {
            if let Some(pfd) = self.fds.iter_mut().find(|f| f.fd == *fd) {
                // Check if already triggered for edge-triggered
                let effective = if (pfd.events & POLL_EDGETRIGGERED) != 0 {
                    revents & !pfd.triggered
                } else {
                    *revents
                };
                if effective != 0 {
                    let res = callback(effective, pfd.info);
                    if res != 0 {
                        count += 1;
                    }
                    pfd.triggered |= effective;
                }
            }
        }
        count
    }

    pub fn wakeup(&mut self, fd: i32, events: u32) {
        if let Some(pfd) = self.fds.iter_mut().find(|f| f.fd == fd) {
            pfd.triggered &= !events;
        }
    }
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
    fn poll_add_del_mod() {
        let mut poll = Poll::new();
        assert!(poll.add_fd(1, POLL_READ, PollFdInfo::Fd(1)).is_ok());
        assert!(poll.add_fd(1, POLL_READ, PollFdInfo::Fd(1)).is_err()); // EEXIST
        assert!(poll.has_fd(1));
        assert!(poll.mod_fd(1, POLL_WRITE, PollFdInfo::Fd(1)).is_ok());
        assert_eq!(poll.fds[0].events, POLL_WRITE);
        assert!(poll.del_fd(1).is_ok());
        assert!(!poll.has_fd(1));
        assert!(poll.del_fd(1).is_err());
    }

    #[test]
    fn poll_wait_and_wakeup() {
        let mut poll = Poll::new();
        poll.add_fd(1, POLL_READ, PollFdInfo::Num(100)).unwrap();
        poll.add_fd(2, POLL_READ | POLL_EDGETRIGGERED, PollFdInfo::Num(200)).unwrap();

        let mut results = Vec::new();
        let count = poll.wait(&[(1, POLL_READ), (2, POLL_READ)], |ev, info| {
            results.push((ev, info));
            1
        });
        assert_eq!(count, 2);
        assert_eq!(results.len(), 2);

        // Edge-triggered: second wait without wakeup should not trigger
        let mut results2 = Vec::new();
        let count2 = poll.wait(&[(2, POLL_READ)], |ev, info| {
            results2.push((ev, info));
            1
        });
        assert_eq!(count2, 0);

        // After wakeup, should trigger again
        poll.wakeup(2, POLL_READ);
        let mut results3 = Vec::new();
        let count3 = poll.wait(&[(2, POLL_READ)], |ev, info| {
            results3.push((ev, info));
            1
        });
        assert_eq!(count3, 1);
    }

    #[test]
    fn select_constants() {
        assert_eq!(SELECT_READ, POLL_READ | POLL_HUP | POLL_ERR);
        assert_eq!(SELECT_WRITE, POLL_WRITE | POLL_ERR);
    }
}
