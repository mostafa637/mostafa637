//! `fs/pty.c` — pty constants and state with packet mode.

use crate::tty::{Termios, Winsize};

pub const PTY_MAX: usize = 256;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PtyType { Master, Slave }

impl Default for PtyType { fn default() -> Self { Self::Master } }

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct PtyState {
    pub locked: bool,
    pub packet_mode: bool,
    pub num: u32,
    pub pty_type: Option<PtyType>,
}

impl PtyState {
    pub fn new(num: u32, pty_type: PtyType) -> Self {
        Self { num, pty_type: Some(pty_type), locked: pty_type == PtyType::Slave, packet_mode: false }
    }
    pub fn is_locked(&self) -> bool { self.locked }
    pub fn lock(&mut self) { self.locked = true; }
    pub fn unlock(&mut self) { self.locked = false; }
    pub fn set_packet_mode(&mut self, enabled: bool) { self.packet_mode = enabled; }
}

pub const TIOCSPTLCK: u32 = 0x80047431;
pub const TIOCGPTN: u32 = 0x80047430;
pub const TIOCPKT: u32 = 0x40047438;
pub const TIOCGPKT: u32 = 0x80047438;

/// Pty pair, matching C's master holds ref to slave
#[derive(Debug)]
pub struct PtyPair {
    pub master: PtyState,
    pub slave: PtyState,
    pub master_termios: Termios,
    pub slave_termios: Termios,
    pub slave_winsize: Winsize,
    pub slave_uid: u32,
    pub slave_gid: u32,
    pub slave_perms: u32,
}

impl PtyPair {
    pub fn new(num: u32) -> Self {
        Self {
            master: PtyState::new(num, PtyType::Master),
            slave: PtyState::new(num, PtyType::Slave),
            master_termios: Termios::default(),
            slave_termios: Termios::default(),
            slave_winsize: Winsize::default(),
            slave_uid: 0,
            slave_gid: 0,
            slave_perms: 0o620,
        }
    }

    pub fn init_slave_inode(&mut self, uid: u32, gid: u32) {
        self.slave_uid = uid;
        self.slave_gid = gid;
        self.slave_perms = 0o620;
    }

    pub fn hangup(&mut self) {
        // Simulate hangup: clear other references
        self.master.locked = false;
        self.slave.locked = true;
    }

    pub fn slave_open(&self) -> Result<(), i32> {
        if self.slave.is_locked() { return Err(-5); } // EIO
        Ok(())
    }
}

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

    #[test]
    fn pty_pair_init_and_hangup() {
        let mut pair = PtyPair::new(1);
        pair.init_slave_inode(1000, 1000);
        assert_eq!(pair.slave_uid, 1000);
        assert_eq!(pair.slave_perms, 0o620);
        assert!(pair.slave_open().is_err()); // locked initially
        pair.slave.unlock();
        assert!(pair.slave_open().is_ok());
        pair.hangup();
        assert!(pair.slave.is_locked());
    }
}
