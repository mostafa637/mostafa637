# x87 memory forms — implementation findings

## x86asm v0.30.0 probe

The 32-bit decoder returns `DataSize=32` and `AddrSize=32` for these x87 memory forms; the actual operand width is exposed by `MemBytes`:

| Instruction | Bytes | `MemBytes` | Operand order from x86asm |
|---|---|---:|---|
| `FLD m32` | `D9 /0` | 4 | `Args[0] = x86asm.Mem` |
| `FST m32` | `D9 /2` | 4 | `Args[0] = x86asm.Mem` |
| `FSTP m32` | `D9 /3` | 4 | `Args[0] = x86asm.Mem` |
| `FLD m64` | `DD /0` | 8 | `Args[0] = x86asm.Mem` |
| `FST m64` | `DD /2` | 8 | `Args[0] = x86asm.Mem` |
| `FSTP m64` | `DD /3` | 8 | `Args[0] = x86asm.Mem` |

For register forms, x86asm returns `x86asm.Reg` values in the FPU range (`F0` through `F7`), so the adapter must distinguish `x86asm.Mem` from `x86asm.Reg` before converting operands.

The SIB probe `D9 44 B3 04` is decoded as `FLD dword ptr [ebx+4*esi+0x4]` with `MemBytes=4`.

## Semantics references

The upstream iSH implementation converts `m32` through a C `float` and `m64` through a C `double`; it does not store raw x80 bytes for these forms. The Pure Go implementation follows this by converting m32 through `math.Float32frombits`/`math.Float32bits` and m64 through `math.Float64frombits`/`math.Float64bits`, then bridging through `fpu.Value.FromFloat64`/`ToFloat64`.

Reference: [Intel-derived ENTER/FPU instruction reference](https://www.felixcloutier.com/x86/)
Reference: upstream iSH source `upstream/ish-ios/emu/fpu.c` in this repository, memory load/store helpers `fpu_ldm32`, `fpu_ldm64`, `fpu_stm32`, and `fpu_stm64`.

## Result

Commit `9a4663f` adds the x86asm adapter, `FPUMemWidth`, direct guest-memory reads/writes, m32/m64 IEEE-754 conversion, SIB support, and decoder/executor tests. `m80` remains intentionally unsupported.
