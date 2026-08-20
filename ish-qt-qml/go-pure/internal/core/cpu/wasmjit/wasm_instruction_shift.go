package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitShift(inst machinecode.Instruction) []byte {
	out := prepareShift(inst)
	out = append(out, shiftValue(inst)...)
	out = append(out, shiftWrite(inst)...)
	return append(out, shiftFlags(inst)...)
}

func prepareShift(inst machinecode.Instruction) []byte {
	out := append(localCode(inst.Dst), 0x21, 20)
	out = append(out, localCode(inst.Dst)...)
	out = append(out, constCode(widthMask(inst.Width))...)
	out = append(out, 0x83, 0x21, 17)
	return append(out, shiftCount(inst)...)
}

func shiftCount(inst machinecode.Instruction) []byte {
	mask := int64(0x1f)
	if inst.Width == 8 {
		mask = 0x3f
	}
	var out []byte
	if inst.Src >= 0 {
		out = append(localCode(inst.Src), constCode(mask)...)
		out = append(out, 0x83, 0x21, 18)
	} else {
		out = append(constCode(inst.Imm&mask), 0x21, 18)
	}
	return appendRotateCount(out, inst)
}

func appendRotateCount(out []byte, inst machinecode.Instruction) []byte {
	if inst.ShiftKind != machinecode.ShiftRCL && inst.ShiftKind != machinecode.ShiftRCR {
		return out
	}
	bits := int64(inst.Width)*8 + 1
	out = append(out, localCode(18)...)
	out = append(out, constCode(bits)...)
	return append(out, 0x82, 0x21, 18)
}

func shiftValue(inst machinecode.Instruction) []byte {
	switch inst.ShiftKind {
	case machinecode.ShiftSHL:
		return binaryShift(0x86)
	case machinecode.ShiftSHR:
		return binaryShift(0x88)
	case machinecode.ShiftSAR:
		return signedShift(inst)
	case machinecode.ShiftROL, machinecode.ShiftROR:
		return rotateShift(inst, inst.ShiftKind == machinecode.ShiftROR)
	case machinecode.ShiftRCL, machinecode.ShiftRCR:
		return rotateCarryShift(inst, inst.ShiftKind == machinecode.ShiftRCR)
	default:
		return nil
	}
}

func binaryShift(op byte) []byte {
	out := append(localCode(17), localCode(18)...)
	return append(out, op, 0x21, 19)
}

func signedShift(inst machinecode.Instruction) []byte {
	out := signExtendValue(inst.Width)
	out = append(out, localCode(18)...)
	return append(out, 0x87, 0x21, 19)
}

func signExtendValue(width uint8) []byte {
	shift := int64(64 - uint(width)*8)
	if shift == 0 {
		return localCode(17)
	}
	out := append(localCode(17), constCode(shift)...)
	out = append(out, 0x86)
	out = append(out, constCode(shift)...)
	return append(out, 0x87)
}
