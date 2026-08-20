package wasmjit

import "golang.org/x/arch/x86/x86asm"

func decodeReg(arg x86asm.Arg) (int16, bool) {
	reg, ok := arg.(x86asm.Reg)
	if !ok {
		return 0, false
	}
	return normalizeReg(reg)
}

func normalizeReg(reg x86asm.Reg) (int16, bool) {
	switch {
	case reg >= x86asm.AL && reg <= x86asm.BL:
		return int16(reg - x86asm.AL), true
	case reg >= x86asm.AH && reg <= x86asm.BH:
		return int16(reg - x86asm.AH), true
	case reg >= x86asm.SPB && reg <= x86asm.DIB:
		return int16(reg-x86asm.SPB) + 4, true
	case reg >= x86asm.R8B && reg <= x86asm.R15B:
		return int16(reg-x86asm.R8B) + 8, true
	case reg >= x86asm.AX && reg <= x86asm.DI:
		return int16(reg - x86asm.AX), true
	case reg >= x86asm.R8W && reg <= x86asm.R15W:
		return int16(reg-x86asm.R8W) + 8, true
	case reg >= x86asm.EAX && reg <= x86asm.EDI:
		return int16(reg - x86asm.EAX), true
	case reg >= x86asm.R8L && reg <= x86asm.R15L:
		return int16(reg-x86asm.R8L) + 8, true
	case reg >= x86asm.RAX && reg <= x86asm.R15:
		return int16(reg - x86asm.RAX), true
	default:
		return 0, false
	}
}

func decodeImm(arg x86asm.Arg) (int64, bool) {
	imm, ok := arg.(x86asm.Imm)
	return int64(imm), ok
}
