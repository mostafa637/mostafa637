//! Word-for-word differential test of `cpu.rs` + `fpu.rs` against the C
//! original.
//!
//! `tests/fixtures/fpu_reference.txt` is produced by `tools/fpu-dump.c`,
//! compiled against the *unmodified* iSH `emu/fpu.c`, `emu/cpu.h` and
//! `emu/float80.c`. After **every** operation the C driver dumps the entire
//! `struct cpu_state` — all eight x87 registers, the status and control words,
//! and every flag byte and bit — and this test requires the Rust port to
//! reproduce each of those words exactly.
//!
//! The flag macros from `emu/cpu.h` (`ZF`/`SF`/`CF`/`OF`/`PF`/`AF`) plus
//! `collapse_flags` and `expand_flags` are checked the same way.
//!
//! Regenerate with:
//!
//! ```text
//! ./tools/gen_fpu_reference.sh /path/to/ish-checkout
//! ```

use ish_emu::cpu::{CpuState, FSW_TOP_MASK};
use ish_emu::float80::{Float80, RoundingMode};
use ish_emu::fpu::{FpuConst, FPU_CONSTS};

const FIXTURE: &str = concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/tests/fixtures/fpu_reference.txt"
);

// ---- the same operand pools the C driver uses ---------------------------

struct MemVal {
    bits64: u64,
    f80: (u64, u16),
}

const MEMVALS: [MemVal; 12] = [
    MemVal {
        bits64: 0x0000_0000_0000_0000,
        f80: (0x0000_0000_0000_0000, 0x0000),
    },
    MemVal {
        bits64: 0x0000_0000_0000_0001,
        f80: (0x8000_0000_0000_0000, 0x3fff),
    },
    MemVal {
        bits64: 0xffff_ffff_ffff_ffff,
        f80: (0x8000_0000_0000_0000, 0xbfff),
    },
    MemVal {
        bits64: 0x0000_0000_7fff_ffff,
        f80: (0xc000_0000_0000_0000, 0x4000),
    },
    MemVal {
        bits64: 0xffff_ffff_8000_0000,
        f80: (0xa000_0000_0000_0000, 0x4000),
    },
    MemVal {
        bits64: 0x0000_0000_0000_7fff,
        f80: (0x9a20_9a84_fbcf_f799, 0x3ffd),
    },
    MemVal {
        bits64: 0x0000_0000_ffff_8000,
        f80: (0x8000_0000_0000_0000, 0x0000),
    },
    MemVal {
        bits64: 0x0000_0000_0000_9c40,
        f80: (0x0000_0000_0000_0001, 0x0000),
    },
    MemVal {
        bits64: 0x4009_21fb_5444_2d18,
        f80: (0xc90f_daa2_2168_c235, 0x4000),
    },
    MemVal {
        bits64: 0xc009_21fb_5444_2d18,
        f80: (0xffff_ffff_ffff_ffff, 0x7ffe),
    },
    MemVal {
        bits64: 0x7fef_ffff_ffff_ffff,
        f80: (0x7fff_ffff_ffff_ffff, 0x0000),
    },
    MemVal {
        bits64: 0x3ff0_0000_0000_0000,
        f80: (0xc000_0000_0000_0000, 0xffff),
    },
];

const POOL: [(u64, u16); 15] = [
    (0x8000_0000_0000_0000, 0x3fff), //  1.0
    (0x8000_0000_0000_0000, 0xbfff), // -1.0
    (0xc000_0000_0000_0000, 0x4000), //  3.0
    (0x9a20_9a84_fbcf_f799, 0x3ffd), //  log10(2)
    (0x0000_0000_0000_0000, 0x0000), // +0
    (0x0000_0000_0000_0000, 0x8000), // -0
    (0x8000_0000_0000_0000, 0x7fff), // +inf
    (0xc000_0000_0000_0000, 0x7fff), //  nan
    (0x0000_0000_0000_0001, 0x0000), //  smallest denormal
    (0x8000_0000_0000_0000, 0x0001), //  smallest normal
    (0xffff_ffff_ffff_ffff, 0x7ffe), //  largest finite
    (0xd49a_784b_cd1b_8afe, 0x4000), //  log2(10)
    (0xb172_17f7_d1cf_79ac, 0x3ffe), //  ln2
    (0xa000_0000_0000_0000, 0x4002), //  10.0
    (0x4000_0000_0000_0000, 0x3fff), //  unormal (unsupported)
];

/// The C driver's `lcg()`, bit for bit.
struct Lcg(u64);

impl Lcg {
    fn next(&mut self) -> u64 {
        self.0 = self
            .0
            .wrapping_mul(6364136223846793005)
            .wrapping_add(1442695040888963407);
        self.0 >> 11
    }
}

/// Rebuild the exact state pool `build_states()` in fpu-dump.c produces.
fn build_states() -> Vec<CpuState> {
    let mut lcg = Lcg(0x9e37_79b9_7f4a_7c15);
    let mut out = Vec::with_capacity(24);
    for s in 0..24usize {
        let mut cpu = CpuState::default();
        for i in 0..8usize {
            let (signif, sign_exp) = if s < 4 {
                POOL[(s * 8 + i) % POOL.len()]
            } else {
                POOL[(lcg.next() as usize) % POOL.len()]
            };
            cpu.fp[i] = Float80::from_bits(signif, sign_exp);
        }
        cpu.set_top((s % 8) as u8);
        cpu.fcw = (lcg.next() & 0x0fff) as u16;
        cpu.fsw = (cpu.fsw & !0x47ff) | ((lcg.next() & 0x4700) as u16);
        cpu.cf = (lcg.next() & 1) as u8;
        cpu.of = (lcg.next() & 1) as u8;
        cpu.zf = lcg.next() & 1 != 0;
        cpu.sf = lcg.next() & 1 != 0;
        cpu.pf = lcg.next() & 1 != 0;
        cpu.af = lcg.next() & 1 != 0;
        cpu.cf_bit = lcg.next() & 1 != 0;
        cpu.of_bit = lcg.next() & 1 != 0;
        cpu.res = lcg.next() as u32;
        cpu.op1 = lcg.next() as u32;
        cpu.op2 = lcg.next() as u32;
        cpu.flags_res = (lcg.next() & 0xf) as u8;
        // C: cpu->eflags = (cpu->eflags & 0x3fff) | ((lcg() & 0x3ffff) << 14)
        cpu.eflags_high = (lcg.next() & 0x3_ffff) as u32;
        out.push(cpu);
    }
    out
}

const STATE_WORDS: usize = 32;

fn dump_state(cpu: &CpuState) -> Vec<String> {
    let mut v = Vec::with_capacity(STATE_WORDS);
    v.push(format!("{:x}", cpu.top()));
    v.push(format!("{:x}", cpu.fsw));
    v.push(format!("{:x}", cpu.fcw));
    v.push(format!("{:x}", cpu.cf));
    v.push(format!("{:x}", cpu.of));
    v.push(format!("{:x}", cpu.zf as u32));
    v.push(format!("{:x}", cpu.sf as u32));
    v.push(format!("{:x}", cpu.pf as u32));
    v.push(format!("{:x}", cpu.af as u32));
    v.push(format!("{:x}", cpu.cf_bit as u32));
    v.push(format!("{:x}", cpu.of_bit as u32));
    v.push(format!("{:x}", cpu.res));
    v.push(format!("{:x}", cpu.op1));
    v.push(format!("{:x}", cpu.op2));
    v.push(format!("{:x}", cpu.flags_res));
    v.push(format!("{:x}", cpu.eflags()));
    for i in 0..8 {
        v.push(format!("{:016x}", cpu.fp[i].signif));
        v.push(format!("{:04x}", cpu.fp[i].sign_exp));
    }
    v
}

const WORD_NAMES: [&str; 16] = [
    "top",
    "fsw",
    "fcw",
    "cf",
    "of",
    "zf",
    "sf",
    "pf",
    "af",
    "cf_bit",
    "of_bit",
    "res",
    "op1",
    "op2",
    "flags_res",
    "eflags",
];

fn word_name(i: usize) -> String {
    if i < 16 {
        WORD_NAMES[i].to_string()
    } else {
        let r = (i - 16) / 2;
        let part = if (i - 16).is_multiple_of(2) {
            "signif"
        } else {
            "signExp"
        };
        format!("fp[{r}].{part}")
    }
}

fn apply_op(cpu: &mut CpuState, op: &str, args: &[&str]) {
    let a = |i: usize| -> usize { args[i].parse().expect("numeric arg") };
    match op {
        "pop" => cpu.fpu_pop(),
        "incstp" => cpu.fpu_incstp(),
        "xch" => cpu.fpu_xch(a(0)),
        "ld" => cpu.fpu_ld(a(0)),
        "st" => cpu.fpu_st(a(0)),
        "ldc" => cpu.fpu_ldc(match a(0) {
            0 => FpuConst::One,
            1 => FpuConst::Log2t,
            2 => FpuConst::Log2e,
            3 => FpuConst::Pi,
            4 => FpuConst::Log2,
            5 => FpuConst::Ln2,
            _ => FpuConst::Zero,
        }),
        "com" => cpu.fpu_com(a(0)),
        "comi" => cpu.fpu_comi(a(0)),
        "cmovb" => cpu.fpu_cmovb(a(0)),
        "cmove" => cpu.fpu_cmove(a(0)),
        "cmovbe" => cpu.fpu_cmovbe(a(0)),
        "cmovu" => cpu.fpu_cmovu(a(0)),
        "cmovnb" => cpu.fpu_cmovnb(a(0)),
        "cmovne" => cpu.fpu_cmovne(a(0)),
        "cmovnbe" => cpu.fpu_cmovnbe(a(0)),
        "cmovnu" => cpu.fpu_cmovnu(a(0)),

        "prem" => cpu.fpu_prem(),
        "rndint" => cpu.fpu_rndint(),
        "scale" => cpu.fpu_scale(),
        "abs" => cpu.fpu_abs(),
        "chs" => cpu.fpu_chs(),
        "sqrt" => cpu.fpu_sqrt(),
        "2xm1" => cpu.fpu_2xm1(),
        "tst" => cpu.fpu_tst(),
        "xam" => cpu.fpu_xam(),
        "xtract" => cpu.fpu_xtract(),
        "sin" => cpu.fpu_sin(),
        "cos" => cpu.fpu_cos(),
        "patan" => cpu.fpu_patan(),
        "yl2x" => cpu.fpu_yl2x(),
        "clex" => cpu.fpu_clex(),

        "add" => cpu.fpu_add(a(0), a(1)),
        "sub" => cpu.fpu_sub(a(0), a(1)),
        "subr" => cpu.fpu_subr(a(0), a(1)),
        "mul" => cpu.fpu_mul(a(0), a(1)),
        "div" => cpu.fpu_div(a(0), a(1)),
        "divr" => cpu.fpu_divr(a(0), a(1)),

        // the C driver stashes the memory result somewhere visible
        "stcw16" => cpu.res = cpu.fpu_stcw16() as u32,
        "stenv32" => {
            let env = cpu.fpu_stenv32();
            cpu.res = env.control;
            cpu.op1 = env.status;
            cpu.op2 = env.tag;
        }
        "save32" => {
            let st = cpu.fpu_save32();
            cpu.res = st.env.control;
            cpu.op1 = st.env.status;
            cpu.fp[0] = Float80::from_le_bytes(st.regs[0]);
            cpu.fp[1] = Float80::from_le_bytes(st.regs[7]);
        }
        "restore32" => {
            let mut st = cpu.fpu_save32();
            st.env.status = 0x1234;
            st.env.control = 0x0fff;
            st.regs[3][0] ^= 0xff;
            cpu.fpu_restore32(&st);
        }

        other => {
            let mv = &MEMVALS[a(0)];
            let i16v = mv.bits64 as i16;
            let i32v = mv.bits64 as i32;
            let i64v = mv.bits64 as i64;
            let f32v = f32::from_bits(mv.bits64 as u32);
            let f64v = f64::from_bits(mv.bits64);
            let f80v = Float80::from_bits(mv.f80.0, mv.f80.1);
            match other {
                "ild16" => cpu.fpu_ild16(i16v),
                "ild32" => cpu.fpu_ild32(i32v),
                "ild64" => cpu.fpu_ild64(i64v),
                "ldm32" => cpu.fpu_ldm32(f32v),
                "ldm64" => cpu.fpu_ldm64(f64v),
                "ldm80" => cpu.fpu_ldm80(f80v),
                "ist16" => cpu.res = cpu.fpu_ist16() as u16 as u32,
                "ist32" => cpu.res = cpu.fpu_ist32() as u32,
                "ist64" => {
                    let o = cpu.fpu_ist64();
                    cpu.res = o as u32;
                    cpu.op1 = (o >> 32) as u32;
                }
                "stm32" => cpu.res = cpu.fpu_stm32().to_bits(),
                "stm64" => {
                    let b = cpu.fpu_stm64().to_bits();
                    cpu.res = b as u32;
                    cpu.op1 = (b >> 32) as u32;
                }
                "stm80" => cpu.fp[1] = cpu.fpu_stm80(),
                "icom16" => cpu.fpu_icom16(i16v),
                "icom32" => cpu.fpu_icom32(i32v),
                "comm32" => cpu.fpu_comm32(f32v),
                "comm64" => cpu.fpu_comm64(f64v),
                "iadd16" => cpu.fpu_iadd16(i16v),
                "isub16" => cpu.fpu_isub16(i16v),
                "isubr16" => cpu.fpu_isubr16(i16v),
                "imul16" => cpu.fpu_imul16(i16v),
                "idiv16" => cpu.fpu_idiv16(i16v),
                "idivr16" => cpu.fpu_idivr16(i16v),
                "iadd32" => cpu.fpu_iadd32(i32v),
                "isub32" => cpu.fpu_isub32(i32v),
                "isubr32" => cpu.fpu_isubr32(i32v),
                "imul32" => cpu.fpu_imul32(i32v),
                "idiv32" => cpu.fpu_idiv32(i32v),
                "idivr32" => cpu.fpu_idivr32(i32v),
                "addm32" => cpu.fpu_addm32(f32v),
                "subm32" => cpu.fpu_subm32(f32v),
                "subrm32" => cpu.fpu_subrm32(f32v),
                "mulm32" => cpu.fpu_mulm32(f32v),
                "divm32" => cpu.fpu_divm32(f32v),
                "divrm32" => cpu.fpu_divrm32(f32v),
                "addm64" => cpu.fpu_addm64(f64v),
                "subm64" => cpu.fpu_subm64(f64v),
                "subrm64" => cpu.fpu_subrm64(f64v),
                "mulm64" => cpu.fpu_mulm64(f64v),
                "divm64" => cpu.fpu_divm64(f64v),
                "divrm64" => cpu.fpu_divrm64(f64v),
                "ldcw16" => cpu.fpu_ldcw16(mv.bits64 as u16),
                unknown => panic!("unknown op {unknown}"),
            }
        }
    }
}

#[test]
fn fpu_state_matches_c_word_for_word() {
    let text = std::fs::read_to_string(FIXTURE)
        .unwrap_or_else(|e| panic!("cannot read {FIXTURE}: {e}\nRun tools/gen_fpu_reference.sh"));
    let states = build_states();

    let mut checks: u64 = 0;
    let mut failures: u64 = 0;
    let mut samples: Vec<String> = Vec::new();

    for line in text.lines() {
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let t: Vec<&str> = line.split_whitespace().collect();

        match t[0] {
            "P" => {
                let op = t[1];
                let n = t.len();
                // the state index is explicit in the fixture: the C driver
                // skips states for some ops (fyl2x), so a running counter
                // would drift and replay the op against the wrong state.
                let s: usize = t[2].parse().expect("state index");
                assert!(n >= 3 + STATE_WORDS, "short P line: {line}");
                let args = &t[3..n - STATE_WORDS];
                let want = &t[n - STATE_WORDS..];

                let mut cpu = states[s].clone();
                apply_op(&mut cpu, op, args);

                let got = dump_state(&cpu);
                for (i, (g, w)) in got.iter().zip(want.iter()).enumerate() {
                    checks += 1;
                    // the C prints the top field both on its own and inside
                    // fsw; both must match
                    if g != w {
                        failures += 1;
                        if samples.len() < 25 {
                            samples.push(format!(
                                "  {op} [{}] {}\n      C   : {}\n      Rust: {}",
                                args.join(" "),
                                word_name(i),
                                w,
                                g
                            ));
                        }
                    }
                }
            }
            "F" => {
                // state index, then 12 inputs, 6 macro results, 8
                // post-collapse words and 3 post-expand words
                assert_eq!(t.len(), 2 + 29, "bad F line: {line}");
                let h = |i: usize| u32::from_str_radix(t[i + 1], 16).unwrap();
                // start from the same pooled state the C driver used, so the
                // bits of `eflags` above the bitfields (which collapse_flags
                // leaves alone) are present too. The twelve inputs printed on
                // the line are exactly this state's flag fields.
                let s: usize = t[1].parse().expect("state index");
                let mut cpu = states[s].clone();
                cpu.res = h(1);
                cpu.op1 = h(2);
                cpu.op2 = h(3);
                cpu.flags_res = h(4) as u8;
                cpu.cf = h(5) as u8;
                cpu.of = h(6) as u8;
                cpu.zf = h(7) != 0;
                cpu.sf = h(8) != 0;
                cpu.pf = h(9) != 0;
                cpu.af = h(10) != 0;
                cpu.cf_bit = h(11) != 0;
                cpu.of_bit = h(12) != 0;

                let macros = [
                    cpu.zf_eval() as u32,
                    cpu.sf_eval() as u32,
                    cpu.cf_eval() as u32,
                    cpu.of_eval() as u32,
                    cpu.pf_eval() as u32,
                    cpu.af_eval() as u32,
                ];
                let names = ["ZF", "SF", "CF", "OF", "PF", "AF"];
                for (i, got) in macros.iter().enumerate() {
                    checks += 1;
                    if *got != h(13 + i) {
                        failures += 1;
                        if samples.len() < 25 {
                            samples.push(format!(
                                "  flags {line}\n      macro {}: C {} Rust {got}",
                                names[i],
                                t[14 + i]
                            ));
                        }
                    }
                }

                cpu.collapse_flags();
                let after = [
                    cpu.zf as u32,
                    cpu.sf as u32,
                    cpu.pf as u32,
                    cpu.af as u32,
                    cpu.cf_bit as u32,
                    cpu.of_bit as u32,
                    cpu.flags_res as u32,
                    cpu.eflags(),
                ];
                let after_names = [
                    "zf",
                    "sf",
                    "pf",
                    "af",
                    "cf_bit",
                    "of_bit",
                    "flags_res",
                    "eflags",
                ];
                for (i, got) in after.iter().enumerate() {
                    checks += 1;
                    if *got != h(19 + i) {
                        failures += 1;
                        if samples.len() < 25 {
                            samples.push(format!(
                                "  collapse {line}\n      {}: C {:x} Rust {got:x}",
                                after_names[i],
                                h(19 + i)
                            ));
                        }
                    }
                }

                cpu.expand_flags();
                let expanded = [cpu.cf as u32, cpu.of as u32, cpu.flags_res as u32];
                for (i, got) in expanded.iter().enumerate() {
                    checks += 1;
                    if *got != h(27 + i) {
                        failures += 1;
                        if samples.len() < 25 {
                            samples.push(format!(
                                "  expand {line}\n      field {i}: C {:x} Rust {got:x}",
                                h(27 + i)
                            ));
                        }
                    }
                }
            }
            other => panic!("unknown record kind {other}"),
        }
    }

    assert!(
        failures == 0,
        "{failures} of {checks} state words differ from the C reference; first {}:\n{}",
        samples.len(),
        samples.join("\n")
    );
    assert!(
        checks > 400_000,
        "only {checks} checks ran - fixture looks truncated"
    );
    eprintln!("fpu differential: {checks} state words matched the C reference exactly");
}

/// The rounding control in `fcw` and the `float80` rounding mode are the same
/// enum, and `fpu_ldcw16` is what connects them. Pinned here because the whole
/// differential test depends on that mapping.
#[test]
fn rc_encoding_matches_the_f80_rounding_mode() {
    let mut cpu = CpuState::default();
    for (rc, mode) in [
        (0u8, RoundingMode::RoundToNearest),
        (1, RoundingMode::RoundDown),
        (2, RoundingMode::RoundUp),
        (3, RoundingMode::RoundChop),
    ] {
        cpu.fpu_ldcw16((rc as u16) << 10);
        assert_eq!(ish_emu::float80::rounding_mode(), mode, "rc = {rc}");
    }
    ish_emu::float80::set_rounding_mode(RoundingMode::RoundToNearest);
}

/// The seven x87 constants must be the exact bit patterns from `fpu_consts[]`.
#[test]
fn fpu_const_table_is_bit_identical() {
    let expected = [
        (0x8000_0000_0000_0000u64, 0x3fffu16),
        (0xd49a_784b_cd1b_8afe, 0x4000),
        (0xb8aa_3b29_5c17_f0bc, 0x3fff),
        (0xc90f_daa2_2168_c235, 0x4000),
        (0x9a20_9a84_fbcf_f799, 0x3ffd),
        (0xb172_17f7_d1cf_79ac, 0x3ffe),
        (0x0000_0000_0000_0000, 0x0000),
    ];
    for (i, (signif, sign_exp)) in expected.iter().enumerate() {
        assert_eq!(
            FPU_CONSTS[i],
            Float80::from_bits(*signif, *sign_exp),
            "constant {i}"
        );
    }
}

/// The state generator here has to produce byte-identical states to the C
/// driver's, otherwise every other comparison in this file is meaningless.
#[test]
fn state_pool_rebuilds_deterministically() {
    let a = build_states();
    let b = build_states();
    assert_eq!(a.len(), 24);
    assert_eq!(a, b);
    // state 0 is hand-picked: pool[(0*8+i) % 15]
    for (i, p) in POOL.iter().take(8).enumerate() {
        assert_eq!(a[0].fp[i], Float80::from_bits(p.0, p.1), "state0 fp[{i}]");
    }
    // top cycles 0..7 and the fsw write must not disturb it
    for (s, cpu) in a.iter().enumerate() {
        assert_eq!(cpu.top(), (s % 8) as u8, "state {s} top");
        assert_eq!(
            cpu.fsw & FSW_TOP_MASK,
            ((s % 8) as u16) << 11,
            "state {s} top in fsw"
        );
    }
}
