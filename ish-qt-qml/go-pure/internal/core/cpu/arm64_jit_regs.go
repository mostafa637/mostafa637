package cpu

import "golang.org/x/arch/arm64/arm64asm"

type arm64RegRef struct {
	index uint8
	sp    bool
	zero  bool
}

func arm64RegArg(arg arm64asm.Arg) (arm64RegRef, bool) {
	if reg, ok := arg.(arm64asm.Reg); ok {
		return arm64Reg(reg)
	}
	if reg, ok := arg.(arm64asm.RegSP); ok {
		return arm64RegSP(reg)
	}
	return arm64RegRef{}, false
}

func arm64Reg(reg arm64asm.Reg) (arm64RegRef, bool) {
	if reg >= arm64asm.X0 && reg <= arm64asm.X30 {
		return arm64RegRef{index: uint8(reg - arm64asm.X0)}, true
	}
	if reg == arm64asm.XZR || reg == arm64asm.WZR {
		return arm64RegRef{zero: true}, true
	}
	if reg >= arm64asm.W0 && reg <= arm64asm.W30 {
		return arm64RegRef{index: uint8(reg - arm64asm.W0)}, true
	}
	return arm64RegRef{}, false
}

func arm64RegSP(reg arm64asm.RegSP) (arm64RegRef, bool) {
	value := arm64asm.Reg(reg)
	if value == arm64asm.SP || value == arm64asm.WSP {
		return arm64RegRef{sp: true}, true
	}
	return arm64Reg(value)
}

func arm64EncodedReg(enc uint32, shift uint32, sp bool) arm64RegRef {
	field := (enc >> shift) & 31
	if field == 31 {
		if sp {
			return arm64RegRef{sp: true}
		}
		return arm64RegRef{zero: true}
	}
	return arm64RegRef{index: uint8(field)}
}

func arm64Width(enc uint32) uint8 {
	if enc&(1<<31) != 0 {
		return 8
	}
	return 4
}
