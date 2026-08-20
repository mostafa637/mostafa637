package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

func decodeMove(inst x86asm.Inst, address uint64) (machinecode.Instruction, error) {
	nextPC := address + uint64(inst.Len)
	dst, ok := decodeReg(inst.Args[0])
	if !ok {
		if ref, valid := decodeMem(inst.Args[0], nextPC); valid {
			src, srcOK := decodeReg(inst.Args[1])
			if srcOK {
				return memoryInstruction(machinecode.OpStore64, -1, src, ref), nil
			}
		}
		return machinecode.Instruction{}, ErrUnsupported
	}
	if src, ok := decodeReg(inst.Args[1]); ok {
		return machinecode.Instruction{Op: machinecode.OpMOVReg, Dst: dst, Src: src}, nil
	}
	if ref, ok := decodeMem(inst.Args[1], nextPC); ok {
		return memoryInstruction(machinecode.OpLoad64, dst, -1, ref), nil
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
