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
| `fpu.rs`     | `emu/fpu.{h,c}`       | 506     | not started (x87 instructions, needs `float80` ✔) |
| `cpu.rs`     | `emu/cpu.h`           | 235     | not started (register file, lazy EFLAGS, FPU/MMX state) |
| `mmu.rs`     | `emu/mmu.h`, `tlb.{h,c}` | 154  | not started (4 GiB address space, TLB) |
| `decode.rs`  | `emu/decode.h`, `modrm.h` | 1,522 | not started (x86 decoder + instruction semantics) |
| `vec.rs`     | `emu/vec.{h,c}`       | 817     | not started (SSE) |
| `mmx.rs`     | `emu/mmx.c`           | 180     | not started (MMX) |

`f80_rem` is declared in `emu/float80.h` but has no definition anywhere in the
iSH tree, so there is nothing to port for it.

## Verification

```console
$ cargo test
running 23 tests   (unit tests in src/float80.rs)
running 2 tests    (tests/differential.rs)
test result: ok
```

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
│   └── float80.rs              # emu/float80.{h,c}
├── tests/
│   ├── differential.rs         # bit-exact replay of the C reference
│   └── fixtures/
│       └── f80_reference.txt   # 125k results from the unmodified C
└── tools/
    ├── f80-dump.c              # reference generator (not part of iSH)
    └── gen_f80_reference.sh
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
* **`debug_assert`-free.** The C assertions are debug-build checks over an
  upstream edge case; the Rust port returns the same bits instead of panicking.

## License

iSH is licensed under GPLv3, with contributions after commit `0e3a4144` also
under GPLv2. This is a derivative work of that code, so it carries the same
terms: **GPL-2.0-or-later**.
