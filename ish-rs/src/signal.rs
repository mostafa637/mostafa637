//! `kernel/signal.h` — signal numbers, masks, and siginfo layouts.
//!
//! This is a leaf-plus module: the constants and mask helpers are independent,
//! while `sighand` and delivery depend on `task` and the execution engine.
//! This file ports the constants, `sigset_t_`, `sigaction_`, `siginfo_`, and
//! the mask helpers that `task.rs` and future `signal.c` ports need.

/// `sigset_t_` — 64-bit mask in iSH.
pub type SigSet = u64;

/// `NUM_SIGS`
pub const NUM_SIGS: usize = 64;

/// Signal numbers, `SIGHUP_` .. `SIGSYS_`.
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

/// `SIG_ERR_`, `SIG_DFL_`, `SIG_IGN_`
pub const SIG_ERR: i32 = -1;
pub const SIG_DFL: u32 = 0;
pub const SIG_IGN: u32 = 1;

/// `SA_*`
pub const SA_SIGINFO: u32 = 4;
pub const SA_NODEFER: u32 = 0x4000_0000;
pub const SA_RESETHAND: u32 = 0x8000_0000;

/// `SI_*`
pub const SI_USER: i32 = 0;
pub const SI_TIMER: i32 = -2;
pub const SI_TKILL: i32 = -6;
pub const SI_KERNEL: i32 = 128;
pub const TRAP_TRACE: i32 = 2;
pub const SEGV_MAPERR: i32 = 1;
pub const SEGV_ACCERR: i32 = 2;

/// `SIG_BLOCK_`, `SIG_UNBLOCK_`, `SIG_SETMASK_`
pub const SIG_BLOCK: u32 = 0;
pub const SIG_UNBLOCK: u32 = 1;
pub const SIG_SETMASK: u32 = 2;

/// `SS_*`
pub const SS_ONSTACK: u32 = 1;
pub const SS_DISABLE: u32 = 2;
pub const MINSIGSTKSZ: u32 = 2048;

/// `struct sigaction_` — packed guest ABI.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct SigAction {
    pub handler: u32,
    pub flags: u32,
    pub restorer: u32,
    pub mask: SigSet,
}

/// `union sigval_`
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct SigVal {
    pub int: i32,
    pub ptr: u32,
}

/// Simplified `siginfo_` — only the fields needed for mask tests now;
/// full delivery will extend this.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct SigInfo {
    pub sig: i32,
    pub errno: i32,
    pub code: i32,
    // The union is omitted for now; the full port will add it after `task`.
}

/// `struct stack_t_`
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct StackT {
    pub stack: u32,
    pub flags: u32,
    pub size: u32,
}

/// Mask helpers, mirroring C's `sig_mask`, `sigset_has`, etc.

#[inline]
pub fn sig_mask(sig: u32) -> SigSet {
    assert!((1..NUM_SIGS as u32).contains(&sig));
    1u64 << (sig - 1)
}

#[inline]
pub fn sigset_has(set: SigSet, sig: u32) -> bool {
    (set & sig_mask(sig)) != 0
}

#[inline]
pub fn sigset_add(set: &mut SigSet, sig: u32) {
    *set |= sig_mask(sig);
}

#[inline]
pub fn sigset_del(set: &mut SigSet, sig: u32) {
    *set &= !sig_mask(sig);
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sig_mask_matches_c() {
        assert_eq!(sig_mask(1), 1);
        assert_eq!(sig_mask(2), 2);
        assert_eq!(sig_mask(9), 1 << 8);
    }

    #[test]
    fn sigset_helpers_match_c() {
        let mut set = 0u64;
        sigset_add(&mut set, SIGINT);
        sigset_add(&mut set, SIGKILL);
        assert!(sigset_has(set, SIGINT));
        assert!(sigset_has(set, SIGKILL));
        assert!(!sigset_has(set, SIGHUP));
        sigset_del(&mut set, SIGINT);
        assert!(!sigset_has(set, SIGINT));
    }

    #[test]
    fn constants_match_c_header() {
        assert_eq!(SIGHUP, 1);
        assert_eq!(SIGKILL, 9);
        assert_eq!(SIGSYS, 31);
        assert_eq!(SA_SIGINFO, 4);
    }
}
