//! `kernel/task.{h,c}` — the process table and task-owned kernel state.
//!
//! iSH keeps one [`Task`] for every guest thread and a small, reusable PID
//! table.  The original C implementation stores raw pointers in a fixed
//! `pids[MAX_PID + 1]` array and relies on `pids_lock` to make those pointers
//! safe to inspect.  Rust represents the same relationships explicitly:
//!
//! * [`TaskTable`] owns tasks, thread groups, and the observable part of the
//!   PID table;
//! * a task refers to its parent and thread group by PID rather than a raw
//!   pointer;
//! * a group is indexed by its leader's PID, and the PID table carries the
//!   session/process-group memberships that make a PID remain occupied after a
//!   task has been reaped.
//!
//! This is intentionally an *explicit* kernel object, rather than a global
//! `__thread current` pointer.  A caller selects [`TaskTable::set_current`]
//! before issuing a current-task syscall.  That removes the C aliasing while
//! preserving the semantics of `pid_get`, `pid_get_task`, and
//! `pid_get_task_zombie`.
//!
//! The host thread launcher in the bottom half of `kernel/task.c` cannot land
//! until the asbestos execution engine and interrupt dispatcher are executable
//! in Rust.  Its state-facing half is here now: PID allocation, parent/child
//! links, `task_create_`, `task_destroy`, address-space attachment, task names,
//! credentials, and the pieces of `struct tgroup` required by process-group,
//! identity, TLS, and resource syscalls.

use std::cell::{RefCell, RefMut};
use std::collections::{BTreeMap, BTreeSet};
use std::rc::Rc;

use crate::cpu::CpuState;
use crate::mmap::Mm;
use crate::personality::ADDR_NO_RANDOMIZE as PERSONALITY_ADDR_NO_RANDOMIZE;
use crate::resource::{Rlimit, Rusage, RLIMIT_NLIMITS};
use crate::user::{Fault, User};

/// `pid_t_` from `misc.h`.
pub type Pid = i32;
/// `uid_t_` from `misc.h`.
pub type Uid = u32;
/// `gid_t_` is also `uid_t_` in iSH's i386 ABI.
pub type Gid = u32;
/// `addr_t` from `misc.h`.
pub type Addr = u32;

/// `MAX_PID` from `kernel/task.h`.
pub const MAX_PID: Pid = 1 << 15;
/// `MAX_GROUPS` from `kernel/task.h`.
pub const MAX_GROUPS: usize = 32;
/// `ADDR_NO_RANDOMIZE_` from `kernel/personality.h`.
///
/// This constant now lives in `crate::personality`; this re-export preserves
/// the previous public path for existing callers and tests.
pub const ADDR_NO_RANDOMIZE: u32 = PERSONALITY_ADDR_NO_RANDOMIZE;

/// A task's attached `struct mm`.
///
/// `kernel/task.c` copies the raw `struct mm *` in `task_create_`; `fork.c`
/// subsequently either retains that object or replaces it with a COW copy.
/// `Rc<RefCell<_>>` expresses precisely that first, shared-pointer stage while
/// keeping mutation confined to the single-threaded Rust kernel model.  The
/// C-visible `Mm::refcount` remains explicit inside [`Mm`], because later fork
/// code is responsible for the same retain/release calls as the C.
pub type SharedMm = Rc<RefCell<Mm>>;

/// The task fields that represent Unix credentials.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Credentials {
    /// real UID (`uid`)
    pub uid: Uid,
    /// real GID (`gid`)
    pub gid: Gid,
    /// effective UID (`euid`)
    pub euid: Uid,
    /// effective GID (`egid`)
    pub egid: Gid,
    /// saved UID (`suid`)
    pub suid: Uid,
    /// saved GID (`sgid`)
    pub sgid: Gid,
}

/// The ported, non-host-thread portion of `struct task`.
///
/// Fields owned by filesystem, signal, ptrace, socket-restart, and host-thread
/// modules are deliberately not represented yet.  Keeping them out is safer
/// than inventing incomplete ownership rules; each will be added with the
/// module that owns its operations.  The fields below are the ones the ported
/// process-management calls can observe today.
pub struct Task {
    /// `cpu`
    pub cpu: CpuState,
    /// `mm`.  `None` is the valid state immediately after a bare
    /// `task_create_(NULL)`; `init.c` assigns its mm afterwards.
    pub mm: Option<SharedMm>,

    /// `pid`, immutable after allocation.
    pub pid: Pid,
    /// `tgid`, the leader PID of this task's thread group.
    pub tgid: Pid,
    /// The key of this task's [`ThreadGroup`], also its leader PID.
    pub(crate) group_id: Option<Pid>,

    /// `uid`, `gid`, `euid`, `egid`, `suid`, and `sgid`.
    pub credentials: Credentials,
    /// `ngroups`
    pub ngroups: usize,
    /// `groups[MAX_GROUPS]`.  The fixed backing array matters: `setgroups`
    /// may partially overwrite it before a guest-memory fault is reported.
    pub groups: [Gid; MAX_GROUPS],
    /// `comm[16]`, including its NUL terminator when one is present.
    pub comm: [u8; 16],
    /// `did_exec`
    pub did_exec: bool,

    /// `parent`, represented by PID rather than a raw pointer.
    pub parent: Option<Pid>,
    /// The observable equivalent of the C `children` intrusive list.
    pub children: BTreeSet<Pid>,

    /// `blocked`, `pending`, and `waiting`; signal delivery is the next owner
    /// of their operations, but `task_create_`'s inheritance/reset behaviour is
    /// already represented here.
    pub blocked: u64,
    pub pending: u64,
    pub waiting: u64,
    /// `saved_mask` and `has_saved_mask`.
    pub saved_mask: u64,
    pub has_saved_mask: bool,

    /// `clear_tid`
    pub clear_tid: Addr,
    /// `robust_list`
    pub robust_list: Addr,
    /// `exit_code`
    pub exit_code: u32,
    /// `zombie`
    pub zombie: bool,
    /// `exiting`
    pub exiting: bool,
}

impl Default for Task {
    fn default() -> Self {
        Self {
            cpu: CpuState::default(),
            mm: None,
            pid: 0,
            tgid: 0,
            group_id: None,
            credentials: Credentials::default(),
            ngroups: 0,
            groups: [0; MAX_GROUPS],
            comm: [0; 16],
            did_exec: false,
            parent: None,
            children: BTreeSet::new(),
            blocked: 0,
            pending: 0,
            waiting: 0,
            saved_mask: 0,
            has_saved_mask: false,
            clear_tid: 0,
            robust_list: 0,
            exit_code: 0,
            zombie: false,
            exiting: false,
        }
    }
}

impl Task {
    /// The C's `superuser()` macro for this task.
    pub fn is_superuser(&self) -> bool {
        self.credentials.euid == 0
    }

    /// Borrow the task's mm mutably.
    ///
    /// `User` takes `&mut Mem`, and `Mm` owns that `Mem`; putting the borrow in
    /// one helper prevents individual syscall ports from accidentally holding
    /// both a mutable task borrow and an address-space borrow across unrelated
    /// state changes.
    pub fn mm_mut(&self) -> Option<RefMut<'_, Mm>> {
        self.mm.as_ref().map(|mm| mm.borrow_mut())
    }

    /// `user_read_task(task, ...)` for a task that owns an address space.
    pub fn user_read(&mut self, addr: Addr, out: &mut [u8]) -> Result<(), Fault> {
        let mm = self.mm.as_ref().ok_or(Fault)?;
        User::new(&mut mm.borrow_mut().mem).read(addr, out)
    }

    /// `user_write_task(task, ...)` for a task that owns an address space.
    pub fn user_write(&mut self, addr: Addr, bytes: &[u8]) -> Result<(), Fault> {
        let mm = self.mm.as_ref().ok_or(Fault)?;
        User::new(&mut mm.borrow_mut().mem).write(addr, bytes)
    }

    /// `user_write_task_ptrace(task, ...)`.
    pub fn user_write_ptrace(&mut self, addr: Addr, bytes: &[u8]) -> Result<(), Fault> {
        let mm = self.mm.as_ref().ok_or(Fault)?;
        User::new(&mut mm.borrow_mut().mem).write_ptrace(addr, bytes)
    }

    /// `user_read_string` with this task as the current task.
    pub fn user_read_string(&mut self, addr: Addr, out: &mut [u8]) -> Result<(), Fault> {
        let mm = self.mm.as_ref().ok_or(Fault)?;
        User::new(&mut mm.borrow_mut().mem).read_string(addr, out)
    }

    /// Replace the prefix of `comm` exactly as C's `strcpy(current->comm, src)`
    /// does: copy through the first NUL and leave bytes after it untouched.
    ///
    /// Callers must provide a terminator within the 16-byte source, just as
    /// `sys_prctl` does after forcing `name[15] = '\\0'`.
    pub fn copy_comm_cstr(&mut self, src: &[u8; 16]) {
        for (i, byte) in src.iter().copied().enumerate() {
            self.comm[i] = byte;
            if byte == 0 {
                break;
            }
        }
    }

    /// The NUL-terminated portion of `comm`, useful for diagnostics and tests.
    pub fn comm_bytes(&self) -> &[u8] {
        let end = self
            .comm
            .iter()
            .position(|&b| b == 0)
            .unwrap_or(self.comm.len());
        &self.comm[..end]
    }

    /// Encode the complete fixed C `groups` array in the guest's little-endian
    /// ABI.  Keeping this explicit avoids assuming the host's endianness.
    pub(crate) fn groups_as_le_bytes(&self) -> [u8; MAX_GROUPS * 4] {
        let mut bytes = [0u8; MAX_GROUPS * 4];
        for (i, group) in self.groups.iter().copied().enumerate() {
            bytes[i * 4..i * 4 + 4].copy_from_slice(&group.to_le_bytes());
        }
        bytes
    }

    /// Decode [`Task::groups_as_le_bytes`].  This is deliberately applied even
    /// after a failed `user_read`: C has copied every prefix it reached into the
    /// real `groups` array before it returns `_EFAULT`.
    pub(crate) fn set_groups_from_le_bytes(&mut self, bytes: &[u8; MAX_GROUPS * 4]) {
        for (i, group) in self.groups.iter_mut().enumerate() {
            *group = u32::from_le_bytes(bytes[i * 4..i * 4 + 4].try_into().unwrap());
        }
    }

    fn clone_for_task_create(&self) -> Self {
        // `*task = *parent` in task_create_.  Fields reset by the remainder of
        // that function are overwritten below by TaskTable::create_task.
        Self {
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

/// The ported state of `struct tgroup`.
///
/// The fields that belong to `signal.c`, `time.c`, `resource.c`, `exit.c`, and
/// tty/filesystem code will grow this type as those modules arrive.  Its PID,
/// session, process-group, thread-list, and personality state already has a
/// single owner here.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ThreadGroup {
    /// `leader`
    pub leader: Pid,
    /// `threads`
    pub threads: BTreeSet<Pid>,
    /// `sid`
    pub sid: Pid,
    /// `pgid`
    pub pgid: Pid,
    /// `personality`
    pub personality: u32,
    /// `stopped`
    pub stopped: bool,
    /// `doing_group_exit`
    pub doing_group_exit: bool,
    /// `group_exit_code`
    pub group_exit_code: u32,
    /// `rusage`, accumulated by the exit path as its threads leave.
    pub rusage: Rusage,
    /// `limits[RLIMIT_NLIMITS_]`.
    pub limits: [Rlimit; RLIMIT_NLIMITS],
    /// `children_rusage`, accumulated as child groups are reaped.
    pub children_rusage: Rusage,
}

impl ThreadGroup {
    fn new(leader: Pid) -> Self {
        let mut threads = BTreeSet::new();
        threads.insert(leader);
        Self {
            leader,
            threads,
            // A freshly zero-initialized C tgroup gets these values before
            // init.c's task_setsid call establishes its first session.
            sid: 0,
            pgid: 0,
            personality: 0,
            stopped: false,
            doing_group_exit: false,
            group_exit_code: 0,
            rusage: Rusage::default(),
            limits: [Rlimit::default(); RLIMIT_NLIMITS],
            children_rusage: Rusage::default(),
        }
    }
}

/// The observable contents of one `struct pid`.
///
/// C stores intrusive lists of thread groups in `session` and `pgroup`.  Sets
/// capture the only observable operation on those lists here: whether a PID is
/// occupied and which group/session a lookup resolves to.
#[derive(Debug, Default)]
pub(crate) struct PidSlot {
    pub(crate) task: Option<Pid>,
    pub(crate) sessions: BTreeSet<Pid>,
    pub(crate) pgroups: BTreeSet<Pid>,
}

impl PidSlot {
    fn is_empty(&self) -> bool {
        self.task.is_none() && self.sessions.is_empty() && self.pgroups.is_empty()
    }
}

/// A failure caused by an invalid task-table operation, rather than a guest
/// errno result.  Guest-facing syscalls turn the applicable cases into their
/// documented i386 errno values.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TaskError {
    /// The requested PID has no live task.
    NoSuchTask,
    /// A task that should have a thread group does not have one.
    MissingThreadGroup,
    /// The fixed PID namespace is exhausted.
    PidExhausted,
    /// A group was requested for a task that already owns one.
    AlreadyHasThreadGroup,
}

impl core::fmt::Display for TaskError {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::NoSuchTask => f.write_str("no such task"),
            Self::MissingThreadGroup => f.write_str("task has no thread group"),
            Self::PidExhausted => f.write_str("the iSH PID table is full"),
            Self::AlreadyHasThreadGroup => f.write_str("task already owns a thread group"),
        }
    }
}

impl std::error::Error for TaskError {}

/// The C `pids` array, its associated tasks/groups, and explicit `current`.
///
/// The original uses a 32,769-entry static array.  A sparse map has the same
/// behavior while avoiding allocation for every unused PID.  A missing map
/// entry is a zero-initialized C `struct pid`; [`TaskTable::pid_exists`] uses
/// the same `pid_empty` predicate as `task.c`.
pub struct TaskTable {
    pub(crate) tasks: BTreeMap<Pid, Task>,
    pub(crate) groups: BTreeMap<Pid, ThreadGroup>,
    pub(crate) pids: BTreeMap<Pid, PidSlot>,
    cur_pid: Pid,
    current: Option<Pid>,
}

impl Default for TaskTable {
    fn default() -> Self {
        Self::new()
    }
}

impl TaskTable {
    /// An empty table, equivalent to the static zero-initialized C `pids`
    /// array before `become_first_process`.
    pub fn new() -> Self {
        Self {
            tasks: BTreeMap::new(),
            groups: BTreeMap::new(),
            pids: BTreeMap::new(),
            cur_pid: 0,
            current: None,
        }
    }

    /// Construct the useful part of `become_first_process` / `construct_task`:
    /// PID 1, one thread group, a fresh mm, root credentials, and a session and
    /// process group rooted at PID 1.
    ///
    /// Files, cwd/root fds, signal handlers, rlimits, and the host signal setup
    /// belong to their unported owners, so this deliberately does not claim to
    /// be all of `init.c`.
    pub fn bootstrap() -> Self {
        let mut table = Self::new();
        let init = table
            .create_task(None)
            .expect("an empty PID table has PID 1");
        table
            .create_thread_group(init)
            .expect("a new task has no thread group");
        {
            let task = table.task_mut(init).unwrap();
            task.mm = Some(Rc::new(RefCell::new(Mm::new())));
        }
        {
            let group = table.groups.get_mut(&init).unwrap();
            group.personality = ADDR_NO_RANDOMIZE;
            // init.c copies its static ABI-compatible rlimit table after the
            // group has been zero-initialized.
            group.limits = crate::resource::INITIAL_LIMITS;
        }
        // The C group starts with sid/pgid zero and init.c calls task_setsid.
        assert_eq!(table.task_setsid(init), init);
        table.set_current(init).unwrap();
        table
    }

    /// `pid_empty` plus the bounds check in `pid_get`.
    pub fn pid_exists(&self, pid: Pid) -> bool {
        if !(0..=MAX_PID).contains(&pid) {
            return false;
        }
        self.pids.get(&pid).is_some_and(|slot| !slot.is_empty())
    }

    /// `pid_get_task_zombie`: return a task even if it is a zombie.
    pub fn pid_get_task_zombie(&self, pid: Pid) -> Option<&Task> {
        if !self.pid_exists(pid) {
            return None;
        }
        let task_pid = self.pids.get(&pid)?.task?;
        self.tasks.get(&task_pid)
    }

    /// `pid_get_task`: a zombie task is hidden from ordinary lookup.
    pub fn pid_get_task(&self, pid: Pid) -> Option<&Task> {
        let task = self.pid_get_task_zombie(pid)?;
        (!task.zombie).then_some(task)
    }

    /// Mutable `pid_get_task_zombie` for the process-management implementation.
    pub(crate) fn pid_get_task_zombie_mut(&mut self, pid: Pid) -> Option<&mut Task> {
        if !self.pid_exists(pid) {
            return None;
        }
        let task_pid = self.pids.get(&pid)?.task?;
        self.tasks.get_mut(&task_pid)
    }

    /// Look up a task directly by its stable PID key.
    pub fn task(&self, pid: Pid) -> Option<&Task> {
        self.tasks.get(&pid)
    }

    /// Mutably look up a task directly by its stable PID key.
    pub fn task_mut(&mut self, pid: Pid) -> Option<&mut Task> {
        self.tasks.get_mut(&pid)
    }

    /// Look up a thread group by its leader PID.
    pub fn thread_group(&self, leader: Pid) -> Option<&ThreadGroup> {
        self.groups.get(&leader)
    }

    /// Mutably look up a thread group by its leader PID.
    ///
    /// Resource accounting, exit handling, and future signal/time syscall
    /// ports own fields in this structure. Exposing the same mutable lookup as
    /// [`TaskTable::task_mut`] lets an embedding supply that state without
    /// reintroducing C's raw group pointers.
    pub fn thread_group_mut(&mut self, leader: Pid) -> Option<&mut ThreadGroup> {
        self.groups.get_mut(&leader)
    }

    /// The task selected as C's thread-local `current`.
    pub fn current_pid(&self) -> Option<Pid> {
        self.current
    }

    /// Borrow C's thread-local `current` task.
    pub fn current(&self) -> Option<&Task> {
        self.current.and_then(|pid| self.tasks.get(&pid))
    }

    /// Mutably borrow C's thread-local `current` task.
    pub fn current_mut(&mut self) -> Option<&mut Task> {
        self.current.and_then(|pid| self.tasks.get_mut(&pid))
    }

    /// Select a live task as C's thread-local `current`.
    pub fn set_current(&mut self, pid: Pid) -> Result<(), TaskError> {
        if self.pid_get_task(pid).is_none() {
            return Err(TaskError::NoSuchTask);
        }
        self.current = Some(pid);
        Ok(())
    }

    /// Clear `current`.  This models the interval before a host task thread has
    /// entered `task_thread` and set C's `__thread current` pointer.
    pub fn clear_current(&mut self) {
        self.current = None;
    }

    /// `task_create_`.
    ///
    /// As in the C, a child begins as a shallow copy of its parent and then the
    /// fields reset by `task_create_` are overwritten: pending signals, child
    /// lists, TID/robust-list addresses, and `did_exec`.  It is deliberately
    /// *not* attached to the copied thread group yet; `fork.c::copy_task` owns
    /// that step, and the public group helpers expose it for subsequent ports.
    pub fn create_task(&mut self, parent: Option<Pid>) -> Result<Pid, TaskError> {
        let template = match parent {
            Some(parent) => self
                .pid_get_task_zombie(parent)
                .ok_or(TaskError::NoSuchTask)?
                .clone_for_task_create(),
            None => Task::default(),
        };

        let pid = self.allocate_pid()?;
        let mut task = template;
        task.pid = pid;
        task.parent = parent;
        task.children.clear();
        // These assignments are the post-copy initialization in task_create_.
        task.pending = 0;
        task.clear_tid = 0;
        task.robust_list = 0;
        task.did_exec = false;

        self.pids.entry(pid).or_default().task = Some(pid);
        self.tasks.insert(pid, task);
        if let Some(parent) = parent {
            self.tasks
                .get_mut(&parent)
                .expect("parent was checked before PID allocation")
                .children
                .insert(pid);
        }
        Ok(pid)
    }

    /// Give a bare task the one-member thread group `init.c` constructs for its
    /// first process.  The group starts with session/process-group IDs zero;
    /// callers establish them with [`TaskTable::task_setsid`].
    pub fn create_thread_group(&mut self, leader: Pid) -> Result<Pid, TaskError> {
        let task = self.tasks.get(&leader).ok_or(TaskError::NoSuchTask)?;
        if task.group_id.is_some() {
            return Err(TaskError::AlreadyHasThreadGroup);
        }
        self.groups.insert(leader, ThreadGroup::new(leader));
        let task = self.tasks.get_mut(&leader).unwrap();
        task.group_id = Some(leader);
        task.tgid = leader;
        Ok(leader)
    }

    /// The group-copy half of `fork.c::copy_task` for a non-`CLONE_THREAD`
    /// child.  `tgroup_copy` retains the old session and process-group
    /// membership, makes the child its own leader, and starts its thread list
    /// with just that child.
    ///
    /// Files, sighands, and mms are intentionally left to their owning fork
    /// code; this method only establishes the task/group topology that both
    /// `group.c` and those later owners use.
    pub fn copy_thread_group_for(&mut self, child: Pid) -> Result<Pid, TaskError> {
        let old_id = self
            .tasks
            .get(&child)
            .ok_or(TaskError::NoSuchTask)?
            .group_id
            .ok_or(TaskError::MissingThreadGroup)?;
        let old_group = self
            .groups
            .get(&old_id)
            .ok_or(TaskError::MissingThreadGroup)?
            .clone();

        let mut group = old_group;
        group.leader = child;
        group.threads.clear();
        group.threads.insert(child);
        // `tgroup_copy` resets these fields after a shallow copy.
        group.stopped = false;
        group.doing_group_exit = false;
        group.group_exit_code = 0;
        // fork.c's tgroup_copy retains the group's own usage but gives a new
        // process group no already-reaped children.
        group.children_rusage = Rusage::default();
        self.groups.insert(child, group.clone());
        {
            let task = self.tasks.get_mut(&child).unwrap();
            task.group_id = Some(child);
            task.tgid = child;
        }
        if group.sid != 0 {
            self.pids
                .entry(group.sid)
                .or_default()
                .sessions
                .insert(child);
        }
        if group.pgid != 0 {
            self.pids
                .entry(group.pgid)
                .or_default()
                .pgroups
                .insert(child);
        }
        Ok(child)
    }

    /// The `CLONE_THREAD` branch of `fork.c::copy_task`: add an already-created
    /// child to its inherited group.  `task_create_` copied `tgid` and the
    /// group pointer, so this is normally the only state transition needed.
    pub fn attach_thread_to_group(&mut self, child: Pid) -> Result<(), TaskError> {
        let group_id = self
            .tasks
            .get(&child)
            .ok_or(TaskError::NoSuchTask)?
            .group_id
            .ok_or(TaskError::MissingThreadGroup)?;
        let group = self
            .groups
            .get_mut(&group_id)
            .ok_or(TaskError::MissingThreadGroup)?;
        group.threads.insert(child);
        let task = self.tasks.get_mut(&child).unwrap();
        task.tgid = group.leader;
        Ok(())
    }

    /// The pointer assignment in `task_set_mm`.  Retain/release belongs to the
    /// caller exactly as it does in C; this function deliberately does not
    /// mutate `Mm::refcount`.
    pub fn task_set_mm(&mut self, pid: Pid, mm: SharedMm) -> Result<(), TaskError> {
        let task = self.tasks.get_mut(&pid).ok_or(TaskError::NoSuchTask)?;
        task.mm = Some(mm);
        Ok(())
    }

    /// `task_destroy`'s process-table and parent/sibling-list work.
    ///
    /// C deliberately does *not* release the mm, fd table, sighand, or remove
    /// the task from its thread group here; its callers have already performed
    /// those owner-specific operations.  This function follows that division:
    /// it removes only the task pointer and parent-child link.  A PID slot is
    /// retained while its session or process-group list is non-empty, exactly
    /// as `pid_empty` requires.
    pub fn destroy_task(&mut self, pid: Pid) -> Result<(), TaskError> {
        let task = self.tasks.remove(&pid).ok_or(TaskError::NoSuchTask)?;
        if let Some(parent) = task.parent {
            if let Some(parent_task) = self.tasks.get_mut(&parent) {
                parent_task.children.remove(&pid);
            }
        }
        if let Some(slot) = self.pids.get_mut(&pid) {
            slot.task = None;
        }
        self.cleanup_pid_slot(pid);
        if self.current == Some(pid) {
            self.current = None;
        }
        Ok(())
    }

    /// Mark a task as a zombie without removing its PID-table task pointer.
    /// This is the state distinction `pid_get_task` and
    /// `pid_get_task_zombie` expose to wait/group/signal code.
    pub fn mark_zombie(&mut self, pid: Pid, exit_code: u32) -> Result<(), TaskError> {
        let task = self
            .pid_get_task_zombie_mut(pid)
            .ok_or(TaskError::NoSuchTask)?;
        task.zombie = true;
        task.exit_code = exit_code;
        Ok(())
    }

    /// Whether task is the leader of its thread group (`task_is_leader`).
    pub fn task_is_leader(&self, pid: Pid) -> bool {
        let Some(task) = self.tasks.get(&pid) else {
            return false;
        };
        let Some(group_id) = task.group_id else {
            return false;
        };
        self.groups
            .get(&group_id)
            .is_some_and(|group| group.leader == pid)
    }

    pub(crate) fn group_of_task(&self, pid: Pid) -> Option<Pid> {
        self.tasks.get(&pid)?.group_id
    }

    pub(crate) fn group_mut(&mut self, id: Pid) -> Option<&mut ThreadGroup> {
        self.thread_group_mut(id)
    }

    pub(crate) fn pid_slot(&self, pid: Pid) -> Option<&PidSlot> {
        self.pids.get(&pid).filter(|slot| !slot.is_empty())
    }

    pub(crate) fn add_session_member(&mut self, pid: Pid, group: Pid) {
        self.pids.entry(pid).or_default().sessions.insert(group);
    }

    pub(crate) fn remove_session_member(&mut self, pid: Pid, group: Pid) {
        if let Some(slot) = self.pids.get_mut(&pid) {
            slot.sessions.remove(&group);
        }
        self.cleanup_pid_slot(pid);
    }

    pub(crate) fn add_pgroup_member(&mut self, pid: Pid, group: Pid) {
        self.pids.entry(pid).or_default().pgroups.insert(group);
    }

    pub(crate) fn remove_pgroup_member(&mut self, pid: Pid, group: Pid) {
        if let Some(slot) = self.pids.get_mut(&pid) {
            slot.pgroups.remove(&group);
        }
        self.cleanup_pid_slot(pid);
    }

    fn cleanup_pid_slot(&mut self, pid: Pid) {
        if self.pids.get(&pid).is_some_and(PidSlot::is_empty) {
            self.pids.remove(&pid);
        }
    }

    fn allocate_pid(&mut self) -> Result<Pid, TaskError> {
        // The C loop increments first, wraps after MAX_PID, and then asks
        // pid_empty.  Preserve that order so a fresh table starts at PID 1.
        for _ in 0..MAX_PID {
            self.cur_pid += 1;
            if self.cur_pid > MAX_PID {
                self.cur_pid = 1;
            }
            if !self.pid_exists(self.cur_pid) {
                return Ok(self.cur_pid);
            }
        }
        // The C has no full-table escape and would loop forever.  Returning a
        // typed failure is the one safe Rust representation of that otherwise
        // unobservable host failure.
        Err(TaskError::PidExhausted)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bootstrap_matches_the_part_of_init_that_owns_tasks() {
        let table = TaskTable::bootstrap();
        assert_eq!(table.current_pid(), Some(1));
        let init = table.current().unwrap();
        assert_eq!(init.pid, 1);
        assert_eq!(init.tgid, 1);
        assert!(init.mm.is_some());
        assert!(init.is_superuser());
        let group = table.thread_group(1).unwrap();
        assert_eq!(group.leader, 1);
        assert_eq!(group.sid, 1);
        assert_eq!(group.pgid, 1);
        assert_eq!(group.personality, ADDR_NO_RANDOMIZE);
        assert!(table.pid_slot(1).unwrap().sessions.contains(&1));
        assert!(table.pid_slot(1).unwrap().pgroups.contains(&1));
    }

    #[test]
    fn task_create_shallow_copies_then_resets_task_c_fields() {
        let mut table = TaskTable::bootstrap();
        {
            let parent = table.current_mut().unwrap();
            parent.credentials.uid = 1000;
            parent.credentials.euid = 1001;
            parent.groups[..2].copy_from_slice(&[4, 9]);
            parent.ngroups = 2;
            parent.comm[..6].copy_from_slice(b"parent");
            parent.blocked = 0x11;
            parent.pending = 0x22;
            parent.waiting = 0x33;
            parent.saved_mask = 0x44;
            parent.has_saved_mask = true;
            parent.clear_tid = 0xfeed;
            parent.robust_list = 0xbeef;
            parent.did_exec = true;
        }
        let child = table.create_task(Some(1)).unwrap();
        let child_task = table.task(child).unwrap();
        assert_eq!(child, 2);
        assert_eq!(child_task.parent, Some(1));
        assert_eq!(
            child_task.tgid, 1,
            "raw task_create copies the group identity"
        );
        assert_eq!(child_task.credentials.uid, 1000);
        assert_eq!(child_task.credentials.euid, 1001);
        assert_eq!(&child_task.groups[..2], &[4, 9]);
        assert_eq!(child_task.comm_bytes(), b"parent");
        assert_eq!(child_task.blocked, 0x11);
        assert_eq!(child_task.waiting, 0x33);
        assert_eq!(child_task.saved_mask, 0x44);
        assert!(child_task.has_saved_mask);
        assert_eq!(child_task.pending, 0, "task_create resets pending");
        assert_eq!(child_task.clear_tid, 0);
        assert_eq!(child_task.robust_list, 0);
        assert!(!child_task.did_exec);
        assert!(table.task(1).unwrap().children.contains(&child));
        assert!(table.pid_get_task(child).is_some());
    }

    #[test]
    fn zombie_lookup_is_separate_from_live_lookup_and_destroy_unlinks_parent() {
        let mut table = TaskTable::bootstrap();
        let child = table.create_task(Some(1)).unwrap();
        table.mark_zombie(child, 0x2300).unwrap();
        assert!(table.pid_get_task(child).is_none());
        assert_eq!(table.pid_get_task_zombie(child).unwrap().exit_code, 0x2300);
        table.destroy_task(child).unwrap();
        assert!(table.pid_get_task_zombie(child).is_none());
        assert!(!table.task(1).unwrap().children.contains(&child));
        assert!(!table.pid_exists(child));
    }

    #[test]
    fn group_copy_and_thread_attach_follow_fork_topology() {
        let mut table = TaskTable::bootstrap();
        let process_child = table.create_task(Some(1)).unwrap();
        table.copy_thread_group_for(process_child).unwrap();
        let group = table.thread_group(process_child).unwrap();
        assert_eq!(group.leader, process_child);
        assert_eq!(group.sid, 1);
        assert_eq!(group.pgid, 1);
        assert_eq!(group.threads, BTreeSet::from([process_child]));
        assert!(table.pid_slot(1).unwrap().sessions.contains(&process_child));
        assert!(table.pid_slot(1).unwrap().pgroups.contains(&process_child));

        let thread_child = table.create_task(Some(1)).unwrap();
        table.attach_thread_to_group(thread_child).unwrap();
        assert_eq!(table.task(thread_child).unwrap().tgid, 1);
        assert!(table
            .thread_group(1)
            .unwrap()
            .threads
            .contains(&thread_child));
    }

    #[test]
    fn comm_copy_stops_at_nul_and_preserves_the_c_strcpy_tail() {
        let mut task = Task {
            comm: *b"abcdefghijklmnop",
            ..Task::default()
        };
        let mut source = [0u8; 16];
        source[..3].copy_from_slice(b"hi\0");
        task.copy_comm_cstr(&source);
        assert_eq!(&task.comm[..3], b"hi\0");
        assert_eq!(&task.comm[3..], b"defghijklmnop");
    }
}
