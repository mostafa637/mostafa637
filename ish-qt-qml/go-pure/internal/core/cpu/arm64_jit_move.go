package cpu

import "golang.org/x/arch/arm64/arm64asm"

func arm64MoveImmediate(s *ARM64State, inst arm64asm.Inst) error {
	imm, ok := inst.Args[1].(arm64asm.Imm64)
	if !ok {
		return ErrARM64Unsupported
	}
	value := imm.Imm
	if arm64Width(inst.Enc) == 4 {
		value = uint64(uint32(value))
	}
	destination := arm64EncodedReg(inst.Enc, 0, false)
	s.write(destination, value)
	return nil
}

func init() {
	registerARM64(arm64asm.MOV, arm64MoveImmediate)
	registerARM64(arm64asm.MOVZ, arm64MoveImmediate)
}
