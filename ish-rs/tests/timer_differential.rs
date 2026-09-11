//! Differential replay of unmodified `util/timer.c`.
//!
//! `tools/timer-dump.c` links the upstream timer and runs its real worker
//! thread against a scripted platform: the clock only advances when a scripted
//! sleep completes, sleeps are released or poked by the driver, and `free` is
//! wrapped so the transcript says whether the caller or the worker freed the
//! timer. The driver pokes only workers it has watched park, so the transcript
//! is identical on every run.
//!
//! The Rust side runs the same corpus through [`ish_emu::timer`] with a
//! [`TimerHost`] that mirrors that platform — including the rule that an
//! interrupted worker stays parked until the driver acknowledges the poke — and
//! compares every record: clock reads, sleeps, callbacks, sets and their
//! replaced specs, state snapshots, and who freed the timer.
//!
//! Regenerate the C oracle locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_timer_reference.sh
//! ```

use std::sync::{Arc, Condvar, Mutex};

use ish_emu::sync::Timespec;
use ish_emu::timer::{Timer, TimerClock, TimerHost, TimerSpec, TimerState, WorkerId};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/timer_reference.txt"
);

fn stamp(time: Timespec) -> String {
    format!("{}.{:09}", time.sec, time.nsec)
}

// ---------------------------------------------------------------------------
// the fixture cursor
// ---------------------------------------------------------------------------

/// The C harness's transcript, in the order it printed it.
struct Corpus {
    lines: Vec<String>,
}

impl Corpus {
    fn load() -> Self {
        let text = std::fs::read_to_string(FIXTURE).expect("timer reference fixture");
        let lines: Vec<String> = text
            .lines()
            .filter(|line| !line.starts_with('#'))
            .map(str::to_owned)
            .collect();
        assert!(!lines.is_empty(), "the timer fixture is empty");
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

/// What the driver and the worker coordinate through: the C harness's
/// `script_lock` / `in_sleep` / `release_sleep` / `interrupted_ack`, plus the
/// wrapped-`free` bookkeeping.
#[derive(Default)]
struct Script {
    now: Timespec,
    in_sleep: bool,
    release_sleep: bool,
    interrupted_ack: bool,
    signalled: bool,
    sleep_count: usize,
    callbacks: usize,
    workers_started: usize,
    freed_by: Option<&'static str>,
}

struct ScriptedHost {
    script: Mutex<Script>,
    condvar: Condvar,
    /// The transcript, appended under `script`.
    log: Mutex<Vec<String>>,
    /// How many workers have finished, so the driver waits for the worker
    /// thread itself before it frees a timer the worker no longer refers to.
    workers_finished: Arc<(Mutex<usize>, Condvar)>,
}

impl ScriptedHost {
    fn new() -> Arc<Self> {
        Arc::new(Self {
            script: Mutex::new(Script {
                now: Timespec::new(100, 0),
                ..Script::default()
            }),
            condvar: Condvar::new(),
            log: Mutex::new(Vec::new()),
            workers_finished: Arc::new((Mutex::new(0), Condvar::new())),
        })
    }

    /// Append a record to the transcript. Also used by the driver, so that the
    /// worker's records and the driver's records end up in one ordered list.
    fn emit(&self, line: impl Into<String>) {
        self.log.lock().unwrap().push(line.into());
    }

    /// Compare every record produced since the last call against the fixture.
    fn replay_into(&self, replay: &mut Replay<'_>) {
        let lines: Vec<String> = std::mem::take(&mut *self.log.lock().unwrap());
        for line in lines {
            replay.expect(&line);
        }
    }

    // -- driver side --------------------------------------------------------

    /// Wait until the worker has parked in sleep number `expected`.
    fn wait_for_sleep(&self, expected: usize) {
        let mut script = self.script.lock().unwrap();
        while script.sleep_count < expected || !script.in_sleep {
            script = self
                .condvar
                .wait_timeout(script, std::time::Duration::from_millis(1))
                .unwrap()
                .0;
        }
    }

    /// Let the parked sleep complete: the clock advances by the request.
    fn release_sleep(&self) {
        let mut script = self.script.lock().unwrap();
        script.release_sleep = true;
        self.condvar.notify_all();
    }

    /// Let an interrupted worker resume.
    fn acknowledge_poke(&self) {
        let mut script = self.script.lock().unwrap();
        script.interrupted_ack = true;
        self.condvar.notify_all();
    }

    fn wait_for_callbacks(&self, expected: usize) {
        let mut script = self.script.lock().unwrap();
        while script.callbacks < expected {
            script = self
                .condvar
                .wait_timeout(script, std::time::Duration::from_millis(1))
                .unwrap()
                .0;
        }
    }

    /// Record `R running=0` once the worker has left its loop, then wait for the
    /// worker thread itself, so a following free is the last reference this
    /// side holds — C's `thread_running == false` case.
    fn wait_for_worker(&self, timer: &Timer) {
        while timer.state().thread_running {
            std::thread::sleep(std::time::Duration::from_millis(1));
        }
        self.emit("R running=0");
        let started = self.script.lock().unwrap().workers_started;
        let (finished, condvar) = &*self.workers_finished;
        let mut guard = finished.lock().unwrap();
        while *guard < started {
            guard = condvar
                .wait_timeout(guard, std::time::Duration::from_millis(1))
                .unwrap()
                .0;
        }
    }

    /// Wait for the worker's own drop to record the free.
    fn wait_for_worker_free(&self) {
        let start = std::time::Instant::now();
        while self.script.lock().unwrap().freed_by.is_none() {
            assert!(
                start.elapsed() < std::time::Duration::from_secs(5),
                "the worker never freed the timer"
            );
            std::thread::sleep(std::time::Duration::from_millis(1));
        }
    }
}

impl TimerHost for ScriptedHost {
    fn now(&self, _clock: TimerClock) -> Timespec {
        let now = self.script.lock().unwrap().now;
        self.emit(format!("C {}", stamp(now)));
        now
    }

    fn start(&self, body: Box<dyn FnOnce() + Send>) -> WorkerId {
        let id = {
            let mut script = self.script.lock().unwrap();
            script.workers_started += 1;
            WorkerId::new(script.workers_started as u64)
        };
        let finished = Arc::clone(&self.workers_finished);
        std::thread::Builder::new()
            .name(format!("scripted-timer-{}", id.index()))
            .spawn(move || {
                body();
                // The worker's own reference has just dropped; tell the driver.
                let (count, condvar) = &*finished;
                *count.lock().unwrap() += 1;
                condvar.notify_all();
            })
            .expect("the scripted timer thread could not be started");
        id
    }

    fn poke(&self, _worker: WorkerId) {
        let mut script = self.script.lock().unwrap();
        script.signalled = true;
        self.condvar.notify_all();
    }

    fn sleep(&self, remaining: Timespec) {
        let mut script = self.script.lock().unwrap();
        script.sleep_count += 1;
        let request = format!("S#{} {}", script.sleep_count, stamp(remaining));
        drop(script);
        self.emit(request);

        let mut script = self.script.lock().unwrap();
        script.in_sleep = true;
        script.signalled = false;
        while !script.release_sleep && !script.signalled {
            let (guard, _) = self
                .condvar
                .wait_timeout(script, std::time::Duration::from_millis(1))
                .unwrap();
            script = guard;
        }
        if script.signalled {
            // The driver decides when the interrupted worker resumes, so
            // whatever it prints about the poke comes first.
            script.signalled = false;
            script.in_sleep = false;
            while !script.interrupted_ack {
                let (guard, _) = self
                    .condvar
                    .wait_timeout(script, std::time::Duration::from_millis(1))
                    .unwrap();
                script = guard;
            }
            script.interrupted_ack = false;
        } else {
            script.now = script.now.checked_add_once(remaining);
            script.release_sleep = false;
            script.in_sleep = false;
        }
    }
}

/// The timer's drop hook: it reports who freed the timer, exactly like the C
/// harness's wrapped `free`.
struct FreeSpy {
    host: Arc<ScriptedHost>,
    main_thread: std::thread::ThreadId,
}

impl Drop for FreeSpy {
    fn drop(&mut self) {
        let who = if std::thread::current().id() == self.main_thread {
            "caller"
        } else {
            "thread"
        };
        // Record the free before publishing the flag, so a driver that sees
        // the flag also sees the record.
        self.host.emit(format!("F {who}"));
        self.host.script.lock().unwrap().freed_by = Some(who);
    }
}

/// One timer plus the driver's view of the transcript.
struct Harness {
    host: Arc<ScriptedHost>,
    timer: Option<Timer>,
}

impl Harness {
    fn new(host: &Arc<ScriptedHost>) -> Self {
        // The spy lives inside the timer's own allocation (through the
        // callback), so the drop that frees the timer fires it.
        let spy = Arc::new(FreeSpy {
            host: Arc::clone(host),
            main_thread: std::thread::current().id(),
        });
        let timer_host = Arc::clone(host);
        let timer = Timer::new(
            TimerClock::Monotonic,
            Arc::clone(host) as Arc<dyn TimerHost>,
            Box::new(move |state| {
                let _keepalive = &spy;
                let mut script = timer_host.script.lock().unwrap();
                script.callbacks += 1;
                let record = format!(
                    "K {} now={} start={} end={} interval={} active={} running={}",
                    script.callbacks,
                    stamp(script.now),
                    stamp(state.start),
                    stamp(state.end),
                    stamp(state.interval),
                    u8::from(state.active),
                    u8::from(state.thread_running)
                );
                drop(script);
                timer_host.emit(record);
            }),
        );
        Self {
            host: Arc::clone(host),
            timer: Some(timer),
        }
    }

    fn state(&self) -> TimerState {
        self.timer.as_ref().expect("the timer is live").state()
    }

    fn log_new(&mut self, label: &str) {
        let state = self.state();
        self.host.emit(format!(
            "N {label} now={} active={} running={} dead={}",
            stamp(self.host.script.lock().unwrap().now),
            u8::from(state.active),
            u8::from(state.thread_running),
            u8::from(state.dead)
        ));
    }

    fn log_state(&mut self, label: &str) {
        let state = self.state();
        self.host.emit(format!(
            "X {label} start={} end={} interval={} active={} running={} dead={}",
            stamp(state.start),
            stamp(state.end),
            stamp(state.interval),
            u8::from(state.active),
            u8::from(state.thread_running),
            u8::from(state.dead)
        ));
    }

    /// `timer_set` with a `NULL` old spec, which is what a fresh timer needs.
    fn set_fresh(&mut self, spec: TimerSpec, label: &str) {
        if let Some(timer) = self.timer.as_ref() {
            timer.set(spec);
        }
        self.host.emit(format!("T {label} rc=0 old=NULL"));
    }

    fn set(&mut self, spec: TimerSpec, label: &str) {
        let old = self
            .timer
            .as_ref()
            .expect("the timer is live")
            .set(spec)
            .expect("the replaced spec");
        self.host.emit(format!(
            "T {label} rc=0 old_value={} old_interval={}",
            stamp(old.value),
            stamp(old.interval)
        ));
    }

    /// `timer_free`. The drop hook records who freed the timer; the caller
    /// waits for the worker's own free with [`ScriptedHost::wait_for_worker_free`]
    /// after acknowledging the poke.
    fn free(&mut self) {
        // Drop the previous phase's marker so a stale one cannot satisfy a wait.
        self.host.script.lock().unwrap().freed_by = None;
        self.timer.take().expect("the timer is live").free();
    }
}

/// A sub-second helper, so the corpus reads like the C driver.
fn nanos(nsec: i64) -> Timespec {
    Timespec::new(0, nsec)
}

/// The whole corpus, in the order the C driver ran it.
#[test]
fn replays_every_record_of_the_c_corpus() {
    let corpus = Corpus::load();
    let mut replay = Replay::new(&corpus);
    let host = ScriptedHost::new();

    // Phase 1: a fresh timer, then a zero-value set that starts no worker.
    let mut timer = Harness::new(&host);
    timer.log_new("fresh");
    timer.set_fresh(TimerSpec::default(), "zero");
    timer.log_state("zero");
    host.replay_into(&mut replay);

    // Phase 2: arm it. The worker parks in the first sleep.
    timer.set(
        TimerSpec {
            value: nanos(500_000_000),
            interval: nanos(250_000_000),
        },
        "arm",
    );
    host.wait_for_sleep(1);
    timer.log_state("arm");
    host.release_sleep();
    host.wait_for_callbacks(1);
    host.wait_for_sleep(2);
    timer.log_state("fired1");
    host.replay_into(&mut replay);

    // Phase 3: the worker is sleeping the interval out. Rescheduling pokes it,
    // so the sleep returns EINTR and it sleeps for the new value instead.
    timer.set(
        TimerSpec {
            value: Timespec::new(2, 0),
            interval: nanos(0),
        },
        "requeue",
    );
    host.acknowledge_poke();
    host.wait_for_sleep(3);
    timer.log_state("requeue");
    host.replay_into(&mut replay);
    host.release_sleep();
    host.wait_for_callbacks(2);
    host.wait_for_worker(timer.timer.as_ref().unwrap());
    timer.log_state("done");
    host.replay_into(&mut replay);
    timer.free();
    host.replay_into(&mut replay);

    // Phase 4: deactivate a running timer without freeing it.
    let mut second = Harness::new(&host);
    second.log_new("second");
    second.set_fresh(
        TimerSpec {
            value: nanos(500_000_000),
            interval: nanos(0),
        },
        "arm2",
    );
    host.wait_for_sleep(4);
    second.log_state("arm2");
    host.replay_into(&mut replay);
    second.set(TimerSpec::default(), "stop");
    host.acknowledge_poke();
    host.wait_for_worker(second.timer.as_ref().unwrap());
    second.log_state("stop");
    host.replay_into(&mut replay);
    second.free();
    host.replay_into(&mut replay);

    // Phase 5: free a timer while its worker is mid-sleep. The worker sees the
    // cleared `active` flag, leaves the loop, and frees the timer itself.
    let mut third = Harness::new(&host);
    third.log_new("third");
    third.set_fresh(
        TimerSpec {
            value: nanos(750_000_000),
            interval: nanos(0),
        },
        "arm3",
    );
    host.wait_for_sleep(5);
    third.log_state("arm3");
    host.emit(format!(
        "D free_while_sleeping in_sleep={}",
        u8::from(host.script.lock().unwrap().in_sleep)
    ));
    host.replay_into(&mut replay);
    third.free();
    host.acknowledge_poke();
    host.wait_for_worker_free();
    host.replay_into(&mut replay);

    // Phase 6: a nanosecond count above one second. `timespec_add` carries once
    // and never re-checks, so `end` stays unnormalized; the worker's remaining
    // time is computed against it, and the callback fires when that
    // unnormalized deadline is reached.
    let mut fourth = Harness::new(&host);
    fourth.log_new("fourth");
    fourth.set_fresh(
        TimerSpec {
            value: nanos(1_500_000_000),
            interval: nanos(0),
        },
        "carry",
    );
    host.wait_for_sleep(6);
    fourth.log_state("carry");
    host.replay_into(&mut replay);
    host.release_sleep();
    host.wait_for_callbacks(3);
    host.wait_for_worker(fourth.timer.as_ref().unwrap());
    fourth.log_state("carry_done");
    host.replay_into(&mut replay);
    fourth.free();
    host.replay_into(&mut replay);

    let (callbacks, sleeps, clock) = {
        let script = host.script.lock().unwrap();
        (script.callbacks, script.sleep_count, script.now)
    };
    host.emit(format!(
        "Z callbacks={callbacks} sleeps={sleeps} clock={}",
        stamp(clock)
    ));
    host.replay_into(&mut replay);
    replay.finish();
}

/// The fixture must keep covering the details that are easy to lose: a zero
/// value starts no worker, a poke reschedules a parked sleep, and the
/// single-subtract carry leaves an unnormalized deadline behind.
#[test]
fn the_fixture_keeps_the_timer_quirks_under_test() {
    let corpus = Corpus::load();
    let has = |needle: &str| corpus.lines.iter().any(|line| line.contains(needle));
    assert!(has("T zero rc=0 old=NULL"), "the fresh-timer set");
    assert!(has("active=0 running=0"), "a zero set starts no worker");
    assert!(has("S#2 0.250000000"), "the interval re-arm");
    assert!(has("S#3 2.000000000"), "a poke reschedules the sleep");
    assert!(has("end=103.1000000000"), "the single-subtract carry");
    assert!(has("S#6 1.500000000"), "the unnormalized remaining");
    assert!(has("F caller"), "the inline free");
    assert!(has("F thread"), "the worker's own free");
    assert!(
        has("D free_while_sleeping in_sleep=1"),
        "the mid-sleep free"
    );
}
