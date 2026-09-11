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
| `decode.rs`  | `emu/decode.h`, `modrm.h` | 1,522 | not started (x86 decoder + instruction semantics) |

`cpu.rs` models the parts of `struct cpu_state` that the ported code touches.
Three fields are not there yet because nothing uses them: `struct mmu *mmu` and
`bool *poked_ptr` (pointers, which need `mmu.rs`) and `long cycle` (engine
bookkeeping).

`f80_rem` is declared in `emu/float80.h` but has no definition anywhere in the
iSH tree, so there is nothing to port for it.

## Verification

```console
$ cargo test --tests
running 64 tests   (unit tests in src/)
running 2 tests    (tests/differential.rs)      -> 125,612 float80 results
running 4 tests    (tests/fpu_differential.rs)  -> 502,552 cpu_state words
running 2 tests    (tests/tlb_differential.rs)  -> 172,312 tlb state words
running 2 tests    (tests/vec_differential.rs)  ->   9,794 vec/mmx results
test result: ok. 74 passed
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
│   ├── tlb.rs                  # emu/tlb.{h,c}
│   └── vec.rs                  # emu/vec.{h,c} + emu/mmx.c
├── tests/
│   ├── differential.rs         # bit-exact replay of the float80 reference
│   ├── fpu_differential.rs     # word-exact replay of the cpu/fpu reference
│   ├── tlb_differential.rs     # word-exact replay of the tlb reference
│   ├── vec_differential.rs     # word-exact replay of the vec/mmx reference
│   └── fixtures/
│       ├── f80_reference.txt   # 125k results from the unmodified C
│       ├── fpu_reference.txt   # 15.7k full cpu_state dumps from the C
│       ├── tlb_reference.txt   # 56 full struct tlb dumps from the C
│       └── vec_reference.txt   # 9.8k vec/mmx results from the C
└── tools/
    ├── f80-dump.c              # float80 reference generator (not part of iSH)
    ├── fpu-dump.c              # cpu/fpu reference generator
    ├── tlb-dump.c              # mmu/tlb reference generator
    ├── vec-dump.c              # vec/mmx reference generator
    ├── gen_vec_ops.py          # derives both op tables from emu/vec.h
    ├── vec-ops.inc             # generated C op table (166 entries)
    ├── gen_f80_reference.sh
    ├── gen_fpu_reference.sh
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
