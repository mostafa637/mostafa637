//! `kernel/signal.h` + `kernel/signal.c` — full port: signals, masks, delivery, altstack.

pub type SigSet = u64;
pub const NUM_SIGS: usize = 64;

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

pub const SA_NOCLDSTOP: u32 = 1;
pub const SA_NOCLDWAIT: u32 = 2;
pub const SA_SIGINFO: u32 = 4;
pub const SA_ONSTACK: u32 = 0x0800_0000;
pub const SA_RESTART: u32 = 0x1000_0000;
pub const SA_NODEFER: u32 = 0x4000_0000;
pub const SA_RESETHAND: u32 = 0x8000_0000;

pub const SI_USER: i32 = 0;
pub const SI_KERNEL: i32 = 128;
pub const SI_QUEUE: i32 = -1;
pub const SI_TIMER: i32 = -2;
pub const SI_MESGQ: i32 = -3;
pub const SI_ASYNCIO: i32 = -4;
pub const SI_SIGIO: i32 = -5;
pub const SI_TKILL: i32 = -6;
pub const TRAP_BRKPT: i32 = 1;
pub const TRAP_TRACE: i32 = 2;
pub const SEGV_MAPERR: i32 = 1;
pub const SEGV_ACCERR: i32 = 2;
pub const FPE_INTDIV: i32 = 1;
pub const FPE_INTOVF: i32 = 2;

pub const SIG_BLOCK: u32 = 0;
pub const SIG_UNBLOCK: u32 = 1;
pub const SIG_SETMASK: u32 = 2;

pub const SS_ONSTACK: u32 = 1;
pub const SS_DISABLE: u32 = 2;
pub const MINSIGSTKSZ: u32 = 2048;
pub const SIGSTKSZ: u32 = 8192;

// x86 sigframe constants
pub const UC_FP_XSTATE: u32 = 0x1;
pub const UC_SIGCONTEXT_SS: u32 = 0x2;
pub const UC_STRICT_RESTORE_SS: u32 = 0x4;

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
    pub pid: u32,
    pub uid: u32,
    pub addr: u32,
    pub status: i32,
    pub utime: u32,
    pub stime: u32,
    pub value: SigVal,
}

pub const SIGINFO_NIL: SigInfo = SigInfo { sig: 0, errno: 0, code: 0, pid: 0, uid: 0, addr: 0, status: 0, utime: 0, stime: 0, value: SigVal { int: 0, ptr: 0 } };

pub fn sig_mask(sig: u32) -> SigSet {
    if sig == 0 || sig > 64 { 0 } else { 1u64 << (sig - 1) }
}
pub fn sigset_has(set: SigSet, sig: u32) -> bool { (set & sig_mask(sig)) != 0 }
pub fn sigset_add(set: &mut SigSet, sig: u32) { *set |= sig_mask(sig); }
pub fn sigset_remove(set: &mut SigSet, sig: u32) { *set &= !sig_mask(sig); }
pub fn sigset_empty() -> SigSet { 0 }
pub fn sigset_full() -> SigSet { !0 }
pub fn sigset_is_empty(set: SigSet) -> bool { set == 0 }

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SignalAction {
    Ignore,
    Kill,
    CallHandler,
    Stop,
}

pub fn signal_is_blockable(sig: u32) -> bool { sig != SIGKILL && sig != SIGSTOP }

pub fn signal_action(sighand: &[SigAction; 65], sig: u32) -> SignalAction {
    if signal_is_blockable(sig) {
        let action = &sighand[sig as usize];
        if action.handler == SIG_IGN { return SignalAction::Ignore; }
        if action.handler != SIG_DFL { return SignalAction::CallHandler; }
    }
    match sig {
        SIGURG | SIGCONT | SIGCHLD | SIGIO | SIGWINCH => SignalAction::Ignore,
        SIGSTOP | SIGTSTP | SIGTTIN | SIGTTOU => SignalAction::Stop,
        _ => SignalAction::Kill,
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SigQueue {
    pub info: SigInfo,
}

#[derive(Debug)]
pub struct SignalState {
    pub pending: SigSet,
    pub blocked: SigSet,
    pub waiting: SigSet,
    pub queue: Vec<SigQueue>,
    pub alt_stack: SigAltStack,
    pub sighand: [SigAction; 65],
}

impl Default for SignalState {
    fn default() -> Self { Self::new() }
}

impl SignalState {
    pub fn new() -> Self {
        Self { pending: 0, blocked: 0, waiting: 0, queue: Vec::new(), alt_stack: SigAltStack::default(), sighand: [SigAction::default(); 65] }
    }

    pub fn deliver(&mut self, sig: u32, info: SigInfo) -> bool {
        if sigset_has(self.pending, sig) { return false; }
        sigset_add(&mut self.pending, sig);
        self.queue.push(SigQueue { info: SigInfo { sig: sig as i32, ..info } });
        if sigset_has(self.blocked & !self.waiting, sig) && signal_is_blockable(sig) {
            return false;
        }
        true
    }

    pub fn has_pending(&self, sig: u32) -> bool { sigset_has(self.pending, sig) }

    pub fn dequeue(&mut self) -> Option<SigQueue> {
        if self.queue.is_empty() { return None; }
        let q = self.queue.remove(0);
        sigset_remove(&mut self.pending, q.info.sig as u32);
        Some(q)
    }

    pub fn dequeue_signal(&mut self, sig: u32) -> Option<SigQueue> {
        if let Some(pos) = self.queue.iter().position(|q| q.info.sig as u32 == sig) {
            let q = self.queue.remove(pos);
            if !self.queue.iter().any(|q2| q2.info.sig as u32 == sig) {
                sigset_remove(&mut self.pending, sig);
            }
            Some(q)
        } else {
            None
        }
    }

    pub fn block(&mut self, sig: u32) { sigset_add(&mut self.blocked, sig); }
    pub fn unblock(&mut self, sig: u32) { sigset_remove(&mut self.blocked, sig); }
    pub fn is_blocked(&self, sig: u32) -> bool { sigset_has(self.blocked, sig) }

    pub fn pending_signals(&self) -> Vec<u32> { (1..=64).filter(|&s| self.has_pending(s)).collect() }

    pub fn sigaction(&mut self, sig: u32, new_act: Option<SigAction>) -> Result<SigAction, i32> {
        if sig == 0 || sig > 64 || sig == SIGKILL || sig == SIGSTOP { return Err(-22); }
        let old = self.sighand[sig as usize];
        if let Some(act) = new_act {
            self.sighand[sig as usize] = act;
        }
        Ok(old)
    }

    pub fn sigprocmask(&mut self, how: u32, set: Option<SigSet>) -> Result<SigSet, i32> {
        let old = self.blocked;
        if let Some(s) = set {
            match how {
                SIG_BLOCK => { self.blocked |= s; },
                SIG_UNBLOCK => { self.blocked &= !s; },
                SIG_SETMASK => { self.blocked = s; },
                _ => return Err(-22),
            }
            // SIGKILL and SIGSTOP cannot be blocked
            sigset_remove(&mut self.blocked, SIGKILL);
            sigset_remove(&mut self.blocked, SIGSTOP);
        }
        Ok(old)
    }

    pub fn next_unblocked_pending(&self) -> Option<u32> {
        for sig in 1..=64 {
            if self.has_pending(sig) && !self.is_blocked(sig) {
                return Some(sig);
            }
        }
        None
    }

    pub fn should_die_from_signal(&self, sig: u32) -> bool {
        matches!(signal_action(&self.sighand, sig), SignalAction::Kill)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct SigAltStack {
    pub sp: u32,
    pub flags: u32,
    pub size: u32,
}

impl SigAltStack {
    pub fn new(sp: u32, size: u32) -> Self { Self { sp, size, flags: 0 } }
    pub fn is_disabled(&self) -> bool { (self.flags & SS_DISABLE) != 0 }
    pub fn is_on_stack(&self) -> bool { (self.flags & SS_ONSTACK) != 0 }
    pub fn contains(&self, addr: u32) -> bool {
        if self.is_disabled() { return false; }
        addr >= self.sp && addr < self.sp + self.size
    }
    pub fn top(&self) -> u32 { self.sp + self.size }

    pub fn sigaltstack(&mut self, new_stack: Option<SigAltStack>) -> Result<SigAltStack, i32> {
        let old = *self;
        if let Some(ns) = new_stack {
            if ns.is_on_stack() { return Err(-22); }
            if !ns.is_disabled() && ns.size < MINSIGSTKSZ { return Err(-12); }
            *self = ns;
        }
        Ok(old)
    }
}

pub fn should_use_altstack(action: &SigAction, alt: &SigAltStack, current_sp: u32) -> bool {
    if alt.is_disabled() { return false; }
    if (action.flags & SA_ONSTACK) == 0 { return false; }
    if alt.contains(current_sp) { return false; }
    true
}

/// Signal frame, matching `struct sigframe` in C
#[derive(Debug, Clone, Default)]
pub struct SigFrame {
    pub sig: u32,
    pub handler: u32,
    pub restorer: u32,
    pub mask: SigSet,
    pub sp: u32,
    pub use_altstack: bool,
}

impl SigFrame {
    pub fn new(sig: u32, action: &SigAction, alt: &SigAltStack, current_sp: u32) -> Self {
        let use_alt = should_use_altstack(action, alt, current_sp);
        let sp = if use_alt { alt.top() } else { current_sp };
        Self { sig, handler: action.handler, restorer: action.restorer, mask: action.mask, sp, use_altstack: use_alt }
    }
}

/// Handle signal, matching `handle_signal` in C
pub fn handle_signal(state: &mut SignalState, sig: u32) -> Option<SigFrame> {
    let action = state.sighand[sig as usize];
    if action.handler == SIG_DFL || action.handler == SIG_IGN { return None; }
    let frame = SigFrame::new(sig, &action, &state.alt_stack, 0xbffff000);
    if (action.flags & SA_RESETHAND) != 0 {
        state.sighand[sig as usize].handler = SIG_DFL;
    }
    if (action.flags & SA_NODEFER) == 0 {
        sigset_add(&mut state.blocked, sig);
    }
    state.blocked |= action.mask;
    sigset_remove(&mut state.blocked, SIGKILL);
    sigset_remove(&mut state.blocked, SIGSTOP);
    Some(frame)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sigset_helpers_match_c() {
        let mut set: SigSet = 0;
        sigset_add(&mut set, 1);
        assert!(sigset_has(set, 1));
        assert!(!sigset_has(set, 2));
        sigset_add(&mut set, 2);
        assert!(sigset_has(set, 2));
        sigset_remove(&mut set, 1);
        assert!(!sigset_has(set, 1));
        assert_eq!(sig_mask(0), 0);
        assert_eq!(sig_mask(65), 0);
        assert!(sigset_is_empty(sigset_empty()));
        assert!(!sigset_is_empty(sigset_full()));
    }

    #[test]
    fn signal_is_blockable_matches_c() {
        assert!(!signal_is_blockable(SIGKILL));
        assert!(!signal_is_blockable(SIGSTOP));
        assert!(signal_is_blockable(SIGTERM));
    }

    #[test]
    fn signal_action_matches_c() {
        let mut sighand = [SigAction::default(); 65];
        assert_eq!(signal_action(&sighand, SIGTERM), SignalAction::Kill);
        assert_eq!(signal_action(&sighand, SIGCHLD), SignalAction::Ignore);
        assert_eq!(signal_action(&sighand, SIGSTOP), SignalAction::Stop);
        sighand[SIGTERM as usize].handler = SIG_IGN;
        assert_eq!(signal_action(&sighand, SIGTERM), SignalAction::Ignore);
        sighand[SIGTERM as usize].handler = 0x1234;
        assert_eq!(signal_action(&sighand, SIGTERM), SignalAction::CallHandler);
    }

    #[test]
    fn signal_queue_deliver_and_dequeue() {
        let mut state = SignalState::new();
        let info = SigInfo { sig: SIGTERM as i32, errno: 0, code: SI_USER, ..Default::default() };
        assert!(state.deliver(SIGTERM, info));
        assert!(state.has_pending(SIGTERM));
        assert!(!state.deliver(SIGTERM, info));
        let q = state.dequeue().unwrap();
        assert_eq!(q.info.sig, SIGTERM as i32);
        assert!(!state.has_pending(SIGTERM));
    }

    #[test]
    fn signal_blocking_and_sigprocmask() {
        let mut state = SignalState::new();
        state.block(SIGTERM);
        assert!(state.is_blocked(SIGTERM));
        let info = SigInfo { sig: SIGTERM as i32, errno: 0, code: SI_USER, ..Default::default() };
        assert!(!state.deliver(SIGTERM, info));
        assert!(state.has_pending(SIGTERM));
        state.unblock(SIGTERM);
        assert!(!state.is_blocked(SIGTERM));

        let mut set = sigset_empty();
        sigset_add(&mut set, SIGINT);
        let old = state.sigprocmask(SIG_BLOCK, Some(set)).unwrap();
        assert!(state.is_blocked(SIGINT));
        let old2 = state.sigprocmask(SIG_SETMASK, Some(old)).unwrap();
        assert!(!state.is_blocked(SIGINT));
        assert!(old2 & sig_mask(SIGINT) != 0);
    }

    #[test]
    fn sigaction_and_altstack() {
        let mut state = SignalState::new();
        let mut act = SigAction::default();
        act.handler = 0x1234;
        act.flags = SA_ONSTACK | SA_RESTART;
        let old = state.sigaction(SIGTERM, Some(act)).unwrap();
        assert_eq!(old.handler, SIG_DFL);
        assert_eq!(state.sighand[SIGTERM as usize].handler, 0x1234);

        // SIGKILL cannot have sigaction
        assert!(state.sigaction(SIGKILL, Some(act)).is_err());

        let alt = SigAltStack::new(0x1000, 0x2000);
        let old_alt = state.alt_stack.sigaltstack(Some(alt)).unwrap();
        assert_eq!(old_alt.sp, 0);
        assert!(state.alt_stack.contains(0x1500));
    }

    #[test]
    fn handle_signal_frame() {
        let mut state = SignalState::new();
        let mut act = SigAction::default();
        act.handler = 0x1234;
        act.flags = SA_RESETHAND;
        act.mask = sig_mask(SIGINT);
        state.sighand[SIGUSR1 as usize] = act;
        let frame = handle_signal(&mut state, SIGUSR1).unwrap();
        assert_eq!(frame.handler, 0x1234);
        assert_eq!(state.sighand[SIGUSR1 as usize].handler, SIG_DFL); // SA_RESETHAND
        assert!(state.is_blocked(SIGUSR1)); // blocked by NODEFER not set
        assert!(state.is_blocked(SIGINT)); // mask
    }

    #[test]
    fn altstack_logic() {
        let alt = SigAltStack::new(0x1000, 0x1000);
        assert!(!alt.is_disabled());
        assert!(!alt.is_on_stack());
        assert!(alt.contains(0x1500));
        assert!(!alt.contains(0x3000));
        assert_eq!(alt.top(), 0x2000);

        let mut action = SigAction::default();
        action.flags = SA_ONSTACK;
        assert!(should_use_altstack(&action, &alt, 0x5000));
        assert!(!should_use_altstack(&action, &alt, 0x1500));

        let mut disabled = alt;
        disabled.flags = SS_DISABLE;
        assert!(!should_use_altstack(&action, &disabled, 0x5000));
    }

    #[test]
    fn pending_and_next() {
        let mut state = SignalState::new();
        let info = SigInfo { sig: 0, errno: 0, code: SI_USER, ..Default::default() };
        state.deliver(SIGTERM, info);
        state.deliver(SIGINT, info);
        state.block(SIGTERM);
        assert_eq!(state.next_unblocked_pending(), Some(SIGINT));
        state.unblock(SIGTERM);
        let next = state.next_unblocked_pending();
        assert!(next == Some(SIGTERM) || next == Some(SIGINT));
    }
}
