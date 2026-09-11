//! Differential replay of unmodified `util/fchdir.c`.
//!
//! `tools/fchdir-dump.c` includes the upstream file, so the transcript is
//! produced by the same `static lock_t` the port models, and wraps `fchdir` at
//! link time, so every host call is recorded with the descriptor it was handed
//! *while the caller still holds the lock*. The corpus has two workers
//! contending for that lock: they never print, they only set flags and record
//! what the wrapped call saw, and the driver prints a value only once the flag
//! that makes it deterministic is up. The one deliberately time-dependent
//! record is `T w2 blocked`: a bounded pause gives a lock that is not really
//! held every chance to let the second worker in.
//!
//! The Rust side runs the same corpus through [`ish_emu::fchdir`] with a
//! scripted [`FchdirHost`] that mirrors that platform — the two directory
//! descriptors the corpus opens succeed, anything else fails — and compares
//! every record: the lock probe, the descriptor the host was handed, whether
//! the lock was held at that moment, the owner field, and the number of host
//! calls made while the second worker was knocking.
//!
//! Regenerate the C oracle locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_fchdir_reference.sh
//! ```

use std::collections::BTreeMap;
use std::sync::{Condvar, Mutex};
use std::time::Duration;

use ish_emu::fchdir::{FchdirHost, FchdirLock};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/fchdir_reference.txt"
);

/// The two directory descriptors the corpus opens, and the descriptor it hands
/// to a failing call.
const FD_A: i32 = 3;
const FD_B: i32 = 4;
const FD_NONE: i32 = -1;

// ---------------------------------------------------------------------------
// the fixture cursor
// ---------------------------------------------------------------------------

/// The C harness's transcript, in the order it printed it.
struct Corpus {
    lines: Vec<String>,
}

impl Corpus {
    fn load() -> Self {
        let text = std::fs::read_to_string(FIXTURE).expect("fchdir reference fixture");
        let lines: Vec<String> = text
            .lines()
            .filter(|line| !line.starts_with('#'))
            .map(str::to_owned)
            .collect();
        assert!(!lines.is_empty(), "the fchdir fixture is empty");
        Self { lines }
    }
}

/// Consumes transcript records in order and collects every divergence.
struct Replay<'a> {
    lines: &'a [String],
    next: usize,
    mismatches: Vec<String>,
}

impl<'a> Replay<'a> {
    fn new(corpus: &'a Corpus) -> Self {
        Self {
            lines: &corpus.lines,
            next: 0,
            mismatches: Vec::new(),
        }
    }

    fn expect(&mut self, record: &str) {
        let index = self.next;
        self.next += 1;
        match self.lines.get(index) {
            Some(line) if line == record => {}
            Some(line) => self
                .mismatches
                .push(format!("record {index}: C `{line}` vs Rust `{record}`")),
            None => self
                .mismatches
                .push(format!("the C oracle has no record for `{record}`")),
        }
    }

    fn finish(self) {
        assert!(
            self.mismatches.is_empty(),
            "{} divergence(s) from the C oracle:\n{}",
            self.mismatches.len(),
            self.mismatches.join("\n")
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
// the scripted platform
// ---------------------------------------------------------------------------

/// One worker's slot in the transcript's flags: the same five the C harness
/// keeps, and the two values the wrapped call recorded for it.
#[derive(Debug, Default, Clone, Copy)]
struct Worker {
    started: bool,
    in_fchdir: bool,
    released: bool,
    unlocked: bool,
    /// The descriptor this worker's host call was handed.
    fd: i32,
    /// What that call returned for it.
    rc: i32,
}

#[derive(Debug, Default)]
struct State {
    /// Every wrapped host call, in order.
    calls: usize,
    /// What the last call was handed and what it returned, which is what the
    /// single-threaded records report.
    last_fd: i32,
    last_rc: i32,
    workers: BTreeMap<&'static str, Worker>,
}

/// The C harness's wrapped `fchdir` plus its scripted flags.
///
/// The guest's `fchdir` succeeds for the two descriptors the corpus opened and
/// fails for anything else, which is what the real call did; there is no
/// working directory to change, because a scripted host must not touch the test
/// process's — that is global state every other test thread shares.
#[derive(Default)]
struct ScriptedHost {
    state: Mutex<State>,
    condvar: Condvar,
}

impl ScriptedHost {
    fn with_state<T>(&self, body: impl FnOnce(&mut State) -> T) -> T {
        let mut state = self.state.lock().unwrap();
        body(&mut state)
    }

    fn wait_for(&self, test: impl Fn(&State) -> bool) {
        let mut state = self.state.lock().unwrap();
        while !test(&state) {
            state = self.condvar.wait(state).unwrap();
        }
    }

    /// The host call itself, attributed to `worker` when a worker made it.
    fn call(&self, worker: Option<&'static str>, dirfd: i32) -> i32 {
        let rc = if dirfd == FD_A || dirfd == FD_B {
            0
        } else {
            -1
        };
        let mut state = self.state.lock().unwrap();
        state.calls += 1;
        state.last_fd = dirfd;
        state.last_rc = rc;
        if let Some(name) = worker {
            let record = state.workers.entry(name).or_default();
            record.fd = dirfd;
            record.in_fchdir = true;
            self.condvar.notify_all();
            // A C call site's relative-path work happens here, with the lock
            // held; the driver ends the pause.
            while !state.workers[&name].released {
                state = self.condvar.wait(state).unwrap();
            }
            state.workers.get_mut(&name).unwrap().rc = rc;
        }
        rc
    }

    // -- worker side --------------------------------------------------------

    fn mark_started(&self, worker: &'static str) {
        self.with_state(|state| state.workers.entry(worker).or_default().started = true);
        self.condvar.notify_all();
    }

    fn mark_unlocked(&self, worker: &'static str) {
        self.with_state(|state| state.workers.entry(worker).or_default().unlocked = true);
        self.condvar.notify_all();
    }

    // -- driver side --------------------------------------------------------

    fn wait_started(&self, worker: &'static str) {
        self.wait_for(|state| state.workers.get(worker).is_some_and(|w| w.started));
    }

    fn wait_in_fchdir(&self, worker: &'static str) {
        self.wait_for(|state| state.workers.get(worker).is_some_and(|w| w.in_fchdir));
    }

    fn wait_unlocked(&self, worker: &'static str) {
        self.wait_for(|state| state.workers.get(worker).is_some_and(|w| w.unlocked));
    }

    fn release(&self, worker: &'static str) {
        self.with_state(|state| state.workers.get_mut(worker).unwrap().released = true);
        self.condvar.notify_all();
    }

    fn calls(&self) -> usize {
        self.with_state(|state| state.calls)
    }

    fn last_fd(&self) -> i32 {
        self.with_state(|state| state.last_fd)
    }

    fn last_rc(&self) -> i32 {
        self.with_state(|state| state.last_rc)
    }

    fn worker_fd(&self, worker: &'static str) -> i32 {
        self.with_state(|state| state.workers[&worker].fd)
    }

    fn worker_rc(&self, worker: &'static str) -> i32 {
        self.with_state(|state| state.workers[&worker].rc)
    }

    /// A view of this host as the C's thread-local `who` made it: the call is
    /// attributed to `worker`, and it pauses until the driver releases it.
    fn as_worker(&self, worker: &'static str) -> WorkerView<'_> {
        WorkerView { host: self, worker }
    }
}

/// The driver's calls, which are not attributed to a worker and do not pause.
struct DriverView<'a>(&'a ScriptedHost);

impl FchdirHost for DriverView<'_> {
    fn fchdir(&self, dirfd: i32) -> i32 {
        self.0.call(None, dirfd)
    }
}

/// One worker's calls, which are attributed and gated.
struct WorkerView<'a> {
    host: &'a ScriptedHost,
    worker: &'static str,
}

impl FchdirHost for WorkerView<'_> {
    fn fchdir(&self, dirfd: i32) -> i32 {
        self.host.call(Some(self.worker), dirfd)
    }
}

/// The corpus's `settle`: long enough that a lock which is not really held
/// cannot hide, short enough to be irrelevant to a correct run.
fn settle() {
    std::thread::sleep(Duration::from_millis(100));
}

// ---------------------------------------------------------------------------
// the replay
// ---------------------------------------------------------------------------

#[test]
fn replays_every_record_of_the_c_corpus() {
    let corpus = Corpus::load();
    let mut replay = Replay::new(&corpus);
    let host = ScriptedHost::default();
    let lock = FchdirLock::new();
    let driver = DriverView(&host);
    let w1 = host.as_worker("w1");
    let w2 = host.as_worker("w2");

    replay.expect(&format!("G dirfds {FD_A} {FD_B}"));

    // 1. One thread at a time: the lock starts free, is taken around the host
    //    call, and is free again after `unlock_fchdir`.
    replay.expect(&format!(
        "T probe-before held={}",
        u8::from(lock.is_locked())
    ));
    let guard = lock.lock_fchdir(&driver, FD_A);
    // The C's `rc = last_rc;` before printing, so the record cannot depend on
    // the order the arguments of one `printf` happen to be evaluated in.
    let (fd, rc) = (host.last_fd(), host.last_rc());
    let owner = lock.owner_is_current_thread();
    replay.expect(&format!("T self fd={fd} rc={rc} owner={}", u8::from(owner)));
    lock.unlock_fchdir(guard);
    replay.expect(&format!(
        "T self released held={}",
        u8::from(lock.is_locked())
    ));

    // A host call that fails is still made, still under the lock, and its
    // result goes nowhere: `lock_fchdir` returns void in C.
    let guard = lock.lock_fchdir(&driver, FD_NONE);
    let (fd, rc) = (host.last_fd(), host.last_rc());
    let owner = lock.owner_is_current_thread();
    replay.expect(&format!("T self fd={fd} rc={rc} owner={}", u8::from(owner)));
    lock.unlock_fchdir(guard);
    replay.expect(&format!(
        "T self released held={}",
        u8::from(lock.is_locked())
    ));

    // 2. Two threads: w1 holds the lock while its host call is in flight, w2 is
    //    already knocking, and it cannot get in until w1 unlocks.
    std::thread::scope(|scope| {
        scope.spawn(|| {
            host.mark_started("w1");
            let guard = lock.lock_fchdir(&w1, FD_A);
            lock.unlock_fchdir(guard);
            host.mark_unlocked("w1");
        });
        host.wait_started("w1");
        scope.spawn(|| {
            host.mark_started("w2");
            let guard = lock.lock_fchdir(&w2, FD_B);
            lock.unlock_fchdir(guard);
            host.mark_unlocked("w2");
        });
        host.wait_started("w2");
        host.wait_in_fchdir("w1");
        let (fd, held) = (host.worker_fd("w1"), lock.is_locked());
        replay.expect(&format!("T w1 in fd={fd} held={}", u8::from(held)));
        settle();
        // The second worker is knocking and still has not reached the host.
        let calls = host.calls();
        replay.expect(&format!("T w2 blocked calls={calls}"));
        host.release("w1");
        host.wait_unlocked("w1");
        let rc = host.worker_rc("w1");
        replay.expect(&format!("T w1 done rc={rc}"));
        host.wait_in_fchdir("w2");
        let (fd, held) = (host.worker_fd("w2"), lock.is_locked());
        replay.expect(&format!("T w2 in fd={fd} held={}", u8::from(held)));
        host.release("w2");
        host.wait_unlocked("w2");
        let rc = host.worker_rc("w2");
        replay.expect(&format!("T w2 done rc={rc}"));
    });

    replay.expect(&format!("T end held={}", u8::from(lock.is_locked())));
    let calls = host.calls();
    replay.expect(&format!("T calls {calls}"));

    replay.finish();
}

/// The properties the port's documentation rests on, so a regeneration that
/// silently weakened the corpus is caught even if it stayed self-consistent.
#[test]
fn the_fixture_keeps_the_fchdir_quirks_under_test() {
    let text = std::fs::read_to_string(FIXTURE).expect("fchdir reference fixture");
    let has = |needle: &str| text.lines().any(|line| line.contains(needle));

    assert!(has("G dirfds 3 4"), "the corpus's two directories");
    assert!(
        has("T probe-before held=0"),
        "C's `static lock_t` is not held at the start"
    );
    assert!(
        has("T self fd=3 rc=0 owner=1"),
        "the host call is made with the descriptor the caller passed, under the lock"
    );
    assert!(
        has("T self fd=-1 rc=-1 owner=1"),
        "a failing host call is still made, still under the lock"
    );
    assert!(
        has("T w1 in fd=3 held=1"),
        "the first worker's call runs while it holds the lock"
    );
    assert!(
        has("T w2 blocked calls=3"),
        "the second worker knocks and cannot get in: exactly three host calls so far"
    );
    assert!(
        has("T w2 in fd=4 held=1"),
        "and it gets in, holding the lock, once the first one releases"
    );
    assert!(
        has("T w1 done rc=0") && has("T w2 done rc=0"),
        "each worker's host call reports the host's own result"
    );
    assert!(
        has("T end held=0"),
        "unlock_fchdir releases the lock, so it is free at the end"
    );
    assert!(
        has("T calls 4"),
        "exactly four host calls: two from the driver and one per worker"
    );
    assert_eq!(
        text.matches("T self fd=").count(),
        2,
        "the two single-threaded calls are pinned"
    );
}
