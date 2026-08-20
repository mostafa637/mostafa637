package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

func decodeBranch(inst x86asm.Inst, address uint64, op machinecode.Op, cond uint8) (machinecode.Instruction, error) {
	rel, ok := inst.Args[0].(x86asm.Rel)
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	nextPC := address + uint64(inst.Len)
	return machinecode.Instruction{Op: op, Target: uint64(int64(nextPC) + int64(rel)), Fallthrough: nextPC, Cond: cond}, nil
}

func decodeCondition(op x86asm.Op) (uint8, bool) {
	for index, item := range []x86asm.Op{x86asm.JO, x86asm.JB, x86asm.JE, x86asm.JBE, x86asm.JS, x86asm.JP, x86asm.JL, x86asm.JLE} {
		if op == item {
			return uint8(index), true
		}
	}
	for index, item := range []x86asm.Op{x86asm.JNO, x86asm.JAE, x86asm.JNE, x86asm.JA, x86asm.JNS, x86asm.JNP, x86asm.JGE, x86asm.JG} {
		if op == item {
			return uint8(index) | 0x80, true
		}
	}
	return 0, false
}

func decodeReturn(inst x86asm.Inst, address uint64) (machinecode.Instruction, error) {
	var cleanup int64
	if inst.Args[0] != nil {
		imm, ok := inst.Args[0].(x86asm.Imm)
		if !ok || imm < 0 {
			return machinecode.Instruction{}, ErrUnsupported
		}
		cleanup = int64(imm)
	}
	return machinecode.Instruction{Op: machinecode.OpRET, Imm: cleanup, Fallthrough: address + uint64(inst.Len)}, nil
}
