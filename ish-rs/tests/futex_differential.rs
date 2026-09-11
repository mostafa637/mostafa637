//! Differential replay of unmodified `kernel/futex.c`.
//!
//! `tools/futex-dump.c` includes the upstream futex table and drives it
//! directly, with no threads: it builds waiters by hand the way `futex_wait`
//! does inside its locked section, runs wakes and requeues over those queues,
//! and prints the table after every step. The harness's wrapped
//! `pthread_cond_wait` / `pthread_cond_timedwait` decide what a park returns and
//! whether a signal lands while the task sleeps, and the wrapped
//! `pthread_cond_broadcast` names the waiter `notify` reached. `current` is a
//! real task over a real address space: page 0x100 is mapped and holds the dword
//! every futex address points at, page 0x200 is left unmapped.
//!
//! The Rust side runs the same corpus through [`ish_emu::futex`] with a
//! [`SyncHost`] and [`Parker`] that mirror that platform, and compares every
//! record: the guest dword, each wait and its outcome, the table after every
//! step, each waiter's whereabouts, wake and requeue counts, and the two
//! robust-list calls. The queues are transacted with the same calls the C
//! harness makes — `FutexTable::lock`, `TableGuard::get_unlocked`,
//! `Futex::enqueue`, `Waiter::dequeue`, `TableGuard::put_unlocked` — so
//! each hand-built waiter holds the one reference that the C one holds.
//!
//! One record cannot be observed directly: `N broadcast=wN`, the waiter a
//! `notify` reached. C saw it by wrapping the condvar's broadcast; the port's
//! [`Cond`] is a plain condition variable with no notification hook. The replay
//! derives those records from the *observed* queue order
//! ([`FutexTable::queue_of`], the same order the `H` and `Q` records pin) and
//! the rule that a wake takes waiters from the head — so a queue in the wrong
//! order still fails the comparison.
//!
//! Regenerate the C oracle locally with:
//!
//! ```text
//! ISH_SRC=/path/to/ish ./tools/gen_futex_reference.sh
//! ```

use std::cell::{Cell, RefCell};
use std::rc::Rc;
use std::sync::MutexGuard;

use ish_emu::futex::{
    mm_identity, sys_get_robust_list, sys_set_robust_list, FutexTable, Waiter, FUTEX_CMD_MASK_,
    FUTEX_PRIVATE_FLAG_, FUTEX_REQUEUE_, FUTEX_WAIT_, FUTEX_WAKE_, ROBUST_LIST_HEAD_SIZE,
};
use ish_emu::memory::P_RWX;
use ish_emu::mmap::Mm;
use ish_emu::sync::{Cond, ParkResult, Parker, SyncHost, Timespec, Waiting};
use ish_emu::task::TaskTable;

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/futex_reference.txt"
);

/// The page the corpus maps, and the two guest addresses it waits on:
/// `dword_at(0x100)` and `dword_at(0x140)`.
const PAGE_A: u32 = 0x100;
const ADDR_A: u32 = PAGE_A << 12;
const ADDR_B: u32 = 0x140 << 12;
/// Where the corpus puts the guest `struct timespec_` for a timed wait.
const TIMEOUT_ADDR: u32 = ADDR_A + 64;
/// `dword_at(0x200) + 8`: inside a page that is never mapped.
const FAULT_TIMEOUT_ADDR: u32 = (0x200 << 12) + 8;
/// Where `sys_get_robust_list` writes its two outputs.
const LIST_OUT: u32 = ADDR_A + 128;
const LEN_OUT: u32 = ADDR_A + 132;

/// C's `%#x`: zero prints without a prefix.
fn hex(value: u32) -> String {
    if value == 0 {
        "0".to_owned()
    } else {
        format!("{value:#x}")
    }
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
        let text = std::fs::read_to_string(FIXTURE).expect("futex reference fixture");
        let lines: Vec<String> = text
            .lines()
            .filter(|line| !line.starts_with('#'))
            .map(str::to_owned)
            .collect();
        assert!(!lines.is_empty(), "the futex fixture is empty");
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

/// `current->pending`, `current->blocked`, and the scripted clock. C reads the
/// first two through `is_signal_pending` inside `wait_for`, and the clock
/// through `clock_gettime` when it has a relative timeout to turn into a
/// deadline.
struct ScriptedHost {
    pending: Cell<u64>,
    blocked: Cell<u64>,
    now: Cell<Timespec>,
    waiting: RefCell<Option<Waiting>>,
    should_unwind: Cell<bool>,
    pending_unwind: Cell<bool>,
}

impl ScriptedHost {
    fn new() -> Self {
        Self {
            pending: Cell::new(0),
            blocked: Cell::new(0),
            // The corpus's `script_now`.
            now: Cell::new(Timespec::new(100, 0)),
            waiting: RefCell::new(None),
            should_unwind: Cell::new(false),
            pending_unwind: Cell::new(false),
        }
    }
}

impl SyncHost for ScriptedHost {
    fn signal_pending(&self) -> bool {
        (self.pending.get() & !self.blocked.get()) != 0
    }

    fn monotonic_now(&self) -> Timespec {
        self.now.get()
    }

    fn set_waiting(&self, waiting: Option<Waiting>) {
        *self.waiting.borrow_mut() = waiting;
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
        self.pending_unwind.replace(false)
    }
}

/// The corpus's wrapped `pthread_cond_wait` / `pthread_cond_timedwait`: the park
/// returns a scripted status, and `pending_after` is the C's assignment of
/// `current->pending = 1` just before it returns.
struct ScriptedParker<'a> {
    host: &'a ScriptedHost,
    script: Cell<(i32, bool)>,
    log: RefCell<Vec<String>>,
}

impl<'a> ScriptedParker<'a> {
    fn new(host: &'a ScriptedHost) -> Self {
        Self {
            host,
            script: Cell::new((0, false)),
            log: RefCell::new(Vec::new()),
        }
    }

    /// Arm the next park: the status pthread returns, and whether a signal is
    /// pending by the time it does.
    fn arm(&self, park_rc: i32, pending_after: bool) {
        self.script.set((park_rc, pending_after));
    }

    /// The `P` records the parks produced since the last drain.
    fn take_log(&self) -> Vec<String> {
        std::mem::take(&mut self.log.borrow_mut())
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
        let (park_rc, pending_after) = self.script.get();
        assert!(
            self.host.waiting.borrow().is_some(),
            "wait_for must publish the waiting cond before it parks"
        );
        self.log.borrow_mut().push(match deadline {
            None => format!(
                "P wait - rc={park_rc} pending_after={}",
                u8::from(pending_after)
            ),
            Some(deadline) => format!(
                "P timedwait {}.{:09} rc={park_rc} pending_after={}",
                deadline.sec,
                deadline.nsec,
                u8::from(pending_after)
            ),
        });
        if pending_after {
            // The C wrapper assigns `current->pending = 1` while the task is
            // still parked, which is what the post-wait check then sees.
            self.host.pending.set(1);
        }
        let result = match park_rc {
            0 => ParkResult::Woken,
            110 => ParkResult::TimedOut,
            other => ParkResult::Failed(other),
        };
        (result, guard)
    }
}

// ---------------------------------------------------------------------------
// the corpus's guest state
// ---------------------------------------------------------------------------

/// One `step_wait` call: the arguments, and how the park is scripted.
struct WaitStep {
    label: &'static str,
    addr: u32,
    val: u32,
    /// 0 = no timeout pointer, 1 = a mapped guest timespec, 2 = an unmapped one.
    timeout_mode: u32,
    park_rc: i32,
    pending_after: bool,
    pending: u64,
    blocked: u64,
}

/// The corpus's nine waits, in the order the C `main` runs them.
const WAIT_STEPS: [WaitStep; 9] = [
    WaitStep {
        label: "unmapped",
        addr: 0x200 << 12,
        val: 7,
        timeout_mode: 0,
        park_rc: 0,
        pending_after: false,
        pending: 0,
        blocked: 0,
    },
    WaitStep {
        label: "mismatch",
        addr: ADDR_A,
        val: 8,
        timeout_mode: 0,
        park_rc: 0,
        pending_after: false,
        pending: 0,
        blocked: 0,
    },
    WaitStep {
        label: "woken",
        addr: ADDR_A,
        val: 7,
        timeout_mode: 0,
        park_rc: 0,
        pending_after: false,
        pending: 0,
        blocked: 0,
    },
    WaitStep {
        label: "timeout",
        addr: ADDR_A,
        val: 7,
        timeout_mode: 1,
        park_rc: 110,
        pending_after: false,
        pending: 0,
        blocked: 0,
    },
    WaitStep {
        label: "timeout-error",
        addr: ADDR_A,
        val: 7,
        timeout_mode: 1,
        park_rc: 22,
        pending_after: false,
        pending: 0,
        blocked: 0,
    },
    WaitStep {
        label: "pending-before",
        addr: ADDR_A,
        val: 7,
        timeout_mode: 0,
        park_rc: 0,
        pending_after: false,
        pending: 0x10,
        blocked: 0,
    },
    WaitStep {
        label: "pending-masked",
        addr: ADDR_A,
        val: 7,
        timeout_mode: 0,
        park_rc: 0,
        pending_after: false,
        pending: 0x10,
        blocked: 0x10,
    },
    WaitStep {
        label: "pending-during",
        addr: ADDR_A,
        val: 7,
        timeout_mode: 1,
        park_rc: 0,
        pending_after: true,
        pending: 0,
        blocked: 0,
    },
    WaitStep {
        label: "timeout-fault",
        addr: ADDR_A,
        val: 7,
        timeout_mode: 2,
        park_rc: 0,
        pending_after: false,
        pending: 0,
        blocked: 0,
    },
];

/// One `sys_futex` call from the corpus's later sections.
struct FutexCall {
    op: u32,
    uaddr: u32,
    val: u32,
    /// The syscall's fourth argument: a timeout pointer for a wait, `val2` for a
    /// requeue.
    val2: u32,
    uaddr2: u32,
}

/// The C corpus's task, address space, table, and hand-built waiters.
struct Harness {
    tasks: TaskTable,
    pid: i32,
    /// C's `struct mem *`, as the port's address-space token.
    mm: usize,
    table: FutexTable,
    /// C's `static long waiter_ids`.
    next_waiter_id: u64,
    /// C's `struct probe_waiter waiters[4]`.
    slots: Vec<Option<Rc<Waiter>>>,
}

impl Harness {
    fn new() -> Self {
        let mut tasks = TaskTable::new();
        let pid = tasks.create_task(None).expect("the corpus task");
        let mm = Rc::new(RefCell::new(Mm::new()));
        assert_eq!(
            mm.borrow_mut().mem.map_nothing(PAGE_A, 1, P_RWX),
            0,
            "the corpus maps page 0x100"
        );
        tasks.task_set_mm(pid, Rc::clone(&mm)).expect("the mm");
        tasks.set_current(pid).expect("the corpus task is current");
        let mm = mm_identity(tasks.task(pid).expect("the corpus task")).expect("the mm is set");
        let mut harness = Self {
            tasks,
            pid,
            mm,
            table: FutexTable::new(),
            next_waiter_id: 0,
            slots: vec![None; 4],
        };
        harness.write_guest_dword(ADDR_A, 7);
        harness
    }

    // -- guest memory -------------------------------------------------------

    /// `set_guest_dword`: the word every futex address points at.
    fn write_guest_dword(&mut self, addr: u32, value: u32) {
        let pid = self.pid;
        let task = self.tasks.task_mut(pid).expect("the corpus task");
        task.user_write(addr, &value.to_le_bytes())
            .expect("the corpus dword is mapped");
    }

    fn read_guest_dword(&mut self, addr: u32) -> u32 {
        let pid = self.pid;
        let task = self.tasks.task_mut(pid).expect("the corpus task");
        let mut bytes = [0u8; 4];
        task.user_read(addr, &mut bytes)
            .expect("the corpus reads a mapped dword");
        u32::from_le_bytes(bytes)
    }

    /// The corpus's guest `struct timespec_`: `{1, 500000000}`.
    fn write_guest_timeout(&mut self) {
        let mut bytes = [0u8; 8];
        bytes[0..4].copy_from_slice(&1u32.to_le_bytes());
        bytes[4..8].copy_from_slice(&500_000_000u32.to_le_bytes());
        let pid = self.pid;
        let task = self.tasks.task_mut(pid).expect("the corpus task");
        task.user_write(TIMEOUT_ADDR, &bytes)
            .expect("the corpus timespec is mapped");
    }

    fn mapped_pages(&self) -> i32 {
        let mm = self
            .tasks
            .task(self.pid)
            .expect("the corpus task")
            .mm
            .clone()
            .expect("the mm");
        let pages = mm.borrow().mem.pgdir_used;
        pages
    }

    /// The waiter in `slots[slot]`, which the corpus must have built first.
    fn waiter(&self, slot: usize) -> Rc<Waiter> {
        Rc::clone(
            self.slots[slot]
                .as_ref()
                .expect("the corpus built this waiter"),
        )
    }

    // -- the corpus's locked helpers ----------------------------------------

    /// `corpus_build(&waiters[slot], addr)`: number a waiter the way the C
    /// harness numbers them, take a reference to the futex, and queue it.
    fn build(&mut self, slot: usize, addr: u32) {
        let id = self.next_waiter_id + 1;
        self.next_waiter_id = id;
        let guard = self.table.lock();
        let futex = guard.get_unlocked(self.mm, addr);
        let waiter = Rc::new(Waiter::new(id, &futex));
        futex.enqueue(Rc::clone(&waiter));
        guard.unlock();
        self.slots[slot] = Some(waiter);
    }

    /// `corpus_release(&waiters[slot])`: leave the queue and put the waiter's
    /// own reference to whichever futex it points at now.
    fn release(&mut self, slot: usize) {
        let waiter = self.waiter(slot);
        let guard = self.table.lock();
        waiter.dequeue();
        if let Some(futex) = waiter.futex() {
            guard.put_unlocked(&futex);
        }
        guard.unlock();
    }

    /// `reset_futexes`: an empty table and the waiter numbering back to one. The
    /// waiters themselves stay in their slots, exactly as the C array does.
    fn reset(&mut self) {
        self.table = FutexTable::new();
        self.next_waiter_id = 0;
    }

    // -- the transcripts ----------------------------------------------------

    /// `dump_table`, ordered by address.
    fn dump_table(&self, replay: &mut Replay<'_>, tag: &str) {
        let entries = self.table.entries();
        for entry in &entries {
            replay.expect(&format!(
                "H {tag} addr={} refs={} queue={}",
                hex(entry.addr),
                entry.refcount,
                entry.queue
            ));
        }
        replay.expect(&format!("H {tag} entries={}", entries.len()));
    }

    /// `dump_waiter`: the waiter's id, the address it is queued on, and whether
    /// its node is still linked into that queue.
    fn dump_waiter(&self, replay: &mut Replay<'_>, slot: usize, tag: &str) {
        let waiter = self.waiter(slot);
        let queued = self
            .table
            .queue_of(self.mm, waiter.addr())
            .contains(&waiter.id());
        replay.expect(&format!(
            "Q {tag} id={} futex={} queued={}",
            waiter.id(),
            hex(waiter.addr()),
            u8::from(queued)
        ));
    }

    /// The `N broadcast=` records the C harness's wrapped condvar produced.
    ///
    /// `futex_wakelike` notifies the head of the queue first and removes it as
    /// it goes, so the targets are the queue's head up to `wake_max` — read from
    /// the queue the port actually holds.
    fn expect_notifications(&self, replay: &mut Replay<'_>, uaddr: u32, wake_max: u32) {
        for id in self
            .table
            .queue_of(self.mm, uaddr)
            .iter()
            .take(wake_max as usize)
        {
            replay.expect(&format!("N broadcast=w{id}"));
        }
    }

    // -- the corpus's steps -------------------------------------------------

    /// One `step_wait`: a full `sys_futex(FUTEX_WAIT_)` with the park scripted.
    fn wait_step(
        &mut self,
        replay: &mut Replay<'_>,
        host: &ScriptedHost,
        parker: &ScriptedParker<'_>,
        step: &WaitStep,
    ) {
        host.pending.set(step.pending);
        host.blocked.set(step.blocked);
        parker.arm(step.park_rc, step.pending_after);
        let timeout = match step.timeout_mode {
            0 => 0,
            1 => {
                self.write_guest_timeout();
                TIMEOUT_ADDR
            }
            mode => {
                assert_eq!(mode, 2, "unknown timeout mode");
                FAULT_TIMEOUT_ADDR
            }
        };
        replay.expect(&format!(
            "W {} addr={} val={} timeout={} park={} pending_before={} blocked_before={}",
            step.label,
            hex(step.addr),
            step.val,
            step.timeout_mode,
            step.park_rc,
            step.pending,
            step.blocked
        ));
        let pid = self.pid;
        let task = self.tasks.task_mut(pid).expect("the corpus task");
        let rc = self.table.sys_futex(
            task,
            step.addr,
            FUTEX_WAIT_,
            step.val,
            timeout,
            0,
            0,
            host,
            parker,
        );
        for line in parker.take_log() {
            replay.expect(&line);
        }
        replay.expect(&format!(
            "W {} rc={rc} pending_after={}",
            step.label,
            host.pending.get()
        ));
        host.pending.set(0);
        host.blocked.set(0);
    }

    /// `futex_wake`, with the notifications it will produce reported first.
    fn wake(&mut self, replay: &mut Replay<'_>, uaddr: u32, wake_max: u32) -> i32 {
        self.expect_notifications(replay, uaddr, wake_max);
        self.table.futex_wake(self.mm, uaddr, wake_max)
    }

    /// `futex_wakelike`, with the wake pass's notifications reported first.
    fn wakelike(
        &mut self,
        replay: &mut Replay<'_>,
        op: u32,
        uaddr: u32,
        wake_max: u32,
        requeue_max: u32,
        requeue_addr: u32,
    ) -> i32 {
        self.expect_notifications(replay, uaddr, wake_max);
        self.table
            .futex_wakelike(op, self.mm, uaddr, wake_max, requeue_max, requeue_addr)
    }

    /// `sys_futex` from the corpus's later sections.
    fn call(
        &mut self,
        replay: &mut Replay<'_>,
        host: &dyn SyncHost,
        parker: &impl Parker,
        call: &FutexCall,
    ) -> i32 {
        if matches!(call.op & FUTEX_CMD_MASK_, FUTEX_WAKE_ | FUTEX_REQUEUE_) {
            self.expect_notifications(replay, call.uaddr, call.val);
        }
        let pid = self.pid;
        let task = self.tasks.task_mut(pid).expect("the corpus task");
        self.table.sys_futex(
            task,
            call.uaddr,
            call.op,
            call.val,
            call.val2,
            call.uaddr2,
            0,
            host,
            parker,
        )
    }

    // -- the robust-list calls ----------------------------------------------

    /// `sys_set_robust_list` and the field it stores.
    fn set_robust_list(&mut self, list: u32, len: u32) -> (i32, u32) {
        let pid = self.pid;
        let task = self.tasks.task_mut(pid).expect("the corpus task");
        let rc = sys_set_robust_list(task, list, len);
        (rc, task.robust_list)
    }

    /// `sys_get_robust_list`.
    fn get_robust_list(&mut self, pid: i32, list_ptr: u32, len_ptr: u32) -> i32 {
        sys_get_robust_list(&mut self.tasks, pid, list_ptr, len_ptr)
    }
}

// ---------------------------------------------------------------------------
// the replay
// ---------------------------------------------------------------------------

#[test]
fn replays_every_record_of_the_c_corpus() {
    let corpus = Corpus::load();
    let mut replay = Replay::new(&corpus);
    let host = ScriptedHost::new();
    let parker = ScriptedParker::new(&host);
    let mut harness = Harness::new();

    // The guest word, and the one mapped page.
    replay.expect(&format!(
        "G dword={} mapped_pages={}",
        harness.read_guest_dword(ADDR_A),
        harness.mapped_pages()
    ));

    // 1. `futex_load`'s failure modes, the successful wait, the timeout paths,
    //    the pending-signal paths, and the faulting timeout pointer.
    for step in &WAIT_STEPS {
        harness.wait_step(&mut replay, &host, &parker, step);
    }
    harness.dump_table(&mut replay, "after-waits");

    // 2. The wait published its waiter and removed it again: the table is empty.
    //    The corpus resets before building queues of its own.
    harness.reset();

    // 3. Wake counting: three waiters on A, one on B.
    for slot in 0..3 {
        harness.build(slot, ADDR_A);
    }
    harness.build(3, ADDR_B);
    harness.dump_table(&mut replay, "three-one");
    let rc = harness.wake(&mut replay, ADDR_A, 0);
    replay.expect(&format!("K wake0 rc={rc}"));
    let rc = harness.wake(&mut replay, ADDR_A, 2);
    replay.expect(&format!("K wake2 rc={rc}"));
    harness.dump_table(&mut replay, "after-wake2");
    for slot in 0..4 {
        harness.dump_waiter(&mut replay, slot, "after-wake2");
    }
    // Each woken waiter puts its own reference back as it returns.
    harness.release(0);
    harness.release(1);
    harness.dump_table(&mut replay, "after-wake2-returns");
    let rc = harness.wake(&mut replay, ADDR_A, 5);
    replay.expect(&format!("K wake5 rc={rc}"));
    harness.release(2);
    harness.dump_table(&mut replay, "after-wake5");

    // 4. Requeue: one woken, two moved, each moving taking its reference.
    for slot in 0..3 {
        harness.build(slot, ADDR_A);
    }
    harness.dump_table(&mut replay, "requeue-before");
    let rc = harness.wakelike(&mut replay, FUTEX_REQUEUE_, ADDR_A, 1, 2, ADDR_B);
    replay.expect(&format!("K requeue rc={rc}"));
    harness.dump_table(&mut replay, "requeue-after");
    for slot in 0..4 {
        harness.dump_waiter(&mut replay, slot, "requeue-after");
    }

    // 5. Draining: A is empty, B holds three, and returning every waiter takes
    //    both entries out of the table.
    let rc = harness.wake(&mut replay, ADDR_A, 10);
    replay.expect(&format!("K wake-rest rc={rc}"));
    let rc = harness.wake(&mut replay, ADDR_B, 10);
    replay.expect(&format!("K wake-target rc={rc}"));
    for slot in 0..4 {
        harness.release(slot);
    }
    harness.dump_table(&mut replay, "drained");

    // 6. The syscall-level dispatch: the private flag, an unsupported operation,
    //    and a requeue through `sys_futex`.
    harness.reset();
    let rc = harness.call(
        &mut replay,
        &host,
        &parker,
        &FutexCall {
            op: FUTEX_WAKE_ | FUTEX_PRIVATE_FLAG_,
            uaddr: ADDR_A,
            val: 1,
            val2: 0,
            uaddr2: 0,
        },
    );
    replay.expect(&format!("K private-wake rc={rc}"));
    let rc = harness.call(
        &mut replay,
        &host,
        &parker,
        &FutexCall {
            op: 9,
            uaddr: ADDR_A,
            val: 1,
            val2: 0,
            uaddr2: 0,
        },
    );
    replay.expect(&format!("K unsupported rc={rc}"));
    harness.build(0, ADDR_A);
    harness.build(1, ADDR_A);
    let rc = harness.call(
        &mut replay,
        &host,
        &parker,
        &FutexCall {
            op: FUTEX_REQUEUE_,
            uaddr: ADDR_A,
            val: 0,
            val2: 2,
            uaddr2: ADDR_B,
        },
    );
    replay.expect(&format!("K sys-requeue rc={rc}"));
    harness.dump_table(&mut replay, "after-sys-requeue");
    for slot in 0..2 {
        harness.dump_waiter(&mut replay, slot, "after-sys-requeue");
    }
    let rc = harness.wake(&mut replay, ADDR_B, 10);
    replay.expect(&format!("K wake-target2 rc={rc}"));
    for slot in 0..2 {
        harness.release(slot);
    }
    harness.dump_table(&mut replay, "end");

    // 7. The robust-list calls.
    let (rc, robust) = harness.set_robust_list(0x1000, 8);
    replay.expect(&format!("L set-len8 rc={rc} robust={}", hex(robust)));
    let (rc, robust) = harness.set_robust_list(0x1000, 12);
    replay.expect(&format!("L set-len12 rc={rc} robust={}", hex(robust)));
    let rc = harness.get_robust_list(harness.pid, LIST_OUT, LEN_OUT);
    let list = harness.read_guest_dword(LIST_OUT);
    let len = harness.read_guest_dword(LEN_OUT);
    replay.expect(&format!("L get-self rc={rc} list={} len={len}", hex(list)));
    let other = harness.pid + 1;
    assert!(
        !harness.tasks.pid_exists(other),
        "the corpus asks about a pid that does not exist"
    );
    let rc = harness.get_robust_list(other, LIST_OUT, LEN_OUT);
    replay.expect(&format!("L get-other rc={rc}"));
    let (rc, robust) = harness.set_robust_list(0, 12);
    replay.expect(&format!("L set-null rc={rc} robust={}", hex(robust)));

    replay.finish();
}

/// The records the port's documented quirks depend on, so a regeneration that
/// silently changed the corpus would be caught even if it stayed
/// self-consistent.
#[test]
fn the_fixture_keeps_the_futex_quirks_under_test() {
    let text = std::fs::read_to_string(FIXTURE).expect("futex reference fixture");
    let has = |needle: &str| text.lines().any(|line| line.contains(needle));

    assert!(has("G dword=7 mapped_pages=1"), "the corpus's guest word");
    assert!(has("W unmapped rc=-14"), "an unreadable word is EFAULT");
    assert!(
        has("W mismatch rc=-11"),
        "a word that does not match is EAGAIN"
    );
    assert!(has("W woken rc=0"), "a woken wait succeeds");
    assert!(
        has("P timedwait 101.500000000") && has("W timeout rc=-110"),
        "the scripted clock plus the guest timeout is the deadline, and its expiry is ETIMEDOUT"
    );
    assert!(
        has("W timeout-error rc=0"),
        "a park failure that is not a timeout is reported as success"
    );
    assert!(
        has("W pending-before rc=-4"),
        "a pending signal returns EINTR before parking"
    );
    assert!(
        has("W pending-masked rc=0 pending_after=16"),
        "a blocked signal does not stop the wait"
    );
    assert!(
        has("W pending-during rc=-4 pending_after=1"),
        "a signal that lands during the wait is EINTR"
    );
    assert!(
        has("timeout-fault addr=0x100000 val=7 timeout=2") && has("W timeout-fault rc=-14"),
        "an unmapped timespec is EFAULT before anything is queued"
    );
    assert!(has("H after-waits entries=0"), "no wait leaks its futex");

    assert!(
        has("H three-one addr=0x100000 refs=3 queue=3"),
        "each hand-built waiter holds its own reference"
    );
    assert!(has("K wake0 rc=0"), "a zero wake wakes nobody");
    assert!(has("K wake2 rc=2"), "a wake stops at its count");
    assert!(
        has("H after-wake2-returns addr=0x100000 refs=1 queue=1"),
        "a woken waiter holds its reference until it returns"
    );
    assert!(has("K wake5 rc=1"), "a wake stops at the queue's end");

    assert!(has("N broadcast=w5"), "the requeue's wake pass");
    assert!(has("K requeue rc=3"), "one woken plus two requeued");
    assert!(
        has("H requeue-after addr=0x100000 refs=1 queue=0"),
        "the source keeps the requeue's own reference"
    );
    assert!(
        has("H requeue-after addr=0x140000 refs=3 queue=3"),
        "each moved waiter brings its reference"
    );
    assert!(
        has("Q requeue-after id=6 futex=0x140000 queued=1"),
        "a moved waiter is re-pointed at the target"
    );
    assert!(
        has("N broadcast=w4") && has("N broadcast=w6") && has("N broadcast=w7"),
        "a requeue moves waiters to the target's tail: w4 was queued first"
    );
    assert!(
        has("K wake-rest rc=0") && has("K wake-target rc=3"),
        "the emptied source, then the target's three"
    );
    assert!(has("H drained entries=0"), "everything drains");

    assert!(has("K private-wake rc=0"), "the private flag is masked off");
    assert!(
        has("K unsupported rc=-38"),
        "an unknown operation is ENOSYS"
    );
    assert!(
        has("K sys-requeue rc=2"),
        "a requeue with a zero wake count"
    );
    assert!(
        has("H after-sys-requeue addr=0x140000 refs=2 queue=2"),
        "the moved waiters' references arrived through sys_futex"
    );
    assert!(has("K wake-target2 rc=2"), "and they can be woken there");

    assert!(
        has("L set-len8 rc=-22"),
        "the robust-list length is checked"
    );
    assert!(
        has("L set-len12 rc=0 robust=0x1000"),
        "and the address is stored unchecked"
    );
    assert!(
        has("L get-self rc=0 list=0x1000 len=12"),
        "a task may read its own list back"
    );
    assert!(has("L get-other rc=-1"), "another pid is EPERM");
    assert!(
        has("L set-null rc=0 robust=0"),
        "a NULL address with the right length succeeds"
    );

    // The trailer is the C harness's own bookkeeping. Nothing in the corpus
    // closes an fd or sends a signal, and its broadcast count is the number of
    // `N` records since the last reset.
    let trailer = text
        .lines()
        .find(|line| line.starts_with("# asbestos_invalidations"))
        .expect("the corpus trailer");
    assert!(
        trailer.ends_with("fd_closes 0 signals_sent 0 broadcasts 2 signals 0"),
        "the trailer's counts: `{trailer}`"
    );
    let last_section = text
        .split("K private-wake")
        .nth(1)
        .expect("the syscall section");
    assert_eq!(
        last_section.matches("N broadcast=").count(),
        2,
        "the trailer's broadcast count is the corpus's own"
    );
    assert_eq!(ROBUST_LIST_HEAD_SIZE, 12);
}
