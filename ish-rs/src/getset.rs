//! `kernel/getset.c` — process identity and credential syscalls.
//!
//! This module is deliberately small and literal: these syscalls read or
//! update fields that belong to [`crate::task::Task`].  Guest buffers still go
//! through `kernel/user.c`'s port, so a multi-word getter retains C's partial
//! write behavior when a later address faults.

use crate::group::ESRCH;
use crate::task::{Addr, Gid, Pid, Task, TaskTable, Uid, ADDR_NO_RANDOMIZE, MAX_GROUPS};

/// `_EPERM` from `kernel/errno.h`.
pub const EPERM: i32 = -1;
/// `_EFAULT` from `kernel/errno.h`.
pub const EFAULT: i32 = -14;
/// `_EINVAL` from `kernel/errno.h`.
pub const EINVAL: i32 = -22;

/// `sys_getpid`.
pub fn sys_getpid(task: &Task) -> Pid {
    task.tgid
}

/// `sys_gettid`.
pub fn sys_gettid(task: &Task) -> Pid {
    task.pid
}

/// `sys_getppid`.
pub fn sys_getppid(task: &Task) -> Pid {
    task.parent.unwrap_or(0)
}

/// `sys_getuid32`.
pub fn sys_getuid32(task: &Task) -> Uid {
    task.credentials.uid
}

/// `sys_getuid` — the old ABI truncates to 16 bits.
pub fn sys_getuid(task: &Task) -> Uid {
    task.credentials.uid & 0xffff
}

/// `sys_geteuid32`.
pub fn sys_geteuid32(task: &Task) -> Uid {
    task.credentials.euid
}

/// `sys_geteuid` — the old ABI truncates to 16 bits.
pub fn sys_geteuid(task: &Task) -> Uid {
    task.credentials.euid & 0xffff
}

/// `sys_setuid`.
pub fn sys_setuid(task: &mut Task, uid: Uid) -> i32 {
    if task.is_superuser() {
        task.credentials.uid = uid;
        task.credentials.suid = uid;
    } else if uid != task.credentials.uid && uid != task.credentials.suid {
        return EPERM;
    }
    task.credentials.euid = uid;
    0
}

/// `sys_setresuid`.
pub fn sys_setresuid(task: &mut Task, ruid: Uid, euid: Uid, suid: Uid) -> i32 {
    const UNCHANGED: Uid = u32::MAX; // `(uid_t_) -1`
    if !task.is_superuser()
        && ((ruid != UNCHANGED
            && ruid != task.credentials.uid
            && ruid != task.credentials.euid
            && ruid != task.credentials.suid)
            || (euid != UNCHANGED
                && euid != task.credentials.uid
                && euid != task.credentials.euid
                && euid != task.credentials.suid)
            || (suid != UNCHANGED
                && suid != task.credentials.uid
                && suid != task.credentials.euid
                && suid != task.credentials.suid))
    {
        return EPERM;
    }

    if ruid != UNCHANGED {
        task.credentials.uid = ruid;
    }
    if euid != UNCHANGED {
        task.credentials.euid = euid;
    }
    if suid != UNCHANGED {
        task.credentials.suid = suid;
    }
    0
}

/// `sys_getresuid`.
///
/// Values are captured before the first guest write, like the C arguments to
/// `user_put`; a fault on the second or third write leaves the earlier words in
/// memory and returns `_EFAULT`.
pub fn sys_getresuid(task: &mut Task, ruid_addr: Addr, euid_addr: Addr, suid_addr: Addr) -> i32 {
    let (ruid, euid, suid) = (
        task.credentials.uid,
        task.credentials.euid,
        task.credentials.suid,
    );
    if task.user_write(ruid_addr, &ruid.to_le_bytes()).is_err() {
        return EFAULT;
    }
    if task.user_write(euid_addr, &euid.to_le_bytes()).is_err() {
        return EFAULT;
    }
    if task.user_write(suid_addr, &suid.to_le_bytes()).is_err() {
        return EFAULT;
    }
    0
}

/// `sys_setreuid`.
pub fn sys_setreuid(task: &mut Task, ruid: Uid, euid: Uid) -> i32 {
    sys_setresuid(task, ruid, euid, u32::MAX)
}

/// `sys_getgid32`.
pub fn sys_getgid32(task: &Task) -> Gid {
    task.credentials.gid
}

/// `sys_getgid` — the old ABI truncates to 16 bits.
pub fn sys_getgid(task: &Task) -> Gid {
    task.credentials.gid & 0xffff
}

/// `sys_getegid32`.
pub fn sys_getegid32(task: &Task) -> Gid {
    task.credentials.egid
}

/// `sys_getegid` — the old ABI truncates to 16 bits.
pub fn sys_getegid(task: &Task) -> Gid {
    task.credentials.egid & 0xffff
}

/// `sys_setgid`.
pub fn sys_setgid(task: &mut Task, gid: Gid) -> i32 {
    if task.is_superuser() {
        task.credentials.gid = gid;
        task.credentials.sgid = gid;
    } else if gid != task.credentials.gid && gid != task.credentials.sgid {
        return EPERM;
    }
    task.credentials.egid = gid;
    0
}

/// `sys_setresgid`.
pub fn sys_setresgid(task: &mut Task, rgid: Gid, egid: Gid, sgid: Gid) -> i32 {
    const UNCHANGED: Gid = u32::MAX; // `(uid_t_) -1`
    if !task.is_superuser()
        && ((rgid != UNCHANGED
            && rgid != task.credentials.gid
            && rgid != task.credentials.egid
            && rgid != task.credentials.sgid)
            || (egid != UNCHANGED
                && egid != task.credentials.gid
                && egid != task.credentials.egid
                && egid != task.credentials.sgid)
            || (sgid != UNCHANGED
                && sgid != task.credentials.gid
                && sgid != task.credentials.egid
                && sgid != task.credentials.sgid))
    {
        return EPERM;
    }

    if rgid != UNCHANGED {
        task.credentials.gid = rgid;
    }
    if egid != UNCHANGED {
        task.credentials.egid = egid;
    }
    if sgid != UNCHANGED {
        task.credentials.sgid = sgid;
    }
    0
}

/// `sys_getresgid`.
pub fn sys_getresgid(task: &mut Task, rgid_addr: Addr, egid_addr: Addr, sgid_addr: Addr) -> i32 {
    let (rgid, egid, sgid) = (
        task.credentials.gid,
        task.credentials.egid,
        task.credentials.sgid,
    );
    if task.user_write(rgid_addr, &rgid.to_le_bytes()).is_err() {
        return EFAULT;
    }
    if task.user_write(egid_addr, &egid.to_le_bytes()).is_err() {
        return EFAULT;
    }
    if task.user_write(sgid_addr, &sgid.to_le_bytes()).is_err() {
        return EFAULT;
    }
    0
}

/// `sys_setregid`.
pub fn sys_setregid(task: &mut Task, rgid: Gid, egid: Gid) -> i32 {
    sys_setresgid(task, rgid, egid, u32::MAX)
}

/// `sys_getgroups`.
pub fn sys_getgroups(task: &mut Task, size: u32, list: Addr) -> i32 {
    if size == 0 {
        return task.ngroups as i32;
    }
    if (size as usize) < task.ngroups {
        return EINVAL;
    }
    let mut bytes = Vec::with_capacity(task.ngroups * 4);
    for group in task.groups[..task.ngroups].iter().copied() {
        bytes.extend_from_slice(&group.to_le_bytes());
    }
    if task.user_write(list, &bytes).is_err() {
        return EFAULT;
    }
    task.ngroups as i32
}

/// `sys_setgroups`.
///
/// The C feeds its fixed `groups` array directly to `user_read`.  The temporary
/// byte array here is copied back whether the read succeeds or faults so the
/// partially written prefix remains visible, then `ngroups` changes only after
/// a successful complete read.
pub fn sys_setgroups(task: &mut Task, size: u32, list: Addr) -> i32 {
    let size = size as usize;
    if size > MAX_GROUPS {
        return EINVAL;
    }
    let mut all_groups = task.groups_as_le_bytes();
    let result = task.user_read(list, &mut all_groups[..size * 4]);
    task.set_groups_from_le_bytes(&all_groups);
    if result.is_err() {
        return EFAULT;
    }
    task.ngroups = size;
    0
}

/// `sys_capget` — the C implementation is intentionally a no-op.
pub fn sys_capget(_task: &Task, _header_addr: Addr, _data_addr: Addr) -> i32 {
    0
}

/// `sys_capset` — the C implementation is intentionally a no-op.
pub fn sys_capset(_task: &Task, _header_addr: Addr, _data_addr: Addr) -> i32 {
    0
}

/// `sys_personality`, using the selected `current` task.
///
/// iSH exposes only `ADDR_NO_RANDOMIZE`; it reports the existing personality
/// for a query or that exact request and never changes the stored word.
pub fn sys_personality(table: &TaskTable, persona: u32) -> i32 {
    let Some(current) = table.current() else {
        return ESRCH;
    };
    let Some(group_id) = current.group_id else {
        return ESRCH;
    };
    let Some(group) = table.thread_group(group_id) else {
        return ESRCH;
    };
    if persona == u32::MAX || persona == ADDR_NO_RANDOMIZE {
        group.personality as i32
    } else {
        EINVAL
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::P_RWX;
    use crate::mmu::PAGE_BITS;
    use crate::task::TaskTable;

    fn mapped_task() -> Task {
        let table = TaskTable::bootstrap();
        let task = table.current().unwrap().clone_for_test();
        task.mm_mut().unwrap().mem.map_nothing(0x100, 1, P_RWX);
        task
    }

    trait CloneForTest {
        fn clone_for_test(&self) -> Task;
    }
    impl CloneForTest for Task {
        fn clone_for_test(&self) -> Task {
            // Tests need an independently mutable task, not a second entry in
            // the same TaskTable. This is the public-state equivalent of the
            // shallow C task copy and is intentionally local to tests.
            Task {
                cpu: self.cpu.clone(),
                mm: self.mm.clone(),
                pid: self.pid,
                tgid: self.tgid,
                group_id: self.group_id,
                credentials: self.credentials,
                ngroups: self.ngroups,
                groups: self.groups,
                comm: self.comm,
                did_exec: self.did_exec,
                parent: self.parent,
                children: self.children.clone(),
                blocked: self.blocked,
                pending: self.pending,
                waiting: self.waiting,
                saved_mask: self.saved_mask,
                has_saved_mask: self.has_saved_mask,
                clear_tid: self.clear_tid,
                robust_list: self.robust_list,
                exit_code: self.exit_code,
                zombie: self.zombie,
                exiting: self.exiting,
            }
        }
    }

    #[test]
    fn identity_getters_keep_the_old_16_bit_abi_truncation() {
        let mut task = mapped_task();
        task.credentials.uid = 0xabcd_1234;
        task.credentials.euid = 0x7fff_ffff;
        task.credentials.gid = 0xfedc_5678;
        task.credentials.egid = 0x1111_2222;
        assert_eq!(sys_getpid(&task), 1);
        assert_eq!(sys_gettid(&task), 1);
        assert_eq!(sys_getppid(&task), 0);
        assert_eq!(sys_getuid32(&task), 0xabcd_1234);
        assert_eq!(sys_getuid(&task), 0x1234);
        assert_eq!(sys_geteuid(&task), 0xffff);
        assert_eq!(sys_getgid(&task), 0x5678);
        assert_eq!(sys_getegid(&task), 0x2222);
    }

    #[test]
    fn uid_and_gid_permission_rules_match_the_c_branches() {
        let mut task = mapped_task();
        assert_eq!(sys_setuid(&mut task, 1000), 0);
        assert_eq!(task.credentials.uid, 1000);
        assert_eq!(task.credentials.suid, 1000);
        assert_eq!(task.credentials.euid, 1000);
        // Non-root may select its real/saved uid but not an arbitrary one.
        assert_eq!(sys_setuid(&mut task, 1000), 0);
        assert_eq!(sys_setuid(&mut task, 2000), EPERM);
        assert_eq!(sys_setresuid(&mut task, u32::MAX, 1000, u32::MAX), 0);
        assert_eq!(sys_setresuid(&mut task, 2000, u32::MAX, u32::MAX), EPERM);

        task.credentials.euid = 0;
        assert_eq!(sys_setgid(&mut task, 3000), 0);
        assert_eq!(task.credentials.gid, 3000);
        assert_eq!(task.credentials.sgid, 3000);
        assert_eq!(task.credentials.egid, 3000);
        assert_eq!(sys_setresgid(&mut task, 4000, 5000, 6000), 0);
        assert_eq!(
            (
                task.credentials.gid,
                task.credentials.egid,
                task.credentials.sgid
            ),
            (4000, 5000, 6000)
        );
    }

    #[test]
    fn res_getters_write_each_word_in_order_and_keep_a_prefix_on_fault() {
        let mut task = mapped_task();
        task.credentials.uid = 0x1111_1111;
        task.credentials.euid = 0x2222_2222;
        task.credentials.suid = 0x3333_3333;
        let page = 0x100 << PAGE_BITS;
        assert_eq!(
            sys_getresuid(&mut task, page, page + 4, 0x200 << PAGE_BITS),
            EFAULT
        );
        let mm = task.mm_mut().unwrap();
        let first = mm.mem.pt(0x100).unwrap();
        assert_eq!(
            &first.data.bytes[first.offset..first.offset + 8],
            &[0x11, 0x11, 0x11, 0x11, 0x22, 0x22, 0x22, 0x22]
        );
    }

    #[test]
    fn groups_observe_size_rules_little_endian_values_and_partial_input() {
        let mut task = mapped_task();
        let page = 0x100 << PAGE_BITS;
        let input = [3u32, 0x1122_3344, u32::MAX];
        let mut bytes = Vec::new();
        for value in input {
            bytes.extend_from_slice(&value.to_le_bytes());
        }
        task.user_write(page, &bytes).unwrap();
        assert_eq!(sys_setgroups(&mut task, 3, page), 0);
        assert_eq!(task.ngroups, 3);
        assert_eq!(&task.groups[..3], &input);
        assert_eq!(sys_getgroups(&mut task, 0, 0), 3);
        assert_eq!(sys_getgroups(&mut task, 2, page), EINVAL);

        // getgroups writes exactly ngroups words.
        task.user_write(page + 32, &[0xaa; 16]).unwrap();
        assert_eq!(sys_getgroups(&mut task, 3, page + 32), 3);
        {
            let mm = task.mm_mut().unwrap();
            let entry = mm.mem.pt(0x100).unwrap();
            assert_eq!(
                &entry.data.bytes[entry.offset + 32..entry.offset + 44],
                bytes.as_slice()
            );
            assert_eq!(
                &entry.data.bytes[entry.offset + 44..entry.offset + 48],
                &[0xaa; 4]
            );
        }

        // A cross-page setgroups fault updates the first word in C's backing
        // array but leaves ngroups unchanged.
        task.groups[..3].copy_from_slice(&[10, 20, 30]);
        task.ngroups = 3;
        let end = page + 4092;
        task.user_write(end, &0xaabb_ccddu32.to_le_bytes()).unwrap();
        assert_eq!(sys_setgroups(&mut task, 2, end), EFAULT);
        assert_eq!(task.groups[0], 0xaabb_ccdd);
        assert_eq!(task.groups[1], 20);
        assert_eq!(task.ngroups, 3);
    }

    #[test]
    fn personality_only_reports_the_supported_value() {
        let table = TaskTable::bootstrap();
        assert_eq!(sys_personality(&table, u32::MAX), ADDR_NO_RANDOMIZE as i32);
        assert_eq!(
            sys_personality(&table, ADDR_NO_RANDOMIZE),
            ADDR_NO_RANDOMIZE as i32
        );
        assert_eq!(sys_personality(&table, 0), EINVAL);
    }

    #[test]
    fn capability_stubs_succeed_without_touching_guest_memory() {
        let task = mapped_task();
        assert_eq!(sys_capget(&task, 0, 0), 0);
        assert_eq!(sys_capset(&task, 0, 0), 0);
    }
}
