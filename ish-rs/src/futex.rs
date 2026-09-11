//! `kernel/futex.{h,c}` — the futex table and the two robust-list calls.
//!
//! A futex is an address in the guest's address space plus a queue of waiters.
//! The kernel keeps one such object per (address space, address) pair, refcounts
//! it, and hash-indexes it; a wait parks on the object's condition variable
//! while releasing the table lock, and a wake notifies waiters in queue order.
//! `FUTEX_REQUEUE` moves waiters between two of those queues and transfers their
//! references with them.
//!
//! # What the port keeps
//!
//! * **The hash and the identity.** C hashes `(addr ^ (unsigned long) mem)`
//!   into 4096 buckets and identifies a futex by the `(mem, addr)` pair, so two
//!   *different* address spaces never share a futex even at the same address.
//!   [`FutexTable`] keeps the same bucket count, the same hash expression, and
//!   the same identity rule, with a stable address-space token standing in for
//!   the C pointer. The bucket index itself is a build artefact (it depends on
//!   where the host put `struct mem`), so it is not part of the reference
//!   fixture — but the arithmetic is ported and pinned by a unit test.
//! * **The reference counting, and what it protects.** `futex_put` frees the
//!   object when the count reaches zero, and C asserts the queue is empty at
//!   that point; the port keeps the assert. The woken waiter, not the waker,
//!   drops the reference it took when it started waiting — which is why a
//!   wake can leave a futex in the table with a queue of zero.
//! * **The requeue's bookkeeping.** Each moved waiter takes its reference to
//!   the target (`refcount--` on the source, `++` on the target) and is
//!   re-pointed at it, so the waiter knows where to put the reference back.
//!   C calls this "sketchy as hell"; the port does the same thing in the same
//!   order, assert included.
//! * **The wait contract inherited from `util/sync.c`.** `futex_wait` reports
//!   `EFAULT` for an unreadable word, `EAGAIN` when the word does not match,
//!   and otherwise exactly what `wait_for` returns — including its rule that a
//!   failed wait which is not `ETIMEDOUT` is reported as success.
//! * **`sys_set_robust_list` checks the length only.** A null or bogus address
//!   is accepted; the address is stored and never validated. `sys_get_robust_list`
//!   answers for the calling task only and returns `EPERM` for anything else,
//!   *including a pid that does not exist*, because C compares the looked-up
//!   task against `current`.
//!
//! # Deliberate differences
//!
//! * **`futex_get`'s "returns holding the lock" is a type, not a comment.** C
//!   documents it in prose and relies on the caller remembering; here
//!   [`FutexTable::get`] hands back a [`TableGuard`] alongside the object, and
//!   the unlocked pair is only reachable through that guard.
//! * **The timeout is read with `Task::user_read`.** C's `user_get` on a
//!   `struct timespec_` copies two `dword_t`s, and the assignment to `long`
//!   zero-extends them; the port keeps the zero-extension, so a guest that
//!   passes a negative `tv_nsec` gets a huge positive one, exactly as in C.
//! * **A task with no address space returns `EFAULT` instead of crashing.** C
//!   dereferences `current->mem` unconditionally in `futex_load`; the port has
//!   no way to reproduce a null dereference and reports the fault instead.
//! * **`STRACE` is not ported.** C prints for a non-private operation and for
//!   an unsupported one before returning `_ENOSYS`; there is no tracer here, so
//!   only the return value survives.
//! * **The waiter holds no strong reference to its futex.** C keeps a raw
//!   pointer that can dangle if the reference counting is violated; the port
//!   keeps a weak one, so the same bug is a missing notify rather than a use
//!   after free. The bookkeeping above makes it unreachable.

use std::cell::{Cell, RefCell};
use std::rc::{Rc, Weak};

use crate::memory::Mem;
use crate::mmu::{pgoffset, MemType, PAGE_SIZE};
use crate::sync::{notify, wait_for, Cond, Lock, LockGuard, Parker, SyncHost, Timespec};
use crate::task::{Pid, Task, TaskTable};

/// `FUTEX_WAIT_`
pub const FUTEX_WAIT_: u32 = 0;
/// `FUTEX_WAKE_`
pub const FUTEX_WAKE_: u32 = 1;
/// `FUTEX_REQUEUE_`
pub const FUTEX_REQUEUE_: u32 = 3;
/// `FUTEX_PRIVATE_FLAG_`
pub const FUTEX_PRIVATE_FLAG_: u32 = 128;
/// `FUTEX_CMD_MASK_` — `~(FUTEX_PRIVATE_FLAG_)`, which clears that one bit and
/// leaves everything above it alone, exactly as the C `int` arithmetic does.
pub const FUTEX_CMD_MASK_: u32 = !FUTEX_PRIVATE_FLAG_;
/// `FUTEX_HASH_BITS`
pub const FUTEX_HASH_BITS: u32 = 12;
/// `FUTEX_HASH_SIZE`
pub const FUTEX_HASH_SIZE: usize = 1 << FUTEX_HASH_BITS;

/// `_EFAULT`, the guest number for "the word is not readable".
pub const EFAULT: i32 = -14;
/// `_EAGAIN`, iSH's number for "the word does not match".
pub const EAGAIN: i32 = -11;
/// `_EINVAL`
pub const EINVAL: i32 = -22;
/// `_EPERM`
pub const EPERM: i32 = -1;
/// `_ENOSYS`
pub const ENOSYS: i32 = -38;

/// The size of `struct robust_list_head_`: `addr`, `offset`, `list_op_pending`.
pub const ROBUST_LIST_HEAD_SIZE: u32 = 12;

/// An address-space identity: C's `struct mem *`, as a comparable token.
pub type MmId = usize;

/// The identity of a futex: one per `(address space, address)`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct FutexKey {
    /// The address space this futex belongs to.
    pub mm: MmId,
    /// The guest address the waiters are sleeping on.
    pub addr: u32,
}

/// `FUTEX_HASH(addr, mem)`: C's `(addr ^ (unsigned long) mem) % FUTEX_HASH_SIZE`.
///
/// Only the low bits of the address-space token take part, which is what makes
/// the bucket index a build artefact rather than a contract: both operands are
/// XORed and taken modulo a power of two.
#[must_use]
pub fn futex_hash(mm: MmId, addr: u32) -> usize {
    ((u64::from(addr) ^ mm as u64) % FUTEX_HASH_SIZE as u64) as usize
}

/// The `(mm, addr)` identity of a task's address space, or `None` when the task
/// has no `mm` at all.
#[must_use]
pub fn mm_identity(task: &Task) -> Option<MmId> {
    task.mm
        .as_ref()
        .map(|mm| Rc::as_ptr(mm).cast::<()>() as usize)
}

/// `futex_load`: read the guest dword at `addr`.
///
/// C takes `read_wrlock(&mem->lock)`, calls `mem_ptr(.., MEM_READ)` and reads
/// the word with one dereference. The port's `Mem` is the address space and its
/// `RefCell` borrow is that lock, so `Mem::ptr` is the whole of the load — with
/// one safety addition at the end of the function.
#[must_use]
pub fn futex_load(mem: &mut Mem, addr: u32) -> Option<u32> {
    if pgoffset(addr) + 4 <= PAGE_SIZE {
        let ptr = mem.ptr(addr, MemType::Read)?;
        // SAFETY: `Mem::ptr` returned a pointer into a live backing object at
        // least one page long, and the whole four bytes lie inside that page.
        return Some(unsafe { ptr.cast::<u32>().read_unaligned() });
    }
    // A word straddling a page boundary: C reads it anyway and would walk off
    // the mapping; the port reads it byte by byte through the page table, so a
    // missing page is a fault rather than an out-of-bounds read.
    let mut bytes = [0u8; 4];
    for (index, byte) in bytes.iter_mut().enumerate() {
        let ptr = mem.ptr(addr + index as u32, MemType::Read)?;
        // SAFETY: single byte inside the page `Mem::ptr` just resolved.
        *byte = unsafe { ptr.read() };
    }
    Some(u32::from_le_bytes(bytes))
}

/// `struct futex_wait`: one waiter's condition variable and where it belongs.
pub struct Waiter {
    id: u64,
    cond: Cond,
    /// `wait->futex`. A requeue rewrites this, which is how the waiter knows
    /// which futex to put its reference back to when it wakes.
    target: RefCell<(u32, Weak<Futex>)>,
}

impl std::fmt::Debug for Waiter {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("Waiter")
            .field("id", &self.id)
            .field("addr", &self.addr())
            .finish()
    }
}

impl Waiter {
    /// A waiter for `futex`, numbered `id`.
    ///
    /// The kernel builds exactly one of these, inside [`FutexTable::futex_wait`],
    /// and numbers it from the table's counter. The differential corpus builds
    /// them by hand — the C harness owns a `struct probe_waiter` and a `long id`
    /// of its own — which is why the id is a parameter rather than being taken
    /// from the table here.
    #[must_use]
    pub fn new(id: u64, futex: &Rc<Futex>) -> Self {
        Self {
            id,
            cond: Cond::new(),
            target: RefCell::new((futex.addr(), Rc::downgrade(futex))),
        }
    }

    /// The waiter's identity, C's `cond_t *` standing in for it in the corpus.
    #[must_use]
    pub fn id(&self) -> u64 {
        self.id
    }

    /// The condition `futex_wakelike` notifies.
    #[must_use]
    pub fn cond(&self) -> &Cond {
        // The tuple is never replaced wholesale, only written in place, so the
        // `Cond` is stable for the waiter's lifetime.
        &self.cond
    }

    /// `wait->futex->addr`: the address this waiter is currently queued on.
    #[must_use]
    pub fn addr(&self) -> u32 {
        self.target.borrow().0
    }

    /// `wait->futex`, if the object is still alive.
    #[must_use]
    pub fn futex(&self) -> Option<Rc<Futex>> {
        self.target.borrow().1.upgrade()
    }

    /// `wait->futex = futex`, the requeue's re-pointing.
    fn point_at(&self, futex: &Rc<Futex>) {
        *self.target.borrow_mut() = (futex.addr(), Rc::downgrade(futex));
    }

    /// `list_remove_safe(&wait.queue)`: leave whichever queue holds this
    /// waiter. Idempotent, like the C is when a waker removed it first.
    pub fn dequeue(&self) {
        if let Some(futex) = self.futex() {
            futex.dequeue(self);
        }
    }
}

/// `struct futex`: a refcounted queue of waiters for one guest address.
pub struct Futex {
    key: FutexKey,
    id: u64,
    refcount: Cell<u32>,
    queue: RefCell<Vec<Rc<Waiter>>>,
}

impl std::fmt::Debug for Futex {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("Futex")
            .field("key", &self.key)
            .field("refcount", &self.refcount.get())
            .field("queue", &self.queue.borrow().len())
            .finish()
    }
}

impl Futex {
    fn new(id: u64, key: FutexKey) -> Self {
        Self {
            key,
            id,
            refcount: Cell::new(1),
            queue: RefCell::new(Vec::new()),
        }
    }

    /// The address this futex is keyed on.
    #[must_use]
    pub fn addr(&self) -> u32 {
        self.key.addr
    }

    /// The address space this futex belongs to.
    #[must_use]
    pub fn key(&self) -> FutexKey {
        self.key
    }

    /// `futex->refcount`.
    #[must_use]
    pub fn refcount(&self) -> u32 {
        self.refcount.get()
    }

    /// The number of waiters currently queued.
    #[must_use]
    pub fn queue_len(&self) -> usize {
        self.queue.borrow().len()
    }

    /// The identity C gets from the allocation's address.
    #[must_use]
    pub fn id(&self) -> u64 {
        self.id
    }

    /// `list_add_tail(&futex->queue, &wait->queue)`.
    pub fn enqueue(self: &Rc<Self>, waiter: Rc<Waiter>) {
        waiter.point_at(self);
        self.queue.borrow_mut().push(waiter);
    }

    /// `list_remove_safe(&wait->queue)`: remove a waiter wherever it sits in
    /// this queue, doing nothing if it is not there.
    pub fn dequeue(&self, waiter: &Waiter) {
        self.queue
            .borrow_mut()
            .retain(|queued| !std::ptr::eq(queued.as_ref(), waiter));
    }

    /// Remove up to `max` waiters from the head of the queue, in queue order.
    ///
    /// This is the loop body shared by `futex_wakelike`'s two passes: a wake
    /// removes the waiters it notifies, a requeue removes the ones it moves.
    fn take_waiters(&self, max: u32) -> Vec<Rc<Waiter>> {
        let mut queue = self.queue.borrow_mut();
        let taken = queue.len().min(max as usize);
        queue.drain(..taken).collect()
    }
}

/// A snapshot of one futex, for the differential comparison and for callers
/// that want to inspect the table.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct FutexEntry {
    /// The guest address the futex is keyed on.
    pub addr: u32,
    /// `futex->refcount`
    pub refcount: u32,
    /// The number of queued waiters.
    pub queue: usize,
}

/// The table lock, held. Every operation that touches the buckets goes through
/// one of these, which is what makes the C's "*_unlocked" pair a type here.
pub struct TableGuard<'a> {
    table: &'a FutexTable,
    guard: Option<LockGuard<'a>>,
}

impl<'a> TableGuard<'a> {
    /// `futex_get_unlocked`: find or create the futex for `(mm, addr)` and take
    /// a reference. The caller must already hold the table lock, which it does
    /// by having this guard.
    #[must_use]
    pub fn get_unlocked(&self, mm: MmId, addr: u32) -> Rc<Futex> {
        self.table.get_unlocked(mm, addr)
    }

    /// `futex_put_unlocked`: drop a reference, freeing the futex — and taking
    /// it out of the table — when the count reaches zero.
    pub fn put_unlocked(&self, futex: &Rc<Futex>) {
        self.table.put_unlocked(futex);
    }

    /// Release the lock. `futex_put` ends with this, and so does a wake that
    /// never touches a second futex.
    pub fn unlock(self) {
        let Self { guard, .. } = self;
        drop(guard);
    }

    /// Hand the raw lock to [`wait_for`], which releases it for the duration of
    /// the wait. The guard comes back through [`TableGuard::resume`].
    #[must_use]
    pub fn into_guard(self) -> LockGuard<'a> {
        self.guard.expect("the table lock is only handed out once")
    }

    /// Take the lock back after a wait, as `futex_wait` does when `wait_for`
    /// returns.
    #[must_use]
    pub fn resume(table: &'a FutexTable, guard: LockGuard<'a>) -> Self {
        Self {
            table,
            guard: Some(guard),
        }
    }
}

impl std::fmt::Debug for TableGuard<'_> {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.debug_struct("TableGuard").finish_non_exhaustive()
    }
}

/// `futex_hash[FUTEX_HASH_SIZE]` plus `futex_lock`.
pub struct FutexTable {
    lock: Lock,
    buckets: RefCell<Vec<Vec<Rc<Futex>>>>,
    next_futex_id: Cell<u64>,
    next_waiter_id: Cell<u64>,
}

impl Default for FutexTable {
    fn default() -> Self {
        Self::new()
    }
}

impl FutexTable {
    /// `init_futex_hash` and the `pthread_key`-free part of the module's setup:
    /// 4096 empty buckets and a lock.
    #[must_use]
    pub fn new() -> Self {
        Self {
            lock: Lock::new(),
            buckets: RefCell::new((0..FUTEX_HASH_SIZE).map(|_| Vec::new()).collect()),
            next_futex_id: Cell::new(0),
            next_waiter_id: Cell::new(0),
        }
    }

    /// Take the table lock — C's `lock(&futex_lock)`.
    #[must_use]
    pub fn lock(&self) -> TableGuard<'_> {
        TableGuard {
            table: self,
            guard: Some(self.lock.lock()),
        }
    }

    /// `futex_get`: find or create the futex for `(mm, addr)`, taking a
    /// reference, and return it with the table lock still held.
    ///
    /// C's malloc-failure branch unlocks and returns `NULL`; a Rust allocation
    /// failure aborts, which is the port's only option and is not reachable in
    /// the reported behavior.
    #[must_use]
    pub fn get(&self, mm: MmId, addr: u32) -> (TableGuard<'_>, Rc<Futex>) {
        let guard = self.lock();
        let futex = guard.get_unlocked(mm, addr);
        (guard, futex)
    }

    /// `futex_put`: drop a reference and release the lock.
    pub fn put(&self, guard: TableGuard<'_>, futex: &Rc<Futex>) {
        guard.put_unlocked(futex);
        guard.unlock();
    }

    fn get_unlocked(&self, mm: MmId, addr: u32) -> Rc<Futex> {
        let key = FutexKey { mm, addr };
        let hash = futex_hash(mm, addr);
        let mut buckets = self.buckets.borrow_mut();
        let bucket = &mut buckets[hash];
        if let Some(existing) = bucket.iter().find(|futex| futex.key == key) {
            existing.refcount.set(existing.refcount.get() + 1);
            return Rc::clone(existing);
        }
        let id = self.next_futex_id.get() + 1;
        self.next_futex_id.set(id);
        let futex = Rc::new(Futex::new(id, key));
        bucket.push(Rc::clone(&futex));
        futex
    }

    fn put_unlocked(&self, futex: &Rc<Futex>) {
        let remaining = futex.refcount.get() - 1;
        futex.refcount.set(remaining);
        if remaining != 0 {
            return;
        }
        // C: assert(list_empty(&futex->queue)) — a freed futex must have no
        // waiters, because each of them holds a reference while it waits.
        assert!(
            futex.queue.borrow().is_empty(),
            "futex {} freed with waiters still queued",
            futex.id()
        );
        let mut buckets = self.buckets.borrow_mut();
        let bucket = &mut buckets[futex_hash(futex.key.mm, futex.key.addr)];
        bucket.retain(|entry| !Rc::ptr_eq(entry, futex));
    }

    /// Every live futex, ordered by address.
    ///
    /// The order is by address rather than by bucket on purpose: the bucket an
    /// address lands in depends on the low bits of the host address of the
    /// address space, so it is not comparable between implementations.
    #[must_use]
    pub fn entries(&self) -> Vec<FutexEntry> {
        let buckets = self.buckets.borrow();
        let mut entries: Vec<FutexEntry> = buckets
            .iter()
            .flatten()
            .map(|futex| FutexEntry {
                addr: futex.addr(),
                refcount: futex.refcount(),
                queue: futex.queue_len(),
            })
            .collect();
        entries.sort_by_key(|entry| entry.addr);
        entries
    }

    /// The ids of the waiters queued at `(mm, addr)`, in queue order.
    ///
    /// This is the queue half of the same inspection surface; the order is the
    /// point of it, because a wake takes waiters from the head. Nothing is
    /// created or referenced: an address with no live futex reports an empty
    /// queue.
    #[must_use]
    pub fn queue_of(&self, mm: MmId, addr: u32) -> Vec<u64> {
        let key = FutexKey { mm, addr };
        let buckets = self.buckets.borrow();
        buckets[futex_hash(mm, addr)]
            .iter()
            .find(|futex| futex.key == key)
            .map(|futex| {
                futex
                    .queue
                    .borrow()
                    .iter()
                    .map(|waiter| waiter.id())
                    .collect()
            })
            .unwrap_or_default()
    }

    /// `futex_wake`: wake up to `val` waiters.
    pub fn futex_wake(&self, mm: MmId, uaddr: u32, val: u32) -> i32 {
        self.futex_wakelike(FUTEX_WAKE_, mm, uaddr, val, 0, 0)
    }

    /// `futex_wakelike`: wake up to `wake_max` waiters, and for
    /// `FUTEX_REQUEUE_` move up to `requeue_max` more to `requeue_addr`.
    ///
    /// The return value counts both, which is what C's `woken += requeued`
    /// does.
    pub fn futex_wakelike(
        &self,
        op: u32,
        mm: MmId,
        uaddr: u32,
        wake_max: u32,
        requeue_max: u32,
        requeue_addr: u32,
    ) -> i32 {
        let (guard, futex) = self.get(mm, uaddr);
        let mut woken = 0u32;
        for waiter in futex.take_waiters(wake_max) {
            notify(waiter.cond());
            woken += 1;
        }

        if op == FUTEX_REQUEUE_ {
            let futex2 = guard.get_unlocked(mm, requeue_addr);
            let mut requeued = 0u32;
            for waiter in futex.take_waiters(requeue_max) {
                // C: // sketchy as hell — the waiter's reference moves with it,
                // so the target gains one and the source loses one.
                waiter.point_at(&futex2);
                futex2.enqueue(waiter);
                assert!(
                    futex.refcount.get() > 1,
                    "the requeue must keep its own reference to the source"
                );
                futex.refcount.set(futex.refcount.get() - 1);
                futex2.refcount.set(futex2.refcount.get() + 1);
                requeued += 1;
            }
            guard.put_unlocked(&futex2);
            woken += requeued;
        }

        self.put(guard, &futex);
        woken as i32
    }

    /// `futex_wait`: sleep until someone wakes this address, or the word stops
    /// matching, or the timeout expires.
    ///
    /// The arguments beyond C's three are the address space the word lives in
    /// and the platform the wait parks on, both injected rather than global.
    #[allow(clippy::too_many_arguments)]
    pub fn futex_wait(
        &self,
        mem: &mut Mem,
        mm: MmId,
        uaddr: u32,
        val: u32,
        timeout: Option<Timespec>,
        host: &dyn SyncHost,
        parker: &impl Parker,
    ) -> i32 {
        let guard = self.lock();
        let futex = guard.get_unlocked(mm, uaddr);
        let Some(loaded) = futex_load(mem, uaddr) else {
            self.put(guard, &futex);
            return EFAULT;
        };
        if loaded != val {
            self.put(guard, &futex);
            return EAGAIN;
        }

        let id = self.next_waiter_id.get() + 1;
        self.next_waiter_id.set(id);
        let waiter = Rc::new(Waiter::new(id, &futex));
        futex.enqueue(Rc::clone(&waiter));

        // The wait releases the table lock, so a wake from another task can
        // reach the queue.
        let (err, raw) = wait_for(waiter.cond(), guard.into_guard(), timeout, host, parker);
        let guard = TableGuard::resume(self, raw);
        // `wait.futex` may point at a different futex now: a requeue moved this
        // waiter and its reference to the new queue.
        let futex = waiter.futex();
        waiter.dequeue();
        match futex {
            Some(futex) => self.put(guard, &futex),
            // Unreachable while the reference counting is correct: the waiter
            // holds a reference, so the futex cannot have been freed. C would
            // use freed memory here.
            None => guard.unlock(),
        }
        err
    }

    /// `sys_futex`.
    ///
    /// `_val3` is the sixth syscall argument, which C only ever prints in a
    /// `STRACE`.
    #[allow(clippy::too_many_arguments)]
    pub fn sys_futex(
        &self,
        task: &mut Task,
        uaddr: u32,
        op: u32,
        val: u32,
        timeout_or_val2: u32,
        uaddr2: u32,
        _val3: u32,
        host: &dyn SyncHost,
        parker: &impl Parker,
    ) -> i32 {
        // C: if (!(op & FUTEX_PRIVATE_FLAG_)) STRACE("!FUTEX_PRIVATE "); the
        // tracer is not ported, so the flag only matters through the mask.
        let mut timeout = None;
        if (op & FUTEX_CMD_MASK_) == FUTEX_WAIT_ && timeout_or_val2 != 0 {
            let mut bytes = [0u8; 8];
            if task.user_read(timeout_or_val2, &mut bytes).is_err() {
                return EFAULT;
            }
            // `struct timespec_` is two `dword_t`s, and C's assignment to the
            // host's `long tv_sec` zero-extends them.
            let sec = u32::from_le_bytes(bytes[0..4].try_into().unwrap());
            let nsec = u32::from_le_bytes(bytes[4..8].try_into().unwrap());
            timeout = Some(Timespec::new(i64::from(sec), i64::from(nsec)));
        }

        let Some(mm) = mm_identity(task) else {
            return EFAULT;
        };
        match op & FUTEX_CMD_MASK_ {
            FUTEX_WAIT_ => {
                let Some(mut mm_ref) = task.mm_mut() else {
                    return EFAULT;
                };
                self.futex_wait(&mut mm_ref.mem, mm, uaddr, val, timeout, host, parker)
            }
            FUTEX_WAKE_ => self.futex_wakelike(FUTEX_WAKE_, mm, uaddr, val, 0, 0),
            FUTEX_REQUEUE_ => {
                self.futex_wakelike(FUTEX_REQUEUE_, mm, uaddr, val, timeout_or_val2, uaddr2)
            }
            // C: FIXME("unsupported futex operation %d", op); return _ENOSYS;
            _ => ENOSYS,
        }
    }
}

/// `sys_set_robust_list`: the length is the only thing C looks at.
pub fn sys_set_robust_list(task: &mut Task, robust_list: u32, len: u32) -> i32 {
    if len != ROBUST_LIST_HEAD_SIZE {
        return EINVAL;
    }
    task.robust_list = robust_list;
    0
}

/// `sys_get_robust_list`: report the caller's list to the caller.
///
/// C looks `pid` up under `pids_lock` and compares the result against
/// `current`, so a pid that does not exist is `EPERM` just like another task's
/// pid — there is no separate "no such process".
pub fn sys_get_robust_list(
    tasks: &mut TaskTable,
    pid: Pid,
    robust_list_ptr: u32,
    len_ptr: u32,
) -> i32 {
    if tasks.current_pid() != Some(pid) {
        return EPERM;
    }
    let Some(task) = tasks.current_mut() else {
        return EPERM;
    };
    if task
        .user_write(robust_list_ptr, &task.robust_list.to_le_bytes())
        .is_err()
    {
        return EFAULT;
    }
    if task
        .user_write(len_ptr, &ROBUST_LIST_HEAD_SIZE.to_le_bytes())
        .is_err()
    {
        return EFAULT;
    }
    0
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::memory::P_RWX;
    use crate::sync::{NoTaskHost, ParkResult, ThreadParker};
    use std::sync::MutexGuard;

    fn mapped_mem(dword: u32) -> Mem {
        let mut mem = Mem::new();
        mem.map_nothing(0x100, 1, P_RWX);
        let ptr = mem.ptr(0x100000, MemType::Write).expect("mapped");
        // SAFETY: writing four bytes inside the page just mapped.
        unsafe { ptr.cast::<u32>().write_unaligned(dword) };
        mem
    }

    #[test]
    fn the_hash_is_cs_expression() {
        // (addr ^ mm) % 4096, both operands taken modulo the table size after
        // the XOR — with the bucket count a power of two, it is the low bits.
        assert_eq!(futex_hash(0, 0x100000), 0);
        assert_eq!(futex_hash(0x720, 0x100000), 0x720);
        assert_eq!(futex_hash(0xfff, 0), 0xfff);
        assert_eq!(futex_hash(0xfff, 0xfff), 0);
        assert_eq!(futex_hash(0x1_0000, 0x1_0000), 0);
        // Only the low 12 bits of either operand can matter.
        assert_eq!(futex_hash(0x1234, 0xabc), futex_hash(0x1234, 0xabc));
        assert_eq!(futex_hash(0x1234, 0x1abc), futex_hash(0x1234, 0xabc));
        // The hash of an address in an address space whose token has the same
        // low bits is the same bucket, which is why the token is opaque.
        assert_eq!(
            futex_hash(0x2000 + 0xabc, 0x1234),
            futex_hash(0xabc, 0x1234)
        );
        assert_eq!(futex_hash(0x1234, 0xabc), 2184);
        assert!(futex_hash(0x123, 0x456) < FUTEX_HASH_SIZE);
    }

    #[test]
    fn a_futex_is_shared_per_address_space_and_address() {
        let table = FutexTable::new();
        let (finish, first) = table.get(1, 0x100000);
        let second = finish.get_unlocked(1, 0x100000);
        assert!(Rc::ptr_eq(&first, &second), "same key, same object");
        assert_eq!(first.refcount(), 2);
        let other_addr = finish.get_unlocked(1, 0x100004);
        assert!(!Rc::ptr_eq(&first, &other_addr));
        let other_mm = finish.get_unlocked(2, 0x100000);
        assert!(
            !Rc::ptr_eq(&first, &other_mm),
            "the same address in another address space is another futex"
        );
        let third = finish.get_unlocked(1, 0x100000);
        assert_eq!(third.refcount(), 3);
        assert_eq!(table.entries().len(), 3);

        // The other two keys go away at zero references; the shared one stays.
        finish.put_unlocked(&other_addr);
        finish.put_unlocked(&other_mm);
        finish.put_unlocked(&third);
        assert_eq!(first.refcount(), 2);
        finish.put_unlocked(&second);
        assert_eq!(first.refcount(), 1);
        finish.put_unlocked(&first);
        finish.unlock();
        assert!(
            table.entries().is_empty(),
            "a futex at zero references is gone"
        );
    }

    #[test]
    fn a_zero_value_word_is_eagain_and_an_unmapped_word_is_efault() {
        let table = FutexTable::new();
        let host = NoTaskHost::new();
        let mut mem = mapped_mem(7);
        let parker = ThreadParker;

        assert_eq!(
            table.futex_wait(&mut mem, 0, 0x100000, 8, None, &host, &parker),
            EAGAIN
        );
        assert_eq!(
            table.futex_wait(&mut mem, 0, 0x200000, 7, None, &host, &parker),
            EFAULT
        );
        assert!(table.entries().is_empty());
    }

    /// A parker that answers immediately, so a wait can be driven without
    /// another thread.
    struct ScriptedParker(Cell<i32>);

    impl Parker for ScriptedParker {
        fn park<'g, T>(
            &self,
            _cond: &Cond,
            guard: MutexGuard<'g, T>,
            _now: Timespec,
            _deadline: Option<Timespec>,
        ) -> (ParkResult, MutexGuard<'g, T>) {
            let result = match self.0.get() {
                0 => ParkResult::Woken,
                110 => ParkResult::TimedOut,
                other => ParkResult::Failed(other),
            };
            (result, guard)
        }
    }

    #[test]
    fn a_wait_publishes_and_removes_itself() {
        let table = FutexTable::new();
        let host = NoTaskHost::new();
        let mut mem = mapped_mem(7);
        let parker = ScriptedParker(Cell::new(0));
        assert_eq!(
            table.futex_wait(&mut mem, 0, 0x100000, 7, None, &host, &parker),
            0
        );
        // A timeout maps to ETIMEDOUT, and a non-timeout failure to success,
        // exactly as `wait_for` specifies.
        assert_eq!(
            table.futex_wait(
                &mut mem,
                0,
                0x100000,
                7,
                None,
                &host,
                &ScriptedParker(Cell::new(110))
            ),
            crate::sync::ETIMEDOUT
        );
        assert_eq!(
            table.futex_wait(
                &mut mem,
                0,
                0x100000,
                7,
                None,
                &host,
                &ScriptedParker(Cell::new(22))
            ),
            0
        );
        assert!(table.entries().is_empty(), "no wait leaks its futex");
    }

    /// Build a waiter the way `futex_wait` does: take the reference first,
    /// then queue the waiter with it. The reference is the waiter's, not the
    /// harness's, and it moves with the waiter if it is requeued.
    fn build_waiter(table: &FutexTable, mm: MmId, addr: u32) -> (Rc<Waiter>, Rc<Futex>) {
        let guard = table.lock();
        let futex = guard.get_unlocked(mm, addr);
        let waiter = Rc::new(Waiter::new(0, &futex));
        futex.enqueue(Rc::clone(&waiter));
        guard.unlock();
        (waiter, futex)
    }

    /// What a returning waiter does: leave the queue and put its reference to
    /// whichever futex it is queued on now.
    fn release_waiter(table: &FutexTable, waiter: &Rc<Waiter>) {
        let guard = table.lock();
        waiter.dequeue();
        if let Some(futex) = waiter.futex() {
            guard.put_unlocked(&futex);
        }
        guard.unlock();
    }

    #[test]
    fn wake_counts_and_requeue_moves_references() {
        let table = FutexTable::new();

        // Three waiters on A, one on B — the corpus's "three-one" state, with
        // no extra references: each waiter holds its own.
        let waiters: Vec<_> = (0..3).map(|_| build_waiter(&table, 0, 0x100000)).collect();
        let (target_waiter, _) = build_waiter(&table, 0, 0x140000);
        assert_eq!(
            table.entries(),
            vec![
                FutexEntry {
                    addr: 0x100000,
                    refcount: 3,
                    queue: 3
                },
                FutexEntry {
                    addr: 0x140000,
                    refcount: 1,
                    queue: 1
                },
            ]
        );

        assert_eq!(
            table.futex_wake(0, 0x100000, 0),
            0,
            "a zero wake wakes nobody"
        );
        assert_eq!(table.futex_wake(0, 0x100000, 2), 2);
        assert_eq!(
            table.entries(),
            vec![
                FutexEntry {
                    addr: 0x100000,
                    refcount: 3,
                    queue: 1
                },
                FutexEntry {
                    addr: 0x140000,
                    refcount: 1,
                    queue: 1
                },
            ],
            "a woken waiter keeps its reference until it returns"
        );

        release_waiter(&table, &waiters[0].0);
        release_waiter(&table, &waiters[1].0);
        assert_eq!(table.entries()[0].refcount, 1);
        assert_eq!(table.futex_wake(0, 0x100000, 5), 1);
        release_waiter(&table, &waiters[2].0);
        assert_eq!(
            table.entries(),
            vec![FutexEntry {
                addr: 0x140000,
                refcount: 1,
                queue: 1
            }],
            "the emptied source left the table"
        );

        // A fresh queue for the requeue: one woken, two moved.
        let moved: Vec<_> = (0..3).map(|_| build_waiter(&table, 0, 0x100000)).collect();
        assert_eq!(table.entries()[0].refcount, 3, "requeue-before");
        assert_eq!(
            table.futex_wakelike(FUTEX_REQUEUE_, 0, 0x100000, 1, 2, 0x140000),
            3
        );
        assert_eq!(
            table.entries(),
            vec![
                FutexEntry {
                    addr: 0x100000,
                    refcount: 1,
                    queue: 0
                },
                FutexEntry {
                    addr: 0x140000,
                    refcount: 3,
                    queue: 3
                },
            ],
            "the moved waiters left the source queue and took their references"
        );
        assert_eq!(moved[1].0.addr(), 0x140000, "a moved waiter is re-pointed");
        assert_eq!(moved[2].0.addr(), 0x140000);
        assert_eq!(moved[0].0.addr(), 0x100000, "the woken one is not");

        // Everything returns, and the table drains.
        for (waiter, _) in &moved {
            release_waiter(&table, waiter);
        }
        release_waiter(&table, &target_waiter);
        assert!(table.entries().is_empty());
    }

    #[test]
    fn robust_list_length_and_ownership_rules() {
        let mut tasks = TaskTable::new();
        let pid = tasks.create_task(None).expect("a task");
        tasks.set_current(pid).expect("current");
        let task = tasks.task_mut(pid).expect("the task");

        assert_eq!(sys_set_robust_list(task, 0x1000, 8), EINVAL);
        assert_eq!(sys_set_robust_list(task, 0x1000, ROBUST_LIST_HEAD_SIZE), 0);
        assert_eq!(task.robust_list, 0x1000);
        // The address is stored, never validated: a null one is accepted.
        assert_eq!(sys_set_robust_list(task, 0, ROBUST_LIST_HEAD_SIZE), 0);
        assert_eq!(task.robust_list, 0);

        // Another task, and a pid that does not exist, are both EPERM.
        let other = tasks.create_task(None).expect("a second task");
        assert_eq!(sys_get_robust_list(&mut tasks, other, 0x100, 0x104), EPERM);
        assert_eq!(sys_get_robust_list(&mut tasks, 9999, 0x100, 0x104), EPERM);
    }
}
