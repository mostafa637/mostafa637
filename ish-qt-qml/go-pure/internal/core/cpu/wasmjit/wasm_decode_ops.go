package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

func decodeMove(inst x86asm.Inst) (machinecode.Instruction, error) {
	dst, ok := decodeReg(inst.Args[0])
	if !ok {
		if base, disp, valid := decodeMem(inst.Args[0]); valid {
			src, srcOK := decodeReg(inst.Args[1])
			if srcOK {
				return machinecode.Instruction{Op: machinecode.OpStore64, Dst: base, Src: src, Imm: disp}, nil
			}
		}
		return machinecode.Instruction{}, ErrUnsupported
	}
	if src, ok := decodeReg(inst.Args[1]); ok {
		return machinecode.Instruction{Op: machinecode.OpMOVReg, Dst: dst, Src: src}, nil
	}
	if base, disp, ok := decodeMem(inst.Args[1]); ok {
		return machinecode.Instruction{Op: machinecode.OpLoad64, Dst: dst, Src: base, Imm: disp}, nil
	}
	imm, ok := decodeImm(inst.Args[1])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	return machinecode.Instruction{Op: machinecode.OpMOVImm, Dst: dst, Imm: imm}, nil
}

func decodeArithmetic(inst x86asm.Inst, op machinecode.Op) (machinecode.Instruction, error) {
	dst, ok := decodeReg(inst.Args[0])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	if src, ok := decodeReg(inst.Args[1]); ok {
		return machinecode.Instruction{Op: regOp(op), Dst: dst, Src: src}, nil
	}
	imm, ok := decodeImm(inst.Args[1])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	return machinecode.Instruction{Op: op, Dst: dst, Imm: imm}, nil
}

func regOp(op machinecode.Op) machinecode.Op {
	switch op {
	case machinecode.OpADDImm:
		return machinecode.OpADDReg
	case machinecode.OpSUBImm:
		return machinecode.OpSUBReg
	default:
		return op
	}
}
