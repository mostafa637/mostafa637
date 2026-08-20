package cpu

import "golang.org/x/arch/arm64/arm64asm"

type ARM64Handler func(*ARM64State, arm64asm.Inst) error

var arm64Handlers [1 << 16]ARM64Handler

func registerARM64(op arm64asm.Op, handler ARM64Handler) {
	arm64Handlers[uint16(op)] = handler
}

func dispatchARM64(state *ARM64State, inst arm64asm.Inst) error {
	handler := arm64Handlers[uint16(inst.Op)]
	if handler == nil {
		return ErrARM64Unsupported
	}
	return handler(state, inst)
}

func (s *ARM64State) StepARM64() error {
	pc := s.PC
	inst, err := DisassembleARM64(s.Memory, Address64(pc))
	if err != nil {
		return err
	}
	s.PC = pc + 4
	return dispatchARM64(s, inst)
}
