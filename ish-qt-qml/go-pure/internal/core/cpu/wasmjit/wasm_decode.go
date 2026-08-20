package wasmjit

import (
	"fmt"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

func decodeX86(src []byte) ([]machinecode.Instruction, error) {
	var out []machinecode.Instruction
	for len(src) > 0 {
		inst, err := x86asm.Decode(src, 64)
		if err != nil {
			return nil, err
		}
		item, err := decodeX86Inst(inst)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
		src = src[inst.Len:]
	}
	return out, nil
}

func decodeX86Inst(inst x86asm.Inst) (machinecode.Instruction, error) {
	switch inst.Op {
	case x86asm.NOP:
		return machinecode.Instruction{Op: machinecode.OpNOP}, nil
	case x86asm.RET:
		return machinecode.Instruction{Op: machinecode.OpRET}, nil
	case x86asm.SYSCALL:
		return machinecode.Instruction{Op: machinecode.OpSyscall}, nil
	case x86asm.MOV:
		return decodeMove(inst)
	case x86asm.ADD:
		return decodeArithmetic(inst, machinecode.OpADDImm)
	case x86asm.SUB:
		return decodeArithmetic(inst, machinecode.OpSUBImm)
	case x86asm.AND:
		return decodeArithmetic(inst, machinecode.OpANDImm)
	case x86asm.OR:
		return decodeArithmetic(inst, machinecode.OpORImm)
	case x86asm.XOR:
		return decodeArithmetic(inst, machinecode.OpXORImm)
	case x86asm.CMP:
		return decodeArithmetic(inst, machinecode.OpCMPImm)
	default:
		return machinecode.Instruction{}, fmt.Errorf("wasmjit: unsupported x86 op %s", inst.Op)
	}
}
