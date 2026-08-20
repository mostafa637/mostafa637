package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func flagSignWidth(width uint8) []byte {
	return bitValue(localCode(19), int64(width)*8-1)
}

func shiftCarry(inst machinecode.Instruction) []byte {
	out := localCode(17)
	if inst.ShiftKind == machinecode.ShiftSHL {
		out = append(out, constCode(int64(inst.Width)*8)...)
		out = append(out, localCode(18)...)
		out = append(out, 0x7d)
	} else {
		out = append(out, localCode(18)...)
		out = append(out, 0x42, 1, 0x7d)
	}
	out = append(out, 0x88)
	out = append(out, constCode(1)...)
	return append(out, 0x83)
}

func rotateCarry(inst machinecode.Instruction) []byte {
	if inst.ShiftKind == machinecode.ShiftROL {
		return bitValue(localCode(19), 0)
	}
	if inst.ShiftKind == machinecode.ShiftRCL {
		return carryFromTop(localCode(20), int64(inst.Width)*8)
	}
	if inst.ShiftKind == machinecode.ShiftRCR {
		return carryFromCount(localCode(20))
	}
	return bitValue(localCode(19), int64(inst.Width)*8-1)
}

func carryFromTop(value []byte, bits int64) []byte {
	out := append(value, constCode(bits)...)
	out = append(out, localCode(18)...)
	out = append(out, 0x7d, 0x88)
	out = append(out, constCode(1)...)
	return append(out, 0x83)
}

func carryFromCount(value []byte) []byte {
	out := append(value, localCode(18)...)
	out = append(out, constCode(1)...)
	out = append(out, 0x7d, 0x88)
	out = append(out, constCode(1)...)
	return append(out, 0x83)
}

func rotateOverflow(inst machinecode.Instruction) []byte {
	bits := int64(inst.Width) * 8
	left := bitValue(localCode(19), bits-1)
	if inst.ShiftKind == machinecode.ShiftROL {
		return xorBits(left, bitValue(localCode(19), 0))
	}
	if inst.ShiftKind == machinecode.ShiftRCL {
		return xorBits(left, bitValue(localCode(20), bits-1))
	}
	return xorBits(left, bitValue(localCode(19), bits-2))
}

func shiftOverflow(inst machinecode.Instruction) []byte {
	if inst.ShiftKind == machinecode.ShiftSAR {
		return constCode(0)
	}
	if inst.ShiftKind == machinecode.ShiftSHR {
		return bitValue(localCode(17), int64(inst.Width)*8-1)
	}
	return xorBits(bitValue(localCode(17), int64(inst.Width)*8-1),
		bitValue(localCode(19), int64(inst.Width)*8-1))
}

func bitValue(value []byte, bit int64) []byte {
	out := append(value, constCode(bit)...)
	out = append(out, 0x88)
	out = append(out, constCode(1)...)
	return append(out, 0x83)
}

func xorBits(left, right []byte) []byte {
	out := append(left, right...)
	return append(out, 0x85)
}
