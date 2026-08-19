# x87 compare/status — implementation findings

## x86asm v0.30.0 corpus

The 32-bit x86asm corpus confirms these forms and operand conventions:

| Instruction | Example encoding | Operand/width | Stack effect |
|---|---|---|---|
| `FCOM m32` | `D8 11` | memory `MemBytes=4` | no pop |
| `FCOMP m32` | `D8 19` | memory `MemBytes=4` | pop one |
| `FCOM m64` | `DC 11` | memory `MemBytes=8` | no pop |
| `FCOMP m64` | `DC 19` | memory `MemBytes=8` | pop one |
| `FCOM ST(i)` | `D8 D0+i` | register | no pop |
| `FCOMP ST(i)` | `D8 D8+i` | register | pop one |
| `FCOMPP` | `DE D9` | implicit ST(0), ST(1) | pop two |
| `FUCOM ST(i)` | `DD E0+i` | register | no pop |
| `FUCOMP ST(i)` | `DD E8+i` | register | pop one |
| `FUCOMPP` | `DA E9` | implicit ST(0), ST(1) | pop two |
| `FCOMI ST(0),ST(i)` | `DB F0+i` | register | no pop, EFLAGS |
| `FUCOMI ST(0),ST(i)` | `DB E8+i` | register | no pop, EFLAGS |
| `FCOMIP ST(0),ST(i)` | `DF F0+i` | register | pop one, EFLAGS |
| `FUCOMIP ST(0),ST(i)` | `DF E8+i` | register | pop one, EFLAGS |
| `FNSTSW AX` | `DF E0` | AX | no pop |
| `FNSTSW m16` | `DD 38` | memory `MemBytes=2` | no pop |

The corpus uses x87 registers represented as x86asm register values (`F0`..`F7`), while Intel formatting adds implicit `ST(0)`/`ST(1)` operands for zero-operand forms.

## iSH upstream semantics

The upstream `fpu_compare` sets x87 condition bits as follows: `C1=0`; `C0 = ST(0) < operand`; `C3 = ST(0) == operand`; and unordered values set `C0=C2=C3=1`. `fpu_comparei` instead clears and sets integer flags: `CF = less`, `ZF = equal`, `PF = 0`, and unordered sets `CF=ZF=PF=1`.

References:

- Upstream iSH: `upstream/ish-ios/emu/fpu.c`, `fpu_compare` and `fpu_comparei`.
- x86asm corpus: `/tmp/go-pure-modcache/golang.org/x/arch@v0.30.0/x86/x86asm/testdata/decode.txt`, x87 compare/status region around lines 5902–6155.
- Intel-derived instruction reference: https://www.felixcloutier.com/x86/fcom:fcomp:fcompp
- Intel-derived status word reference: https://www.felixcloutier.com/x86/fnstsw:fnstcw

## Scope

The first implementation should cover register compare/pop and m32/m64 compare memory forms, plus `FCOMI/FUCOMI` and their pop variants if x86asm operand decoding is unambiguous. `FNSTSW` can serialize `FSW` after the condition-bit mapping is established. x87 exception masks and full invalid-operation exception delivery remain outside this focused pass.
