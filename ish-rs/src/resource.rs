//! `kernel/resource.{h,c}` — resource limits, usage accounting, and scheduler
//! compatibility syscalls.
//!
//! The C implementation has two kinds of host input: per-host-thread CPU usage
//! (`getrusage(RUSAGE_THREAD)`) and the number of online CPUs (`sysconf`).  They
//! are explicit through [`ResourceHost`] here rather than being silently
//! approximated with a Rust wall clock.  This keeps the guest ABI and all of the
//! C-side ordering/permission rules portable to iOS while leaving the platform
//! adapter responsible for obtaining those host values.
//!
//! Resource limits and child usage belong to a thread group, just as they do in
//! C's `struct tgroup`.  The storage is added in `task.rs`; this module is the
//! sole owner of the guest layouts and syscall rules that operate on it.

use crate::group::{EPERM, ESRCH};
use crate::task::{Addr, Pid, Task, TaskTable};

/// Invalid argument.
pub const EINVAL: i32 = -22;
/// Bad guest address.
pub const EFAULT: i32 = -14;

/// Number of resource-limit slots in iSH's i386 ABI.
pub const RLIMIT_NLIMITS: usize = 16;
/// `RLIM_INFINITY_`.
pub const RLIM_INFINITY: u64 = u64::MAX;

/// Resource numbers from `kernel/resource.h`.
pub const RLIMIT_CPU: u32 = 0;
pub const RLIMIT_FSIZE: u32 = 1;
pub const RLIMIT_DATA: u32 = 2;
pub const RLIMIT_STACK: u32 = 3;
pub const RLIMIT_CORE: u32 = 4;
pub const RLIMIT_RSS: u32 = 5;
pub const RLIMIT_NPROC: u32 = 6;
pub const RLIMIT_NOFILE: u32 = 7;
pub const RLIMIT_MEMLOCK: u32 = 8;
pub const RLIMIT_AS: u32 = 9;
pub const RLIMIT_LOCKS: u32 = 10;
pub const RLIMIT_SIGPENDING: u32 = 11;
pub const RLIMIT_MSGQUEUE: u32 = 12;
pub const RLIMIT_NICE: u32 = 13;
pub const RLIMIT_RTPRIO: u32 = 14;
pub const RLIMIT_RTTIME: u32 = 15;

/// A `struct rlimit_`: two 64-bit quantities in the i386 guest ABI.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Rlimit {
    /// `rlim_cur`
    pub cur: u64,
    /// `rlim_max`
    pub max: u64,
}

impl Rlimit {
    /// Size of a guest `struct rlimit_` / `struct rlimit64`.
    pub const SIZE: usize = 16;

    /// Construct a limit pair in a constant initializer.
    pub const fn new(cur: u64, max: u64) -> Self {
        Self { cur, max }
    }

    /// Decode the explicit little-endian guest representation.
    pub fn from_le_bytes(bytes: [u8; Self::SIZE]) -> Self {
        Self {
            cur: u64::from_le_bytes(bytes[0..8].try_into().unwrap()),
            max: u64::from_le_bytes(bytes[8..16].try_into().unwrap()),
        }
    }

    /// Encode the explicit little-endian guest representation.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        bytes[0..8].copy_from_slice(&self.cur.to_le_bytes());
        bytes[8..16].copy_from_slice(&self.max.to_le_bytes());
        bytes
    }
}

/// The legacy 32-bit `struct rlimit32_` guest layout.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Rlimit32 {
    /// `rlim_cur`
    pub cur: u32,
    /// `rlim_max`
    pub max: u32,
}

impl Rlimit32 {
    /// Size of the legacy guest structure.
    pub const SIZE: usize = 8;

    /// Decode the explicit little-endian guest representation.
    pub fn from_le_bytes(bytes: [u8; Self::SIZE]) -> Self {
        Self {
            cur: u32::from_le_bytes(bytes[0..4].try_into().unwrap()),
            max: u32::from_le_bytes(bytes[4..8].try_into().unwrap()),
        }
    }

    /// Encode the explicit little-endian guest representation.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        bytes[0..4].copy_from_slice(&self.cur.to_le_bytes());
        bytes[4..8].copy_from_slice(&self.max.to_le_bytes());
        bytes
    }
}

impl From<Rlimit32> for Rlimit {
    fn from(limit: Rlimit32) -> Self {
        Self {
            cur: u64::from(limit.cur),
            max: u64::from(limit.max),
        }
    }
}

impl From<Rlimit> for Rlimit32 {
    fn from(limit: Rlimit) -> Self {
        Self {
            cur: limit.cur as u32,
            max: limit.max as u32,
        }
    }
}

/// `struct timeval_` from `kernel/time.h`, retained here because resource usage
/// owns the two time values until `time.c` is ported.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Timeval {
    /// Seconds.
    pub sec: u32,
    /// Microseconds.
    pub usec: u32,
}

impl Timeval {
    /// Guest layout size.
    pub const SIZE: usize = 8;

    fn write_to(self, bytes: &mut [u8]) {
        bytes[0..4].copy_from_slice(&self.sec.to_le_bytes());
        bytes[4..8].copy_from_slice(&self.usec.to_le_bytes());
    }
}

/// The i386-compatible `struct rusage_` from `kernel/resource.h`.
///
/// iSH currently populates only `utime` and `stime`; the remaining fields are
/// nevertheless represented and written as part of the exact 72-byte guest
/// layout.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Rusage {
    /// `ru_utime`
    pub utime: Timeval,
    /// `ru_stime`
    pub stime: Timeval,
    /// `ru_maxrss`
    pub maxrss: u32,
    /// `ru_ixrss`
    pub ixrss: u32,
    /// `ru_idrss`
    pub idrss: u32,
    /// `ru_isrss`
    pub isrss: u32,
    /// `ru_minflt`
    pub minflt: u32,
    /// `ru_majflt`
    pub majflt: u32,
    /// `ru_nswap`
    pub nswap: u32,
    /// `ru_inblock`
    pub inblock: u32,
    /// `ru_oublock`
    pub oublock: u32,
    /// `ru_msgsnd`
    pub msgsnd: u32,
    /// `ru_msgrcv`
    pub msgrcv: u32,
    /// `ru_nsignals`
    pub nsignals: u32,
    /// `ru_nvcsw`
    pub nvcsw: u32,
    /// `ru_nivcsw`
    pub nivcsw: u32,
}

impl Rusage {
    /// Size of iSH's `struct rusage_`: 18 i386 words.
    pub const SIZE: usize = 72;

    /// Encode the guest structure without relying on host layout or endian.
    pub fn to_le_bytes(self) -> [u8; Self::SIZE] {
        let mut bytes = [0; Self::SIZE];
        self.utime.write_to(&mut bytes[0..8]);
        self.stime.write_to(&mut bytes[8..16]);
        let words = [
            self.maxrss,
            self.ixrss,
            self.idrss,
            self.isrss,
            self.minflt,
            self.majflt,
            self.nswap,
            self.inblock,
            self.oublock,
            self.msgsnd,
            self.msgrcv,
            self.nsignals,
            self.nvcsw,
            self.nivcsw,
        ];
        for (index, word) in words.into_iter().enumerate() {
            let offset = 16 + index * 4;
            bytes[offset..offset + 4].copy_from_slice(&word.to_le_bytes());
        }
        bytes
    }
}

/// Host values consumed by the two resource syscalls that deliberately expose
/// host state in iSH.
///
/// An embedding supplies this from its platform layer.  Linux iSH obtains the
/// first value with `getrusage(RUSAGE_THREAD)` and the second with
/// `sysconf(_SC_NPROCESSORS_ONLN)`; the trait keeps those APIs out of the
/// portable Rust core and makes their behavior deterministic in tests.
pub trait ResourceHost {
    /// Return current host-thread CPU usage in the guest's 32-bit timeval form.
    fn current_thread_rusage(&self) -> Rusage;

    /// Return the count used to construct the affinity bitset.
    fn online_cpus(&self) -> usize;
}

/// A deterministic [`ResourceHost`] useful to embeddings without host telemetry
/// and to tests.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct StaticResourceHost {
    /// Returned by [`ResourceHost::current_thread_rusage`].
    pub rusage: Rusage,
    /// Returned by [`ResourceHost::online_cpus`].
    pub cpus: usize,
}

impl ResourceHost for StaticResourceHost {
    fn current_thread_rusage(&self) -> Rusage {
        self.rusage
    }

    fn online_cpus(&self) -> usize {
        self.cpus
    }
}

/// Initial limits copied by `init.c` into the first thread group.
pub(crate) const INITIAL_LIMITS: [Rlimit; RLIMIT_NLIMITS] = [
    Rlimit::new(RLIM_INFINITY, RLIM_INFINITY),   // CPU
    Rlimit::new(RLIM_INFINITY, RLIM_INFINITY),   // FSIZE
    Rlimit::new(RLIM_INFINITY, RLIM_INFINITY),   // DATA
    Rlimit::new(8 * 1024 * 1024, RLIM_INFINITY), // STACK
    Rlimit::new(0, RLIM_INFINITY),               // CORE
    Rlimit::new(RLIM_INFINITY, RLIM_INFINITY),   // RSS
    Rlimit::new(1024, 1024),                     // NPROC
    Rlimit::new(1024, 4096),                     // NOFILE
    Rlimit::new(64 * 1024, 64 * 1024),           // MEMLOCK
    Rlimit::new(RLIM_INFINITY, RLIM_INFINITY),   // AS
    Rlimit::new(RLIM_INFINITY, RLIM_INFINITY),   // LOCKS
    Rlimit::new(1024, 1024),                     // SIGPENDING
    Rlimit::new(819_200, 819_200),               // MSGQUEUE
    Rlimit::new(0, 0),                           // NICE
    Rlimit::new(0, 0),                           // RTPRIO
    Rlimit::new(RLIM_INFINITY, RLIM_INFINITY),   // RTTIME
];

/// `RUSAGE_SELF_`, represented in the raw `dword_t` syscall argument.
pub const RUSAGE_SELF: u32 = 0;
/// `RUSAGE_CHILDREN_` after C converts `-1` to that raw `dword_t` argument.
pub const RUSAGE_CHILDREN: u32 = u32::MAX;
/// `SCHED_OTHER_`.
pub const SCHED_OTHER: i32 = 0;

fn resource_index(resource: u32) -> Result<usize, i32> {
    if resource < RLIMIT_NLIMITS as u32 {
        Ok(resource as usize)
    } else {
        Err(EINVAL)
    }
}

fn current_pid(table: &TaskTable) -> Result<Pid, i32> {
    table.current_pid().ok_or(ESRCH)
}

fn current_task_mut(table: &mut TaskTable) -> Result<&mut Task, i32> {
    let pid = current_pid(table)?;
    table.task_mut(pid).ok_or(ESRCH)
}

fn rlimit_get(table: &TaskTable, pid: Pid, resource: u32) -> Result<Rlimit, i32> {
    let index = resource_index(resource)?;
    let group_id = table.group_of_task(pid).ok_or(ESRCH)?;
    table
        .thread_group(group_id)
        .map(|group| group.limits[index])
        .ok_or(ESRCH)
}

fn rlimit_set(table: &mut TaskTable, pid: Pid, resource: u32, limit: Rlimit) -> Result<(), i32> {
    let index = resource_index(resource)?;
    let group_id = table.group_of_task(pid).ok_or(ESRCH)?;
    let group = table.group_mut(group_id).ok_or(ESRCH)?;
    group.limits[index] = limit;
    Ok(())
}

fn read_rlimit(task: &mut Task, addr: Addr) -> Result<Rlimit, i32> {
    let mut bytes = [0; Rlimit::SIZE];
    task.user_read(addr, &mut bytes).map_err(|_| EFAULT)?;
    Ok(Rlimit::from_le_bytes(bytes))
}

fn write_bytes(task: &mut Task, addr: Addr, bytes: &[u8]) -> i32 {
    task.user_write(addr, bytes).map_or(EFAULT, |_| 0)
}

fn check_setrlimit(table: &TaskTable, pid: Pid, resource: u32, new_limit: Rlimit) -> i32 {
    // C deliberately returns success for root before looking up the old limit.
    if table.task(pid).is_some_and(Task::is_superuser) {
        return 0;
    }
    match rlimit_get(table, pid, resource) {
        Err(err) => err,
        Ok(old_limit) if new_limit.max > old_limit.max => EPERM,
        Ok(_) => 0,
    }
}

/// Return `resource`'s current soft limit for the selected task.
///
/// C's `rlimit()` terminates the entire emulator for an invalid resource.  A
/// typed `Err(EINVAL)` retains the valid-call result without turning malformed
/// guest state into a Rust process abort.
pub fn rlimit(table: &TaskTable, resource: u32) -> Result<u64, i32> {
    let pid = current_pid(table)?;
    rlimit_get(table, pid, resource).map(|limit| limit.cur)
}

fn getrlimit32(table: &mut TaskTable, resource: u32, addr: Addr, old: bool) -> i32 {
    let pid = match current_pid(table) {
        Ok(pid) => pid,
        Err(err) => return err,
    };
    let limit = match rlimit_get(table, pid, resource) {
        Ok(limit) => limit,
        Err(err) => return err,
    };
    // do_getrlimit32 narrows into the local rlimit32 first. The old ABI then
    // clamps only a *negative-looking narrowed value*, rather than the full
    // 64-bit source value; this distinction is observable for exact 4 GiB.
    let mut limit32 = Rlimit32::from(limit);
    if old {
        if limit32.cur > i32::MAX as u32 {
            limit32.cur = i32::MAX as u32;
        }
        if limit32.max > i32::MAX as u32 {
            limit32.max = i32::MAX as u32;
        }
    }
    let bytes = limit32.to_le_bytes();
    match current_task_mut(table) {
        Ok(task) => write_bytes(task, addr, &bytes),
        Err(err) => err,
    }
}

/// `sys_getrlimit32` / the modern `getrlimit` syscall entry.
pub fn sys_getrlimit32(table: &mut TaskTable, resource: u32, rlim_addr: Addr) -> i32 {
    getrlimit32(table, resource, rlim_addr, false)
}

/// `sys_old_getrlimit32`, including its `INT_MAX` infinity compatibility clamp.
pub fn sys_old_getrlimit32(table: &mut TaskTable, resource: u32, rlim_addr: Addr) -> i32 {
    getrlimit32(table, resource, rlim_addr, true)
}

/// `sys_setrlimit32`.
///
/// Despite its historical name, iSH's C entry reads a full `struct rlimit_`
/// (two 64-bit fields), not `struct rlimit32_`; retain that observable ABI
/// quirk rather than narrowing the input to the legacy getter's layout.
pub fn sys_setrlimit32(table: &mut TaskTable, resource: u32, rlim_addr: Addr) -> i32 {
    let pid = match current_pid(table) {
        Ok(pid) => pid,
        Err(err) => return err,
    };
    let limit = match current_task_mut(table).and_then(|task| read_rlimit(task, rlim_addr)) {
        Ok(limit) => limit,
        Err(err) => return err,
    };
    let err = check_setrlimit(table, pid, resource, limit);
    if err < 0 {
        return err;
    }
    match rlimit_set(table, pid, resource, limit) {
        Ok(()) => 0,
        Err(err) => err,
    }
}

/// `sys_prlimit64`.
///
/// As in C, an old-limit write is performed before the new limit is read.  That
/// matters when the two guest addresses alias, and also means an old-limit
/// write fault leaves the stored limit untouched.
pub fn sys_prlimit64(
    table: &mut TaskTable,
    pid: Pid,
    resource: u32,
    new_limit_addr: Addr,
    old_limit_addr: Addr,
) -> i32 {
    if pid != 0 {
        return EINVAL;
    }
    let current = match current_pid(table) {
        Ok(pid) => pid,
        Err(err) => return err,
    };

    if old_limit_addr != 0 {
        let limit = match rlimit_get(table, current, resource) {
            Ok(limit) => limit,
            Err(err) => return err,
        };
        let bytes = limit.to_le_bytes();
        let err = match current_task_mut(table) {
            Ok(task) => write_bytes(task, old_limit_addr, &bytes),
            Err(err) => err,
        };
        if err < 0 {
            return err;
        }
    }

    if new_limit_addr != 0 {
        let limit = match current_task_mut(table).and_then(|task| read_rlimit(task, new_limit_addr))
        {
            Ok(limit) => limit,
            Err(err) => return err,
        };
        let err = check_setrlimit(table, current, resource, limit);
        if err < 0 {
            return err;
        }
        return match rlimit_set(table, current, resource, limit) {
            Ok(()) => 0,
            Err(err) => err,
        };
    }
    0
}

/// The `rusage_get_current` host boundary in `resource.c`.
pub fn rusage_get_current(host: &impl ResourceHost) -> Rusage {
    host.current_thread_rusage()
}

fn timeval_add(dst: &mut Timeval, src: Timeval) {
    dst.sec = dst.sec.wrapping_add(src.sec);
    dst.usec = dst.usec.wrapping_add(src.usec);
    // The C source deliberately carries at most once, because valid timeval
    // operands each have a sub-million microsecond component.
    if dst.usec >= 1_000_000 {
        dst.usec -= 1_000_000;
        dst.sec = dst.sec.wrapping_add(1);
    }
}

/// `rusage_add`: only the two time fields are accumulated by iSH.
pub fn rusage_add(dst: &mut Rusage, src: Rusage) {
    timeval_add(&mut dst.utime, src.utime);
    timeval_add(&mut dst.stime, src.stime);
}

/// `sys_getrusage`.
pub fn sys_getrusage(
    table: &mut TaskTable,
    host: &impl ResourceHost,
    who: u32,
    rusage_addr: Addr,
) -> i32 {
    let rusage = match who {
        RUSAGE_SELF => rusage_get_current(host),
        RUSAGE_CHILDREN => {
            let pid = match current_pid(table) {
                Ok(pid) => pid,
                Err(err) => return err,
            };
            let Some(group_id) = table.group_of_task(pid) else {
                return ESRCH;
            };
            let Some(group) = table.thread_group(group_id) else {
                return ESRCH;
            };
            group.children_rusage
        }
        _ => return EINVAL,
    };
    let bytes = rusage.to_le_bytes();
    match current_task_mut(table) {
        Ok(task) => write_bytes(task, rusage_addr, &bytes),
        Err(err) => err,
    }
}

/// `sys_sched_getaffinity`.
pub fn sys_sched_getaffinity(
    table: &mut TaskTable,
    host: &impl ResourceHost,
    pid: Pid,
    cpusetsize: u32,
    cpuset_addr: Addr,
) -> i32 {
    if pid != 0 && table.pid_get_task(pid).is_none() {
        return ESRCH;
    }

    let cpus = host.online_cpus();
    // iSH intentionally uses `cpus / 8 + 1`, including one extra byte whenever
    // the count is exactly divisible by eight.
    let byte_count = cpus / 8 + 1;
    if (cpusetsize as usize) < byte_count {
        return EINVAL;
    }
    let mut cpuset = vec![0u8; byte_count];
    for cpu in 0..cpus {
        cpuset[cpu / 8] |= 1 << (cpu % 8);
    }
    match current_task_mut(table) {
        Ok(task) => match write_bytes(task, cpuset_addr, &cpuset) {
            0 => byte_count as i32,
            err => err,
        },
        Err(err) => err,
    }
}

/// `sys_sched_setaffinity` is intentionally a success stub in iSH.
pub fn sys_sched_setaffinity(_pid: Pid, _cpusetsize: u32, _cpuset_addr: Addr) -> i32 {
    0
}

/// `sys_getpriority` is intentionally fixed at the nicest C-visible value.
pub fn sys_getpriority(_which: i32, _who: Pid) -> i32 {
    20
}

/// `sys_setpriority` is intentionally a success stub.
pub fn sys_setpriority(_which: i32, _who: Pid, _priority: i32) -> i32 {
    0
}

/// `sys_sched_getparam` writes a zero `sched_priority` word.
pub fn sys_sched_getparam(table: &mut TaskTable, _pid: Pid, param_addr: Addr) -> i32 {
    let bytes = 0i32.to_le_bytes();
    match current_task_mut(table) {
        Ok(task) => write_bytes(task, param_addr, &bytes),
        Err(err) => err,
    }
}

/// `sys_sched_getscheduler` always reports `SCHED_OTHER_`.
pub fn sys_sched_getscheduler(_pid: Pid) -> i32 {
    SCHED_OTHER
}

/// `sys_sched_setscheduler` accepts only `SCHED_OTHER_` with priority zero.
pub fn sys_sched_setscheduler(
    table: &mut TaskTable,
    _pid: Pid,
    policy: i32,
    param_addr: Addr,
) -> i32 {
    if policy != SCHED_OTHER {
        return EINVAL;
    }
    let mut bytes = [0; 4];
    let task = match current_task_mut(table) {
        Ok(task) => task,
        Err(err) => return err,
    };
    if task.user_read(param_addr, &mut bytes).is_err() {
        return EFAULT;
    }
    if i32::from_le_bytes(bytes) == 0 {
        0
    } else {
        EINVAL
    }
}

/// `sys_sched_get_priority_max` accepts only `SCHED_OTHER_`.
pub fn sys_sched_get_priority_max(policy: i32) -> i32 {
    if policy == SCHED_OTHER {
        0
    } else {
        EINVAL
    }
}

/// `sys_ioprio_get` is intentionally a zero-returning stub.
pub fn sys_ioprio_get(_which: i32, _who: i32) -> i32 {
    0
}

/// `sys_ioprio_set` is intentionally a success stub.
pub fn sys_ioprio_set(_which: i32, _who: i32, _ioprio: i32) -> i32 {
    0
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::P_RWX;
    use crate::mmu::PAGE_BITS;

    const PAGE: Addr = 0x100 << PAGE_BITS;

    fn table_with_memory() -> TaskTable {
        let mut table = TaskTable::bootstrap();
        table
            .current_mut()
            .unwrap()
            .mm_mut()
            .unwrap()
            .mem
            .map_nothing(0x100, 2, P_RWX);
        table
    }

    fn read_current(table: &mut TaskTable, addr: Addr, size: usize) -> Vec<u8> {
        let mut bytes = vec![0; size];
        table
            .current_mut()
            .unwrap()
            .user_read(addr, &mut bytes)
            .unwrap();
        bytes
    }

    fn write_current(table: &mut TaskTable, addr: Addr, bytes: &[u8]) {
        table
            .current_mut()
            .unwrap()
            .user_write(addr, bytes)
            .unwrap();
    }

    #[test]
    fn initial_limits_and_both_getrlimit_abis_match_init_c() {
        let mut table = table_with_memory();
        assert_eq!(rlimit(&table, RLIMIT_STACK), Ok(8 * 1024 * 1024));
        assert_eq!(rlimit(&table, RLIMIT_NOFILE), Ok(1024));

        assert_eq!(sys_getrlimit32(&mut table, RLIMIT_STACK, PAGE), 0);
        assert_eq!(
            Rlimit32::from_le_bytes(
                read_current(&mut table, PAGE, Rlimit32::SIZE)
                    .try_into()
                    .unwrap()
            ),
            Rlimit32 {
                cur: 8 * 1024 * 1024,
                max: u32::MAX,
            }
        );

        assert_eq!(sys_old_getrlimit32(&mut table, RLIMIT_STACK, PAGE + 16), 0);
        assert_eq!(
            Rlimit32::from_le_bytes(
                read_current(&mut table, PAGE + 16, Rlimit32::SIZE)
                    .try_into()
                    .unwrap()
            ),
            Rlimit32 {
                cur: 8 * 1024 * 1024,
                max: i32::MAX as u32,
            }
        );
        // do_getrlimit32 narrows before the legacy wrapper considers the
        // signed limit. Exact 4 GiB therefore becomes zero, not INT_MAX.
        table.group_mut(1).unwrap().limits[RLIMIT_CORE as usize] = Rlimit::new(1 << 32, 1 << 32);
        assert_eq!(sys_old_getrlimit32(&mut table, RLIMIT_CORE, PAGE + 32), 0);
        assert_eq!(
            Rlimit32::from_le_bytes(
                read_current(&mut table, PAGE + 32, Rlimit32::SIZE)
                    .try_into()
                    .unwrap()
            ),
            Rlimit32::default()
        );
        assert_eq!(sys_getrlimit32(&mut table, 99, PAGE), EINVAL);
    }

    #[test]
    fn setrlimit_preserves_the_c_root_and_nonroot_rules() {
        let mut table = table_with_memory();
        let raised = Rlimit::new(8_000, 8_000);
        write_current(&mut table, PAGE, &raised.to_le_bytes());
        assert_eq!(sys_setrlimit32(&mut table, RLIMIT_NOFILE, PAGE), 0);
        assert_eq!(rlimit(&table, RLIMIT_NOFILE), Ok(8_000));

        table.current_mut().unwrap().credentials.euid = 1000;
        let too_high = Rlimit::new(9_000, 9_000);
        write_current(&mut table, PAGE, &too_high.to_le_bytes());
        assert_eq!(sys_setrlimit32(&mut table, RLIMIT_NOFILE, PAGE), EPERM);
        assert_eq!(rlimit(&table, RLIMIT_NOFILE), Ok(8_000));

        let lower_max_but_higher_cur = Rlimit::new(99_999, 7_000);
        write_current(&mut table, PAGE, &lower_max_but_higher_cur.to_le_bytes());
        assert_eq!(sys_setrlimit32(&mut table, RLIMIT_NOFILE, PAGE), 0);
        let group = table.thread_group(1).unwrap();
        assert_eq!(
            group.limits[RLIMIT_NOFILE as usize],
            Rlimit::new(99_999, 7_000),
            "C does not impose a cur <= max check here"
        );
    }

    #[test]
    fn prlimit_writes_old_before_reading_an_aliasing_new_limit() {
        let mut table = table_with_memory();
        let proposed = Rlimit::new(7, 8);
        write_current(&mut table, PAGE, &proposed.to_le_bytes());

        // `old_limit_addr` aliases `new_limit_addr`: the old value is written
        // first, then read back as the new value. This is C's exact ordering.
        assert_eq!(sys_prlimit64(&mut table, 0, RLIMIT_NOFILE, PAGE, PAGE), 0);
        assert_eq!(rlimit(&table, RLIMIT_NOFILE), Ok(1024));
        assert_eq!(
            Rlimit::from_le_bytes(
                read_current(&mut table, PAGE, Rlimit::SIZE)
                    .try_into()
                    .unwrap()
            ),
            Rlimit::new(1024, 4096)
        );

        assert_eq!(sys_prlimit64(&mut table, 1, RLIMIT_NOFILE, PAGE, 0), EINVAL);
        assert_eq!(sys_prlimit64(&mut table, 0, 99, 0, PAGE + 32), EINVAL);
        // With neither pointer present, C never validates resource at all.
        assert_eq!(sys_prlimit64(&mut table, 0, 99, 0, 0), 0);
        assert_eq!(
            sys_prlimit64(&mut table, 0, RLIMIT_NOFILE, 0x9000_0000, 0),
            EFAULT
        );
    }

    #[test]
    fn rusage_wire_layout_children_and_time_carry_match_c() {
        let mut table = table_with_memory();
        let host = StaticResourceHost {
            rusage: Rusage {
                utime: Timeval {
                    sec: 1,
                    usec: 900_000,
                },
                stime: Timeval { sec: 2, usec: 3 },
                maxrss: 77,
                ..Rusage::default()
            },
            cpus: 1,
        };
        assert_eq!(sys_getrusage(&mut table, &host, RUSAGE_SELF, PAGE), 0);
        let bytes = read_current(&mut table, PAGE, Rusage::SIZE);
        assert_eq!(&bytes[0..4], &1u32.to_le_bytes());
        assert_eq!(&bytes[4..8], &900_000u32.to_le_bytes());
        assert_eq!(&bytes[8..12], &2u32.to_le_bytes());
        assert_eq!(&bytes[16..20], &77u32.to_le_bytes());

        let group = table.group_mut(1).unwrap();
        group.children_rusage = Rusage {
            utime: Timeval {
                sec: 4,
                usec: 800_000,
            },
            stime: Timeval {
                sec: 5,
                usec: 700_000,
            },
            ..Rusage::default()
        };
        assert_eq!(
            sys_getrusage(&mut table, &host, RUSAGE_CHILDREN, PAGE + 128),
            0
        );
        let bytes = read_current(&mut table, PAGE + 128, Rusage::SIZE);
        assert_eq!(&bytes[0..8], &[4, 0, 0, 0, 0, 0x35, 0x0c, 0]);

        let mut total = Rusage {
            utime: Timeval {
                sec: 1,
                usec: 800_000,
            },
            stime: Timeval {
                sec: 2,
                usec: 900_000,
            },
            maxrss: 9,
            ..Rusage::default()
        };
        rusage_add(
            &mut total,
            Rusage {
                utime: Timeval {
                    sec: 3,
                    usec: 300_000,
                },
                stime: Timeval {
                    sec: 4,
                    usec: 200_000,
                },
                maxrss: 10,
                ..Rusage::default()
            },
        );
        assert_eq!(
            total.utime,
            Timeval {
                sec: 5,
                usec: 100_000
            }
        );
        assert_eq!(
            total.stime,
            Timeval {
                sec: 7,
                usec: 100_000
            }
        );
        assert_eq!(total.maxrss, 9, "only time fields are added in C");
        assert_eq!(sys_getrusage(&mut table, &host, 1, PAGE), EINVAL);
    }

    #[test]
    fn forked_thread_groups_copy_limits_and_own_usage_but_not_child_usage() {
        let mut table = table_with_memory();
        {
            let group = table.group_mut(1).unwrap();
            group.limits[RLIMIT_NOFILE as usize] = Rlimit::new(71, 72);
            group.rusage = Rusage {
                utime: Timeval { sec: 3, usec: 4 },
                ..Rusage::default()
            };
            group.children_rusage = Rusage {
                stime: Timeval { sec: 5, usec: 6 },
                ..Rusage::default()
            };
        }
        let child = table.create_task(Some(1)).unwrap();
        table.copy_thread_group_for(child).unwrap();
        let child_group = table.thread_group(child).unwrap();
        assert_eq!(
            child_group.limits[RLIMIT_NOFILE as usize],
            Rlimit::new(71, 72)
        );
        assert_eq!(child_group.rusage.utime, Timeval { sec: 3, usec: 4 });
        assert_eq!(child_group.children_rusage, Rusage::default());
    }

    #[test]
    fn affinity_keeps_ishs_extra_byte_and_live_pid_rule() {
        let mut table = table_with_memory();
        let host = StaticResourceHost {
            cpus: 10,
            ..StaticResourceHost::default()
        };
        assert_eq!(sys_sched_getaffinity(&mut table, &host, 0, 1, PAGE), EINVAL);
        assert_eq!(sys_sched_getaffinity(&mut table, &host, 0, 2, PAGE), 2);
        assert_eq!(read_current(&mut table, PAGE, 2), vec![0xff, 0x03]);

        let child = table.create_task(Some(1)).unwrap();
        assert_eq!(sys_sched_getaffinity(&mut table, &host, child, 2, PAGE), 2);
        table.mark_zombie(child, 0).unwrap();
        assert_eq!(
            sys_sched_getaffinity(&mut table, &host, child, 2, PAGE),
            ESRCH
        );
    }

    #[test]
    fn scheduler_priority_and_ioprio_stubs_keep_their_narrow_contracts() {
        let mut table = table_with_memory();
        assert_eq!(sys_sched_getparam(&mut table, 123, PAGE), 0);
        assert_eq!(read_current(&mut table, PAGE, 4), vec![0; 4]);
        assert_eq!(sys_sched_getparam(&mut table, 123, 0x9000_0000), EFAULT);

        write_current(&mut table, PAGE, &0i32.to_le_bytes());
        assert_eq!(sys_sched_setscheduler(&mut table, 4, SCHED_OTHER, PAGE), 0);
        write_current(&mut table, PAGE, &1i32.to_le_bytes());
        assert_eq!(
            sys_sched_setscheduler(&mut table, 4, SCHED_OTHER, PAGE),
            EINVAL
        );
        assert_eq!(
            sys_sched_setscheduler(&mut table, 4, 1, 0x9000_0000),
            EINVAL
        );
        assert_eq!(sys_sched_getscheduler(123), SCHED_OTHER);
        assert_eq!(sys_sched_get_priority_max(SCHED_OTHER), 0);
        assert_eq!(sys_sched_get_priority_max(1), EINVAL);
        assert_eq!(sys_sched_setaffinity(123, 0, 0), 0);
        assert_eq!(sys_getpriority(1, 2), 20);
        assert_eq!(sys_setpriority(1, 2, -20), 0);
        assert_eq!(sys_ioprio_get(1, 2), 0);
        assert_eq!(sys_ioprio_set(1, 2, 3), 0);
    }
}
