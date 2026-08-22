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
	conds := map[x86asm.Op]uint8{
		x86asm.JO: 0, x86asm.JNO: 1, x86asm.JB: 2, x86asm.JAE: 3,
		x86asm.JE: 4, x86asm.JNE: 5, x86asm.JBE: 6, x86asm.JA: 7,
		x86asm.JS: 8, x86asm.JNS: 9, x86asm.JP: 10, x86asm.JNP: 11,
		x86asm.JL: 12, x86asm.JGE: 13, x86asm.JLE: 14, x86asm.JG: 15,
	}
	c, ok := conds[op]
	return c, ok
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
