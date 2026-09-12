//! `fs/pty.c` — pty constants and state.

use crate::tty::{Termios, Winsize};

/// Pty number max
pub const PTY_MAX: usize = 256;

/// Pty state
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PtyType {
    Master,
    Slave,
}

/// Pty lock state
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct PtyState {
    pub locked: bool,
    pub packet_mode: bool,
    pub num: u32,
    pub pty_type: Option<PtyType>,
}

impl Default for PtyType {
    fn default() -> Self { Self::Master }
}

impl PtyState {
    pub fn new(num: u32, pty_type: PtyType) -> Self {
        Self { num, pty_type: Some(pty_type), locked: pty_type == PtyType::Slave, packet_mode: false }
    }

    pub fn is_locked(&self) -> bool { self.locked }
    pub fn lock(&mut self) { self.locked = true; }
    pub fn unlock(&mut self) { self.locked = false; }
}

/// TIOCSPTLCK, TIOCGPTN, etc.
pub const TIOCSPTLCK: u32 = 0x80047431;
pub const TIOCGPTN: u32 = 0x80047430;
pub const TIOCPKT: u32 = 0x40047438;
pub const TIOCGPKT: u32 = 0x80047438;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pty_state_lock() {
        let mut pty = PtyState::new(1, PtyType::Slave);
        assert!(pty.is_locked());
        pty.unlock();
        assert!(!pty.is_locked());
        pty.lock();
        assert!(pty.is_locked());
    }

    #[test]
    fn pty_type() {
        let master = PtyState::new(0, PtyType::Master);
        assert_eq!(master.pty_type, Some(PtyType::Master));
        assert!(!master.is_locked());
    }
}
