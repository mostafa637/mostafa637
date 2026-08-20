package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func shiftFlags(inst machinecode.Instruction) []byte {
	if inst.ShiftKind >= machinecode.ShiftROL && inst.ShiftKind <= machinecode.ShiftRCR {
		return rotateFlags(inst)
	}
	active := shiftActive()
	out := appendShiftFlag(nil, shiftCarry(inst), active, 0)
	out = appendShiftFlag(out, flagZero(localCode(19)), active, 6)
	out = appendShiftFlag(out, flagSignWidth(inst.Width), active, 7)
	out = appendShiftFlag(out, flagParity(localCode(19)), active, 2)
	return appendShiftFlag(out, shiftOverflow(inst), oneCount(), 11)
}

func rotateFlags(inst machinecode.Instruction) []byte {
	active := shiftActive()
	out := appendShiftFlag(nil, rotateCarry(inst), active, 0)
	return appendShiftFlag(out, rotateOverflow(inst), oneCount(), 11)
}

func shiftActive() []byte {
	return append(localCode(18), 0x50, 0x45)
}

func oneCount() []byte {
	out := append(localCode(18), constCode(1)...)
	return append(out, 0x51)
}

func appendShiftFlag(out, value, cond []byte, bit byte) []byte {
	out = append(out, value...)
	out = append(out, localCode(16)...)
	out = append(out, constCode(int64(bit))...)
	out = append(out, 0x88)
	out = append(out, constCode(1)...)
	out = append(out, 0x83)
	out = append(out, cond...)
	out = append(out, 0x1b)
	out = append(out, constCode(int64(bit))...)
	out = append(out, 0x86, 0x20, 16, 0x84, 0x21, 16)
	return out
}
