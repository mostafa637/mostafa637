//! `kernel/sockrestart.c` — socket restart helpers for interrupted syscalls.

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum SockRestartType {
    #[default]
    None,
    Connect,
    Accept,
    Recv,
    Send,
}

#[derive(Debug, Clone, Default)]
pub struct SockRestart {
    pub restart_type: SockRestartType,
    pub fd: i32,
    pub addr: u32,
    pub addrlen: u32,
    pub flags: u32,
}

impl SockRestart {
    pub fn new(restart_type: SockRestartType, fd: i32) -> Self {
        Self { restart_type, fd, ..Default::default() }
    }
    pub fn is_restarting(&self) -> bool { !matches!(self.restart_type, SockRestartType::None) }
    pub fn clear(&mut self) { self.restart_type = SockRestartType::None; }
}

pub fn should_restart_syscall(error: i32, signal_pending: bool) -> bool {
    // EINTR = -4 in guest
    error == -4 && !signal_pending
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sock_restart_type() {
        let mut r = SockRestart::new(SockRestartType::Connect, 3);
        assert!(r.is_restarting());
        assert_eq!(r.fd, 3);
        r.clear();
        assert!(!r.is_restarting());
    }

    #[test]
    fn should_restart() {
        assert!(should_restart_syscall(-4, false));
        assert!(!should_restart_syscall(-4, true));
        assert!(!should_restart_syscall(-22, false));
    }
}
