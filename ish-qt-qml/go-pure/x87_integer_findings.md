# x87 integer memory — implementation findings

## x86asm v0.30.0 probe

All tested forms decode with `DataSize=32` and `AddrSize=32`; the memory width is exposed by `MemBytes`:

| Instruction | Encoding family | `MemBytes` | Meaning |
|---|---|---:|---|
| `FILD m16/m32/m64` | `DF/DB/DF /0` | 2/4/8 | signed integer load and push |
| `FIST m16/m32/m64` | `DF/DB/DF /2` | 2/4/8 | rounded signed integer store |
| `FISTP m16/m32/m64` | `DF/DB/DF /3` | 2/4/8 | rounded signed integer store and pop |
| `FISTTP m16/m32/m64` | `DF/DB/DF /1` | 2/4/8 | truncating signed integer store and pop |

The first operand is `x86asm.Mem`; register-only forms are not valid for this family. SIB and segment addressing are handled through the existing `x86Operand32` helper.

## Semantics reference

The upstream iSH implementation uses signed `fpu_ild16`, `fpu_ild32`, and `fpu_ild64` loads. `fpu_ist16` and `fpu_ist32` convert using the current x87 integer conversion and replace an out-of-range result with `INT16_MIN` or `INT32_MIN`; `fpu_ist64` stores the full signed 64-bit result. The Pure Go implementation follows this policy. `FISTTP` uses a Pure Go wrapper that converts through `ToFloat64`, handles NaN and int64 saturation, and applies `math.Trunc`; this avoids an incorrect `ToInt64RoundZero` result for fractional values in the current x80 dependency.

Reference: [upstream iSH fpu.c](../upstream/ish-ios/emu/fpu.c), integer load/store helpers around `fpu_ild16`, `fpu_ild32`, `fpu_ild64`, `fpu_ist16`, `fpu_ist32`, and `fpu_ist64`.
Reference: [Intel-derived x86 instruction reference](https://www.felixcloutier.com/x86/).

## Scope

`m80` integer forms remain intentionally unsupported in this commit. The implementation does not yet model x87 exception/status flags or programmable rounding-control state; the existing pure-Go `fpu.Value.ToInt64` policy is used for `FIST/FISTP`, while `ToInt64RoundZero` is used for `FISTTP`.
