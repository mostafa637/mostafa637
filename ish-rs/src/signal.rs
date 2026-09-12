//! `kernel/signal.h` + `kernel/signal.c` — signal numbers, masks, and delivery.
//!
//! The C implementation has complex locking and thread wakeup via pthread.
//! This Rust port keeps the pure logic (mask helpers, action determination,
//! queue management) and abstracts the host wakeup via trait.

/// `sigset_t_` — 64-bit mask in iSH.
pub type SigSet = u64;

/// `NUM_SIGS`
pub const NUM_SIGS: usize = 64;

/// Signal numbers
pub const SIGHUP: u32 = 1;
pub const SIGINT: u32 = 2;
pub const SIGQUIT: u32 = 3;
pub const SIGILL: u32 = 4;
pub const SIGTRAP: u32 = 5;
pub const SIGABRT: u32 = 6;
pub const SIGIOT: u32 = 6;
pub const SIGBUS: u32 = 7;
pub const SIGFPE: u32 = 8;
pub const SIGKILL: u32 = 9;
pub const SIGUSR1: u32 = 10;
pub const SIGSEGV: u32 = 11;
pub const SIGUSR2: u32 = 12;
pub const SIGPIPE: u32 = 13;
pub const SIGALRM: u32 = 14;
pub const SIGTERM: u32 = 15;
pub const SIGSTKFLT: u32 = 16;
pub const SIGCHLD: u32 = 17;
pub const SIGCONT: u32 = 18;
pub const SIGSTOP: u32 = 19;
pub const SIGTSTP: u32 = 20;
pub const SIGTTIN: u32 = 21;
pub const SIGTTOU: u32 = 22;
pub const SIGURG: u32 = 23;
pub const SIGXCPU: u32 = 24;
pub const SIGXFSZ: u32 = 25;
pub const SIGVTALRM: u32 = 26;
pub const SIGPROF: u32 = 27;
pub const SIGWINCH: u32 = 28;
pub const SIGIO: u32 = 29;
pub const SIGPWR: u32 = 30;
pub const SIGSYS: u32 = 31;

pub const SIG_ERR: i32 = -1;
pub const SIG_DFL: u32 = 0;
pub const SIG_IGN: u32 = 1;

pub const SA_SIGINFO: u32 = 4;
pub const SA_NODEFER: u32 = 0x4000_0000;
pub const SA_RESETHAND: u32 = 0x8000_0000;

pub const SI_USER: i32 = 0;
pub const SI_TIMER: i32 = -2;
pub const SI_TKILL: i32 = -6;
pub const SI_KERNEL: i32 = 128;
pub const TRAP_TRACE: i32 = 2;
pub const SEGV_MAPERR: i32 = 1;
pub const SEGV_ACCERR: i32 = 2;

pub const SIG_BLOCK: u32 = 0;
pub const SIG_UNBLOCK: u32 = 1;
pub const SIG_SETMASK: u32 = 2;

pub const SS_ONSTACK: u32 = 1;
pub const SS_DISABLE: u32 = 2;
pub const MINSIGSTKSZ: u32 = 2048;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct SigAction {
    pub handler: u32,
    pub flags: u32,
    pub restorer: u32,
    pub mask: SigSet,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct SigVal {
    pub int: i32,
    pub ptr: u32,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct SigInfo {
    pub sig: i32,
    pub errno: i32,
    pub code: i32,
}

pub const SIGINFO_NIL: SigInfo = SigInfo { sig: 0, errno: 0, code: 0 };

/// Sigset helpers matching C macros.
pub fn sig_mask(sig: u32) -> SigSet {
    if sig == 0 || sig > 64 {
        0
    } else {
        1u64 << (sig - 1)
    }
}

pub fn sigset_has(set: SigSet, sig: u32) -> bool {
    (set & sig_mask(sig)) != 0
}

pub fn sigset_add(set: &mut SigSet, sig: u32) {
    *set |= sig_mask(sig);
}

pub fn sigset_remove(set: &mut SigSet, sig: u32) {
    *set &= !sig_mask(sig);
}

/// Signal action determination, matching C `signal_action`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SignalAction {
    Ignore,
    Kill,
    CallHandler,
    Stop,
}

pub fn signal_is_blockable(sig: u32) -> bool {
    sig != SIGKILL && sig != SIGSTOP
}

pub fn signal_action(sighand: &[SigAction; 65], sig: u32) -> SignalAction {
    if signal_is_blockable(sig) {
        let action = &sighand[sig as usize];
        if action.handler == SIG_IGN {
            return SignalAction::Ignore;
        }
        if action.handler != SIG_DFL {
            return SignalAction::CallHandler;
        }
    }

    match sig {
        SIGURG | SIGCONT | SIGCHLD | SIGIO | SIGWINCH => SignalAction::Ignore,
        SIGSTOP | SIGTSTP | SIGTTIN | SIGTTOU => SignalAction::Stop,
        _ => SignalAction::Kill,
    }
}

/// Signal queue entry, matching C `struct sigqueue`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SigQueue {
    pub info: SigInfo,
}

/// Task signal state, simplified from C's task fields.
#[derive(Debug, Default)]
pub struct SignalState {
    pub pending: SigSet,
    pub blocked: SigSet,
    pub waiting: SigSet,
    pub queue: Vec<SigQueue>,
}

impl SignalState {
    pub fn new() -> Self {
        Self::default()
    }

    /// `deliver_signal_unlocked` logic without locking.
    pub fn deliver(&mut self, sig: u32, info: SigInfo) -> bool {
        if sigset_has(self.pending, sig) {
            return false; // already pending
        }
        sigset_add(&mut self.pending, sig);
        self.queue.push(SigQueue { info: SigInfo { sig: sig as i32, ..info } });

        // If blocked and not waiting, don't wake
        if sigset_has(self.blocked & !self.waiting, sig) && signal_is_blockable(sig) {
            return false;
        }
        true // would wake
    }

    pub fn has_pending(&self, sig: u32) -> bool {
        sigset_has(self.pending, sig)
    }

    pub fn dequeue(&mut self) -> Option<SigQueue> {
        if self.queue.is_empty() {
            return None;
        }
        let q = self.queue.remove(0);
        sigset_remove(&mut self.pending, q.info.sig as u32);
        Some(q)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sigset_helpers_match_c() {
        assert_eq!(sig_mask(1), 1);
        assert_eq!(sig_mask(2), 2);
        assert_eq!(sig_mask(64), 1u64 << 63);
        assert_eq!(sig_mask(0), 0);
        assert_eq!(sig_mask(65), 0);

        let mut set = 0;
        sigset_add(&mut set, SIGHUP);
        assert!(sigset_has(set, SIGHUP));
        assert!(!sigset_has(set, SIGINT));
        sigset_remove(&mut set, SIGHUP);
        assert!(!sigset_has(set, SIGHUP));
    }

    #[test]
    fn signal_is_blockable_matches_c() {
        assert!(!signal_is_blockable(SIGKILL));
        assert!(!signal_is_blockable(SIGSTOP));
        assert!(signal_is_blockable(SIGTERM));
    }

    #[test]
    fn signal_action_matches_c() {
        let mut actions = [SigAction::default(); 65];
        // Default actions
        assert_eq!(signal_action(&actions, SIGCHLD), SignalAction::Ignore);
        assert_eq!(signal_action(&actions, SIGSTOP), SignalAction::Stop);
        assert_eq!(signal_action(&actions, SIGTERM), SignalAction::Kill);

        // SIG_IGN
        actions[SIGTERM as usize].handler = SIG_IGN;
        assert_eq!(signal_action(&actions, SIGTERM), SignalAction::Ignore);

        // Custom handler
        actions[SIGTERM as usize].handler = 0x1234;
        assert_eq!(signal_action(&actions, SIGTERM), SignalAction::CallHandler);

        // SIGKILL and SIGSTOP cannot be ignored
        actions[SIGKILL as usize].handler = SIG_IGN;
        assert_eq!(signal_action(&actions, SIGKILL), SignalAction::Kill);
    }

    #[test]
    fn signal_delivery_and_queue() {
        let mut state = SignalState::new();
        assert!(state.deliver(SIGTERM, SIGINFO_NIL));
        assert!(state.has_pending(SIGTERM));
        assert_eq!(state.queue.len(), 1);

        // Second delivery of same signal should not add
        assert!(!state.deliver(SIGTERM, SIGINFO_NIL));
        assert_eq!(state.queue.len(), 1);

        // Blocked signal should not wake but still pending
        state.blocked = sig_mask(SIGUSR1);
        assert!(!state.deliver(SIGUSR1, SIGINFO_NIL)); // blocked, no wake
        assert!(state.has_pending(SIGUSR1));

        // Dequeue
        let q = state.dequeue().unwrap();
        assert_eq!(q.info.sig, SIGTERM as i32);
        assert!(!state.has_pending(SIGTERM));
    }
}
