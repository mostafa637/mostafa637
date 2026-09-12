//! `kernel/fork.c` — clone flags and task creation full port.

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

pub const IMPLEMENTED_FLAGS: u32 = CLONE_VM | CLONE_FILES | CLONE_FS | CLONE_SIGHAND | CLONE_SYSVSEM | CLONE_VFORK | CLONE_THREAD | CLONE_SETTLS | CLONE_CHILD_SETTID | CLONE_PARENT_SETTID | CLONE_CHILD_CLEARTID | CLONE_DETACHED;

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
        Self { flags, stack, ptid, tls, ctid }
    }
    pub fn is_thread(&self) -> bool { (self.flags & CLONE_THREAD) != 0 }
    pub fn is_vfork(&self) -> bool { (self.flags & CLONE_VFORK) != 0 }
    pub fn has_unimplemented_flags(&self) -> bool { (self.flags & !IMPLEMENTED_FLAGS & !CSIGNAL) != 0 }
    pub fn exit_signal(&self) -> u32 { self.flags & CSIGNAL }
    pub fn is_valid(&self) -> bool {
        // CLONE_THREAD requires CLONE_SIGHAND and CLONE_VM, etc.
        if self.is_thread() {
            if (self.flags & CLONE_SIGHAND) == 0 { return false; }
            if (self.flags & CLONE_VM) == 0 { return false; }
        }
        if (self.flags & CLONE_SIGHAND) != 0 && (self.flags & CLONE_VM) == 0 {
            return false;
        }
        true
    }
}

pub fn should_copy_group(flags: u32) -> bool { (flags & CLONE_THREAD) == 0 }
pub fn should_copy_mm(flags: u32) -> bool { (flags & CLONE_VM) == 0 }
pub fn should_copy_files(flags: u32) -> bool { (flags & CLONE_FILES) == 0 }
pub fn should_copy_fs(flags: u32) -> bool { (flags & CLONE_FS) == 0 }
pub fn should_copy_sighand(flags: u32) -> bool { (flags & CLONE_SIGHAND) == 0 }

#[derive(Debug, Clone, Default)]
pub struct TaskCopy {
    pub copy_group: bool,
    pub copy_mm: bool,
    pub copy_files: bool,
    pub copy_fs: bool,
    pub copy_sighand: bool,
    pub is_thread: bool,
    pub is_vfork: bool,
    pub new_stack: Option<u32>,
    pub tls: Option<u32>,
}

impl TaskCopy {
    pub fn from_clone_args(args: &CloneArgs) -> Self {
        Self {
            copy_group: should_copy_group(args.flags),
            copy_mm: should_copy_mm(args.flags),
            copy_files: should_copy_files(args.flags),
            copy_fs: should_copy_fs(args.flags),
            copy_sighand: should_copy_sighand(args.flags),
            is_thread: args.is_thread(),
            is_vfork: args.is_vfork(),
            new_stack: if args.stack != 0 { Some(args.stack) } else { None },
            tls: if (args.flags & CLONE_SETTLS) != 0 { Some(args.tls) } else { None },
        }
    }
}

/// Fork result, matching C's do_fork
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ForkResult {
    Child(u32),
    Parent(u32),
    Error(i32),
}

pub fn do_fork_check(args: &CloneArgs) -> Result<TaskCopy, i32> {
    if args.has_unimplemented_flags() { return Err(-22); } // EINVAL
    if !args.is_valid() { return Err(-22); }
    Ok(TaskCopy::from_clone_args(args))
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
        let args = CloneArgs::new(CLONE_VM | CLONE_THREAD | CLONE_SIGHAND, 0, 0, 0, 0);
        assert!(args.is_thread());
        assert!(!args.is_vfork());
        assert!(!args.has_unimplemented_flags());
        assert_eq!(args.exit_signal(), 0);
        assert!(args.is_valid());

        let args2 = CloneArgs::new(CLONE_NEWNS, 0, 0, 0, 0);
        assert!(args2.has_unimplemented_flags());

        let args3 = CloneArgs::new(17, 0, 0, 0, 0);
        assert_eq!(args3.exit_signal(), 17);
    }

    #[test]
    fn copy_checks() {
        assert!(should_copy_group(0));
        assert!(!should_copy_group(CLONE_THREAD));
        assert!(should_copy_mm(0));
        assert!(!should_copy_mm(CLONE_VM));
    }

    #[test]
    fn clone_validation() {
        // THREAD without SIGHAND invalid
        let args = CloneArgs::new(CLONE_THREAD | CLONE_VM, 0, 0, 0, 0);
        assert!(!args.is_valid());
        // SIGHAND without VM invalid
        let args2 = CloneArgs::new(CLONE_SIGHAND, 0, 0, 0, 0);
        assert!(!args2.is_valid());
        // Valid thread
        let args3 = CloneArgs::new(CLONE_THREAD | CLONE_VM | CLONE_SIGHAND, 0, 0, 0, 0);
        assert!(args3.is_valid());
    }

    #[test]
    fn task_copy_from_args() {
        let args = CloneArgs::new(CLONE_VM | CLONE_FS, 0x1000, 0, 0x2000, 0);
        let copy = TaskCopy::from_clone_args(&args);
        assert!(!copy.copy_mm);
        assert!(!copy.copy_fs);
        assert!(copy.copy_files);
        assert_eq!(copy.new_stack, Some(0x1000));
        assert!(copy.tls.is_none());

        let args2 = CloneArgs::new(CLONE_VM | CLONE_SETTLS, 0, 0, 0x3000, 0);
        let copy2 = TaskCopy::from_clone_args(&args2);
        assert_eq!(copy2.tls, Some(0x3000));
    }

    #[test]
    fn do_fork_check_test() {
        let args = CloneArgs::new(0, 0, 0, 0, 0);
        assert!(do_fork_check(&args).is_ok());
        let args2 = CloneArgs::new(CLONE_NEWNS, 0, 0, 0, 0);
        assert!(do_fork_check(&args2).is_err());
    }
}
