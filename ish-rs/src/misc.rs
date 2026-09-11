//! `kernel/misc.c` — small process-control syscalls.
//!
//! This file ports iSH's `prctl`, `arch_prctl`, and reboot policy.  The calls
//! are intentionally narrow because the original C is narrow: keepcaps is a
//! success stub, only `PR_SET_NAME` has state, all `arch_prctl` requests fail,
//! and reboot validates magic values but never asks the host to reboot.

use crate::getset::{EFAULT, EINVAL, EPERM};
use crate::task::{Addr, Task};

/// `PRCTL_SET_KEEPCAPS_`.
pub const PR_SET_KEEPCAPS: u32 = 8;
/// `PRCTL_SET_NAME_`.
pub const PR_SET_NAME: u32 = 15;

/// `REBOOT_MAGIC1` interpreted as iSH's signed `int_t`.
pub const REBOOT_MAGIC1: i32 = 0xfee1_deadu32 as i32;
/// `REBOOT_MAGIC2`.
pub const REBOOT_MAGIC2: i32 = 672_274_793;
/// `REBOOT_MAGIC2A`.
pub const REBOOT_MAGIC2A: i32 = 85_072_278;
/// `REBOOT_MAGIC2B`.
pub const REBOOT_MAGIC2B: i32 = 369_367_448;
/// `REBOOT_MAGIC2C`.
pub const REBOOT_MAGIC2C: i32 = 537_993_216;
/// `REBOOT_CMD_CAD_OFF`.
pub const REBOOT_CMD_CAD_OFF: i32 = 0;
/// `REBOOT_CMD_CAD_ON` interpreted as an iSH signed `int_t`.
pub const REBOOT_CMD_CAD_ON: i32 = 0x89ab_cdefu32 as i32;

/// `sys_prctl`.
///
/// `arg3`–`arg5` are deliberately ignored just like their `UNUSED` C
/// parameters.  `PR_SET_NAME` reads at most 15 guest bytes, forces byte 15 to
/// NUL, then uses C `strcpy` semantics to update `task.comm`.
pub fn sys_prctl(
    task: &mut Task,
    option: u32,
    arg2: Addr,
    _arg3: u32,
    _arg4: u32,
    _arg5: u32,
) -> i32 {
    match option {
        PR_SET_KEEPCAPS => 0,
        PR_SET_NAME => {
            let mut name = [0u8; 16];
            if task.user_read_string(arg2, &mut name[..15]).is_err() {
                return EFAULT;
            }
            // `name[sizeof(name) - 1] = '\0'` in C, including the case where
            // the guest supplied fifteen non-NUL bytes.
            name[15] = 0;
            task.copy_comm_cstr(&name);
            0
        }
        _ => EINVAL,
    }
}

/// `sys_arch_prctl`.
pub fn sys_arch_prctl(_task: &Task, _code: i32, _addr: Addr) -> i32 {
    EINVAL
}

/// `sys_reboot`.
///
/// This validates the guest ABI and returns exactly what iSH returns.  It never
/// reboots the host: the two accepted CAD commands are no-ops in the C source.
pub fn sys_reboot(task: &Task, magic: i32, magic2: i32, cmd: i32) -> i32 {
    if !task.is_superuser() {
        return EPERM;
    }
    if magic != REBOOT_MAGIC1
        || !matches!(
            magic2,
            REBOOT_MAGIC2 | REBOOT_MAGIC2A | REBOOT_MAGIC2B | REBOOT_MAGIC2C
        )
    {
        return EINVAL;
    }
    match cmd {
        REBOOT_CMD_CAD_ON | REBOOT_CMD_CAD_OFF => 0,
        _ => EPERM,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::P_RWX;
    use crate::mmu::PAGE_BITS;
    use crate::task::TaskTable;

    fn task_with_memory() -> Task {
        let mut table = TaskTable::bootstrap();
        let task = table.current_mut().unwrap();
        task.mm_mut().unwrap().mem.map_nothing(0x100, 1, P_RWX);
        Task {
            cpu: task.cpu.clone(),
            mm: task.mm.clone(),
            pid: task.pid,
            tgid: task.tgid,
            group_id: task.group_id,
            credentials: task.credentials,
            ngroups: task.ngroups,
            groups: task.groups,
            comm: task.comm,
            did_exec: task.did_exec,
            parent: task.parent,
            children: task.children.clone(),
            blocked: task.blocked,
            pending: task.pending,
            waiting: task.waiting,
            saved_mask: task.saved_mask,
            has_saved_mask: task.has_saved_mask,
            clear_tid: task.clear_tid,
            robust_list: task.robust_list,
            exit_code: task.exit_code,
            zombie: task.zombie,
            exiting: task.exiting,
        }
    }

    #[test]
    fn prctl_keepcaps_is_a_success_stub_and_unknown_options_are_einval() {
        let mut task = task_with_memory();
        assert_eq!(sys_prctl(&mut task, PR_SET_KEEPCAPS, 0, 1, 2, 3), 0);
        assert_eq!(sys_prctl(&mut task, 99, 0, 0, 0, 0), EINVAL);
        assert_eq!(sys_arch_prctl(&task, 0x1002, 0x4444), EINVAL);
    }

    #[test]
    fn prctl_set_name_copies_through_nul_and_keeps_the_old_tail() {
        let mut task = task_with_memory();
        task.comm = *b"abcdefghijklmnop";
        let addr = 0x100 << PAGE_BITS;
        task.user_write(addr, b"shell\0ignored").unwrap();
        assert_eq!(sys_prctl(&mut task, PR_SET_NAME, addr, 0, 0, 0), 0);
        assert_eq!(&task.comm[..6], b"shell\0");
        assert_eq!(&task.comm[6..], b"ghijklmnop");
    }

    #[test]
    fn prctl_set_name_forces_a_terminator_after_fifteen_bytes() {
        let mut task = task_with_memory();
        let addr = 0x100 << PAGE_BITS;
        task.user_write(addr, b"fifteen-bytes!x").unwrap();
        assert_eq!(b"fifteen-bytes!x".len(), 15);
        assert_eq!(sys_prctl(&mut task, PR_SET_NAME, addr, 0, 0, 0), 0);
        assert_eq!(&task.comm[..15], b"fifteen-bytes!x");
        assert_eq!(task.comm[15], 0);
    }

    #[test]
    fn prctl_set_name_faults_for_a_null_or_unmapped_guest_address() {
        let mut task = task_with_memory();
        assert_eq!(sys_prctl(&mut task, PR_SET_NAME, 0, 0, 0, 0), EFAULT);
        assert_eq!(
            sys_prctl(&mut task, PR_SET_NAME, 0x200 << PAGE_BITS, 0, 0, 0),
            EFAULT
        );
    }

    #[test]
    fn reboot_checks_privilege_magics_and_command() {
        let mut task = task_with_memory();
        assert_eq!(
            sys_reboot(&task, REBOOT_MAGIC1, REBOOT_MAGIC2, REBOOT_CMD_CAD_OFF),
            0
        );
        assert_eq!(
            sys_reboot(&task, REBOOT_MAGIC1, REBOOT_MAGIC2A, REBOOT_CMD_CAD_ON),
            0
        );
        assert_eq!(
            sys_reboot(&task, 0, REBOOT_MAGIC2, REBOOT_CMD_CAD_OFF),
            EINVAL
        );
        assert_eq!(
            sys_reboot(&task, REBOOT_MAGIC1, 0, REBOOT_CMD_CAD_OFF),
            EINVAL
        );
        assert_eq!(sys_reboot(&task, REBOOT_MAGIC1, REBOOT_MAGIC2, 123), EPERM);
        task.credentials.euid = 1000;
        assert_eq!(
            sys_reboot(&task, REBOOT_MAGIC1, REBOOT_MAGIC2, REBOOT_CMD_CAD_OFF),
            EPERM
        );
    }
}
