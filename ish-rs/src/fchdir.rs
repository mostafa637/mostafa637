//! `util/fchdir.{h,c}` — one process-wide lock around `fchdir`, for the call
//! sites that can only work through a relative host path.
//!
//! A working directory is a property of the whole process, so the two places
//! iSH needs one have to serialise against every other thread that might be
//! using relative paths:
//!
//! * `fs/real.c`'s `realfs_mknod` — iOS has no `mknodat`, so a FIFO is created
//!   with `mkfifo(path)`: change into the mount's root with
//!   `fchdir(mount->root_fd)`, create the FIFO, leave.
//! * `tools/fakefs.c`'s tar extractor, which creates FIFOs the same way. It is
//!   a host tool, not part of the emulator, so nothing here depends on it.
//!
//! The C is twelve lines: a `static lock_t fchdir_lock`, a `lock` followed by
//! the host call, and an `unlock`. Everything interesting about it is the
//! *shape* of the call site — the host call has to be inside the lock, and the
//! relative-path work that follows it has to be too — so that is what the
//! differential corpus pins, with two workers contending and a wrapped
//! `fchdir` that reports the descriptor it was handed.
//!
//! # Deliberate differences
//!
//! * **The lock is an object, held as a guard.** C's lock is a file-scope
//!   static that a caller has to remember to release; here
//!   [`FchdirLock::lock_fchdir`] returns a [`LockGuard`] and
//!   [`FchdirLock::unlock_fchdir`] is the explicit release the C call sites
//!   perform. Dropping the guard early is a bug the compiler cannot catch, but
//!   the guard keeps the "held" state visible in the type of the code between
//!   the two calls.
//! * **Exactly one lock per process is the caller's obligation.**
//!   [`process_lock`] answers with the crate's stand-in for C's static; a second
//!   [`FchdirLock::new`] is a second lock, and therefore no mutual exclusion at
//!   all, exactly as a second static would be in C.
//! * **The host call is injected.** C calls `fchdir(2)` directly; here a
//!   [`FchdirHost`] does, so a test can script it. [`UnixFchdirHost`] is the
//!   production one.
//! * **`errno` is not modelled.** C's call sites look at neither `fchdir`'s
//!   result nor `errno`: a failed change of directory leaves the relative-path
//!   call to fail on its own.

use std::sync::OnceLock;

use crate::sync::{Lock, LockGuard};

/// The platform's `fchdir(2)`.
///
/// C calls it from inside the lock and ignores what it returns; the port keeps
/// the return value so a host that wants to distinguish the failure can, and
/// so the corpus can record it.
///
/// The trait carries no `Send`/`Sync` bound of its own: a host is called from
/// whichever thread holds the lock, so a host shared by several threads has to
/// be `Sync` in its own right — which is the embedding's business, exactly as
/// C's file-scope call is nobody's business but the linker's. (The corpus's
/// host is shared across threads, and `tests/fchdir_differential.rs` is where
/// that is spelled out.)
pub trait FchdirHost {
    /// `fchdir(dirfd)`: make the host's working directory the one `dirfd` names,
    /// returning 0 on success and -1 on failure.
    fn fchdir(&self, dirfd: i32) -> i32;
}

#[cfg(unix)]
extern "C" {
    /// `int fchdir(int fd)` from the C library.
    fn fchdir(dirfd: i32) -> i32;
}

/// The production host on unix targets: the same `fchdir(2)` C calls.
#[cfg(unix)]
#[derive(Debug, Default, Clone, Copy)]
pub struct UnixFchdirHost;

#[cfg(unix)]
impl FchdirHost for UnixFchdirHost {
    fn fchdir(&self, dirfd: i32) -> i32 {
        // SAFETY: `fchdir` takes and returns an `int` and has no memory
        // contract; -1 is the failure value the C also gets.
        unsafe { fchdir(dirfd) }
    }
}

/// `static lock_t fchdir_lock`, plus the two functions around it.
#[derive(Debug)]
pub struct FchdirLock {
    lock: Lock,
}

impl Default for FchdirLock {
    fn default() -> Self {
        Self::new()
    }
}

impl FchdirLock {
    /// A fresh lock. Use [`process_lock`] instead unless the point is to test
    /// the lock itself: two instances serialise nothing against each other.
    #[must_use]
    pub fn new() -> Self {
        Self { lock: Lock::new() }
    }

    /// `lock_fchdir`: take the lock, then change the host's working directory
    /// to `dirfd`.
    ///
    /// The lock stays held until the caller passes the returned guard to
    /// [`FchdirLock::unlock_fchdir`], which is what keeps the relative-path work
    /// that follows from racing another thread's.
    ///
    /// Like C, a failing `fchdir` is not reported and does not release the lock:
    /// the caller's relative-path call is what fails, and the lock is still
    /// released by the same `unlock_fchdir`.
    #[must_use]
    pub fn lock_fchdir(&self, host: &impl FchdirHost, dirfd: i32) -> LockGuard<'_> {
        let guard = self.lock.lock();
        // C: `fchdir(dirfd);` — result and errno both dropped.
        let _ = host.fchdir(dirfd);
        guard
    }

    /// `unlock_fchdir`: release the lock `lock_fchdir` took.
    pub fn unlock_fchdir(&self, guard: LockGuard<'_>) {
        drop(guard);
    }

    /// Whether the lock is held right now, for a caller that wants to look
    /// without waiting: C's `trylock(&fchdir_lock)`.
    ///
    /// A free lock is taken and released again, and — like C's `trylock` — no
    /// owner is recorded for the probe.
    #[must_use]
    pub fn is_locked(&self) -> bool {
        self.lock.try_lock().is_err()
    }

    /// `pthread_equal(fchdir_lock.owner, pthread_self())`: whether the calling
    /// thread is the one recorded as holding the lock.
    #[must_use]
    pub fn owner_is_current_thread(&self) -> bool {
        self.lock.owner_is_current_thread()
    }
}

/// The one lock every caller must share, C's file-scope `fchdir_lock`.
///
/// A second [`FchdirLock`] would be a second lock: two threads holding
/// different instances would both change the working directory.
#[must_use]
pub fn process_lock() -> &'static FchdirLock {
    static LOCK: OnceLock<FchdirLock> = OnceLock::new();
    LOCK.get_or_init(FchdirLock::new)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::RefCell;
    use std::rc::Rc;
    use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
    use std::sync::{Condvar, Mutex};

    /// A host that records the descriptors it was handed and whether the lock
    /// was held at the moment of the call.
    #[derive(Default)]
    struct RecordingHost {
        calls: RefCell<Vec<(i32, bool)>>,
        lock: RefCell<Option<Rc<FchdirLock>>>,
        rc: i32,
    }

    impl FchdirHost for RecordingHost {
        fn fchdir(&self, dirfd: i32) -> i32 {
            let held = self
                .lock
                .borrow()
                .as_ref()
                .is_some_and(|lock| lock.is_locked());
            self.calls.borrow_mut().push((dirfd, held));
            self.rc
        }
    }

    #[test]
    fn the_host_call_happens_under_the_lock_and_the_lock_outlives_it() {
        let lock = Rc::new(FchdirLock::new());
        let host = RecordingHost::default();
        *host.lock.borrow_mut() = Some(Rc::clone(&lock));

        assert!(!lock.is_locked(), "a fresh lock is free");
        let guard = lock.lock_fchdir(&host, 7);
        assert_eq!(
            host.calls.borrow().as_slice(),
            &[(7, true)],
            "the host call must be made while the lock is held"
        );
        assert!(lock.is_locked(), "and it is still held afterwards");
        assert!(
            lock.owner_is_current_thread(),
            "`lock` records the owning thread"
        );
        lock.unlock_fchdir(guard);
        assert!(!lock.is_locked());
        assert!(!lock.owner_is_current_thread(), "`unlock` clears the owner");
    }

    #[test]
    fn a_failing_host_call_is_still_made_under_the_lock() {
        let lock = Rc::new(FchdirLock::new());
        let host = RecordingHost {
            rc: -1,
            ..RecordingHost::default()
        };
        *host.lock.borrow_mut() = Some(Rc::clone(&lock));
        let guard = lock.lock_fchdir(&host, -1);
        assert_eq!(host.calls.borrow().as_slice(), &[(-1, true)]);
        // `lock_fchdir` returns void in C: the -1 goes nowhere, and the caller
        // still owns the lock and must still release it.
        assert!(lock.is_locked());
        lock.unlock_fchdir(guard);
        assert!(!lock.is_locked());
    }

    /// A host whose call blocks until the test releases it, so the lock can be
    /// observed held from another thread.
    #[derive(Default)]
    struct BlockingHost {
        state: Mutex<(bool, bool)>,
        condvar: Condvar,
        calls: AtomicUsize,
    }

    impl BlockingHost {
        fn wait_until_inside(&self) {
            let mut state = self.state.lock().unwrap();
            while !state.0 {
                state = self.condvar.wait(state).unwrap();
            }
        }

        fn release(&self) {
            let mut state = self.state.lock().unwrap();
            state.1 = true;
            self.condvar.notify_all();
        }
    }

    impl FchdirHost for BlockingHost {
        fn fchdir(&self, _dirfd: i32) -> i32 {
            self.calls.fetch_add(1, Ordering::SeqCst);
            let mut state = self.state.lock().unwrap();
            state.0 = true;
            self.condvar.notify_all();
            while !state.1 {
                state = self.condvar.wait(state).unwrap();
            }
            0
        }
    }

    #[test]
    fn the_lock_keeps_a_second_thread_out_of_the_host_call() {
        let lock = FchdirLock::new();
        let host = BlockingHost::default();
        let knocking = AtomicBool::new(false);

        std::thread::scope(|scope| {
            scope.spawn(|| {
                let guard = lock.lock_fchdir(&host, 3);
                lock.unlock_fchdir(guard);
            });
            host.wait_until_inside();
            assert!(lock.is_locked(), "the first thread holds the lock");

            let (knocking, lock, host) = (&knocking, &lock, &host);
            scope.spawn(move || {
                // The flag goes up immediately before the call, exactly as the
                // corpus's `w2.started` does.
                knocking.store(true, Ordering::SeqCst);
                let guard = lock.lock_fchdir(host, 4);
                lock.unlock_fchdir(guard);
            });
            while !knocking.load(Ordering::SeqCst) {
                std::thread::yield_now();
            }
            // The second thread is knocking and cannot get in while the first
            // one is inside its host call: still exactly one call.
            std::thread::sleep(std::time::Duration::from_millis(50));
            assert_eq!(
                host.calls.load(Ordering::SeqCst),
                1,
                "the second thread reached the host while the lock was held"
            );
            host.release();
        });

        assert_eq!(host.calls.load(Ordering::SeqCst), 2);
        assert!(!lock.is_locked(), "everything unwound");
    }

    #[test]
    fn the_process_lock_is_one_lock() {
        assert!(std::ptr::eq(process_lock(), process_lock()));
        assert!(!process_lock().is_locked());
    }

    /// The ported contract has to hold with the real host too, but a test may
    /// not change this process's working directory: it is global state that
    /// every other test thread shares. The corpus uses a scripted host for the
    /// same reason, so only the failure path is checked here.
    #[cfg(unix)]
    #[test]
    fn unix_host_reports_a_failure_and_keeps_the_lock() {
        let lock = FchdirLock::new();
        let guard = lock.lock_fchdir(&UnixFchdirHost, -1);
        assert!(lock.is_locked(), "a failed fchdir still holds the lock");
        lock.unlock_fchdir(guard);
    }
}
