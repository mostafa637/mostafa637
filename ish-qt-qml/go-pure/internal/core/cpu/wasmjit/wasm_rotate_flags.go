package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func rotateFlags(inst machinecode.Instruction) []byte {
	out := localCode(21)
	out = append(out, WasmOpI64Eqz, WasmOpIf, 0x40, WasmOpElse)
	if inst.ShiftKind == machinecode.ShiftRCL || inst.ShiftKind == machinecode.ShiftRCR {
		out = append(out, localCode(22)...)
	} else {
		out = append(out, rotateCarry(inst)...)
	}
	out = append(out, WasmOpLocalSet, 17)
	out = append(out, rotateOverflow(inst)...)
	out = append(out, WasmOpLocalSet, 18)
	out = append(out, updateRotateFlags()...)
	out = append(out, WasmOpEnd)
	return out
}

func rotateCarry(inst machinecode.Instruction) []byte {
	if inst.ShiftKind == machinecode.ShiftROL {
		return carryFromTop(localCode(20), 0)
	}
	return carryFromTop(localCode(20), 63)
}

func rotateOverflow(inst machinecode.Instruction) []byte {
	out := localCode(21)
	out = append(out, constCode(1)...)
	out = append(out, WasmOpI64Ne, WasmOpIf, 0x7e)
	out = append(out, constCode(0)...)
	out = append(out, WasmOpElse)
	if inst.ShiftKind == machinecode.ShiftROL || inst.ShiftKind == machinecode.ShiftRCL {
		out = append(out, localCode(17)...)
	} else {
		out = append(out, carryFromTop(localCode(20), 63)...)
		out = append(out, carryFromTop(localCode(20), 62)...)
		out = append(out, WasmOpI64Xor)
	}
	return append(out, WasmOpEnd)
}

func updateRotateFlags() []byte {
	mask := int64(^(1 | 1<<11))
	out := localCode(16)
	out = append(out, constCode(mask)...)
	out = append(out, WasmOpI64And)
	out = append(out, localCode(17)...)
	out = append(out, localCode(18)...)
	out = append(out, constCode(11)...)
	out = append(out, WasmOpI64Shl, WasmOpI64Or, WasmOpI64Or)
	return append(out, WasmOpLocalSet, 16)
}
