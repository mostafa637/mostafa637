package cpu

import "golang.org/x/arch/arm64/arm64asm"

func arm64Nop(*ARM64State, arm64asm.Inst) error {
	return nil
}

func arm64Branch(s *ARM64State, inst arm64asm.Inst) error {
	rel, ok := inst.Args[0].(arm64asm.PCRel)
	if !ok {
		return ErrARM64Unsupported
	}
	if inst.Op == arm64asm.BL {
		s.Regs[30] = s.PC
	}
	s.PC = uint64(int64(s.PC-4) + int64(rel))
	return nil
}

func arm64RegisterBranch(s *ARM64State, inst arm64asm.Inst) error {
	ref, ok := arm64RegArg(inst.Args[0])
	if !ok {
		return ErrARM64Unsupported
	}
	s.PC = s.read(ref)
	return nil
}

func init() {
	registerARM64(arm64asm.NOP, arm64Nop)
	registerARM64(arm64asm.B, arm64Branch)
	registerARM64(arm64asm.BL, arm64Branch)
	registerARM64(arm64asm.BR, arm64RegisterBranch)
	registerARM64(arm64asm.RET, arm64RegisterBranch)
}
