# `ish-emu` — a Rust port of the iSH emulator core

A line-by-line Rust translation of the host-independent core of
[iSH](https://github.com/ish-app/ish), the userspace x86 Linux emulator for iOS.

iSH's core is the code under `emu/`: the CPU state, the MMU/TLB, the x86
decoder, and the software FPU. The C original is roughly 4,400 lines in `emu/`
plus the `asbestos` execution engine and the `kernel/` syscall layer. This crate
ports it module by module, and **every ported module is checked bit-for-bit
against the compiled C original**, not just against hand-written expectations.

## Status

| Rust module  | C original            | C lines | Status |
|--------------|-----------------------|---------|--------|
| `float80.rs` | `emu/float80.{h,c}`   | 652     | **done** — 125,612 results verified bit-exact against the C reference |
| `cpu.rs`     | `emu/cpu.h`           | 235     | **done** — register file, lazy EFLAGS, `fsw`/`fcw`, flag macros, `collapse_flags`/`expand_flags` |
| `fpu.rs`     | `emu/fpu.{h,c}`       | 506     | **done** — all 78 x87 operations, whole `cpu_state` compared after each one |
| `mmu.rs`     | `emu/mmu.h`           | 40      | **done** — page arithmetic, `MEM_*` types, the `mmu_ops` translate interface |
| `tlb.rs`     | `emu/tlb.{h,c}`       | 144     | **done** — the 1024-entry software TLB; 172,312 state words verified against the C |
| `vec.rs`     | `emu/vec.{h,c}` + `mmx.c` | 997  | **done** — all 166 SSE/MMX operations, 9,794 results verified against the C |
| `modrm.rs`   | `emu/modrm.h`         | 106     | **done** — ModRM/SIB decoder; 1,283 decodings (10,264 fields) verified against the C |
| `cpuid.rs`   | `emu/cpuid.h`         | 29      | **done** — `do_cpuid` leaves 0/1 and the `CPUID_EDX_*` bits |
| `interrupt.rs` | `emu/interrupt.h`   | 15      | **done** — the 13 `INT_*` vector numbers |
| `decode.rs`  | `emu/decode.h`        | 1,416   | **dispatch ported** — 8 opcode maps × 2 operand sizes; 140,289 events over 37,891 decodings verified against the C. The opcode table is generated from the header |
| `memory.rs`  | `kernel/memory.{h,c}` | 346     | **done** — the guest address space: two-level page table, `pt_map`/`pt_unmap`/`pt_set_flags`/`pt_copy_on_write`/`mem_ptr`/`pt_find_hole`; 428 operations and 395 page records verified against the C |
| `mmap.rs`    | `kernel/mmap.c` + `mm.h` | 243   | **done** — `mmap2`, the old `mmap`, `munmap`, `mremap`, `mprotect`, `brk`, the no-op syscalls and `mm_copy`; 51 operations and the full page table after each verified against the C |
| `errno.rs`   | `kernel/errno.{h,c}`  | 105     | **done** — host→guest errno translation; 4,216 `err_map` inputs and all 10 `errno_map` probes verified against the C, table generated from the host headers |
| `user.rs`    | `kernel/user.c`       | 97      | **done** — the guest↔kernel byte copies (`user_read`/`user_write`/`user_write_task_ptrace`/`user_read_string`/`user_write_string`); 52 operations, 77 guest pages and 192,937 bytes verified against the C |
| `resource.rs` | `kernel/resource.{h,c}` | 265   | **done** — `rlimit`/`rusage` guest ABIs, resource-limit rules, affinity bitmap, and scheduler/priority compatibility calls; 51 deterministic C operations and 52 full state snapshots verified against unmodified C |
| `random.rs`  | `kernel/random.{h,c}` | 35      | **done** — bounded `getrandom`, host-failure mapping, output fault ordering, and explicit iOS/Linux entropy adapter; 9 deterministic C operations verified byte-for-byte |
| `uname.rs`   | `kernel/uname.c`      | 81      | **done** — guest `uname`/`sysinfo` layouts, C string/truncation behavior, and explicit hostname/uptime/memory host data; 10 deterministic C operations verified byte-for-byte |
| `task.rs`    | `kernel/task.{h,c}`   | 346     | **foundation ported** — PID table, parent/child links, task creation/destruction, mm attachment, task credentials/names, thread-group topology, zombie visibility and explicit current-task selection |
| `group.rs`   | `kernel/group.c`      | 131     | **done** — `setpgid`/`getpgid`, `setsid`/`getsid`, session and process-group membership rules |
| `getset.rs`  | `kernel/getset.c`     | 202     | **done** — PID/UID/GID getters and setters, supplementary groups, capability stubs and personality |
| `tls.rs`     | `kernel/tls.c`        | 42      | **done** — i386 `user_desc`, `set_thread_area` and `set_tid_address` |
| `misc.rs`    | `kernel/misc.c`       | 59      | **done** — `prctl`, `arch_prctl` and the host-safe reboot policy |

`cpu.rs` models the parts of `struct cpu_state` that the ported code touches.
Three fields are not there yet because nothing uses them: `struct mmu *mmu` and
`bool *poked_ptr` (pointers, which need `mmu.rs`) and `long cycle` (engine
bookkeeping).

`f80_rem` is declared in `emu/float80.h` but has no definition anywhere in the
iSH tree, so there is nothing to port for it.

## Verification

```console
$ cargo test --tests
running 148 tests  (unit tests in src/)
running 3 tests    (tests/decode_differential.rs)  -> 140,289 decoder events
running 3 tests    (tests/errno_differential.rs)   ->   4,216 err_map inputs
running 2 tests    (tests/mmap_differential.rs)    ->      51 mmap operations
running 2 tests    (tests/differential.rs)         -> 125,612 float80 results
running 4 tests    (tests/fpu_differential.rs)     -> 502,552 cpu_state words
running 2 tests    (tests/memory_differential.rs)  ->     428 page-table operations
running 1 test     (tests/user_differential.rs)    ->     192,937 guest+host bytes
running 3 tests    (tests/modrm_differential.rs)   ->  10,264 decode fields
running 2 tests    (tests/tlb_differential.rs)     -> 172,312 tlb state words
running 2 tests    (tests/vec_differential.rs)     ->   9,794 vec/mmx results
running 1 test     (tests/resource_differential.rs)->      51 resource operations + 52 full state snapshots
running 1 test     (tests/random_differential.rs)  ->       9 getrandom operations
running 1 test     (tests/uname_differential.rs)   ->      10 uname/sysinfo operations
test result: ok. 211 passed
$ cargo clippy --all-targets                   # clean, no warnings
```

### float80

The important test is `tests/differential.rs`. It replays
`tests/fixtures/f80_reference.txt` — 125,612 results produced by
`tools/f80-dump.c` compiled against the **unmodified** iSH `emu/float80.c` — and
requires the significand *and* the packed sign/exponent word to match exactly,
in all four x87 rounding modes, over a corpus of 61 operands (integers, doubles,
denormals, ±0, ±inf, NaN, unormals, the largest finite, the smallest normal and
denormal, and pseudo-random encodings) covering `add`, `sub`, `mul`, `div`,
`mod`, `lt`, `eq`, `uncomparable`, `neg`, `abs`, `round`, `sqrt`, `log2`,
`scale`, `xtract`, `from_int`, `to_int`, `from_double`, `to_double` and the five
predicates.

Regenerate the fixture from any iSH checkout:

```console
$ ./tools/gen_f80_reference.sh /path/to/ish
wrote tests/fixtures/f80_reference.txt (125370 lines)
```

### cpu + fpu

`tests/fpu_differential.rs` does the same job one level up.
`tools/fpu-dump.c` compiles against the unmodified `emu/fpu.c`, `emu/cpu.h` and
`emu/float80.c`, then runs **every** `fpu_*` entry point — 78 operations: stack
manipulation, the eight conditional moves, loads and stores in 16/32/64/80-bit
integer and 32/64/80-bit float forms, all six arithmetic ops in
register-to-register and memory-operand forms, the compares, the transcendentals
and the environment save/restore — against 24 pseudo-random `cpu_state`s. After
each operation it dumps the entire state: all eight x87 registers, `top`, `fsw`,
`fcw`, every flag byte and bit, and the lazy-flag words. The Rust port has to
reproduce all 32 words of every dump, which is 502,552 comparisons. The same
fixture checks the `ZF`/`SF`/`CF`/`OF`/`PF`/`AF` macros from `emu/cpu.h` plus
`collapse_flags` and `expand_flags`.

```console
$ ./tools/gen_fpu_reference.sh /path/to/ish
wrote tests/fixtures/fpu_reference.txt (15718 lines)
```

The differential test was mutation-checked: swapping `f80_sub` for `f80_add`
inside `fpu_sub`, and swapping C0 with C3 in `fpu_xam`, each made it fail (119
and 7 mismatched words respectively).

`sin`, `cos`, `atan2` and `pow` are computed in double precision by iSH itself
("the ancient chinese art of chi ting"); the port calls the same system libm
through Rust's `f64` methods and the fixture confirms the results are identical
bit for bit.

### mmu + tlb

`tests/tlb_differential.rs` is the same idea applied to the memory path.
`tools/tlb-dump.c` compiles the **unmodified** `emu/tlb.c` and runs a
56-step access sequence against a fake `mmu_ops` backend (16 aliased pages with
read-only and unmapped masks). After every step it dumps the observable result
*and* the whole `struct tlb`: all 1024 entries with their `page`,
`page_if_writable` and presence, plus `dirty_page`, `segfault_addr`,
`mem_changes` and a running count of `translate` calls.

That call count is what makes this a test of the *cache* and not just of the
page walker — a port that translated on every access would still return the
right bytes and would fail here at once. The sequence covers cache hits and
misses, cross-page reads and writes in both directions, a fault on the *second*
page of a cross-page access, read-only and unmapped faults, two guest addresses
4 MiB apart that collide in one TLB slot and evict each other, `tlb_refresh`
as both a flush and a no-op, an access on the last mapped page (whose next page
aliases back to the first one in the fake backend), and an access larger than a
page (where the C's unsigned `PAGE_SIZE - size` wraps and takes the fast path
instead of crossing pages).

Raw host pointers are never compared — they differ between processes — so the
fixture records guest-observable state plus an FNV-1a hash of the backing
store, which catches any stray write.

```console
$ ISH_SRC=/path/to/ish ./tools/gen_tlb_reference.sh
wrote tests/fixtures/tlb_reference.txt: 58 lines, 56 steps
```

The test was mutation-checked: 17 targeted breakages of `tlb.rs` — dropping the
high-bit xor from `TLB_INDEX`, making `read_ptr`/`write_ptr` always miss,
marking read entries writable, not resyncing `mem_changes` on flush, not
flushing on refresh, using a saturating instead of a wrapping `PAGE_SIZE -
size`, not setting `dirty_page` on a write hit, splitting a cross-page copy one
byte off, translating the wrong page, and more — were each caught. One ordering
is *not* covered and is documented as such in the test header: `tlb_handle_miss`
translates first and checks `mmu->changes` afterwards, which only matters for a
backend that remaps from inside `translate` (the real `kernel/memory.c` does),
and the fake backend cannot reach `mmu.changes` to reproduce it.

### vec + mmx

`tests/vec_differential.rs` covers the 166 functions `emu/vec.h` declares —
`nm` on the compiled `emu/vec.c` + `emu/mmx.c` reports exactly those 166, with
no orphans on either side. `tools/vec-dump.c` calls all of them over 59 operand
cases; both the C op table and the Rust dispatch match are generated from the
header by `tools/gen_vec_ops.py`, so a new operation cannot be quietly skipped
and an operation missing on the Rust side panics on its name.

Each record dumps the destination register, the source register, a side buffer
for the operations whose destination is a bare scalar, and nine `cpu_state`
flag fields — the last being what proves only the two `ucomi` operations touch
the CPU. The cases are chosen to hit the corners: shift counts just past every
element width, byte pairs summing to exactly 255 and differences landing on 0
and −1 (where the saturating adds and subs change branch), all eight
float-compare predicates, shuffle encodings, insert/extract indices, and
NaN/±inf/denormal/overflow operands for the conversions. The operand values
themselves travel in the fixture header, so the Rust side never duplicates the
C driver's tables.

```console
$ ISH_SRC=/path/to/ish ./tools/gen_vec_reference.sh
wrote tests/fixtures/vec_reference.txt: 9855 lines, 9794 records, 166 operations, 59 cases
```

Mutation-checked: 29 targeted breakages were each caught, from the `dq`
operations covering only the low qword, to a `punpcklbw` iterating upwards, to
a `cvtt` saturating the way Rust's `as` does instead of the way `CVTTSD2SI`
does. One mutant is provably undetectable — `sb > 0xfe` in `paddusb` is
equivalent to `sb > 0xff`, since both store `0xff` when `sb == 0xff`.

### modrm

`tests/modrm_differential.rs` compares 1,283 decodings field by field — eight
fields each (result, final instruction pointer, operand type, `reg`, base,
displacement, `index`, `shift`), 10,264 comparisons. The corpus is exhaustive
where the encoding is: all 256 ModRM bytes, and all 256 SIB bytes under each of
the four `mod` values, including `mod=11` where `rm=100` is an ordinary
register rather than a SIB escape. Three more cases place the operand against an
unmapped page, which pins two behaviours that are easy to get wrong: the decoder
reports the fault by returning false, and the instruction pointer is still left
past the bytes it consumed, because the C's `READ` macro advances `*ip` *before*
it reads.

The fixture describes its own memory image — the backing-store fill, the
unmapped page, every byte the C driver wrote and each starting ip — so the Rust
side rebuilds the same image instead of duplicating the generator's tables.

```console
$ ISH_SRC=/path/to/ish ./tools/gen_modrm_reference.sh
wrote tests/fixtures/modrm_reference.txt: 2568 lines, 1283 cases
```

Mutation-checked: 9 of 10 targeted breakages were caught, including a SIB scale
of 8 folded back to 1, `[ebp]` with no displacement losing its disp32 meaning,
a disp8 that stops being sign-extended, an index of `esp` treated as a real
index register, and a `READ` that only advances the ip on success. The survivor
is `mode != MODE_REG` in the SIB arm, which is unreachable-by-construction — the
`mode == MODE_REG` arm above it already caught that case.

The scale-8 case is worth spelling out, because the C gives no hint that it
matters: `struct modrm`'s `shift` is `enum { times_1, times_2, times_4 }`, but
`modrm->shift = MOD(sib_byte)` stores the raw two bits, so a scale of 8 arrives
as the unnamed `3`. It is live: `asbestos/gen.c` selects an addressing gadget
with `modrm->index * 4 + modrm->shift`, and that table is generated over
`.irp times, 1,2,4,8`. Folding 3 to 0 would silently decode `[eax + ecx*8]` as
`[eax + ecx]`.

### cpuid and interrupt

`emu/cpuid.h` and `emu/interrupt.h` are small enough that their unit tests are
the check: leaf 0 returns `GenuineIntel` in the ebx/edx/ecx order Intel
specifies, leaf 1 and the `default:` case both return
`fpu | cmov | mmx | sse2` (bit 25, plain SSE, is deliberately absent), and the
13 `INT_*` vectors are the constants `kernel/` raises.

### decode

`emu/decode.h` is the one module here that is not a library. It is a 1,416-line
template that `asbestos/gen.c` includes twice — once with `OP_SIZE` 32, once
with 16 — supplying ~150 macros that turn each decoded instruction into gadget
emissions. So it was split by what each part is:

* the **dispatch** — prefix handling, the eight opcode maps, the ModRM and
  immediate reads, the GRP groups, the x87 split, where an instruction ends a
  block — is translated by hand into `src/decode.rs`;
* the **semantics** — what `ADD` or `CVTSS2SD` does — live in the gadget
  backends and are *not* ported. The decoder's job ends at naming the operation
  (`Op`, one variant per macro the header invokes) and its operands, which stay
  as the C's own tokens so nothing is lost or invented.

`tools/decode-dump.c` supplies the same ~150 macros as recorders — each one
stringifies its arguments instead of generating anything — and includes the
header unmodified, both instances. It compiles `-Wall -Wextra` clean, which is
itself a check that the macro contract was read correctly.

The per-opcode step lists in `src/decode_table.rs` are generated from that
reference run rather than transcribed. What makes that sound rather than
circular is a property the generator *proves*: that once the ModRM byte is
consumed, the dispatch depends on it only through a 72-value class function (8
memory classes + 64 register). It checks this by decoding all 256 ModRM bytes
for every opcode of every map in both operand sizes — **1,042,066 decodes** —
and fails if any class disagrees with itself.

```console
$ ISH_SRC=/path/to/ish ./tools/gen_decode_reference.sh
wrote tests/fixtures/decode_reference.txt: 216076 lines, 37891 cases
# class_invariant 1042066 decodes, 14 opcodes excepted: base:0f base:2e ...
$ python3 ./tools/gen_decode_table.py
wrote src/decode_table.rs: 4068 opcodes (114 class-dependent), 789 distinct step lists
```

That check falsified the obvious 16-class version of the rule on its first run.
The x87 opcodes fall through to

```c
switch (insn << 8 | modrm.opcode << 4 | modrm.rm_opcode)
```

in their register form, so they read the `rm` field as well — `d9 e0` is `fchs`
and `d9 e1` is `fabs`, same opcode, same reg field. Hence 8 × 8 register
classes. A second guard, asserting that no opcode in a narrowly-swept map
depends on the class, caught a stale map index in the prefix table.

The 14 excepted opcodes are printed into the fixture rather than hidden in the
generator: they are exactly the `goto restart` / `goto lockrestart` /
operand-size-switch / nested-`0x0f` sites, and the decoder drives them itself.
A unit test checks both halves of that coupling — every other opcode has a
table entry, every prefix opcode is intercepted — which is what makes the
lookup's empty-slot arm unreachable.

`tests/decode_differential.rs` replays all 37,891 corpus cases through the Rust
decoder: **140,289 events** match, in order, plus the result code and the final
instruction pointer. A third test drives the same instructions with ModRM bytes
the corpus never contained — `mod=01` and `mod=10` forms, SIB, disp32-with-no-base
— and requires the same operation out, which is what tests the class abstraction
rather than the corpus.

Mutation-checked: 14 of 15 targeted breakages of `src/decode.rs` were caught,
including a `0x66` that stops switching operand size, the locked `0x66` losing
its `RESTORE_IP` rewind, a class function that forgets the `rm` field, an
immediate read taking its bit count as a byte count, and a read that only
advances the instruction pointer when it succeeds. The survivor edits the
lookup's empty-slot arm, which the coupling test above proves unreachable.

One upstream quirk is reproduced rather than fixed. `lock 0x66` in 32-bit mode
does `RESTORE_IP` — rewinds to the start of the instruction — and hands it to
the 16-bit decoder, prefixes and all. The C comments this "I didn't think this
through"; the port does the same thing, and the corpus covers it.

### What the differential test caught

Two genuine translation bugs, both caused by flattening the C union/bitfield
`signExp` (15-bit `exp` + 1-bit `sign`) into a single Rust `u16`:

* `Float80::round` did `f.sign_exp = EXP_DENORMAL`, clearing the sign bit. The C
  statement `f.exp = EXP_DENORMAL` only writes the exponent bitfield. Every
  exponent write now goes through `Float80::set_exp`, which cannot touch the
  sign.
* `f80_div` returned `NaN` early for `0/0`, skipping the trailing
  `f.sign = a.sign ^ b.sign` that the C original applies, so `-0.0 / +0.0`
  came out `+NaN` instead of `-NaN`.

The vec/mmx harness found two more, both from flattening the C's macro
machinery by hand:

* The `dq` operations (`andps`/`orps`/`xorps`) work on one element spanning the
  whole 128-bit register, because the C's `union vec` has a `__uint128_t dqw`
  member. A first pass treated them as 64-bit and silently left the upper half
  of the register untouched.
* `_VEC_CVT` stores `INT32_MIN` *in the destination type*, so `cvttsd2ss` of a
  NaN produces the float −2³¹ (`0xcf000000`) rather than a float NaN. `gcc`
  compiles it exactly that way; a straightforward `*dst = *src as f32` does not.

### Upstream corner cases, pinned by tests

`tests/differential.rs::upstream_corner_cases_are_reproduced` locks in three
places where the C original is itself awkward, so a future cleanup cannot drift:

* `smallest_normal - smallest_denormal` produces an encoding whose exponent
  field says "denormal" while the integer bit is still set. iSH's *debug* build
  trips `assert(f80_is_supported(f))` at the end of `f80_add` on it; the
  arithmetic result is well defined and the port produces the identical bits
  without panicking. This is why the reference is built with `-DNDEBUG` (how
  iSH ships).
* `f80_log2(+inf)` never terminates, and neither does `f80_log2` of anything
  under `round_up` (the halving `bit` sequence rounds the smallest denormal back
  up to itself). The port keeps this behaviour; the fixture simply does not feed
  it those inputs.
* `f80_scale(0, n)` for large `n` executes `signif <<= 128` on a zero
  significand. gcc/x86 masks the shift count, so nothing happens and the
  exponent is left at `exp - 128`. The port uses `wrapping_shl` to reproduce it
  exactly rather than "fixing" it.

### memory

`kernel/memory.c` is the first module ported from outside `emu/`: it is the
guest address space that `mmu.rs` and `tlb.rs` sit on top of. Everything in the
C file is here — the two-level table (`pgdir[1024]` → 1024 `pt_entry`), the
reference-counted `struct data`, and `pt_map`, `pt_map_nothing`, `pt_unmap`,
`pt_unmap_always`, `pt_set_flags`, `pt_copy_on_write`, `pt_find_hole`,
`pt_is_hole`, `mem_next_page`, `mem_ptr`, `mem_ptr_nofault`, `mem_segv_reason`
and `mem_changed`.

Three things are deliberately different, each documented at the point it bites:

* **There is no host mapping.** The C keeps a real `mmap`ed region per
  `struct data` and calls `mprotect` in `pt_set_flags`. Here the bytes are owned
  by a `Box<[u8]>`, so `pt_set_flags` cannot fail on the host side — its
  `_ENOMEM` (`-12`, `kernel/errno.h:19`) is only reachable from the "not mapped"
  check, and protection is enforced by the guest-side tests in `Mem::ptr`. The
  reference still exercises both `pt_set_flags` outcomes (`R -12` on a hole,
  `R 0` on a mapping).
* **`pt_entry::blocks[2]` is gone.** Those are the asbestos JIT's per-page block
  lists. The JIT is not ported, so `asbestos_invalidate_page` is a counter; the
  fixture records it (`asbestos_invalidations 32`) and the test compares it, so
  invalidation still happens at exactly the same points.
* **`container_of` is a trait impl.** `struct mem` embeds `struct mmu` and
  `mem_mmu_ops.translate` recovers the outer struct by pointer arithmetic. In
  Rust `Mem` implements `MmuOps` directly, keeps its own `changes`, and
  `Mmu::sync_changes()` reconciles the two.

`data->refcount` is not hand-maintained: one `Rc<Data>` per mapped page makes
`Rc::strong_count` *be* the C's refcount, and the fixture prints it. That is what
catches a mapping that fails to drop its predecessor, which a flags-only
comparison would not see.

The reference generator links the real `kernel/memory.c`, so it needs
`tools/stub-include/sqlite3.h`: `memory.c` → `fs/fd.h` → `fs/fake-db.h` →
`<sqlite3.h>`, and those three types are only ever used as pointers. It also
links the real `kernel/errno.c` (whose `EPIPE` path pulls in `current` and
`send_signal`, stubbed with counters) rather than faking the host→guest errno
table. The generator refuses to emit a fixture unless both counters are still
zero, i.e. unless every stub stayed off the path.

```console
$ ISH_SRC=/path/to/ish ./tools/gen_memory_reference.sh
wrote tests/fixtures/memory_reference.txt: 1268 lines, 428 operations
# asbestos_invalidations 32 fd_closes 0 signals_sent 0
```

Host pointers never enter the fixture: each backing object gets a small integer
identity in first-appearance page order, so "these two pages share one object"
survives the trip and the addresses do not. Per mapped page the test compares
flags, offset into the backing object, that identity and the reference count;
plus `pgdir_used` and the change count after every operation.

The corpus is a printed script — fixed corner cases (partial unmap,
`pt_set_flags` on a hole, a write to a read-only page, CoW across a fork, a
ptrace write, grow-down, `pt_find_hole`, `mem_next_page`, re-mapping over a
mapping, a mapping with a non-zero offset) followed by a 400-step deterministic
LCG walk — so the Rust test replays operations rather than duplicating tables.

27 mutations of `src/memory.rs` were injected; **26 were caught**. The survivor
is `hole_end - page == size` → `>= size` in `pt_find_hole`, and it is not a
coverage gap: while the downward scan stays inside a hole that expression grows
by exactly one per page, so it cannot step over `size` without landing on it.
The two forms are the same function; the comment in the source says so.

The sweep earned its keep. `Mem::ptr`'s copy-on-write break passed the page's
old `offset` to `pt_map`, where the C passes `0` because the copy is a fresh
single-page object. Two unit tests had it wrong the other way round, and one
assertion in the corpus (`main page 600: offset`) is what pinned it down. A
fixture field was wrong too: the generator printed the `pt_map` offset as
hard-coded decimal text in a line where every other page-number field is hex,
so the script contradicted the call it described. It now prints the value.

### user

`kernel/user.c` is 97 lines and every syscall argument passes through it. The
whole file is one loop: take the guest pointer, copy up to the end of its page,
ask `mem_ptr` for the next page, repeat. Three things about that loop are
behaviour rather than implementation, and all three are compared:

* **No rollback.** Each page is copied as the walk reaches it, so a fault on
  page *N* leaves pages *0..N-1* written. The corpus has a write that straddles
  a writable page and a hole: it returns 1 *and* the first six bytes are there.
  This is why the fixture dumps guest bytes (`G`) and not just return codes — a
  port that wrote to the wrong page and faulted at the right one would pass a
  return-code-only test.
* **The arithmetic wraps.** `chunk_end` is an `addr_t`, so at the top of the
  address space `(PAGE(p) + 1) << PAGE_BITS` is 0, while the loop bound
  `addr + count` is `size_t` and is not. The corpus reads and writes at
  `0xfffff000` with a count of 8192 to keep that comparison honest.
* **`user_write_string` writes one byte at a time.** That is observable: every
  byte runs `mem_ptr`, and every `mem_ptr` on a writable page invalidates the
  page's compiled blocks. Chunking the string would reach the same bytes with
  the same faults and one sixth of the invalidations. The fixture's trailer
  (`asbestos_invalidations 33`) is what caught the chunked first draft.

The `read_wrlock`/`read_wrunlock` around each C entry point have no counterpart
here; there is one thread.

Two deliberate deviations, both documented at the call site:

* `write_string` takes a `&CStr` rather than a byte slice. The C walks `buf`
  until it finds a NUL and has undefined behaviour if there is none.
* `read_string` keeps the C's behaviour of returning *success* with an
  unterminated buffer when `max` runs out first. The corpus pins both the
  short-`max` case and the exact-fit case.

The corpus also maps page 0 so the `addr == 0` checks are load-bearing: with
nothing at page 0 they are redundant, because the access would fault anyway, and
removing either check would go unnoticed.

```console
$ ISH_SRC=/path/to/ish ./tools/gen_user_reference.sh
wrote tests/fixtures/user_reference.txt: 590 lines, 52 operations, 576K
# asbestos_invalidations 33 fd_closes 0 signals_sent 0
```

23 mutations of `src/user.rs` were injected; **22 were caught**. The survivor is
advancing the *read* walk by one byte instead of by `chunk_end`. It is
equivalent, and only on the read path: the copies overlap, but a read never
changes a guest byte (grow-down maps zeroes and zeroes read back as zeroes), it
faults on the same first unmapped page, and it bumps no counter. The identical
change on the *write* path is caught, because there each extra `mem_ptr`
invalidates a page.

Both fixtures now state their radix in a header line. The first draft of this
one printed byte counts in hex while every other count was decimal, and the test
read them as decimal — a fixture field that silently changes base is worse than
a missing one.

### errno

`kernel/errno.c` is one switch and one side effect, but it has a wrinkle that
makes transcription a bad idea: the *guest* numbers are the i386 Linux ABI, fixed
by `kernel/errno.h`, while the *host* numbers are whatever the compiling platform
calls them. The C gets the second half from `<errno.h>` at build time, so the
port asks for it the same way — `tools/gen_errno_table.py` emits a C program that
prints each `ERRCASE` name with its host value, runs it, and joins the result
against the guest values.

On this Linux host all 82 entries come out as the identity (`host == -guest`),
which is exactly why the mapping looks pointless here and is not: on a Darwin
host `EAGAIN` is 35, not 11. Regenerating on another host produces that host's
table, as the C would.

What is left is the fallback, and it is most of the behaviour: an errno the C
does not know becomes `-(err | 0x1000)`. The corpus runs `err_map` over -16 to
4199, so 4,134 of 4,216 inputs take that path. Two properties fall out and are
both pinned:

* it is an **or**, not an add, so an input with bit 12 already set comes back
  unchanged apart from the sign;
* for a negative input that means a *positive* result — `err_map(-16) == 16` —
  which is what the C does.

The fixture carries the host's own numbers, so a table generated on one platform
and tested on another fails on the host-number assertions rather than silently
comparing the wrong half of the mapping. It also counts the C's
`printk("unknown error")` calls, and the test's count of unknown inputs has to
match — including the two unknown probes among the ten `errno_map` cases.

`errno_map`'s SIGPIPE delivery is returned rather than performed, because signal
delivery is not ported: `Mapped { guest, sigpipe }` says exactly what the C would
have done.

14 mutations were injected, spanning both `src/errno.rs` and the generated table
(a stale `HOST_EPIPE`, a wrong guest number, a dropped entry); **all 14 were
caught**.

### mmap

`kernel/mmap.c` is where the guest asks for address space, and it sits directly
on `memory.rs`. Returns are compared as raw 32-bit words because that is what
these syscalls return: an address on success, a negated errno reinterpreted as
`addr_t` on failure — `-EINVAL` comes back as `ffffffea`.

Four pieces of upstream behaviour are pinned rather than tidied:

* **A non-`MAP_FIXED` hint that is not free is used anyway.** When the hint
  overlaps an existing mapping the C does `addr = 0;`, which reads as "go find a
  hole instead" — but `page` was already assigned from the hint and is never
  recomputed, and `addr` is not read again. The assignment is dead. The corpus
  maps `0x400000` twice with no `MAP_FIXED` and compares the result.
* **`mprotect` replaces the flags, so it clears `P_ANONYMOUS`** — and `mremap`
  only grows anonymous mappings. The corpus mprotects a mapping and then fails to
  grow it, which is the interaction a reader would not predict from either
  function alone.
* **`mremap` uses `PAGE(len)`, not `PAGE_ROUND_UP(len)`.** A sub-page length
  rounds *down* to zero pages, so `mremap(a, 0x100, 0x1000)` takes the grow
  branch, not the shrink branch, and fails because the page it would grow into is
  the one it already owns.
* **`brk`'s shrink leaves a page behind** when the old brk is not page-aligned:
  it unmaps `PAGE(old_brk) - PAGE(new_brk)` pages. Shrinking from `0x1002001` to
  `0x1000000` leaves page `0x1002` mapped, and that stray page is why a later
  `brk` in the corpus is refused.

One place departs from the letter of the original, and says so in the source:
`mremap`'s grow path checks its pages with `entry == NULL && entry->flags !=
pt_flags`, which dereferences `entry` in the branch that just established it is
`NULL`. Any sparse range crashes the C instead of returning an error, so the
reference generator cannot contain that case; the port implements the check that
was plainly meant.

Not ported, both documented at the point they would matter: file-backed mappings
(no fd table, so every descriptor is invalid and `EBADF` is the C's own answer —
the generator asserts `f_get` was reached exactly once, by the one file-backed
attempt) and `struct mm`'s procfs fields.

27 mutations were injected; **24 were caught**. The three survivors are not
coverage gaps: `mmap_common`'s `len == 0` check is redundant with `do_mmap`'s
`pages == 0`, since `page_round_up(0)` is 0; and `mmap2`'s page-to-byte offset
conversion plus the old `mmap`'s choice of offset field over fd field are both
unread, because nothing consumes the offset until a file-backed mapping exists.
Closing the other four survivors took three new corpus cases — a non-page-aligned
`mprotect`, a successful multi-page `mremap` shrink, and a non-page-aligned
`munmap` — plus a unit test that fills the whole `pt_find_hole` scan range to
reach `do_mmap`'s `ENOMEM`.

### task, groups, identity, TLS and resources

The next layer is the task-owned state that turns the address-space primitives
into a usable syscall context. `task.rs` owns an explicit `TaskTable` rather
than reproducing C's raw global `__thread current` pointer: the caller chooses a
live current PID, while the table retains the same observable distinction
between `pid_get_task` (hides zombies) and `pid_get_task_zombie` (does not).

The PID table is sparse in Rust but has the C's fixed range and wrap-around
allocation policy. It also preserves the non-obvious reason `struct pid` has
three links: a PID slot remains occupied while a session or process-group member
still references it, even after its task pointer has gone away. Parent/child
links, C's shallow `task_create_` copy plus field resets, and the topology split
between a new process group and `CLONE_THREAD` are all explicit APIs, ready for
`fork.c`.

`group.rs` ports the exact ordering of `group.c`'s checks: `setpgid` can target a
zombie because it uses `pid->task`, but `getpgid` hides one; a caller may affect
only itself or a direct child; joining a group requires a same-session member;
and a session leader cannot make another process-group change. `setsid` moves
both index memberships and intentionally ignores a syscall argument because
that is what iSH's zero-argument C implementation does.

The identity and TLS syscalls use the existing guest-memory bridge rather than
host pointers. This preserves partial writes in `getresuid`/`getresgid`, and the
more subtle partial overwrite of the fixed supplementary-groups array when
`setgroups` faults mid-copy. `set_thread_area` preserves all unimplemented
`user_desc` bitfield bits and still updates `tls_ptr` before a read-only
descriptor's write-back fails. `PR_SET_NAME` uses C `strcpy` semantics, so bytes
after the terminating NUL in `comm[16]` remain untouched.

`resource.rs` adds the exact 32- and 64-bit rlimit guest layouts (including
C's surprising full-64-bit `setrlimit32` input), iSH's `INT_MAX` compatibility
clamp, root/non-root maximum-limit rule, the old-before-new ordering of
`prlimit64`, rusage wire layout and time-only
accumulation, affinity bitset, and the intentionally narrow scheduler/priority
stubs. `ResourceHost` makes the two host observations in the C source
(per-thread CPU usage and online CPUs) explicit, so an iOS embedding can provide
native telemetry rather than the core silently substituting wall-clock data.
`ThreadGroup` now carries limits, own usage, and children usage; `exit.c` will
connect its reaping transitions to that storage.

`tests/resource_differential.rs` replays 51 calls from
`tests/fixtures/resource_reference.txt`. `tools/resource-dump.c` compiles and
links the **unmodified** `kernel/resource.c`; a deterministic task/group,
two-page guest window, and linker-wrapped `getrusage(RUSAGE_THREAD)` / `sysconf`
inputs make the comparison reproducible locally. After every call the test
compares the raw return value, all 16 `(cur,max)` limit pairs, all 18
`children_rusage` ABI words, effective UID, and an FNV-1a hash of the whole
guest window. C initializes only the two timeval fields of a host rusage, so
the fixture compares that defined prefix and explicitly clears its indeterminate
tail before hashing rather than treating compiler stack garbage as an ABI.

```console
$ ISH_SRC=/path/to/ish ./tools/gen_resource_reference.sh
wrote tests/fixtures/resource_reference.txt: 182 lines, 51 C operations, 52 snapshots
$ cargo test --test resource_differential
… 51 resource operations and 52 full C-state snapshots matched exactly
```

`random.rs` ports `kernel/random.c` with a `RandomSource` host boundary, so an
iOS embedding can use `CCRandomGenerateBytes` and a Linux embedding can use its
native `getrandom` implementation without the portable core inventing entropy.
`tests/random_differential.rs` links the unmodified C file and wraps only its
Linux `syscall(SYS_getrandom, …)` boundary with deterministic bytes. Its nine
records verify the one-MiB limit, zero-length call, ignored flags, host failure,
guest-output fault ordering, and every byte written on successful calls.

```console
$ ISH_SRC=/path/to/ish ./tools/gen_random_reference.sh
wrote tests/fixtures/random_reference.txt: 34 lines, 9 C operations
$ cargo test --test random_differential
… 9 getrandom operations and every deterministic output byte matched C
```

`uname.rs` ports `kernel/uname.c`'s fixed 390-byte `uname` and 60-byte
`sys_info` guest ABIs. `SystemInfoHost` supplies native hostname, uptime/load,
and memory values while `UnameConfig` represents C's two mutable identity
globals and compile-time version suffix. The direct C fixture fixes
`SOURCE_DATE_EPOCH=0`, wraps host `uname`/`sysinfo`, and supplies `get_uptime`;
it verifies the string fields, the 64→32-bit truncations, zeroed `bufferram` and
padding, host calls before guest faults, override precedence, and `snprintf`
truncation.

```console
$ ISH_SRC=/path/to/ish ./tools/gen_uname_reference.sh
wrote tests/fixtures/uname_reference.txt: 50 lines, 10 C operations, 11 snapshots
$ cargo test --test uname_differential
… 10 uname/sysinfo operations and all C-derived guest bytes matched exactly
```

The host thread launcher in `task.c` and signal/tty/filesystem pointers in the
rest of `struct task` remain later engine/kernel ports; they are intentionally
not represented as fake implementations. The new `iSH Rust Core` workflow runs
a focused rustfmt check (without rewriting generated decoder output), every
unit/differential test, and strict clippy on every change under `ish-rs/`.

## Layout

```
ish-rs/
├── Cargo.toml
├── src/
│   ├── lib.rs
│   ├── cpu.rs                  # emu/cpu.h
│   ├── float80.rs              # emu/float80.{h,c}
│   ├── fpu.rs                  # emu/fpu.{h,c}
│   ├── mmu.rs                  # emu/mmu.h
│   ├── decode.rs               # emu/decode.h dispatch
│   ├── decode_table.rs         # generated opcode table (do not edit)
│   ├── modrm.rs                # emu/modrm.h
│   ├── cpuid.rs                # emu/cpuid.h
│   ├── interrupt.rs            # emu/interrupt.h
│   ├── tlb.rs                  # emu/tlb.{h,c}
│   ├── vec.rs                  # emu/vec.{h,c} + emu/mmx.c
│   ├── memory.rs               # kernel/memory.{h,c}
│   ├── user.rs                 # kernel/user.c
│   ├── mmap.rs                 # kernel/mmap.c + kernel/mm.h
│   ├── errno.rs                # kernel/errno.{h,c}
│   ├── errno_table.rs          # generated host->guest errno table (do not edit)
│   ├── resource.rs             # kernel/resource.{h,c}
│   ├── random.rs               # kernel/random.{h,c}
│   ├── uname.rs                # kernel/uname.c
│   ├── task.rs                 # kernel/task.{h,c} state and PID table
│   ├── group.rs                # kernel/group.c
│   ├── getset.rs               # kernel/getset.c
│   ├── tls.rs                  # kernel/tls.c
│   └── misc.rs                 # kernel/misc.c
├── tests/
│   ├── differential.rs         # bit-exact replay of the float80 reference
│   ├── fpu_differential.rs     # word-exact replay of the cpu/fpu reference
│   ├── decode_differential.rs  # event-exact replay of the decoder reference
│   ├── memory_differential.rs  # operation-exact replay of the page-table reference
│   ├── modrm_differential.rs   # field-exact replay of the ModRM/SIB reference
│   ├── user_differential.rs    # byte-exact replay of the user-memory reference
│   ├── errno_differential.rs   # value-exact replay of the errno reference
│   ├── mmap_differential.rs    # word-exact replay of the mmap reference
│   ├── tlb_differential.rs     # word-exact replay of the tlb reference
│   ├── vec_differential.rs     # word-exact replay of the vec/mmx reference
│   ├── resource_differential.rs # state-exact replay of the resource reference
│   ├── random_differential.rs  # byte-exact replay of the random reference
│   ├── uname_differential.rs   # byte-exact replay of the uname/sysinfo reference
│   └── fixtures/
│       ├── f80_reference.txt   # 125k results from the unmodified C
│       ├── fpu_reference.txt   # 15.7k full cpu_state dumps from the C
│       ├── decode_reference.txt# 37.9k instruction decodings from the C
│       ├── memory_reference.txt# 428 page-table operations from the C
│       ├── modrm_reference.txt # 1283 ModRM/SIB decodings from the C
│       ├── user_reference.txt  # 52 user-memory ops, guest+host bytes from the C
│       ├── errno_reference.txt # 4216 err_map inputs + host errno numbers
│       ├── mmap_reference.txt  # 51 mmap ops + the whole page table after each
│       ├── tlb_reference.txt   # 56 full struct tlb dumps from the C
│       ├── vec_reference.txt   # 9.8k vec/mmx results from the C
│       ├── resource_reference.txt # 51 deterministic resource calls from the C
│       ├── random_reference.txt # 9 deterministic getrandom calls from the C
│       └── uname_reference.txt # 10 deterministic uname/sysinfo calls from the C
└── tools/
    ├── f80-dump.c              # float80 reference generator (not part of iSH)
    ├── fpu-dump.c              # cpu/fpu reference generator
    ├── tlb-dump.c              # mmu/tlb reference generator
    ├── decode-dump.c           # decoder reference generator (drives decode.h)
    ├── gen_decode_table.py     # derives src/decode_table.rs from the fixture
    ├── memory-dump.c           # page-table reference generator (links memory.c)
    ├── stub-include/sqlite3.h  # three opaque typedefs memory.c reaches via fd.h
    ├── user-dump.c             # user-memory reference generator (links user.c)
    ├── errno-dump.c            # errno reference generator (links errno.c)
    ├── mmap-dump.c             # mmap reference generator (links mmap.c)
    ├── resource-dump.c         # deterministic resource.c reference generator
    ├── random-dump.c           # deterministic random.c reference generator
    ├── uname-dump.c            # deterministic uname.c reference generator
    ├── gen_errno_table.py      # derives src/errno_table.rs, asking the host
    ├── modrm-dump.c            # ModRM/SIB reference generator
    ├── vec-dump.c              # vec/mmx reference generator
    ├── gen_vec_ops.py          # derives both op tables from emu/vec.h
    ├── vec-ops.inc             # generated C op table (166 entries)
    ├── gen_f80_reference.sh
    ├── gen_fpu_reference.sh
    ├── gen_decode_reference.sh
    ├── gen_memory_reference.sh
    ├── gen_user_reference.sh
    ├── gen_errno_reference.sh
    ├── gen_mmap_reference.sh
    ├── gen_resource_reference.sh
    ├── gen_random_reference.sh
    ├── gen_uname_reference.sh
    ├── gen_modrm_reference.sh
    ├── gen_tlb_reference.sh
    └── gen_vec_reference.sh
```

## Design notes

* **No `f64` shortcuts.** The whole point of `emu/float80.c` is 80-bit extended
  precision; the port keeps the 64-bit significand + 15-bit exponent
  representation and does every operation in `u128`, exactly as the C does.
  `to_f64`/`from_f64` exist only where the C has `f80_to_double`/`f80_from_double`.
* **Faithful over idiomatic.** C quirks are reproduced on purpose (the `rest`
  truncation to `uint64_t` in the rounding helper, the `f80_eq` copy/paste on
  the zero-sign clearing, `sqrt(inf)` returning NaN through Newton iteration).
  Where the port does deviate, it is commented and covered by a test.
* **`__thread` → `thread_local!`.** `f80_rounding_mode` is thread-local in C and
  a `thread_local!` `Cell` here, with `rounding_mode()`, `set_rounding_mode()`
  and a `with_rounding_mode()` RAII-style helper.
* **No bitfields in Rust.** `cpu.h` overlays unions of bitfields on raw
  `dword_t`/`word_t` storage. Each of those becomes named fields plus an
  explicit pack/unpack pair (`eflags()`/`set_eflags()`, `top()`/`set_top()`,
  `rc()`/`set_rc()`). Two subtleties are modelled on purpose: `eflags` keeps the
  bits *above* the declared bitfields (a `popf`/`pushf` round trip preserves
  them), and `set_exp()` on `Float80` can only ever touch the 15-bit exponent.
* **Quirks kept.** `fpu_clex` clears the *EFLAGS* sign flag rather than the FPU
  stack-fault bit, `fpu_stm32` narrows through `f64`, and the conditional moves
  read the materialised `cf` byte and the packed `zf`/`pf` bits instead of the
  lazy `ZF`/`PF` macros. All three are pinned by tests.
* **The TLB borrows the MMU instead of owning it.** In C, `struct tlb` holds a
  `struct mmu *` and `tlb_flush` reads `tlb->mmu->changes` through it. Here
  every TLB call takes `&mut Mmu`, and the TLB keeps only an identity token
  (`mmu_id: *const ()`, compared with `ptr::eq`) so `refresh` can still tell
  "same address space" from "a different one". This is what lets a backend
  change mappings between two operations without unsafe aliasing.
* **`data_minus_addr` → `Option<NonNull<u8>>`.** The C caches
  `host_ptr - guest_page` so the gadgets get a byte pointer with one add; the
  port stores the page base and reconstructs the C value in
  `TlbEntry::data_minus_addr()`. `None` is the C's post-flush zero.
* **Unsigned wraps kept.** `PGOFFSET(addr) > PAGE_SIZE - size` wraps when
  `size` exceeds a page, which sends oversized accesses down the *fast* path;
  `page_round_up(0)` is 0, not 1; and `(PAGE(addr) + 1) << PAGE_BITS` wraps to 0
  for the last guest page. Each is pinned by the differential corpus.
* **`MEM_WRITE_PTRACE` is not `MEM_WRITE`.** The backends compare
  `type == MEM_WRITE` exactly, so a ptrace write bypasses write protection
  (`kernel/user.c` relies on this). The discriminants are reproduced and tested.
* **NaN payloads are a codegen artefact, and are pinned rather than trusted.**
  With both operands NaN, x86 returns the payload of whichever operand the
  compiler loaded into the destination register, and floating-point add and
  multiply are commutative as far as gcc and LLVM are concerned. The port uses
  `*dst = *dst + *src` because rustc and gcc both put `dst` in the register for
  that spelling, while Rust's `*dst += *src` makes LLVM load `src` first and the
  other payload comes out. The vec corpus contains cases whose f32 *and* f64
  views are both NaN, so a compiler that changes its mind fails a test instead
  of diverging quietly.
* **Generated op tables, not hand-copied ones.** `emu/vec.h` declares 166
  functions in 30 signatures. Both the C driver's table and the Rust dispatch
  match come from `tools/gen_vec_ops.py`, so the two sides cannot drift and
  nothing gets silently omitted.
* **`debug_assert`-free.** The C assertions are debug-build checks over an
  upstream edge case; the Rust port returns the same bits instead of panicking.

## License

iSH is licensed under GPLv3, with contributions after commit `0e3a4144` also
under GPLv2. This is a derivative work of that code, so it carries the same
terms: **GPL-2.0-or-later**.
