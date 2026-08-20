package cpu

import (
	"fmt"

	"golang.org/x/arch/arm64/arm64asm"
)

func DecodeARM64(code []byte) (arm64asm.Inst, error) {
	if len(code) < 4 {
		return arm64asm.Inst{}, ErrARM64InvalidInstruction
	}
	inst, err := arm64asm.Decode(code[:4])
	if err != nil {
		return arm64asm.Inst{}, fmt.Errorf("%w: %v", ErrARM64InvalidInstruction, err)
	}
	return inst, nil
}

func DisassembleARM64(memory *Memory64, pc Address64) (arm64asm.Inst, error) {
	if memory == nil {
		return arm64asm.Inst{}, ErrUnmapped
	}
	var code [4]byte
	if err := memory.Read(pc, code[:]); err != nil {
		return arm64asm.Inst{}, err
	}
	return DecodeARM64(code[:])
}
