package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

func decodePush(inst x86asm.Inst, address uint64) (machinecode.Instruction, error) {
	if reg, ok := decodeReg(inst.Args[0]); ok {
		return machinecode.Instruction{Op: machinecode.OpPush64, Src: reg, Width: 8}, nil
	}
	if imm, ok := decodeImm(inst.Args[0]); ok {
		return machinecode.Instruction{Op: machinecode.OpPush64, Src: -1, Imm: imm, Width: 8, MemBase: -1, MemIndex: -1}, nil
	}
	ref, ok := decodeMem(inst.Args[0], address+uint64(inst.Len))
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	return memoryInstruction(machinecode.OpPush64, -1, -1, ref), nil
}

func decodePop(inst x86asm.Inst, address uint64) (machinecode.Instruction, error) {
	if reg, ok := decodeReg(inst.Args[0]); ok {
		return machinecode.Instruction{Op: machinecode.OpPop64, Dst: reg, Width: 8}, nil
	}
	ref, ok := decodeMem(inst.Args[0], address+uint64(inst.Len))
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	return memoryInstruction(machinecode.OpPop64, -1, -1, ref), nil
}
