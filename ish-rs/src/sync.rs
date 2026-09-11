//! `util/sync.{h,c}` — locks, condition variables, and the waiter contract.
//!
//! iSH builds every blocking path in the kernel on three primitives: a mutex
//! (`lock_t`), a condition variable (`cond_t`), and a writer-preferring
//! read/write lock (`wrlock_t`). `wait_for` adds the rule that a wait must end
//! early when the task has an unblocked pending signal, and the per-task
//! `waiting_cond` / `waiting_lock` pair is how `kernel/signal.c` finds the
//! waiter to wake.
//!
//! # What the port keeps
//!
//! * **The two `EINTR` checks and the timeout mapping.** `wait_for` checks for a
//!   pending signal *before* and *after* the blocking wait, and turns any
//!   failed wait into `_ETIMEDOUT`. `wait_for_ignore_signals` does neither: it
//!   waits once and reports only timeouts. `tests/sync_differential.rs` replays
//!   every branch of both against the unmodified C.
//! * **The deadline arithmetic, quirks included.** C computes `now + timeout`
//!   and applies the nanosecond carry only when the sum is *strictly greater*
//!   than `1000000000`, so an exact `1e9` survives as an invalid `tv_nsec`.
//!   The port computes the same value, and the default [`ThreadParker`] reports
//!   [`ParkResult::Failed`] for it — which is what pthread's `EINVAL` produces
//!   in C, and therefore what C's `rc != ETIMEDOUT` rule turns back into
//!   success.
//! * **The published waiter.** While the wait is in flight the port calls
//!   [`SyncHost::set_waiting`] with the [`CondId`]/[`LockId`] pair, mirroring
//!   `current->waiting_cond` / `current->waiting_lock`, and clears it after.
//! * **`trylock` does not claim ownership.** In C only `lock()` stores
//!   `pthread_self()` in `lock->owner`; `trylock` leaves it alone, which
//!   `kernel/signal.c`'s `pthread_equal(lock->owner, pthread_self())` probe and
//!   `fs/lock.c` depend on. The port reproduces that, and the corpus pins it.
//!
//! # Deliberate differences
//!
//! * **Guards instead of a raw mutex pointer.** C hands `wait_for` a `lock_t *`
//!   that pthread unlocks and relocks around the wait. Rust's version takes the
//!   [`LockGuard`] by value and hands it back, which preserves the C invariant
//!   that the caller holds the lock before and after the wait — including the
//!   fact that `lock->owner` stays set for the whole wait.
//! * **`sigunwind_start` cannot longjmp.** C arms a `sigsetjmp` and lets
//!   `sigusr1_handler` jump back into it. There is no safe Rust equivalent, so
//!   the request is explicit: [`sigusr1_handler`] records it, and the next
//!   [`sigunwind_start`] reports [`Unwind::Unwound`] — the same observable
//!   transition without a jump into a frame the caller has left.
//! * **No writer preference.** C configures
//!   `PTHREAD_RWLOCK_PREFER_WRITER_NONRECURSIVE_NP` on glibc;
//!   `std::sync::RwLock` offers no preference knob. The port keeps the same
//!   `val` transitions and the same assertions.
//! * **`is_signal_pending` does not take the sighand lock.** C locks
//!   `current->sighand->lock` unless the caller already holds it, purely to
//!   read `pending` and `blocked` safely. The port asks the host for the
//!   boolean: those masks are kernel state, not state behind the lock being
//!   waited on, so there is nothing to re-enter.

use std::sync::atomic::{AtomicI32, AtomicU64, Ordering};
use std::sync::{
    Condvar, LockResult, Mutex, MutexGuard, RwLock, RwLockReadGuard, RwLockWriteGuard,
};
use std::sync::{PoisonError, TryLockError};
use std::thread::ThreadId;
use std::time::{Duration, Instant};

use crate::task::Pid;

/// `_EINTR`: a wait was interrupted by a pending signal.
pub const EINTR: i32 = -4;
/// `EINVAL`, the status pthread returns for an invalid absolute deadline.
pub const EINVAL: i32 = 22;
/// `_ETIMEDOUT`: a wait expired.
pub const ETIMEDOUT: i32 = -110;
/// `EBUSY`: `trylock` found the lock held.
pub const EBUSY: i32 = 16;

/// `struct timespec` as the kernel uses it.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Timespec {
    /// `tv_sec`
    pub sec: i64,
    /// `tv_nsec`
    pub nsec: i64,
}

impl Timespec {
    /// A time value in seconds and nanoseconds.
    #[must_use]
    pub const fn new(sec: i64, nsec: i64) -> Self {
        Self { sec, nsec }
    }

    /// A monotonic duration as a time value, the shape `clock_gettime` fills in.
    #[must_use]
    pub fn from_duration(duration: Duration) -> Self {
        Self {
            sec: duration.as_secs() as i64,
            nsec: i64::from(duration.subsec_nanos()),
        }
    }

    /// C's `abs_timeout` computation inside `wait_for_ignore_signals`:
    /// `now + timeout`, with the carry applied only when the nanosecond sum is
    /// *strictly greater* than one second.
    ///
    /// The strictness is a C quirk, not a typo: a sum of exactly `1000000000`
    /// stays unnormalized and pthread then rejects that deadline with `EINVAL`.
    /// [`ThreadParker`] reproduces it by reporting [`ParkResult::Failed`].
    #[must_use]
    pub const fn deadline_after(now: Self, timeout: Self) -> Self {
        let mut sec = now.sec + timeout.sec;
        let mut nsec = now.nsec + timeout.nsec;
        if nsec > 1_000_000_000 {
            nsec -= 1_000_000_000;
            sec += 1;
        }
        Self { sec, nsec }
    }

    /// Whether the value could be handed to a real `pthread_cond_timedwait`:
    /// pthread rejects a negative or `>= 1e9` nanosecond field.
    #[must_use]
    pub const fn is_normalized(self) -> bool {
        self.nsec >= 0 && self.nsec < 1_000_000_000
    }

    /// The relative duration between two monotonic readings, saturating at
    /// zero, the value pthread derives from an absolute deadline.
    #[must_use]
    pub fn duration_since(self, earlier: Self) -> Duration {
        let delta = (i128::from(self.sec) - i128::from(earlier.sec)) * 1_000_000_000
            + (i128::from(self.nsec) - i128::from(earlier.nsec));
        if delta <= 0 {
            return Duration::ZERO;
        }
        Duration::from_nanos(u64::try_from(delta).unwrap_or(u64::MAX))
    }
}

/// Identity of a [`Cond`], standing in for the C `cond_t *`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct CondId(u64);

/// Identity of a [`Lock`], standing in for the C `lock_t *`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct LockId(u64);

fn next_id(counter: &AtomicU64) -> u64 {
    counter.fetch_add(1, Ordering::Relaxed) + 1
}

static NEXT_COND_ID: AtomicU64 = AtomicU64::new(0);
static NEXT_LOCK_ID: AtomicU64 = AtomicU64::new(0);

/// The task's published wait, mirroring `current->waiting_cond` and
/// `current->waiting_lock`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Waiting {
    /// The condition the task is sleeping on.
    pub cond: CondId,
    /// The lock that condvar was handed, which C records too so that signal
    /// delivery can reason about who would release it.
    pub lock: LockId,
}

/// `cond_t`: a condition variable.
///
/// C sets `CLOCK_MONOTONIC` as the condvar's clock on Linux; `Condvar` in
/// `std` waits against the monotonic clock there as well, so this type only has
/// to keep the identity C gets from the object's address.
#[derive(Debug)]
pub struct Cond {
    condvar: Condvar,
    id: CondId,
}

impl Default for Cond {
    fn default() -> Self {
        Self::new()
    }
}

impl Cond {
    /// `cond_init`.
    #[must_use]
    pub fn new() -> Self {
        Self {
            condvar: Condvar::new(),
            id: CondId(next_id(&NEXT_COND_ID)),
        }
    }

    /// The object's identity: the port's `cond_t *`.
    #[must_use]
    pub fn id(&self) -> CondId {
        self.id
    }

    /// `notify`: wake every waiter.
    pub fn notify_all(&self) {
        self.condvar.notify_all();
    }

    /// `notify_once`: wake one waiter.
    pub fn notify_one(&self) {
        self.condvar.notify_one();
    }
}

/// `lock_t`: a mutex that remembers which thread holds it.
///
/// `owner` is not bookkeeping for its own sake: `kernel/signal.c` uses
/// `pthread_equal(task->waiting_lock->owner, pthread_self())` to decide whether
/// the task being signalled is the one that would have to release that lock,
/// and `fs/lock.c` probes it while a file lock is contended.
#[derive(Debug)]
pub struct Lock {
    inner: Mutex<()>,
    owner: Mutex<Option<ThreadId>>,
    id: LockId,
}

impl Default for Lock {
    fn default() -> Self {
        Self::new()
    }
}

impl Lock {
    /// `lock_init`.
    #[must_use]
    pub fn new() -> Self {
        Self {
            inner: Mutex::new(()),
            owner: Mutex::new(None),
            id: LockId(next_id(&NEXT_LOCK_ID)),
        }
    }

    /// The object's identity: the port's `lock_t *`.
    #[must_use]
    pub fn id(&self) -> LockId {
        self.id
    }

    /// `lock`: acquire, recording the owner.
    pub fn lock(&self) -> LockGuard<'_> {
        let inner = self.inner.lock().unwrap_or_else(PoisonError::into_inner);
        *self.owner.lock().unwrap_or_else(PoisonError::into_inner) =
            Some(std::thread::current().id());
        LockGuard {
            lock: self,
            inner: Some(inner),
        }
    }

    /// `trylock`: acquire if free, reporting the pthread status.
    ///
    /// Like C, a successful `trylock` does **not** set `owner`; only [`Lock::lock`]
    /// does. That is observable through `signal.c`'s probe and pinned by
    /// `tests/sync_differential.rs`.
    pub fn try_lock(&self) -> Result<LockGuard<'_>, i32> {
        let inner = match self.inner.try_lock() {
            Ok(inner) => inner,
            // C's mutexes are not poisoned; a poisoned one is still held.
            Err(TryLockError::Poisoned(error)) => error.into_inner(),
            Err(TryLockError::WouldBlock) => return Err(EBUSY),
        };
        Ok(LockGuard {
            lock: self,
            inner: Some(inner),
        })
    }

    /// Whether `owner` names the calling thread: C's
    /// `pthread_equal(lock->owner, pthread_self())`.
    #[must_use]
    pub fn owner_is_current_thread(&self) -> bool {
        self.owner
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .is_some_and(|owner| owner == std::thread::current().id())
    }

    /// Whether no thread is recorded as the owner: C's
    /// `pthread_equal(lock->owner, zero_init(pthread_t))`.
    #[must_use]
    pub fn owner_is_clear(&self) -> bool {
        self.owner
            .lock()
            .unwrap_or_else(PoisonError::into_inner)
            .is_none()
    }
}

/// The guard [`Lock::lock`] and [`Lock::try_lock`] hand out.
///
/// C's `unlock` zeroes `owner` and then releases the mutex; dropping this guard
/// does the same, in the same order. [`wait_for`] takes one by value and hands
/// it back, so the owner stays recorded across a wait exactly as in C.
#[derive(Debug)]
pub struct LockGuard<'a> {
    lock: &'a Lock,
    // `Option` because a wait has to hand the guard to the parker, which
    // consumes it, and a type with a `Drop` impl cannot be destructured.
    inner: Option<MutexGuard<'a, ()>>,
}

impl<'a> LockGuard<'a> {
    /// The lock this guard belongs to.
    #[must_use]
    pub fn lock(&self) -> &'a Lock {
        self.lock
    }

    /// Split the guard for a wait, keeping `owner` recorded exactly as C does:
    /// the caller's lock stays owned for the whole wait and is rebuilt around
    /// the returned mutex guard.
    fn into_parts(mut self) -> (&'a Lock, MutexGuard<'a, ()>) {
        let inner = self.inner.take().expect("a lock guard is only split once");
        let lock = self.lock;
        // Dropping `self` here would clear `owner`, which C keeps set while
        // pthread_cond_wait releases and reacquires the mutex.
        std::mem::forget(self);
        (lock, inner)
    }

    fn from_parts(lock: &'a Lock, inner: MutexGuard<'a, ()>) -> Self {
        Self {
            lock,
            inner: Some(inner),
        }
    }
}

impl Drop for LockGuard<'_> {
    fn drop(&mut self) {
        *self
            .lock
            .owner
            .lock()
            .unwrap_or_else(PoisonError::into_inner) = None;
    }
}

/// What a parked wait returned, mirroring pthread's return values.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ParkResult {
    /// The wait was signalled.
    Woken,
    /// `ETIMEDOUT`: the deadline passed.
    TimedOut,
    /// Any other pthread status, such as `EINVAL` for an invalid deadline or
    /// `EINTR` when a signal handler interrupted the wait. C treats every one
    /// of these as a successful, non-timed-out wait.
    Failed(i32),
}

/// The blocking primitive C gets from pthread, injected so the wait itself can
/// be replaced.
///
/// The method is generic over the guard's payload because the futex table parks
/// while holding its own state mutex, and `Condvar` accepts any guard type.
pub trait Parker {
    /// `pthread_cond_wait` / `pthread_cond_timedwait` with `guard` held; returns
    /// with the same guard held. `now` is the monotonic reading the caller used
    /// to build `deadline`, so an implementation can recover the relative
    /// duration pthread would compute from the two.
    fn park<'g, T>(
        &self,
        cond: &Cond,
        guard: MutexGuard<'g, T>,
        now: Timespec,
        deadline: Option<Timespec>,
    ) -> (ParkResult, MutexGuard<'g, T>);
}

/// The production parker: real condition-variable waits.
#[derive(Debug, Default, Clone, Copy)]
pub struct ThreadParker;

impl Parker for ThreadParker {
    fn park<'g, T>(
        &self,
        cond: &Cond,
        guard: MutexGuard<'g, T>,
        now: Timespec,
        deadline: Option<Timespec>,
    ) -> (ParkResult, MutexGuard<'g, T>) {
        let Some(deadline) = deadline else {
            let guard = cond
                .condvar
                .wait(guard)
                .unwrap_or_else(PoisonError::into_inner);
            return (ParkResult::Woken, guard);
        };
        if !deadline.is_normalized() {
            // pthread rejects such a deadline, and C's strict carry test is
            // exactly how one can reach it.
            return (ParkResult::Failed(EINVAL), guard);
        }
        let result: LockResult<_> = cond
            .condvar
            .wait_timeout(guard, deadline.duration_since(now));
        let (guard, timeout) = result.unwrap_or_else(PoisonError::into_inner);
        let outcome = if timeout.timed_out() {
            ParkResult::TimedOut
        } else {
            ParkResult::Woken
        };
        (outcome, guard)
    }
}

/// The host state `util/sync.c` reads from the kernel: the current task's
/// signal masks, the waiter it publishes, and the unwind flag.
///
/// C reaches all of this through the thread-local `current`; a trait keeps the
/// same seams explicit so the waiting rules can be replayed without a running
/// kernel, exactly as `kernel/resource.c` and `kernel/futex.c` are.
pub trait SyncHost {
    /// `is_signal_pending`: `!!(current->pending & ~current->blocked)`, with no
    /// current task meaning "false".
    fn signal_pending(&self) -> bool;

    /// `clock_gettime(CLOCK_MONOTONIC, &now)`.
    fn monotonic_now(&self) -> Timespec;

    /// Publish or clear `current->waiting_cond` / `current->waiting_lock`. C
    /// skips this entirely when there is no current task.
    fn set_waiting(&self, waiting: Option<Waiting>);

    /// `should_unwind`.
    fn should_unwind(&self) -> bool;

    /// Assign `should_unwind`.
    fn set_should_unwind(&self, value: bool);

    /// Record that `sigusr1_handler` jumped, so that the next
    /// [`sigunwind_start`] can report it as the C `sigsetjmp` returning one.
    fn record_pending_unwind(&self);

    /// Consume a recorded jump, reporting whether there was one. This is the
    /// `if (sigsetjmp(unwind_buf, 1))` branch of C's `sigunwind_start`.
    fn take_pending_unwind(&self) -> bool;
}

/// `wait_for_ignore_signals`: wait once, ignoring pending signals.
///
/// C publishes the task as a waiter, releases the lock for the duration of the
/// wait, and reports `_ETIMEDOUT` only when pthread says the wait timed out;
/// every other failure is reported as success.
pub fn wait_for_ignore_signals<'a, P: Parker>(
    cond: &Cond,
    guard: LockGuard<'a>,
    timeout: Option<Timespec>,
    host: &dyn SyncHost,
    parker: &P,
) -> (i32, LockGuard<'a>) {
    let (lock, inner) = guard.into_parts();
    host.set_waiting(Some(Waiting {
        cond: cond.id(),
        lock: lock.id(),
    }));
    // C reads the clock only when it has a relative timeout to add to it.
    let now = timeout.map(|_| host.monotonic_now());
    let deadline =
        timeout.map(|timeout| Timespec::deadline_after(now.unwrap_or_default(), timeout));
    let (result, inner) = parker.park(cond, inner, now.unwrap_or_default(), deadline);
    host.set_waiting(None);
    let guard = LockGuard::from_parts(lock, inner);
    match result {
        ParkResult::TimedOut => (ETIMEDOUT, guard),
        _ => (0, guard),
    }
}

/// `wait_for`: wait, but stop early when the task has a pending signal.
pub fn wait_for<'a, P: Parker>(
    cond: &Cond,
    guard: LockGuard<'a>,
    timeout: Option<Timespec>,
    host: &dyn SyncHost,
    parker: &P,
) -> (i32, LockGuard<'a>) {
    if host.signal_pending() {
        return (EINTR, guard);
    }
    let (err, guard) = wait_for_ignore_signals(cond, guard, timeout, host, parker);
    if err < 0 {
        return (ETIMEDOUT, guard);
    }
    if host.signal_pending() {
        return (EINTR, guard);
    }
    (0, guard)
}

/// `notify`: wake every waiter.
pub fn notify(cond: &Cond) {
    cond.notify_all();
}

/// `notify_once`: wake one waiter.
pub fn notify_once(cond: &Cond) {
    cond.notify_one();
}

/// The result of [`sigunwind_start`].
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Unwind {
    /// The C `sigsetjmp` returned zero: a fresh unwind is armed.
    Fresh,
    /// The C `sigsetjmp` returned one, because `sigusr1_handler` jumped back
    /// into it.
    Unwound,
}

/// `sigunwind_start`: arm an unwind, or report the one that just completed.
pub fn sigunwind_start(host: &dyn SyncHost) -> Unwind {
    if host.take_pending_unwind() {
        // The C branch that returns 1 also clears the flag, because the jump
        // may have interrupted a wait that had re-armed it.
        host.set_should_unwind(false);
        return Unwind::Unwound;
    }
    host.set_should_unwind(true);
    Unwind::Fresh
}

/// `sigunwind_end`: disarm the unwind.
pub fn sigunwind_end(host: &dyn SyncHost) {
    host.set_should_unwind(false);
}

/// `sigusr1_handler`: clear the flag and record the jump when one is armed,
/// reporting whether it unwound.
pub fn sigusr1_handler(host: &dyn SyncHost) -> bool {
    if host.should_unwind() {
        host.set_should_unwind(false);
        host.record_pending_unwind();
        return true;
    }
    false
}

/// A [`SyncHost`] for a caller with no current task — the C `current == NULL`
/// case, where no signal can be pending and no wait is published.
#[derive(Debug)]
pub struct NoTaskHost {
    start: Instant,
    should_unwind: std::cell::Cell<bool>,
    pending_unwind: std::cell::Cell<bool>,
}

impl Default for NoTaskHost {
    fn default() -> Self {
        Self::new()
    }
}

impl NoTaskHost {
    /// A host whose monotonic clock starts now.
    #[must_use]
    pub fn new() -> Self {
        Self {
            start: Instant::now(),
            should_unwind: std::cell::Cell::new(false),
            pending_unwind: std::cell::Cell::new(false),
        }
    }

    /// The state `sigusr1_handler` left for the next [`sigunwind_start`].
    #[must_use]
    pub fn has_pending_unwind(&self) -> bool {
        self.pending_unwind.get()
    }
}

impl SyncHost for NoTaskHost {
    fn signal_pending(&self) -> bool {
        false
    }

    fn monotonic_now(&self) -> Timespec {
        Timespec::from_duration(self.start.elapsed())
    }

    fn set_waiting(&self, _waiting: Option<Waiting>) {}

    fn should_unwind(&self) -> bool {
        self.should_unwind.get()
    }

    fn set_should_unwind(&self, value: bool) {
        self.should_unwind.set(value);
    }

    fn record_pending_unwind(&self) {
        self.pending_unwind.set(true);
    }

    fn take_pending_unwind(&self) -> bool {
        let pending = self.pending_unwind.get();
        self.pending_unwind.set(false);
        pending
    }
}

/// `wrlock_t`'s debug record: the file, line, and task that took the write
/// lock. C keeps them for the same reason — a contended lock wants a name.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct LockDebugRecord {
    /// `file`
    pub file: &'static str,
    /// `line`
    pub line: u32,
    /// `pid`, the write-locking task.
    pub pid: Pid,
}

/// `wrlock_t`: a read/write lock that counts its readers.
///
/// C keeps `val` beside the pthread rwlock for assertions and debugging (`0`
/// unlocked, `-1` write-locked, `> 0` readers) and records `file`/`line`/`pid`
/// on a write lock. The port keeps both, so the observable state matches even
/// though `std::sync::RwLock` has no writer-preference knob.
#[derive(Debug)]
pub struct WrLock {
    inner: RwLock<()>,
    val: AtomicI32,
    debug: Mutex<Option<LockDebugRecord>>,
}

impl Default for WrLock {
    fn default() -> Self {
        Self::new()
    }
}

impl WrLock {
    /// `wrlock_init`.
    #[must_use]
    pub const fn new() -> Self {
        Self {
            inner: RwLock::new(()),
            val: AtomicI32::new(0),
            debug: Mutex::new(None),
        }
    }

    /// `read_wrlock`.
    pub fn read(&self) -> ReadGuard<'_> {
        let guard = self.inner.read().unwrap_or_else(PoisonError::into_inner);
        let previous = self.val.fetch_add(1, Ordering::Relaxed);
        assert!(previous >= 0, "read_wrlock on a write-locked lock");
        ReadGuard { lock: self, guard }
    }

    /// `write_wrlock`.
    pub fn write(&self, file: &'static str, line: u32, pid: Pid) -> WriteGuard<'_> {
        let guard = self.inner.write().unwrap_or_else(PoisonError::into_inner);
        assert_eq!(
            self.val.swap(-1, Ordering::Relaxed),
            0,
            "write_wrlock on a locked lock"
        );
        *self.debug.lock().unwrap_or_else(PoisonError::into_inner) =
            Some(LockDebugRecord { file, line, pid });
        WriteGuard { lock: self, guard }
    }

    /// `lock->val`.
    #[must_use]
    pub fn val(&self) -> i32 {
        self.val.load(Ordering::Relaxed)
    }

    /// `lock->file`, `lock->line`, and `lock->pid`.
    #[must_use]
    pub fn debug(&self) -> Option<LockDebugRecord> {
        *self.debug.lock().unwrap_or_else(PoisonError::into_inner)
    }
}

/// The guard [`WrLock::read`] hands out; dropping it is `read_wrunlock`.
#[derive(Debug)]
pub struct ReadGuard<'a> {
    lock: &'a WrLock,
    guard: RwLockReadGuard<'a, ()>,
}

impl Drop for ReadGuard<'_> {
    fn drop(&mut self) {
        let previous = self.lock.val.fetch_sub(1, Ordering::Relaxed);
        assert!(previous > 0, "read_wrunlock on an unlocked lock");
        let _ = &self.guard;
    }
}

/// The guard [`WrLock::write`] hands out; dropping it is `write_wrunlock`.
#[derive(Debug)]
pub struct WriteGuard<'a> {
    lock: &'a WrLock,
    guard: RwLockWriteGuard<'a, ()>,
}

impl Drop for WriteGuard<'_> {
    fn drop(&mut self) {
        assert_eq!(
            self.lock.val.load(Ordering::Relaxed),
            -1,
            "write_wrunlock on an unlocked write lock"
        );
        self.lock.val.store(0, Ordering::Relaxed);
        *self
            .lock
            .debug
            .lock()
            .unwrap_or_else(PoisonError::into_inner) = None;
        let _ = &self.guard;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::AtomicUsize;
    use std::sync::Arc;

    /// A host whose signal state the tests drive directly. Atomics rather than
    /// `Cell` because the "signal arrives during the wait" case needs a second
    /// thread to deliver it.
    #[derive(Default)]
    struct SignalledHost {
        pending: AtomicU64,
        blocked: AtomicU64,
        now: Timespec,
    }

    impl SyncHost for SignalledHost {
        fn signal_pending(&self) -> bool {
            (self.pending.load(Ordering::SeqCst) & !self.blocked.load(Ordering::SeqCst)) != 0
        }

        fn monotonic_now(&self) -> Timespec {
            self.now
        }

        fn set_waiting(&self, _waiting: Option<Waiting>) {}

        fn should_unwind(&self) -> bool {
            false
        }

        fn set_should_unwind(&self, _value: bool) {}

        fn record_pending_unwind(&self) {}

        fn take_pending_unwind(&self) -> bool {
            false
        }
    }

    #[test]
    fn deadline_arithmetic_matches_the_c_inline_code() {
        // No carry.
        assert_eq!(
            Timespec::deadline_after(
                Timespec::new(100, 250_000_000),
                Timespec::new(1, 500_000_000)
            ),
            Timespec::new(101, 750_000_000)
        );
        // Carry: the sum is strictly greater than one second.
        assert_eq!(
            Timespec::deadline_after(
                Timespec::new(100, 700_000_000),
                Timespec::new(0, 500_000_000)
            ),
            Timespec::new(101, 200_000_000)
        );
        // The quirk: exactly 1e9 stays unnormalized.
        let exact = Timespec::deadline_after(
            Timespec::new(100, 600_000_000),
            Timespec::new(0, 400_000_000),
        );
        assert_eq!(exact, Timespec::new(100, 1_000_000_000));
        assert!(!exact.is_normalized());
        assert_eq!(
            exact.duration_since(Timespec::new(100, 0)).as_nanos(),
            1_000_000_000
        );
        assert_eq!(
            Timespec::new(99, 0).duration_since(Timespec::new(100, 0)),
            Duration::ZERO
        );
    }

    #[test]
    fn try_lock_leaves_the_owner_unset_like_c() {
        let lock = Lock::new();
        assert!(lock.owner_is_clear());
        let guard = lock.lock();
        assert!(lock.owner_is_current_thread());
        assert_eq!(lock.try_lock().err(), Some(EBUSY));
        assert!(
            lock.owner_is_current_thread(),
            "a failed trylock must not disturb the owner"
        );
        drop(guard);
        assert!(lock.owner_is_clear());
        let guard = lock.try_lock().expect("the lock is free");
        assert!(lock.owner_is_clear(), "C's trylock records no owner");
        drop(guard);
    }

    #[test]
    fn write_lock_counters_and_debug_fields_track_c() {
        let lock = WrLock::new();
        assert_eq!(lock.val(), 0);
        assert_eq!(lock.debug(), None);
        let first = lock.read();
        let second = lock.read();
        assert_eq!(lock.val(), 2);
        drop(first);
        assert_eq!(lock.val(), 1);
        drop(second);
        assert_eq!(lock.val(), 0);
        let write = lock.write("src/sync.rs", 7, 47);
        assert_eq!(lock.val(), -1);
        assert_eq!(
            lock.debug(),
            Some(LockDebugRecord {
                file: "src/sync.rs",
                line: 7,
                pid: 47
            })
        );
        drop(write);
        assert_eq!(lock.val(), 0);
        assert_eq!(lock.debug(), None);
    }

    #[test]
    fn no_task_host_has_a_running_monotonic_clock() {
        let host = NoTaskHost::new();
        assert!(!host.signal_pending());
        let first = host.monotonic_now();
        let second = host.monotonic_now();
        assert!(second.sec > first.sec || second.nsec >= first.nsec);
        assert!(!host.has_pending_unwind());
    }

    #[test]
    fn unwind_arming_disarming_and_the_explicit_jump() {
        let host = NoTaskHost::new();
        assert_eq!(sigunwind_start(&host), Unwind::Fresh);
        assert!(host.should_unwind());
        assert!(sigusr1_handler(&host), "the armed unwind fires");
        assert!(!host.should_unwind());
        assert!(host.has_pending_unwind());
        // The next arm reports the completed jump, like the C `sigsetjmp`
        // branch returning 1.
        assert_eq!(sigunwind_start(&host), Unwind::Unwound);
        assert!(!host.has_pending_unwind());
        sigunwind_end(&host);
        assert!(!host.should_unwind());
        assert!(!sigusr1_handler(&host), "no unwind is armed any more");
        assert_eq!(sigunwind_start(&host), Unwind::Fresh);
    }

    /// Wait until `waiters` threads are parked on `cond`, then return holding
    /// the lock so the caller can notify without a lost wakeup.
    ///
    /// Each waiter bumps `ready` while holding `lock`, so once this caller
    /// holds `lock` and sees the full count, every waiter has necessarily
    /// released the mutex through `park` and is committed to waking up.
    fn wait_until_parked<'a>(lock: &'a Lock, ready: &AtomicUsize, waiters: usize) -> LockGuard<'a> {
        loop {
            let guard = lock.lock();
            if ready.load(Ordering::SeqCst) == waiters {
                return guard;
            }
            drop(guard);
            std::thread::yield_now();
        }
    }

    #[test]
    fn real_threads_wake_the_right_waiters() {
        let cond = Arc::new(Cond::new());
        let lock = Arc::new(Lock::new());
        let ready = Arc::new(AtomicUsize::new(0));
        let seen = Arc::new(Mutex::new(Vec::new()));

        // `notify` wakes every waiter.
        std::thread::scope(|scope| {
            for index in 0..3u32 {
                let (cond, lock, ready, seen) = (
                    Arc::clone(&cond),
                    Arc::clone(&lock),
                    Arc::clone(&ready),
                    Arc::clone(&seen),
                );
                scope.spawn(move || {
                    let host = NoTaskHost::new();
                    let guard = lock.lock();
                    ready.fetch_add(1, Ordering::SeqCst);
                    // A real 5s bound so a lost wakeup fails instead of hanging.
                    let (rc, guard) = wait_for(
                        &cond,
                        guard,
                        Some(Timespec::new(5, 0)),
                        &host,
                        &ThreadParker,
                    );
                    drop(guard);
                    assert_eq!(rc, 0);
                    seen.lock().unwrap().push(index);
                });
            }
            let guard = wait_until_parked(&lock, &ready, 3);
            notify(&cond);
            drop(guard);
        });
        assert_eq!(seen.lock().unwrap().len(), 3);
    }

    /// `notify_once` must reach a waiter, and must not reach all of them;
    /// `notify` must then release the rest.
    ///
    /// The upper bound is `waiters - 1` rather than exact equality on purpose.
    /// Rust's `Condvar::notify_one` wakes one *blocked* waiter, but a waiter
    /// that has already registered and has not yet reached the kernel wait can
    /// return from the same notification too — measured at roughly 7% of runs
    /// with two racing waiters, and at none in 1,500 runs when they were given
    /// a settling pause first. C's `pthread_cond_signal` only promises "at
    /// least one" as well, and every C caller re-checks its predicate in a
    /// loop, so the port cannot depend on the stronger behaviour either. What
    /// the test still catches is the mistake that matters: a `notify_once`
    /// implemented as `notify` wakes all four waiters below before the
    /// `notify` at the end.
    #[test]
    fn notify_once_wakes_one_waiter_and_notify_the_rest() {
        const WAITERS: usize = 4;

        let cond = Arc::new(Cond::new());
        let lock = Arc::new(Lock::new());
        let ready = Arc::new(AtomicUsize::new(0));
        let counter = Arc::new(AtomicUsize::new(0));

        std::thread::scope(|scope| {
            for _ in 0..WAITERS {
                let (cond, lock, ready, counter) = (
                    Arc::clone(&cond),
                    Arc::clone(&lock),
                    Arc::clone(&ready),
                    Arc::clone(&counter),
                );
                scope.spawn(move || {
                    let host = NoTaskHost::new();
                    let guard = lock.lock();
                    ready.fetch_add(1, Ordering::SeqCst);
                    let (rc, guard) = wait_for(
                        &cond,
                        guard,
                        Some(Timespec::new(5, 0)),
                        &host,
                        &ThreadParker,
                    );
                    drop(guard);
                    assert_eq!(rc, 0);
                    counter.fetch_add(1, Ordering::Relaxed);
                });
            }
            let guard = wait_until_parked(&lock, &ready, WAITERS);
            notify_once(&cond);
            drop(guard);
            // Give every waiter that is going to return from this one
            // notification time to record itself, then check that the rest are
            // still parked.
            while counter.load(Ordering::Relaxed) == 0 {
                std::thread::yield_now();
            }
            std::thread::sleep(Duration::from_millis(50));
            let woken = counter.load(Ordering::Relaxed);
            assert!(
                (1..WAITERS).contains(&woken),
                "notify_once must wake some, but not all, of the waiters: {woken} of {WAITERS}"
            );
            let guard = lock.lock();
            notify(&cond);
            drop(guard);
        });
        assert_eq!(counter.load(Ordering::Relaxed), WAITERS);
    }

    /// A millisecond-long wait: long enough to cross a scheduler tick, short
    /// enough to keep the suite fast.
    const TICK: Timespec = Timespec::new(0, 1_000_000);

    #[test]
    fn a_pending_signal_interrupts_only_the_signal_aware_wait() {
        let host = SignalledHost {
            pending: AtomicU64::new(1),
            now: Timespec::new(500, 0),
            ..SignalledHost::default()
        };
        let cond = Cond::new();
        let lock = Lock::new();

        // The pre-wait check returns before anything blocks.
        let guard = lock.lock();
        let (rc, guard) = wait_for(&cond, guard, None, &host, &ThreadParker);
        assert_eq!(rc, EINTR, "the pre-wait check must win");

        // The ignoring variant never checks, so it runs to the deadline:
        // `ETIMEDOUT`, never `EINTR`.
        let (rc, guard) = wait_for_ignore_signals(&cond, guard, Some(TICK), &host, &ThreadParker);
        assert_eq!(rc, ETIMEDOUT, "the ignoring variant never checks signals");
        drop(guard);

        // A masked signal is not pending, so even the checking variant waits.
        host.blocked.store(1, Ordering::SeqCst);
        let guard = lock.lock();
        let (rc, guard) = wait_for(&cond, guard, Some(TICK), &host, &ThreadParker);
        assert_eq!(
            rc, ETIMEDOUT,
            "a blocked signal must not interrupt the wait"
        );
        drop(guard);
    }

    #[test]
    fn a_signal_that_arrives_during_the_wait_is_caught_afterwards() {
        let host = Arc::new(SignalledHost {
            now: Timespec::new(500, 0),
            ..SignalledHost::default()
        });
        let cond = Arc::new(Cond::new());
        let lock = Arc::new(Lock::new());

        std::thread::scope(|scope| {
            // Deliver the signal while the main thread is parked, exactly like
            // `deliver_signal` poking `task->waiting_cond`.
            let (cond_for_waker, host_for_waker) = (Arc::clone(&cond), Arc::clone(&host));
            scope.spawn(move || {
                std::thread::sleep(Duration::from_millis(20));
                host_for_waker.pending.store(1, Ordering::SeqCst);
                notify(&cond_for_waker);
            });

            let guard = lock.lock();
            let (rc, guard) = wait_for(
                &cond,
                guard,
                Some(Timespec::new(30, 0)),
                host.as_ref(),
                &ThreadParker,
            );
            drop(guard);
            assert_eq!(rc, EINTR, "the post-wait check must see the signal");
        });
    }
}
