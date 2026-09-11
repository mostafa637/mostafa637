//! `kernel/group.c` — sessions and process groups.
//!
//! A thread group is iSH's process-level object.  `group.c` supplies the small
//! but subtle namespace on top of it: `setpgid`, `getpgid`, `setsid`, and
//! `getsid`.  The C implementation expresses membership with intrusive lists
//! hanging off `struct pid`; [`crate::task::TaskTable`] owns equivalent indexed
//! sets, so the rules below are translated without raw pointers or a global
//! lock.
//!
//! `TaskTable` is single-threaded at this stage of the port.  Each public
//! method that spells `sys_*` reads its selected current task, just as the C
//! function reads `current`.  The `*_for` variants make the current PID explicit
//! for tests and for the future syscall dispatcher.

use crate::task::{Pid, TaskError, TaskTable};

/// `_EPERM` from `kernel/errno.h`.
pub const EPERM: i32 = -1;
/// `_ESRCH` from `kernel/errno.h`.
pub const ESRCH: i32 = -3;

impl TaskTable {
    /// `sys_setpgid`, with [`TaskTable::current_pid`] in place of C's
    /// thread-local `current`.
    pub fn sys_setpgid(&mut self, id: Pid, pgid: Pid) -> i32 {
        let Some(current) = self.current_pid() else {
            return ESRCH;
        };
        self.setpgid_for(current, id, pgid)
    }

    /// `sys_setpgid` with an explicit caller.
    ///
    /// This preserves all of the C's ordering and its deliberately narrow
    /// permission test: the target must be the caller itself or a *direct*
    /// child, a target process group must already contain a group in the same
    /// session, and a session leader may not create a process group.
    pub fn setpgid_for(&mut self, current: Pid, mut id: Pid, mut pgid: Pid) -> i32 {
        if self.pid_get_task(current).is_none() {
            return ESRCH;
        }
        if id == 0 {
            id = current;
        }
        if pgid == 0 {
            pgid = id;
        }

        // C deliberately uses pid_get(id)->task rather than pid_get_task(id),
        // so a zombie is still a legal target in this one syscall.
        let Some(target) = self.pid_slot(id).and_then(|slot| slot.task) else {
            return ESRCH;
        };
        let Some(target_group_id) = self.group_of_task(target) else {
            return ESRCH;
        };
        let Some(target_group) = self.thread_group(target_group_id) else {
            return ESRCH;
        };
        let target_sid = target_group.sid;
        let old_pgid = target_group.pgid;
        let leader = target_group.leader;

        // If joining someone else's pgrp, C takes list_first_entry on its
        // pgroup list and compares the session.  setpgid's own invariant means
        // every group in that list has the same session, so any deterministic
        // member is the equivalent safe representation.
        if id != pgid {
            let Some(group_id) = self
                .pid_slot(pgid)
                .and_then(|slot| slot.pgroups.iter().next().copied())
            else {
                return EPERM;
            };
            let Some(existing_group) = self.thread_group(group_id) else {
                return EPERM;
            };
            if target_sid != existing_group.sid {
                return EPERM;
            }
        }

        let Some(target_task) = self.task(target) else {
            return ESRCH;
        };
        if target != current && target_task.parent != Some(current) {
            return ESRCH;
        }
        // `if (tgroup->sid == tgroup->leader->pid) return _EPERM;`.
        if target_sid == leader {
            return EPERM;
        }

        if old_pgid != pgid {
            self.remove_pgroup_member(old_pgid, target_group_id);
            self.add_pgroup_member(pgid, target_group_id);
            self.group_mut(target_group_id)
                .expect("group was checked before the membership update")
                .pgid = pgid;
        }
        0
    }

    /// `sys_setpgrp`.
    pub fn sys_setpgrp(&mut self) -> i32 {
        self.sys_setpgid(0, 0)
    }

    /// `sys_getpgid`, using the selected current task.
    pub fn sys_getpgid(&self, pid: Pid) -> i32 {
        let Some(current) = self.current_pid() else {
            return ESRCH;
        };
        self.getpgid_for(current, pid)
    }

    /// `sys_getpgid` with an explicit caller.  `current` is used only for the
    /// `pid == 0` case, exactly as in C.
    pub fn getpgid_for(&self, current: Pid, pid: Pid) -> i32 {
        let target = if pid == 0 { current } else { pid };
        let Some(task) = self.pid_get_task(target) else {
            return ESRCH;
        };
        let Some(group_id) = task.group_id else {
            return ESRCH;
        };
        self.thread_group(group_id)
            .map_or(ESRCH, |group| group.pgid)
    }

    /// `sys_getpgrp`.
    pub fn sys_getpgrp(&self) -> i32 {
        self.sys_getpgid(0)
    }

    /// `task_leave_session`.
    ///
    /// TTY release is intentionally absent because the tty subsystem is not
    /// ported yet.  The membership mutation is the part that is observable to
    /// `pid_get`, `setpgid`, and a future tty owner; the later tty port can add
    /// its ownership transition at this one boundary.
    pub fn task_leave_session(&mut self, task_pid: Pid) -> Result<(), TaskError> {
        let group_id = self
            .group_of_task(task_pid)
            .ok_or(TaskError::MissingThreadGroup)?;
        let sid = self
            .thread_group(group_id)
            .ok_or(TaskError::MissingThreadGroup)?
            .sid;
        if sid != 0 {
            self.remove_session_member(sid, group_id);
        }
        Ok(())
    }

    /// `task_setsid`.
    ///
    /// The C helper takes any task pointer but uses its thread group's leader
    /// PID as the new SID.  It records the group under the *passed task's* PID
    /// in the PID table; unusual for a non-leader but reproduced here because
    /// it is visible through `pid_get` and future tty session lookup.
    pub fn task_setsid(&mut self, task_pid: Pid) -> i32 {
        let Some(group_id) = self.group_of_task(task_pid) else {
            return ESRCH;
        };
        let Some(group) = self.thread_group(group_id) else {
            return ESRCH;
        };
        let new_sid = group.leader;
        let old_sid = group.sid;
        let old_pgid = group.pgid;
        if old_pgid == new_sid || old_sid == new_sid {
            return EPERM;
        }

        // task_leave_session removes the old session-list link but deliberately
        // does not rewrite group.sid; the assignments below are the same order
        // as group.c after it returns.
        if old_sid != 0 {
            self.remove_session_member(old_sid, group_id);
        }
        self.add_session_member(task_pid, group_id);
        if old_pgid != 0 {
            self.remove_pgroup_member(old_pgid, group_id);
        }
        self.add_pgroup_member(task_pid, group_id);
        let group = self
            .group_mut(group_id)
            .expect("group was checked before session membership changes");
        group.sid = new_sid;
        group.pgid = new_sid;
        new_sid
    }

    /// `sys_setsid`.
    pub fn sys_setsid(&mut self) -> i32 {
        let Some(current) = self.current_pid() else {
            return ESRCH;
        };
        self.task_setsid(current)
    }

    /// `sys_getsid`.  iSH's C declaration takes no PID argument, and its
    /// syscall table casts the six-register calling convention to that function
    /// type, so any guest argument is intentionally ignored.
    pub fn sys_getsid(&self) -> i32 {
        let Some(current) = self.current() else {
            return ESRCH;
        };
        let Some(group_id) = current.group_id else {
            return ESRCH;
        };
        self.thread_group(group_id).map_or(ESRCH, |group| group.sid)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn process_child(table: &mut TaskTable) -> Pid {
        let child = table.create_task(Some(1)).unwrap();
        table.copy_thread_group_for(child).unwrap();
        child
    }

    #[test]
    fn bootstrap_is_a_session_and_process_group_leader() {
        let mut table = TaskTable::bootstrap();
        assert_eq!(table.sys_getsid(), 1);
        assert_eq!(table.sys_getpgid(0), 1);
        assert_eq!(table.sys_getpgrp(), 1);
        // A session leader cannot use setpgid to create or join a pgrp.
        assert_eq!(table.sys_setpgrp(), EPERM);
        assert_eq!(table.sys_setpgid(0, 0), EPERM);
    }

    #[test]
    fn child_can_make_a_process_group_then_becomes_ineligible_for_setsid() {
        let mut table = TaskTable::bootstrap();
        let child = process_child(&mut table);
        table.set_current(child).unwrap();
        assert_eq!(table.sys_getpid_for_test(), child);
        assert_eq!(table.sys_setpgrp(), 0);
        assert_eq!(table.sys_getpgrp(), child);
        // C tests pgid == leader before it touches session membership.
        assert_eq!(table.sys_setsid(), EPERM);
        assert_eq!(table.thread_group(child).unwrap().sid, 1);
    }

    #[test]
    fn setsid_moves_a_non_leader_group_out_of_its_old_session_and_pgrp() {
        let mut table = TaskTable::bootstrap();
        let child = process_child(&mut table);
        table.set_current(child).unwrap();
        assert_eq!(table.sys_setsid(), child);
        let group = table.thread_group(child).unwrap();
        assert_eq!(group.sid, child);
        assert_eq!(group.pgid, child);
        assert!(table.pid_slot(child).unwrap().sessions.contains(&child));
        assert!(table.pid_slot(child).unwrap().pgroups.contains(&child));
        assert!(!table.pid_slot(1).unwrap().sessions.contains(&child));
        assert!(!table.pid_slot(1).unwrap().pgroups.contains(&child));
        assert_eq!(table.sys_getsid(), child);
    }

    #[test]
    fn setpgid_requires_a_direct_child_and_a_group_in_the_same_session() {
        let mut table = TaskTable::bootstrap();
        let child_a = process_child(&mut table);
        let child_b = process_child(&mut table);

        // PID 1 can place child_a in a pgrp rooted at child_a.
        assert_eq!(table.setpgid_for(1, child_a, 0), 0);
        // child_b starts in the same session and can join that existing pgrp.
        assert_eq!(table.setpgid_for(1, child_b, child_a), 0);
        assert_eq!(table.getpgid_for(1, child_b), child_a);

        // An unrelated caller cannot move a sibling (C returns ESRCH, not
        // EPERM, for this permission failure).
        table.set_current(child_a).unwrap();
        assert_eq!(table.sys_setpgid(child_b, child_a), ESRCH);

        // Give child_a its own session.  PID 1 may no longer move child_b into
        // that process group because the sessions differ.
        // First put child_a back in PID 1's pgrp so setsid is permitted.
        table.set_current(1).unwrap();
        assert_eq!(table.sys_setpgid(child_a, 1), 0);
        table.set_current(child_a).unwrap();
        assert_eq!(table.sys_setsid(), child_a);
        table.set_current(1).unwrap();
        assert_eq!(table.sys_setpgid(child_b, child_a), EPERM);
    }

    #[test]
    fn getpgid_hides_zombies_but_setpgid_uses_the_pid_slot_like_c() {
        let mut table = TaskTable::bootstrap();
        let child = process_child(&mut table);
        table.mark_zombie(child, 0).unwrap();
        assert_eq!(table.sys_getpgid(child), ESRCH);
        // sys_setpgid uses pid->task directly, so the zombie task is still
        // accepted. It is a direct child of PID 1 and not a session leader.
        assert_eq!(table.setpgid_for(1, child, 0), 0);
        assert_eq!(table.thread_group(child).unwrap().pgid, child);
    }

    #[test]
    fn leave_session_only_unlinks_the_session_membership() {
        let mut table = TaskTable::bootstrap();
        let child = process_child(&mut table);
        assert!(table.pid_slot(1).unwrap().sessions.contains(&child));
        table.task_leave_session(child).unwrap();
        assert!(!table.pid_slot(1).unwrap().sessions.contains(&child));
        // task_leave_session does not clear group.sid; task_setsid replaces it
        // immediately afterwards in the C source.
        assert_eq!(table.thread_group(child).unwrap().sid, 1);
    }

    // This stays private to the test module so the production API mirrors the
    // C source rather than adding a redundant current-task getpid method here.
    trait TestGetPid {
        fn sys_getpid_for_test(&self) -> Pid;
    }
    impl TestGetPid for TaskTable {
        fn sys_getpid_for_test(&self) -> Pid {
            self.current().unwrap().tgid
        }
    }
}
