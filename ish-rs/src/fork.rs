//! `kernel/fork.c` — clone flags and task creation constants.

/// `CSIGNAL_`
pub const CSIGNAL: u32 = 0x000000ff;
pub const CLONE_VM: u32 = 0x00000100;
pub const CLONE_FS: u32 = 0x00000200;
pub const CLONE_FILES: u32 = 0x00000400;
pub const CLONE_SIGHAND: u32 = 0x00000800;
pub const CLONE_PTRACE: u32 = 0x00002000;
pub const CLONE_VFORK: u32 = 0x00004000;
pub const CLONE_PARENT: u32 = 0x00008000;
pub const CLONE_THREAD: u32 = 0x00010000;
pub const CLONE_NEWNS: u32 = 0x00020000;
pub const CLONE_SYSVSEM: u32 = 0x00040000;
pub const CLONE_SETTLS: u32 = 0x00080000;
pub const CLONE_PARENT_SETTID: u32 = 0x00100000;
pub const CLONE_CHILD_CLEARTID: u32 = 0x00200000;
pub const CLONE_DETACHED: u32 = 0x00400000;
pub const CLONE_UNTRACED: u32 = 0x00800000;
pub const CLONE_CHILD_SETTID: u32 = 0x01000000;
pub const CLONE_NEWCGROUP: u32 = 0x02000000;
pub const CLONE_NEWUTS: u32 = 0x04000000;
pub const CLONE_NEWIPC: u32 = 0x08000000;
pub const CLONE_NEWUSER: u32 = 0x10000000;
pub const CLONE_NEWPID: u32 = 0x20000000;
pub const CLONE_NEWNET: u32 = 0x40000000;
pub const CLONE_IO: u32 = 0x80000000;

pub const IMPLEMENTED_FLAGS: u32 = CLONE_VM
    | CLONE_FILES
    | CLONE_FS
    | CLONE_SIGHAND
    | CLONE_SYSVSEM
    | CLONE_VFORK
    | CLONE_THREAD
    | CLONE_SETTLS
    | CLONE_CHILD_SETTID
    | CLONE_PARENT_SETTID
    | CLONE_CHILD_CLEARTID
    | CLONE_DETACHED;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct CloneArgs {
    pub flags: u32,
    pub stack: u32,
    pub ptid: u32,
    pub tls: u32,
    pub ctid: u32,
}

impl CloneArgs {
    pub fn new(flags: u32, stack: u32, ptid: u32, tls: u32, ctid: u32) -> Self {
        Self {
            flags,
            stack,
            ptid,
            tls,
            ctid,
        }
    }

    pub fn is_thread(&self) -> bool {
        (self.flags & CLONE_THREAD) != 0
    }

    pub fn is_vfork(&self) -> bool {
        (self.flags & CLONE_VFORK) != 0
    }

    pub fn has_unimplemented_flags(&self) -> bool {
        (self.flags & !IMPLEMENTED_FLAGS & !CSIGNAL) != 0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn clone_flags_match_c() {
        assert_eq!(CLONE_VM, 0x100);
        assert_eq!(CLONE_THREAD, 0x10000);
        assert_eq!(CSIGNAL, 0xff);
    }

    #[test]
    fn clone_args_helpers() {
        let args = CloneArgs::new(CLONE_VM | CLONE_THREAD, 0, 0, 0, 0);
        assert!(args.is_thread());
        assert!(!args.is_vfork());
        assert!(!args.has_unimplemented_flags());

        let args2 = CloneArgs::new(CLONE_NEWNS, 0, 0, 0, 0);
        assert!(args2.has_unimplemented_flags());
    }
}
