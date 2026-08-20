package cpu

import "golang.org/x/arch/arm64/arm64asm"

func arm64AddSubImmediate(s *ARM64State, inst arm64asm.Inst) error {
	if _, ok := inst.Args[2].(arm64asm.ImmShift); !ok {
		return ErrARM64Unsupported
	}
	width := arm64Width(inst.Enc)
	subtract := inst.Op == arm64asm.SUB || inst.Op == arm64asm.SUBS
	spDest := inst.Op == arm64asm.ADD || inst.Op == arm64asm.SUB
	dst := arm64EncodedReg(inst.Enc, 0, spDest)
	src := arm64EncodedReg(inst.Enc, 5, true)
	value := arm64Immediate(inst.Enc)
	result := arm64AddValue(s.read(src), value, subtract, width)
	s.write(dst, result)
	if inst.Op == arm64asm.ADDS || inst.Op == arm64asm.SUBS {
		s.setNZ(result)
	}
	return nil
}

func arm64Immediate(enc uint32) uint64 {
	value := uint64((enc >> 10) & 0xfff)
	if enc&(1<<22) != 0 {
		return value << 12
	}
	return value
}

func arm64AddValue(left, right uint64, subtract bool, width uint8) uint64 {
	if subtract {
		left -= right
	} else {
		left += right
	}
	if width == 4 {
		return uint64(uint32(left))
	}
	return left
}

func init() {
	registerARM64(arm64asm.ADD, arm64AddSubImmediate)
	registerARM64(arm64asm.SUB, arm64AddSubImmediate)
	registerARM64(arm64asm.ADDS, arm64AddSubImmediate)
	registerARM64(arm64asm.SUBS, arm64AddSubImmediate)
}
