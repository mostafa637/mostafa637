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
| `mmu.rs`     | `emu/mmu.h`, `tlb.{h,c}` | 184  | not started (4 GiB address space, TLB) |
| `decode.rs`  | `emu/decode.h`, `modrm.h` | 1,522 | not started (x86 decoder + instruction semantics) |
| `vec.rs`     | `emu/vec.{h,c}`       | 817     | not started (SSE) |
| `mmx.rs`     | `emu/mmx.c`           | 180     | not started (MMX) |

`cpu.rs` models the parts of `struct cpu_state` that the ported code touches.
Three fields are not there yet because nothing uses them: `struct mmu *mmu` and
`bool *poked_ptr` (pointers, which need `mmu.rs`) and `long cycle` (engine
bookkeeping).

`f80_rem` is declared in `emu/float80.h` but has no definition anywhere in the
iSH tree, so there is nothing to port for it.

## Verification

```console
$ cargo test --tests
running 40 tests   (unit tests in src/)
running 2 tests    (tests/differential.rs)      -> 125,612 float80 results
running 4 tests    (tests/fpu_differential.rs)  -> 502,552 cpu_state words
test result: ok
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
│   └── fpu.rs                  # emu/fpu.{h,c}
├── tests/
│   ├── differential.rs         # bit-exact replay of the float80 reference
│   ├── fpu_differential.rs     # word-exact replay of the cpu/fpu reference
│   └── fixtures/
│       ├── f80_reference.txt   # 125k results from the unmodified C
│       └── fpu_reference.txt   # 15.7k full cpu_state dumps from the C
└── tools/
    ├── f80-dump.c              # float80 reference generator (not part of iSH)
    ├── fpu-dump.c              # cpu/fpu reference generator
    ├── gen_f80_reference.sh
    └── gen_fpu_reference.sh
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
* **`debug_assert`-free.** The C assertions are debug-build checks over an
  upstream edge case; the Rust port returns the same bits instead of panicking.

## License

iSH is licensed under GPLv3, with contributions after commit `0e3a4144` also
under GPLv2. This is a derivative work of that code, so it carries the same
terms: **GPL-2.0-or-later**.
