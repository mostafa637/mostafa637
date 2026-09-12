//! `kernel/ptrace.h` — ptrace constants and structures.
//!
//! Leaf header: only depends on `misc.h` types. This ports the constants and
//! the `user_regs_struct_`, `user_fpregs_struct_`, and `user_` layouts.

/// `PTRACE_TRACEME_`
pub const PTRACE_TRACEME: u32 = 0;
pub const PTRACE_PEEKTEXT: u32 = 1;
pub const PTRACE_PEEKDATA: u32 = 2;
pub const PTRACE_PEEKUSER: u32 = 3;
pub const PTRACE_POKETEXT: u32 = 4;
pub const PTRACE_POKEDATA: u32 = 5;
pub const PTRACE_CONT: u32 = 7;
pub const PTRACE_KILL: u32 = 8;
pub const PTRACE_SINGLESTEP: u32 = 9;
pub const PTRACE_GETREGS: u32 = 12;
pub const PTRACE_SETREGS: u32 = 13;
pub const PTRACE_GETFPREGS: u32 = 14;
pub const PTRACE_SETFPREGS: u32 = 15;
pub const PTRACE_SETOPTIONS: u32 = 0x4200;
pub const PTRACE_GETSIGINFO: u32 = 0x4202;

pub const PTRACE_EVENT_FORK: u32 = 1;

/// `struct user_regs_struct_` — i386 general registers as seen by ptrace.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct UserRegsStruct {
    pub ebx: u32,
    pub ecx: u32,
    pub edx: u32,
    pub esi: u32,
    pub edi: u32,
    pub ebp: u32,
    pub eax: u32,
    pub xds: u32,
    pub xes: u32,
    pub xfs: u32,
    pub xgs: u32,
    pub orig_eax: u32,
    pub eip: u32,
    pub xcs: u32,
    pub eflags: u32,
    pub esp: u32,
    pub xss: u32,
}

/// `struct user_fpregs_struct_`
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct UserFpregsStruct {
    pub cwd: u32,
    pub swd: u32,
    pub twd: u32,
    pub fip: u32,
    pub fcs: u32,
    pub foo: u32,
    pub fos: u32,
    pub st_space: [u32; 20],
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ptrace_constants_match_c() {
        assert_eq!(PTRACE_TRACEME, 0);
        assert_eq!(PTRACE_PEEKTEXT, 1);
        assert_eq!(PTRACE_GETREGS, 12);
        assert_eq!(PTRACE_SETOPTIONS, 0x4200);
    }

    #[test]
    fn user_regs_is_68_bytes() {
        assert_eq!(core::mem::size_of::<UserRegsStruct>(), 17 * 4);
    }
}
