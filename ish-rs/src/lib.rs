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
//! | [`vec`]         | `emu/vec.{h,c}` + `mmx.c` | ported, all 166 operations compared against the C |
//! | [`modrm`]       | `emu/modrm.h`             | ported, decode results compared against the C |
//! | [`decode`]      | `emu/decode.h`            | ported dispatch, opcode table generated from the C |
//! | [`memory`]      | `kernel/memory.{h,c}`     | ported, the guest address space behind `mmu`/`tlb` |
//! | [`user`]        | `kernel/user.c`           | ported, the guest<->kernel byte copies, faults included |
//! | [`errno`]       | `kernel/errno.{h,c}`      | ported, host to guest errno translation, table generated |
//! | [`cpuid`]       | `emu/cpuid.h`             | ported |
//! | [`interrupt`]   | `emu/interrupt.h`         | ported |
//!
//! # Provenance and license
//!
//! This is a translation of GPLv3 / GPLv2-or-later code, so it carries the same
//! license: `GPL-2.0-or-later`. See `README.md`.

pub mod cpu;
pub mod cpuid;
pub mod decode;
pub mod errno;
pub mod errno_table;
pub mod decode_table;
pub mod float80;
pub mod fpu;
pub mod interrupt;
pub mod memory;
pub mod mmu;
pub mod modrm;
pub mod tlb;
pub mod user;
pub mod vec;

pub use cpu::CpuState;
pub use float80::Float80;
