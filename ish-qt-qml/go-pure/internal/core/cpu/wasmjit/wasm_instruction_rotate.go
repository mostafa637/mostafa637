package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func rotateShift(inst machinecode.Instruction, right bool) []byte {
	bits := int64(inst.Width) * 8
	out := append(localCode(18), constCode(bits)...)
	out = append(out, 0x82, 0x21, 18)
	if right {
		return append(out, rotateRight(bits)...)
	}
	return append(out, rotateLeft(bits)...)
}

func rotateLeft(bits int64) []byte {
	out := append(localCode(17), localCode(18)...)
	out = append(out, 0x86)
	out = append(out, localCode(17)...)
	out = append(out, constCode(bits)...)
	out = append(out, localCode(18)...)
	out = append(out, 0x7d, 0x88, 0x84, 0x21, 19)
	return out
}

func rotateRight(bits int64) []byte {
	out := append(localCode(17), localCode(18)...)
	out = append(out, 0x88)
	out = append(out, localCode(17)...)
	out = append(out, constCode(bits)...)
	out = append(out, localCode(18)...)
	out = append(out, 0x7d, 0x86, 0x84, 0x21, 19)
	return out
}

func shiftWrite(inst machinecode.Instruction) []byte {
	if inst.Width != 1 && inst.Width != 2 {
		return append(localCode(19), 0x21, byte(inst.Dst))
	}
	out := append(localCode(20), constCode(^widthMask(inst.Width))...)
	out = append(out, 0x83)
	out = append(out, localCode(19)...)
	out = append(out, 0x84, 0x21, byte(inst.Dst))
	return out
}

func rotateCarryShift(inst machinecode.Instruction, right bool) []byte {
	bits := int64(inst.Width) * 8
	out := carryRotateTerms(bits, right)
	out = append(out, constCode(widthMask(inst.Width))...)
	out = append(out, 0x83)
	out = append(out, localCode(20)...)
	out = append(out, localCode(18)...)
	out = append(out, 0x50, 0x45, 0x1b, 0x21, 19)
	return out
}

func carryRotateTerms(bits int64, right bool) []byte {
	first, third := byte(0x86), byte(0x88)
	if right {
		first, third = third, first
	}
	out := safeShift(localCode(17), localCode(18), first)
	out = append(out, carryTerm(bits, right)...)
	out = append(out, 0x84)
	out = append(out, valueTerm(bits, third)...)
	return append(out, 0x84)
}

func carryTerm(bits int64, right bool) []byte {
	value := bitValue(localCode(16), 0)
	amount := append(localCode(18), constCode(1)...)
	if right {
		amount = append(constCode(bits), localCode(18)...)
	}
	if right {
		amount = append(amount, 0x7d)
	} else {
		amount = append(amount, 0x7d)
	}
	return safeShift(value, amount, 0x86)
}

func valueTerm(bits int64, op byte) []byte {
	amount := append(constCode(bits+1), localCode(18)...)
	amount = append(amount, 0x7d)
	return safeShift(localCode(17), amount, op)
}

func safeShift(value, amount []byte, op byte) []byte {
	out := append(value, amount...)
	out = append(out, 0x21, 19)
	out = append(out, localCode(19)...)
	out = append(out, op)
	out = append(out, constCode(0)...)
	out = append(out, localCode(19)...)
	out = append(out, constCode(64)...)
	out = append(out, 0x54, 0x1b)
	return out
}
