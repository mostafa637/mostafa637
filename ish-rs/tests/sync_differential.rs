//! Differential replay of unmodified `util/sync.c`.
//!
//! `tools/sync-dump.c` links the upstream synchronization primitives against a
//! scripted pthread: `pthread_cond_wait` / `pthread_cond_timedwait` decide the
//! wait's fate, `clock_gettime(CLOCK_MONOTONIC)` fixes the timeout origin, and
//! `pthread_cond_broadcast` / `pthread_cond_signal` reveal which condvar
//! `notify` and `notify_once` reached. Every recorded value is therefore
//! independent of the machine that generated the fixture.
//!
//! The Rust side runs the same corpus through [`ish_emu::sync`], with a
//! [`Parker`] that answers exactly what the C harness answered and a
//! [`SyncHost`] built from the same scripted task state, and compares the
//! return values, the published waiter, the deadlines, the notify targets, the
//! lock owner, and the write-lock counter.
//!
//! Regenerate the C oracle locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_sync_reference.sh
//! ```

use std::cell::{Cell, RefCell};
use std::collections::VecDeque;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::{Arc, MutexGuard};

use ish_emu::sync::{
    notify, notify_once, sigunwind_end, sigunwind_start, sigusr1_handler, wait_for,
    wait_for_ignore_signals, Cond, Lock, LockGuard, ParkResult, Parker, SyncHost, Timespec, Unwind,
    Waiting, WrLock, EBUSY,
};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/sync_reference.txt"
);

// ---------------------------------------------------------------------------
// the fixture cursor
// ---------------------------------------------------------------------------

/// The C harness's records, in the order it printed them.
struct Corpus {
    lines: Vec<String>,
}

impl Corpus {
    fn load() -> Self {
        let text = std::fs::read_to_string(FIXTURE).expect("sync reference fixture");
        let lines: Vec<String> = text
            .lines()
            .filter(|line| !line.starts_with('#'))
            .filter(|line| !line.trim().is_empty())
            .map(str::to_owned)
            .collect();
        assert!(!lines.is_empty(), "the sync fixture is empty");
        Self { lines }
    }
}

/// Consumes fixture records in order and collects every divergence instead of
/// stopping at the first one.
struct Replay<'a> {
    lines: &'a [String],
    next: usize,
    mismatches: RefCell<Vec<String>>,
}

impl<'a> Replay<'a> {
    fn new(corpus: &'a Corpus) -> Self {
        Self {
            lines: &corpus.lines,
            next: 0,
            mismatches: RefCell::new(Vec::new()),
        }
    }

    /// Compare one record: `kind` is the C `printf`'s leading letter and
    /// `fields` the rest of the line, built from the Rust port's own output.
    fn expect(&mut self, kind: char, fields: &str) {
        let index = self.next;
        self.next += 1;
        let produced = format!("{kind} {fields}");
        let Some(line) = self.lines.get(index) else {
            self.record(format!("the C oracle has no record for `{produced}`"));
            return;
        };
        if line != &produced {
            self.record(format!("record {index}: C `{line}` vs Rust `{produced}`"));
        }
    }

    fn record(&self, message: String) {
        self.mismatches.borrow_mut().push(message);
    }

    fn finish(self) {
        let mismatches = self.mismatches.into_inner();
        assert!(
            mismatches.is_empty(),
            "{} divergence(s) from the C oracle:\n{}",
            mismatches.len(),
            mismatches.join("\n")
        );
        assert_eq!(
            self.next,
            self.lines.len(),
            "the replay stopped early: {} fixture record(s) were never produced",
            self.lines.len() - self.next
        );
    }
}

// ---------------------------------------------------------------------------
// the scripted host
// ---------------------------------------------------------------------------

/// The scripted `current`: signal masks, the reported clock, the published
/// waiter, and the unwind flags.
///
/// `current` models the C code's `current != NULL` guard: with no task there is
/// nothing to publish, which is how the port's `set_waiting` seam is specified.
struct ScriptedHost {
    current: Cell<bool>,
    pending: Cell<u64>,
    blocked: Cell<u64>,
    now: Cell<Timespec>,
    waiting: RefCell<Option<Waiting>>,
    should_unwind: Cell<bool>,
    pending_unwind: Cell<bool>,
}

impl Default for ScriptedHost {
    fn default() -> Self {
        Self {
            current: Cell::new(true),
            pending: Cell::new(0),
            blocked: Cell::new(0),
            now: Cell::new(Timespec::new(100, 0)),
            waiting: RefCell::new(None),
            should_unwind: Cell::new(false),
            pending_unwind: Cell::new(false),
        }
    }
}

impl SyncHost for ScriptedHost {
    fn signal_pending(&self) -> bool {
        self.current.get() && (self.pending.get() & !self.blocked.get()) != 0
    }

    fn monotonic_now(&self) -> Timespec {
        self.now.get()
    }

    fn set_waiting(&self, waiting: Option<Waiting>) {
        if self.current.get() {
            *self.waiting.borrow_mut() = waiting;
        }
    }

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

/// What the C harness's wrapped pthread recorded for one wait: which flavor of
/// wait pthread was asked for, the absolute deadline it was handed, its raw
/// status, and whether the task was published while the wait was in flight
/// (which is what C's `current->waiting_cond != NULL` test sees).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct ParkLog {
    deadline: Option<Timespec>,
    status: i32,
    published: bool,
}

impl ParkLog {
    fn line(&self, pending_after: bool) -> String {
        let deadline = self.deadline.map_or_else(
            || "-".to_owned(),
            |deadline| format!("{}.{:09}", deadline.sec, deadline.nsec),
        );
        let kind = if self.deadline.is_some() {
            "timedwait"
        } else {
            "wait"
        };
        format!(
            "{kind} {deadline} rc={} pending_after={} during_cond={} during_lock={}",
            self.status,
            u8::from(pending_after),
            u8::from(self.published),
            u8::from(self.published)
        )
    }
}

/// The scripted pthread: a queue of answers in corpus order, plus the log of
/// what each wait observed.
struct ScriptedParker<'h> {
    host: &'h ScriptedHost,
    queue: RefCell<VecDeque<(ParkResult, bool)>>,
    log: RefCell<Vec<ParkLog>>,
}

impl<'h> ScriptedParker<'h> {
    fn new(host: &'h ScriptedHost) -> Self {
        Self {
            host,
            queue: RefCell::new(VecDeque::new()),
            log: RefCell::new(Vec::new()),
        }
    }

    /// Script the next wait: the status pthread returns and whether a signal
    /// lands while the task sleeps.
    fn script(&self, result: ParkResult, pending_after: bool) {
        self.queue.borrow_mut().push_back((result, pending_after));
    }

    /// The wait the last call performed, if it parked at all.
    fn take(&self) -> Option<ParkLog> {
        self.log.borrow_mut().pop()
    }

    /// Fail loudly if a wait parked without having been scripted.
    fn expect_drained(&self) {
        assert!(
            self.queue.borrow().is_empty(),
            "{} scripted wait(s) were never performed; the replay diverged from C's corpus",
            self.queue.borrow().len()
        );
    }
}

impl Parker for ScriptedParker<'_> {
    fn park<'g, T>(
        &self,
        _cond: &Cond,
        guard: MutexGuard<'g, T>,
        _now: Timespec,
        deadline: Option<Timespec>,
    ) -> (ParkResult, MutexGuard<'g, T>) {
        let (result, pending_after) = self
            .queue
            .borrow_mut()
            .pop_front()
            .expect("the corpus asked for an unscripted wait");
        self.log.borrow_mut().push(ParkLog {
            deadline,
            status: match result {
                ParkResult::Woken => 0,
                ParkResult::TimedOut => 110,
                ParkResult::Failed(status) => status,
            },
            published: self.host.waiting.borrow().is_some(),
        });
        if pending_after {
            // The C harness's wrapped wait assigns `current->pending = 1`
            // before returning, which is what the post-wait check sees.
            self.host.pending.set(1);
        }
        (result, guard)
    }
}

// ---------------------------------------------------------------------------
// the corpus
// ---------------------------------------------------------------------------

/// One `wait_case` from the C `main`.
struct WaitCase {
    ignore_signals: bool,
    pending: u64,
    blocked: u64,
    timeout: Option<Timespec>,
    /// The status the harness's wrapped pthread returns.
    result: ParkResult,
    pending_after: bool,
    /// Whether the wait reaches pthread at all; the pre-wait `EINTR` does not.
    parks: bool,
}

/// Run one `wait_case` and compare both of its records.
fn replay_wait(
    replay: &mut Replay<'_>,
    host: &ScriptedHost,
    parker: &ScriptedParker<'_>,
    case: &WaitCase,
) {
    host.pending.set(case.pending);
    host.blocked.set(case.blocked);
    if case.parks {
        parker.script(case.result, case.pending_after);
    }

    // C's `wait_case` hands `wait_for` either a private lock or the task's
    // sighand lock; the port's `LockId` stands in for that pointer, and the
    // only thing the corpus observes about it is that a waiter was published.
    let cond = Cond::new();
    let lock = Lock::new();
    let guard = lock.lock();
    let (rc, guard) = if case.ignore_signals {
        wait_for_ignore_signals(&cond, guard, case.timeout, host, parker)
    } else {
        wait_for(&cond, guard, case.timeout, host, parker)
    };
    let cleared = host.waiting.borrow().is_none();
    drop(guard);

    if let Some(park) = parker.take() {
        replay.expect('P', &park.line(case.pending_after));
    }
    replay.expect(
        if case.ignore_signals { 'I' } else { 'W' },
        &format!(
            "pending={:x} blocked={:x} rc={rc} cleared_after={}",
            case.pending,
            case.blocked,
            u8::from(cleared)
        ),
    );
}

/// `wait_for` with no current task: no signal can be pending and nothing is
/// published, but the wait still happens.
fn replay_no_task(replay: &mut Replay<'_>, host: &ScriptedHost, parker: &ScriptedParker<'_>) {
    host.current.set(false);
    parker.script(ParkResult::Woken, false);
    let cond = Cond::new();
    let lock = Lock::new();
    let guard = lock.lock();
    let (rc, guard) = wait_for(&cond, guard, None, host, parker);
    drop(guard);
    host.current.set(true);

    let park = parker.take().expect("the no-task wait still parks");
    replay.expect('P', &park.line(false));
    replay.expect('W', &format!("no_current rc={rc}"));
}

/// `notify` must wake every waiter of the condvar it is given, and
/// `notify_once` must reach its own condvar and not all of its waiters — the
/// behaviour the C harness observed by naming the object
/// `pthread_cond_broadcast` / `pthread_cond_signal` reached.
///
/// `notify_once`'s assertion is `1..waiters` rather than exact equality because
/// a condition variable only promises to wake one *blocked* waiter: one that
/// has registered and has not yet blocked can return from the same
/// notification too (see `sync::tests::notify_once_wakes_one_waiter_and_notify_the_rest`).
/// Four waiters keep that bound away from "all of them", which is what a
/// `notify_once` implemented as `notify` would produce.
fn replay_notify(replay: &mut Replay<'_>) {
    let cond_a = Arc::new(Cond::new());
    let cond_b = Arc::new(Cond::new());
    let lock = Arc::new(Lock::new());
    let ready = Arc::new(AtomicUsize::new(0));
    let on_a = Arc::new(AtomicUsize::new(0));
    let on_b = Arc::new(AtomicUsize::new(0));

    std::thread::scope(|scope| {
        for _ in 0..3 {
            let (cond, lock, ready, on_a) = (
                Arc::clone(&cond_a),
                Arc::clone(&lock),
                Arc::clone(&ready),
                Arc::clone(&on_a),
            );
            scope.spawn(move || {
                let host = ScriptedHost::default();
                let guard = lock.lock();
                ready.fetch_add(1, Ordering::SeqCst);
                let (rc, guard) = wait_for(
                    &cond,
                    guard,
                    Some(Timespec::new(5, 0)),
                    &host,
                    &ParkerForThreads,
                );
                drop(guard);
                assert_eq!(rc, 0);
                on_a.fetch_add(1, Ordering::SeqCst);
            });
        }
        for _ in 0..4 {
            let (cond, lock, ready, on_b) = (
                Arc::clone(&cond_b),
                Arc::clone(&lock),
                Arc::clone(&ready),
                Arc::clone(&on_b),
            );
            scope.spawn(move || {
                let host = ScriptedHost::default();
                let guard = lock.lock();
                ready.fetch_add(1, Ordering::SeqCst);
                let (rc, guard) = wait_for(
                    &cond,
                    guard,
                    Some(Timespec::new(5, 0)),
                    &host,
                    &ParkerForThreads,
                );
                drop(guard);
                assert_eq!(rc, 0);
                on_b.fetch_add(1, Ordering::SeqCst);
            });
        }

        // Wait until all seven are parked, holding the lock so the
        // notifications cannot be lost; each waiter bumps `ready` while holding
        // that lock.
        let guard = loop {
            let guard = lock.lock();
            if ready.load(Ordering::SeqCst) == 7 {
                break guard;
            }
            drop(guard);
            std::thread::yield_now();
        };
        notify(&cond_a);
        drop(guard);
        while on_a.load(Ordering::SeqCst) != 3 {
            std::thread::yield_now();
        }
        assert_eq!(
            on_b.load(Ordering::SeqCst),
            0,
            "notify(&a) must reach only condvar a"
        );
        replay.expect('C', "broadcast=a");

        notify_once(&cond_b);
        while on_b.load(Ordering::SeqCst) == 0 {
            std::thread::yield_now();
        }
        std::thread::sleep(std::time::Duration::from_millis(20));
        let woken = on_b.load(Ordering::SeqCst);
        assert!(
            (1..4).contains(&woken),
            "notify_once must wake some, but not all, of condvar b's waiters: {woken} of 4"
        );
        replay.expect('C', "signal=b");

        // Release the rest so the scope can join.
        let guard = lock.lock();
        notify(&cond_b);
        drop(guard);
    });
    assert_eq!(on_b.load(Ordering::SeqCst), 4);
}

/// A stateless parker for the thread-based checks, where the corpus is not
/// scripting the waits.
struct ParkerForThreads;

impl Parker for ParkerForThreads {
    fn park<'g, T>(
        &self,
        cond: &Cond,
        guard: MutexGuard<'g, T>,
        now: Timespec,
        deadline: Option<Timespec>,
    ) -> (ParkResult, MutexGuard<'g, T>) {
        ish_emu::sync::ThreadParker.park(cond, guard, now, deadline)
    }
}

/// `sigunwind_start` / `sigunwind_end` / `sigusr1_handler`.
fn replay_unwind(replay: &mut Replay<'_>, host: &ScriptedHost) {
    let result = sigunwind_start(host);
    replay.expect(
        'U',
        &format!(
            "start rc={} flag={}",
            u8::from(result == Unwind::Unwound),
            u8::from(host.should_unwind())
        ),
    );
    replay.expect(
        'U',
        &format!("armed_again flag={}", u8::from(host.should_unwind())),
    );

    sigunwind_end(host);
    replay.expect('U', &format!("end flag={}", u8::from(host.should_unwind())));

    // No unwind is armed, so the handler is a no-op. C's `sigusr1_handler`
    // returns void, so the fixture only records the flag; the port's boolean
    // is asserted here instead.
    assert!(!sigusr1_handler(host), "nothing was armed");
    replay.expect(
        'U',
        &format!("idle_handler flag={}", u8::from(host.should_unwind())),
    );

    let result = sigunwind_start(host);
    replay.expect(
        'U',
        &format!(
            "rearm rc={} flag={}",
            u8::from(result == Unwind::Unwound),
            u8::from(host.should_unwind())
        ),
    );
    sigunwind_end(host);
    replay.expect(
        'U',
        &format!("rearm_end flag={}", u8::from(host.should_unwind())),
    );
}

/// `lock`, `trylock`, and `unlock`.
fn replay_lock(replay: &mut Replay<'_>) {
    let lock = Lock::new();
    let guard: LockGuard<'_> = lock.lock();
    replay.expect(
        'L',
        &format!(
            "locked owner_self={}",
            u8::from(lock.owner_is_current_thread())
        ),
    );
    let status = lock.try_lock().err().unwrap_or(0);
    replay.expect(
        'L',
        &format!(
            "trylock_held rc={status} owner_self={}",
            u8::from(lock.owner_is_current_thread())
        ),
    );
    assert_eq!(status, EBUSY);
    drop(guard);
    replay.expect(
        'L',
        &format!("unlocked owner_cleared={}", u8::from(lock.owner_is_clear())),
    );
    let guard = lock.try_lock().expect("the lock is free again");
    replay.expect(
        'L',
        &format!(
            "trylock_free rc=0 owner_self={}",
            u8::from(lock.owner_is_current_thread())
        ),
    );
    drop(guard);
}

/// `wrlock`, `read_wrlock`, `write_wrlock`, and their unlocks.
fn replay_wrlock(replay: &mut Replay<'_>) {
    let lock = WrLock::new();
    replay.expect('R', &format!("init val={}", lock.val()));

    let first = lock.read();
    let second = lock.read();
    replay.expect('R', &format!("read2 val={}", lock.val()));
    drop(first);
    replay.expect('R', &format!("read1 val={}", lock.val()));
    drop(second);
    replay.expect('R', &format!("read0 val={}", lock.val()));

    // C's corpus runs with `current == NULL`, so its `current_pid()` is 0.
    let write = lock.write("tools/sync-dump.c", 1, 0);
    let debug = lock.debug();
    replay.expect(
        'R',
        &format!(
            "write val={} file={} line_set={} pid={}",
            lock.val(),
            if debug.is_some() { "set" } else { "null" },
            u8::from(debug.is_some_and(|record| record.line != 0)),
            debug.map_or(0, |record| record.pid)
        ),
    );
    drop(write);
    replay.expect(
        'R',
        &format!(
            "write0 val={} file={} line_set={}",
            lock.val(),
            if lock.debug().is_some() {
                "set"
            } else {
                "null"
            },
            u8::from(lock.debug().is_some_and(|record| record.line != 0))
        ),
    );
}

/// The whole corpus, in the order the C `main` ran it.
#[test]
fn replays_every_record_of_the_c_corpus() {
    let corpus = Corpus::load();
    let mut replay = Replay::new(&corpus);
    let host = ScriptedHost::default();
    let parker = ScriptedParker::new(&host);

    let one_second = Timespec::new(1, 0);
    let normal = Timespec::new(1, 500_000_000);

    // wait_for, both EINTR checks included.
    let cases = [
        // A signal is already pending: C returns before reaching pthread.
        WaitCase {
            ignore_signals: false,
            pending: 1,
            blocked: 0,
            timeout: None,
            result: ParkResult::Woken,
            pending_after: false,
            parks: false,
        },
        WaitCase {
            ignore_signals: false,
            pending: 1,
            blocked: 1,
            timeout: None,
            result: ParkResult::Woken,
            pending_after: false,
            parks: true,
        },
        // The signal lands while the task sleeps, caught by the second check.
        WaitCase {
            ignore_signals: false,
            pending: 0,
            blocked: 0,
            timeout: None,
            result: ParkResult::Woken,
            pending_after: true,
            parks: true,
        },
        // The corpus repeats this one with the waited-on lock being the task's
        // `sighand->lock` instead of a private lock; the port's `LockId` stands
        // in for that pointer, and only "a waiter was published" is observable.
        WaitCase {
            ignore_signals: false,
            pending: 1,
            blocked: 1,
            timeout: None,
            result: ParkResult::Woken,
            pending_after: false,
            parks: true,
        },
        WaitCase {
            ignore_signals: false,
            pending: 0,
            blocked: 0,
            timeout: Some(normal),
            result: ParkResult::Woken,
            pending_after: false,
            parks: true,
        },
        WaitCase {
            ignore_signals: false,
            pending: 0,
            blocked: 0,
            timeout: Some(one_second),
            result: ParkResult::TimedOut,
            pending_after: false,
            parks: true,
        },
        // Any other pthread error is not a timeout, so C reports success.
        WaitCase {
            ignore_signals: false,
            pending: 0,
            blocked: 0,
            timeout: Some(one_second),
            result: ParkResult::Failed(22),
            pending_after: false,
            parks: true,
        },
        WaitCase {
            ignore_signals: false,
            pending: 0,
            blocked: 0,
            timeout: Some(one_second),
            result: ParkResult::Failed(4),
            pending_after: false,
            parks: true,
        },
    ];
    for case in &cases {
        replay_wait(&mut replay, &host, &parker, case);
    }

    // The nanosecond carry, and the strict `> 1000000000` test behind it.
    host.now.set(Timespec::new(100, 700_000_000));
    replay_wait(
        &mut replay,
        &host,
        &parker,
        &WaitCase {
            ignore_signals: false,
            pending: 0,
            blocked: 0,
            timeout: Some(Timespec::new(0, 500_000_000)),
            result: ParkResult::Woken,
            pending_after: false,
            parks: true,
        },
    );
    host.now.set(Timespec::new(100, 600_000_000));
    replay_wait(
        &mut replay,
        &host,
        &parker,
        &WaitCase {
            ignore_signals: false,
            pending: 0,
            blocked: 0,
            // Sums to exactly 1e9, which C leaves unnormalized; the harness's
            // pthread then answers EINVAL, and `wait_for` reports success.
            timeout: Some(Timespec::new(0, 400_000_000)),
            result: ParkResult::Failed(22),
            pending_after: false,
            parks: true,
        },
    );
    host.now.set(Timespec::new(200, 250_000_000));
    replay_wait(
        &mut replay,
        &host,
        &parker,
        &WaitCase {
            ignore_signals: false,
            pending: 0,
            blocked: 0,
            timeout: Some(normal),
            result: ParkResult::Woken,
            pending_after: false,
            parks: true,
        },
    );

    // wait_for_ignore_signals: no EINTR checks at all.
    for (pending, timeout, result) in [
        // A pending, unblocked signal is ignored outright.
        (1, None, ParkResult::Woken),
        (0, Some(one_second), ParkResult::TimedOut),
        (0, None, ParkResult::Woken),
    ] {
        replay_wait(
            &mut replay,
            &host,
            &parker,
            &WaitCase {
                ignore_signals: true,
                pending,
                blocked: 0,
                timeout,
                result,
                pending_after: false,
                parks: true,
            },
        );
    }

    replay_no_task(&mut replay, &host, &parker);
    parker.expect_drained();
    replay_notify(&mut replay);
    replay_unwind(&mut replay, &host);
    replay_lock(&mut replay);
    replay_wrlock(&mut replay);

    replay.finish();
}

/// The fixture must keep covering the two `EINTR` checks, the timeout mapping,
/// the carry, and the unnormalized deadline — the same coverage the generator
/// asserts, restated here so a hand-edited fixture cannot silently lose them.
#[test]
fn the_fixture_keeps_the_c_quirks_under_test() {
    let corpus = Corpus::load();
    let has = |needle: &str| corpus.lines.iter().any(|line| line.contains(needle));
    assert!(has("pending=1 blocked=0 rc=-4"), "the pre-wait EINTR");
    assert!(has("pending=0 blocked=0 rc=-4"), "the post-wait EINTR");
    assert!(has("rc=-110"), "the timeout mapping");
    assert!(has("timedwait 101.200000000"), "the nanosecond carry");
    assert!(has("timedwait 100.1000000000"), "the unnormalized deadline");
    assert!(has("rc=22"), "the non-timeout pthread failure");
    assert!(has("no_current"), "the no-current-task wait");
    assert!(
        has("trylock_free rc=0 owner_self=0"),
        "trylock does not own"
    );
    assert!(
        has("write val=-1 file=set line_set=1"),
        "the write-lock record"
    );
    assert!(has("broadcast=a") && has("signal=b"), "the notify targets");
}
