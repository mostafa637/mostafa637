package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

func decodeMUL(inst x86asm.Inst, op x86asm.Op) (machinecode.Instruction, error) {
	out := machinecode.Instruction{Op: mulOp(op)}
	operand, ok := inst.Args[0].(x86asm.Reg)
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	reg, _ := decodeReg(operand)
	out.Dst = 0
	out.Src = reg
	out.Width = uint8(regWidth(operand))
	out.Signed = op == x86asm.IMUL
	return out, nil
}

func mulOp(op x86asm.Op) machinecode.Op {
	if op == x86asm.IMUL {
		return machinecode.OpIMUL64
	}
	return machinecode.OpMUL64
}

func decodeDIV(inst x86asm.Inst, op x86asm.Op) (machinecode.Instruction, error) {
	out := machinecode.Instruction{Op: machinecode.OpDIV64}
	operand, ok := inst.Args[0].(x86asm.Reg)
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	reg, _ := decodeReg(operand)
	out.Dst = 0
	out.Src = reg
	out.Width = uint8(regWidth(operand))
	out.Signed = op == x86asm.IDIV
	return out, nil
}

func decodeExplicitIMUL(inst x86asm.Inst) (machinecode.Instruction, error) {
	count := 0
	for i := 0; i < len(inst.Args); i++ {
		if inst.Args[i] != nil {
			count++
		}
	}
	if count != 2 && count != 3 {
		return machinecode.Instruction{}, ErrUnsupported
	}
	dst, ok := decodeReg(inst.Args[0])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	out := machinecode.Instruction{Op: machinecode.OpIMUL64, Dst: dst, Src: -1}
	regArg, _ := inst.Args[0].(x86asm.Reg)
	out.Width = uint8(regWidth(regArg))
	left, ok := decodeReg(inst.Args[1])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	if count == 3 {
		if imm, ok := decodeImm(inst.Args[2]); ok {
			out.Imm = imm
			out.MulSource = int16(left)
			return out, nil
		}
		return machinecode.Instruction{}, ErrUnsupported
	}
	out.MulSource = int16(left)
	return out, nil
}
