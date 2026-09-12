//! `kernel/ptrace.h` + `kernel/ptrace.c` — ptrace constants and helpers.

/// PTRACE requests
pub const PTRACE_TRACEME: u32 = 0;
pub const PTRACE_PEEKTEXT: u32 = 1;
pub const PTRACE_PEEKDATA: u32 = 2;
pub const PTRACE_PEEKUSER: u32 = 3;
pub const PTRACE_POKETEXT: u32 = 4;
pub const PTRACE_POKEDATA: u32 = 5;
pub const PTRACE_POKEUSER: u32 = 6;
pub const PTRACE_CONT: u32 = 7;
pub const PTRACE_KILL: u32 = 8;
pub const PTRACE_SINGLESTEP: u32 = 9;
pub const PTRACE_GETREGS: u32 = 12;
pub const PTRACE_SETREGS: u32 = 13;
pub const PTRACE_GETFPREGS: u32 = 14;
pub const PTRACE_SETFPREGS: u32 = 15;
pub const PTRACE_ATTACH: u32 = 16;
pub const PTRACE_DETACH: u32 = 17;
pub const PTRACE_GETFPXREGS: u32 = 18;
pub const PTRACE_SETFPXREGS: u32 = 19;
pub const PTRACE_SYSCALL: u32 = 24;
pub const PTRACE_SETOPTIONS: u32 = 0x4200;
pub const PTRACE_GETEVENTMSG: u32 = 0x4201;

/// `struct user_regs_struct_` guest ABI (i386)
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct UserRegsStruct {
    pub ebx: u32,
    pub ecx: u32,
    pub edx: u32,
    pub esi: u32,
    pub edi: u32,
    pub ebp: u32,
    pub eax: u32,
    pub orig_eax: u32,
    pub eip: u32,
    pub eflags: u32,
    pub esp: u32,
}

/// Ptrace state
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct PtraceState {
    pub traced: bool,
    pub stopped: bool,
}

impl PtraceState {
    pub fn new() -> Self { Self::default() }
    pub fn is_traced(&self) -> bool { self.traced }
    pub fn is_stopped(&self) -> bool { self.stopped }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ptrace_constants_match_c() {
        assert_eq!(PTRACE_TRACEME, 0);
        assert_eq!(PTRACE_PEEKTEXT, 1);
        assert_eq!(PTRACE_CONT, 7);
        assert_eq!(PTRACE_KILL, 8);
    }

    #[test]
    fn ptrace_state() {
        let mut state = PtraceState::new();
        assert!(!state.is_traced());
        state.traced = true;
        assert!(state.is_traced());
    }
}
