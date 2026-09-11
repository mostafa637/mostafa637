//! `util/timer.{h,c}` — the timespec helpers and the timer thread.
//!
//! Every iSH timer is a detached thread that recomputes its deadline, sleeps
//! the remainder, and calls a callback. `kernel/time.c` builds `setitimer` on
//! top of it, and a `setitimer` that is re-armed while running relies on the
//! wakeup path exactly as this module implements it.
//!
//! # What the port keeps
//!
//! * **The arithmetic, subtraction included.** `timespec_add` applies the
//!   nanosecond carry *once* and never re-checks, so a value above one second
//!   stays unnormalized — and `timespec_subtract` borrows only when the
//!   nanosecond field demands it, which is how that unnormalized deadline
//!   produces a "negative" remaining time that ends the inner loop.
//!   `tests/timer_differential.rs` replays the unnormalized case against the C.
//! * **The lock protocol.** The worker holds the timer lock except while it
//!   sleeps, exactly as C does, so a `set` from another thread sees a
//!   consistent `start`/`end`/`interval` and the callback runs with the lock
//!   held.
//! * **Poking as an interrupted sleep.** C wakes a sleeping worker with
//!   `pthread_kill(SIGUSR1)`, which makes `nanosleep` return `EINTR` so the
//!   worker recomputes its deadline. [`TimerHost::sleep`] is therefore
//!   specified as an interruptible sleep: a plain `std::thread::sleep` would
//!   run down a stale deadline, so a timer rescheduled from ten seconds to ten
//!   milliseconds would fire ten seconds late.
//! * **Who frees the timer.** `timer_free` frees inline when no worker is
//!   running, and otherwise sets `dead` and lets the worker free the timer on
//!   its way out. The port models that with the timer's own reference count, so
//!   the lifetime matches without an explicit `free`; `tests/timer_differential.rs`
//!   observes both cases through a drop hook, the same way the C oracle wraps
//!   the allocator.
//!
//! # Deliberate differences
//!
//! * **`start`, `end` and `interval` are zero-initialized.** C's `timer_new`
//!   leaves them uninitialized, so the only old spec C can report for a fresh
//!   timer is undefined; the port starts them at zero, and the differential
//!   corpus passes `None` for that first set, exactly as a caller must.
//! * **The callback receives the locked state.** C hands the callback a
//!   `void *data` and lets it reach back into the timer. Since the worker holds
//!   the lock while calling, the port passes the state itself, so the same read
//!   pattern cannot deadlock a non-reentrant mutex.
//! * **`set` returns the old spec.** C writes it through a pointer that may be
//!   `NULL`; [`Timer::set`] returns `Option<TimerSpec>`, which is that
//!   distinction, and drops C's constant `0` return.

use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};
use std::time::Duration;

use crate::sync::Timespec;

/// `clockid_t`, restricted to the two clocks C's own `assert` allows.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TimerClock {
    /// `CLOCK_MONOTONIC`
    Monotonic,
    /// `CLOCK_REALTIME`
    Realtime,
}

/// A `struct timer_spec` / `struct itimerspec`.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct TimerSpec {
    /// `value`: how long until the next fire.
    pub value: Timespec,
    /// `interval`: how long after that until the next one, or zero for a
    /// one-shot.
    pub interval: Timespec,
}

/// Identity of a worker thread, the port's `pthread_t`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct WorkerId(u64);

impl WorkerId {
    /// Mint a worker id.
    ///
    /// The host owns this because the host owns the threads: C's `pthread_t`
    /// comes from `pthread_create`, and a host that spawns scripted workers
    /// needs to name them the same way.
    #[must_use]
    pub const fn new(index: u64) -> Self {
        Self(index)
    }

    /// The number the host minted this id from.
    #[must_use]
    pub const fn index(self) -> u64 {
        self.0
    }
}

static NEXT_WORKER_ID: AtomicU64 = AtomicU64::new(0);

fn next_worker_id() -> WorkerId {
    WorkerId::new(NEXT_WORKER_ID.fetch_add(1, Ordering::Relaxed) + 1)
}

/// The timer's observable state: C's `struct timer` minus the platform bits.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct TimerState {
    /// `start`: when the current wait began.
    pub start: Timespec,
    /// `end`: the absolute deadline the worker is waiting for.
    pub end: Timespec,
    /// `interval`: the reload value, or zero for a one-shot.
    pub interval: Timespec,
    /// `active`
    pub active: bool,
    /// `thread_running`
    pub thread_running: bool,
    /// `dead`: set by `timer_free` when the worker is running, so the worker
    /// knows it is the one that frees the timer.
    pub dead: bool,
    /// C's `pthread_t thread`: the worker to poke. Not part of the port's
    /// observable contract (no two implementations share thread ids), but the
    /// field is here because `set` and `free` read it under the same lock.
    pub worker: Option<WorkerId>,
}

impl Default for TimerState {
    fn default() -> Self {
        // `timer_new` leaves `start`/`end`/`interval` uninitialized in C; see
        // the module docs. Zero is what a caller may rely on before the first
        // `set`.
        Self {
            start: Timespec::new(0, 0),
            end: Timespec::new(0, 0),
            interval: Timespec::new(0, 0),
            active: false,
            thread_running: false,
            dead: false,
            worker: None,
        }
    }
}

impl Timespec {
    /// `timespec_is_zero`.
    #[must_use]
    pub const fn is_zero(self) -> bool {
        self.sec == 0 && self.nsec == 0
    }

    /// `timespec_positive`: strictly after zero.
    #[must_use]
    pub const fn is_positive(self) -> bool {
        self.sec > 0 || (self.sec == 0 && self.nsec > 0)
    }

    /// `timespec_add`: `x + y`, carrying once.
    ///
    /// The carry is a single `if`, not a loop: a nanosecond sum of two seconds
    /// or more keeps an unnormalized `nsec`, which is what the C does and what
    /// `tests/timer_differential.rs` pins.
    #[must_use]
    pub const fn checked_add_once(self, other: Self) -> Self {
        let mut sec = self.sec + other.sec;
        let mut nsec = self.nsec + other.nsec;
        if nsec >= 1_000_000_000 {
            nsec -= 1_000_000_000;
            sec += 1;
        }
        Self { sec, nsec }
    }

    /// `timespec_subtract`: `x - y`, borrowing a second when the nanosecond
    /// field needs it.
    ///
    /// Note the borrow test: unlike [`Timespec::deadline_after`], which is the
    /// waiting contract's version of the same computation, this one compares
    /// only the nanosecond fields, so the result can have a negative second
    /// count with a non-negative nanosecond field — and
    /// [`Timespec::is_positive`] then reports "not expired yet".
    ///
    /// [`Timespec::deadline_after`]: crate::sync::Timespec::deadline_after
    #[must_use]
    pub const fn subtract(self, other: Self) -> Self {
        let mut sec = self.sec;
        let mut nsec = self.nsec;
        if nsec < other.nsec {
            sec -= 1;
            nsec += 1_000_000_000;
        }
        Self {
            sec: sec - other.sec,
            nsec: nsec - other.nsec,
        }
    }

    /// `timespec_normalize`: fold the nanosecond field into the second count,
    /// truncating toward zero so a negative value stays negative.
    #[must_use]
    pub const fn normalize(self) -> Self {
        Self {
            sec: self.sec + self.nsec / 1_000_000_000,
            nsec: self.nsec % 1_000_000_000,
        }
    }
}

/// The platform the timer thread runs on, injected so the clock, the wakeup,
/// and the sleep itself are all scriptable.
pub trait TimerHost: Send + Sync {
    /// The current time on the timer's clock, C's `timespec_now`.
    fn now(&self, clock: TimerClock) -> Timespec;

    /// Start the detached worker that runs `body`, returning its `pthread_t`.
    ///
    /// C's `pthread_create` + `pthread_detach`: the caller never joins.
    fn start(&self, body: Box<dyn FnOnce() + Send>) -> WorkerId;

    /// `pthread_kill(worker, SIGUSR1)`: interrupt `worker`'s sleep so it
    /// recomputes its deadline instead of sleeping out a stale one.
    fn poke(&self, worker: WorkerId);

    /// `nanosleep(&remaining, NULL)`: sleep for up to `remaining`, returning
    /// early when the thread is poked.
    ///
    /// No return value: C's timer ignores both `nanosleep`'s result and its
    /// remainder, and simply recomputes.
    fn sleep(&self, remaining: Timespec);
}

/// The production host: real threads, a real clock, and a condition variable
/// that makes a sleep interruptible.
#[derive(Debug, Default)]
pub struct ThreadHost;

/// The state a `poke` sets and a `sleep` waits on: the port's stand-in for the
/// signal that interrupts `nanosleep`.
#[derive(Debug, Default)]
struct PokeSlot {
    poked: Mutex<bool>,
    condvar: std::sync::Condvar,
}

impl PokeSlot {
    fn interruptible_sleep(&self, remaining: Timespec) {
        let mut poked = self
            .poked
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        if *poked {
            // The poke arrived before the sleep started, like a signal that was
            // already pending when nanosleep was entered: it returns at once.
            *poked = false;
            return;
        }
        if !remaining.is_positive() {
            // C's `nanosleep` with a negative or zero request returns
            // immediately; the worker then recomputes and leaves the loop.
            return;
        }
        let duration = Duration::new(
            remaining.sec.unsigned_abs(),
            u32::try_from(remaining.nsec).unwrap_or(0),
        );
        // `nanosleep` takes a *relative* time even for a realtime clock, which
        // is why this waits on a duration rather than on the deadline.
        let (mut guard, _timed_out) = self
            .condvar
            .wait_timeout(poked, duration)
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        *guard = false;
    }

    fn poke(&self) {
        *self
            .poked
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner) = true;
        self.condvar.notify_all();
    }
}

thread_local! {
    /// The poke slot of the worker running on this thread, if any.
    static CURRENT_WORKER: std::cell::RefCell<Option<Arc<PokeSlot>>> =
        const { std::cell::RefCell::new(None) };

    /// The poke slots of the workers this thread started, keyed by worker id.
    ///
    /// C keeps the `pthread_t` in `struct timer`; the port resolves its
    /// [`WorkerId`] through this table, because `std::thread` has no
    /// "interrupt that thread" call to name.
    static WORKERS: std::cell::RefCell<std::collections::BTreeMap<WorkerId, Arc<PokeSlot>>> =
        const { std::cell::RefCell::new(std::collections::BTreeMap::new()) };
}

impl TimerHost for ThreadHost {
    fn now(&self, clock: TimerClock) -> Timespec {
        match clock {
            TimerClock::Monotonic => {
                // `Instant` is monotonic and its origin is arbitrary, which is
                // all a timer needs: only differences are ever used.
                static START: std::sync::OnceLock<std::time::Instant> = std::sync::OnceLock::new();
                Timespec::from_duration(START.get_or_init(std::time::Instant::now).elapsed())
            }
            TimerClock::Realtime => Timespec::from_duration(
                std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap_or_default(),
            ),
        }
    }

    fn start(&self, body: Box<dyn FnOnce() + Send>) -> WorkerId {
        let id = next_worker_id();
        let slot = Arc::new(PokeSlot::default());
        let thread_slot = Arc::clone(&slot);
        WORKERS.with(|workers| workers.borrow_mut().insert(id, slot));
        std::thread::Builder::new()
            .name(format!("ish-timer-{}", id.index()))
            .spawn(move || {
                CURRENT_WORKER.with(|current| *current.borrow_mut() = Some(thread_slot));
                body();
                CURRENT_WORKER.with(|current| *current.borrow_mut() = None);
            })
            // Dropping the handle detaches the thread, like `pthread_detach`.
            .expect("the timer thread could not be started");
        id
    }

    fn poke(&self, worker: WorkerId) {
        let slot = WORKERS.with(|workers| workers.borrow().get(&worker).cloned());
        if let Some(slot) = slot {
            slot.poke();
        }
    }

    fn sleep(&self, remaining: Timespec) {
        let slot = CURRENT_WORKER.with(|current| current.borrow().clone());
        if let Some(slot) = slot {
            slot.interruptible_sleep(remaining);
        }
    }
}

/// `timer_callback_t`: called with the timer lock held.
pub type TimerCallback = Box<dyn Fn(&TimerState) + Send + Sync>;

struct TimerInner {
    clock: TimerClock,
    callback: TimerCallback,
    state: Mutex<TimerState>,
    host: Arc<dyn TimerHost>,
}

/// `struct timer`.
///
/// The handle owns one reference to the timer's state and a running worker owns
/// another, which is what gives `timer_free` C's "the worker frees it if it is
/// still running" lifetime without an explicit free.
pub struct Timer {
    inner: Arc<TimerInner>,
}

impl std::fmt::Debug for Timer {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("Timer")
            .field("clock", &self.inner.clock)
            .field("state", &self.state())
            .finish()
    }
}

impl Timer {
    /// `timer_new`: an inactive timer on `clock`.
    #[must_use]
    pub fn new(clock: TimerClock, host: Arc<dyn TimerHost>, callback: TimerCallback) -> Self {
        Self {
            inner: Arc::new(TimerInner {
                clock,
                callback,
                state: Mutex::new(TimerState::default()),
                host,
            }),
        }
    }

    /// A snapshot of the timer's state, taken under its lock.
    #[must_use]
    pub fn state(&self) -> TimerState {
        *self
            .inner
            .state
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }

    /// `timer_set`: (re)arm the timer, reporting the spec it replaced.
    ///
    /// C writes the replaced spec through `oldspec`, which may be `NULL`; the
    /// `Option` return is that distinction.
    pub fn set(&self, spec: TimerSpec) -> Option<TimerSpec> {
        let mut state = self
            .inner
            .state
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        let now = self.inner.host.now(self.inner.clock);
        let old = Some(TimerSpec {
            value: state.end.subtract(now),
            interval: state.interval,
        });

        state.start = now;
        state.end = now.checked_add_once(spec.value);
        state.interval = spec.interval;
        state.active = !spec.value.is_zero();
        if state.thread_running {
            // The worker is asleep; poking it makes it recompute `remaining`
            // against the new `end` instead of finishing the stale sleep.
            if let Some(worker) = state.worker {
                self.inner.host.poke(worker);
            }
        } else if state.active {
            state.thread_running = true;
            let inner = Arc::clone(&self.inner);
            state.worker = Some(self.inner.host.start(Box::new(move || timer_thread(inner))));
        }
        old
    }

    /// `timer_free`: stop the timer, freeing it here or letting the worker do
    /// it on its way out.
    pub fn free(self) {
        let mut state = self
            .inner
            .state
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        state.active = false;
        if state.thread_running {
            state.dead = true;
            if let Some(worker) = state.worker {
                self.inner.host.poke(worker);
            }
            drop(state);
            // `self` drops here. The worker still holds a reference, so the
            // timer is freed by the worker instead — C's `free(timer)` inside
            // `timer_thread`.
        }
    }
}

/// `timer_thread`: wait for the deadline, fire, and reload the interval.
fn timer_thread(inner: Arc<TimerInner>) {
    let mut state = inner
        .state
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    loop {
        let mut remaining = state.end.subtract(inner.host.now(inner.clock));
        while state.active && remaining.is_positive() {
            // C unlocks around `nanosleep` so another thread's `timer_set` can
            // rewrite `end` while this one is asleep.
            drop(state);
            inner.host.sleep(remaining);
            state = inner
                .state
                .lock()
                .unwrap_or_else(std::sync::PoisonError::into_inner);
            remaining = state.end.subtract(inner.host.now(inner.clock));
        }
        if state.active {
            (inner.callback)(&state);
        }
        if state.active && state.interval.is_positive() {
            state.start = state.end;
            state.end = state.start.checked_add_once(state.interval);
        } else {
            break;
        }
    }
    state.thread_running = false;
    state.worker = None;
    // If `timer_free` ran while this thread was working, `dead` is set and this
    // drop is the one that frees the timer, exactly like C's `free(timer)`.
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::atomic::AtomicUsize;

    #[test]
    fn timespec_helpers_match_the_c_inline_functions() {
        // `timespec_add` carries once, so an oversized nanosecond field stays.
        assert_eq!(
            Timespec::new(0, 1_500_000_000).checked_add_once(Timespec::new(0, 1_500_000_000)),
            Timespec::new(1, 2_000_000_000),
            "one carry only: 3e9 nanoseconds leaves 2e9 behind"
        );
        assert_eq!(
            Timespec::new(102, 500_000_000).checked_add_once(Timespec::new(0, 1_500_000_000)),
            Timespec::new(103, 1_000_000_000),
            "one carry only: the 1e9 nanosecond field survives"
        );
        assert_eq!(
            Timespec::new(1, 900_000_000).checked_add_once(Timespec::new(0, 200_000_000)),
            Timespec::new(2, 100_000_000)
        );

        // `timespec_subtract` borrows on the nanosecond field alone, so the
        // result may carry a negative second count.
        assert_eq!(
            Timespec::new(103, 1_000_000_000).subtract(Timespec::new(104, 0)),
            Timespec::new(-1, 1_000_000_000)
        );
        assert!(!Timespec::new(-1, 1_000_000_000).is_positive());
        assert_eq!(
            Timespec::new(100, 250_000_000).subtract(Timespec::new(100, 500_000_000)),
            Timespec::new(-1, 750_000_000)
        );
        assert_eq!(
            Timespec::new(100, 750_000_000).subtract(Timespec::new(100, 500_000_000)),
            Timespec::new(0, 250_000_000)
        );
        assert!(Timespec::new(0, 1).is_positive());
        assert!(!Timespec::new(0, 0).is_positive());
        assert!(Timespec::new(0, 0).is_zero());

        // `timespec_normalize` truncates toward zero.
        assert_eq!(
            Timespec::new(0, 2_500_000_000).normalize(),
            Timespec::new(2, 500_000_000)
        );
        assert_eq!(
            Timespec::new(0, -1_500_000_000).normalize(),
            Timespec::new(-1, -500_000_000)
        );
    }

    #[test]
    fn a_zero_value_set_starts_no_worker() {
        let host = Arc::new(ThreadHost);
        let timer = Timer::new(TimerClock::Monotonic, host, Box::new(|_| {}));
        assert_eq!(timer.state(), TimerState::default());
        assert!(timer.set(TimerSpec::default()).is_some());
        let state = timer.state();
        assert!(!state.active);
        assert!(!state.thread_running);
        assert!(!state.dead);
        assert_eq!(
            state.start, state.end,
            "a zero value is a same-instant timer"
        );
        assert_eq!(state.worker, None);
    }

    #[test]
    fn the_worker_fires_and_reloads_the_interval() {
        let host = Arc::new(ThreadHost);
        let fired = Arc::new(AtomicUsize::new(0));
        let intervals = Arc::new(Mutex::new(Vec::new()));
        let counter = Arc::clone(&fired);
        let seen = Arc::clone(&intervals);
        let timer = Timer::new(
            TimerClock::Monotonic,
            Arc::clone(&host) as Arc<dyn TimerHost>,
            Box::new(move |state| {
                counter.fetch_add(1, Ordering::SeqCst);
                seen.lock().unwrap().push(state.interval);
            }),
        );

        // A 10 ms shot with a 10 ms interval: it fires repeatedly until freed.
        timer.set(TimerSpec {
            value: Timespec::new(0, 10_000_000),
            interval: Timespec::new(0, 10_000_000),
        });
        let start = std::time::Instant::now();
        while fired.load(Ordering::SeqCst) < 2 {
            assert!(
                start.elapsed() < Duration::from_secs(5),
                "the interval should have reloaded"
            );
            std::thread::sleep(Duration::from_millis(2));
        }
        assert_eq!(
            intervals.lock().unwrap().first().copied(),
            Some(Timespec::new(0, 10_000_000))
        );
        timer.free();

        // The worker stops: the count must not keep climbing.
        std::thread::sleep(Duration::from_millis(50));
        assert_eq!(
            fired.load(Ordering::SeqCst),
            2,
            "free must stop the timer without an extra fire"
        );
    }

    #[test]
    fn a_running_timer_is_rescheduled_without_waiting_out_the_old_deadline() {
        let host = Arc::new(ThreadHost);
        let fired = Arc::new(AtomicUsize::new(0));
        let counter = Arc::clone(&fired);
        let timer = Timer::new(
            TimerClock::Monotonic,
            Arc::clone(&host) as Arc<dyn TimerHost>,
            Box::new(move |_| {
                counter.fetch_add(1, Ordering::SeqCst);
            }),
        );

        // Ten seconds out, then rescheduled to 10 ms. With a non-interruptible
        // sleep this would fire ten seconds late — the C `EINTR` case.
        timer.set(TimerSpec {
            value: Timespec::new(10, 0),
            interval: Timespec::new(0, 0),
        });
        std::thread::sleep(Duration::from_millis(20));
        let old = timer
            .set(TimerSpec {
                value: Timespec::new(0, 10_000_000),
                interval: Timespec::new(0, 0),
            })
            .expect("the replaced spec");
        assert!(
            old.value.sec >= 9,
            "the old deadline was about ten seconds away: {old:?}"
        );

        let start = std::time::Instant::now();
        while fired.load(Ordering::SeqCst) == 0 {
            assert!(
                start.elapsed() < Duration::from_secs(2),
                "the poke did not reschedule the sleep"
            );
            std::thread::sleep(Duration::from_millis(2));
        }
        timer.free();
    }

    #[test]
    fn free_during_a_sleep_lets_the_worker_finish_without_firing() {
        let host = Arc::new(ThreadHost);
        let fired = Arc::new(AtomicUsize::new(0));
        let counter = Arc::clone(&fired);
        let timer = Timer::new(
            TimerClock::Monotonic,
            Arc::clone(&host) as Arc<dyn TimerHost>,
            Box::new(move |_| {
                counter.fetch_add(1, Ordering::SeqCst);
            }),
        );
        timer.set(TimerSpec {
            value: Timespec::new(5, 0),
            interval: Timespec::new(0, 0),
        });
        // Wait for the worker to park, so the free really does race the sleep.
        let start = std::time::Instant::now();
        while !timer.state().thread_running {
            assert!(start.elapsed() < Duration::from_secs(2));
            std::thread::sleep(Duration::from_millis(1));
        }
        std::thread::sleep(Duration::from_millis(10));
        timer.free();

        // The poke wakes the worker's sleep, it sees `active == false`, and it
        // leaves without calling the callback.
        std::thread::sleep(Duration::from_millis(50));
        assert_eq!(
            fired.load(Ordering::SeqCst),
            0,
            "a freed timer must not fire"
        );
    }
}
