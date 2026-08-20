package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

func decodeTest(inst x86asm.Inst) (machinecode.Instruction, error) {
	dst, ok := decodeReg(inst.Args[0])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	if src, ok := decodeReg(inst.Args[1]); ok {
		return machinecode.Instruction{Op: machinecode.OpTestReg, Dst: dst, Src: src}, nil
	}
	imm, ok := decodeImm(inst.Args[1])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	return machinecode.Instruction{Op: machinecode.OpTestImm, Dst: dst, Imm: imm}, nil
}
