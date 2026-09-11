//! Rust port of the iSH emulator core (`emu/`).
//!
//! iSH (<https://github.com/ish-app/ish>) is a userspace x86 Linux emulator for
//! iOS. Its "core" is the host-independent part under `emu/`: the CPU state,
//! the MMU/TLB, the x86 decoder, and the software FPU. This crate is a
//! line-by-line Rust translation of that core, module by module.
//!
//! # Modules
//!
//! | Rust module     | C original                | Status |
//! |-----------------|---------------------------|--------|
//! | [`float80`]     | `emu/float80.h` + `.c`    | ported, bit-exact against the C reference |
//! | [`cpu`]         | `emu/cpu.h`               | ported, flag logic checked against the C macros |
//! | [`fpu`]         | `emu/fpu.h` + `.c`        | ported, full state compared after every op |
//! | [`mmu`]         | `emu/mmu.h`               | ported, page arithmetic and the translate interface |
//! | [`tlb`]         | `emu/tlb.h` + `.c`        | ported, cache behaviour compared against the C |
//!
//! # Provenance and license
//!
//! This is a translation of GPLv3 / GPLv2-or-later code, so it carries the same
//! license: `GPL-2.0-or-later`. See `README.md`.

pub mod cpu;
pub mod float80;
pub mod fpu;
pub mod mmu;
pub mod tlb;

pub use cpu::CpuState;
pub use float80::Float80;
