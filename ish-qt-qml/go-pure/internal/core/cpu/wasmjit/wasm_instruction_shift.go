package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
)

func emitShift(inst machinecode.Instruction) []byte {
	if inst.ShiftKind >= machinecode.ShiftROL {
		return emitRotate(inst)
	}
	out := shiftPrepare(inst)
	out = append(out, shiftOp(inst)...)
	out = append(out, shiftWrite(inst)...)
	return append(out, shiftFlags(inst)...)
}

func shiftPrepare(inst machinecode.Instruction) []byte {
	out := localCode(inst.Dst)
	out = append(out, WasmOpLocalSet, 19)
	if inst.Src >= 0 {
		out = append(out, localCode(inst.Src)...)
	} else {
		out = append(out, constCode(inst.Imm)...)
	}
	out = append(out, constCode(63)...)
	out = append(out, WasmOpI64And, WasmOpLocalSet, 21)
	return out
}

func shiftOp(inst machinecode.Instruction) []byte {
	out := localCode(19)
	out = append(out, localCode(21)...)
	switch inst.ShiftKind {
	case machinecode.ShiftSHL:
		out = append(out, WasmOpI64Shl)
	case machinecode.ShiftSHR:
		out = append(out, WasmOpI64ShrU)
	case machinecode.ShiftSAR:
		out = append(out, WasmOpI64ShrS)
	}
	return append(out, WasmOpLocalSet, 20)
}

func shiftWrite(inst machinecode.Instruction) []byte {
	return append(localCode(20), WasmOpLocalSet, byte(inst.Dst))
}

func shiftFlags(inst machinecode.Instruction) []byte {
	out := localCode(21)
	out = append(out, WasmOpI64Eqz)
	out = append(out, WasmOpIf, 0x40, WasmOpElse)
	out = append(out, shiftCarry(inst)...)
	out = append(out, WasmOpLocalSet, 17)
	out = append(out, shiftOverflow(inst)...)
	out = append(out, WasmOpLocalSet, 18)
	out = append(out, updateShiftFlags(inst)...)
	out = append(out, WasmOpEnd)
	return out
}

func shiftCarry(inst machinecode.Instruction) []byte {
	out := localCode(19)
	if inst.ShiftKind == machinecode.ShiftSHL {
		out = append(out, constCode(64)...)
		out = append(out, localCode(21)...)
		out = append(out, WasmOpI64Sub, WasmOpI64ShrU)
	} else {
		out = append(out, localCode(21)...)
		out = append(out, constCode(1)...)
		out = append(out, WasmOpI64Sub, WasmOpI64ShrU)
	}
	out = append(out, constCode(1)...)
	return append(out, WasmOpI64And)
}

func shiftOverflow(inst machinecode.Instruction) []byte {
	out := localCode(21)
	out = append(out, constCode(1)...)
	out = append(out, WasmOpI64Ne)
	out = append(out, WasmOpIf, 0x7e)
	out = append(out, constCode(0)...)
	out = append(out, WasmOpElse)
	if inst.ShiftKind == machinecode.ShiftSHL {
		out = append(out, carryFromTop(localCode(20), 63)...)
		out = append(out, carryFromTop(localCode(19), 63)...)
		out = append(out, WasmOpI64Xor)
	} else if inst.ShiftKind == machinecode.ShiftSHR {
		out = append(out, carryFromTop(localCode(19), 63)...)
	} else {
		out = append(out, constCode(0)...)
	}
	out = append(out, WasmOpEnd)
	return out
}

func updateShiftFlags(inst machinecode.Instruction) []byte {
	mask := int64(^(1 | 1<<11 | 1<<6 | 1<<7 | 1<<2))
	out := localCode(16)
	out = append(out, constCode(mask)...)
	out = append(out, WasmOpI64And)
	out = append(out, localCode(17)...)
	out = append(out, localCode(18)...)
	out = append(out, constCode(11)...)
	out = append(out, WasmOpI64Shl, WasmOpI64Or, WasmOpI64Or)
	out = append(out, flagZero(localCode(20))...)
	out = append(out, constCode(6)...)
	out = append(out, WasmOpI64Shl, WasmOpI64Or)
	out = append(out, flagSign(localCode(20))...)
	out = append(out, constCode(7)...)
	out = append(out, WasmOpI64Shl, WasmOpI64Or)
	out = append(out, flagParity(localCode(20))...)
	out = append(out, constCode(2)...)
	out = append(out, WasmOpI64Shl, WasmOpI64Or)
	return append(out, WasmOpLocalSet, 16)
}

func carryFromTop(val []byte, bit byte) []byte {
	out := append(val, constCode(int64(bit))...)
	out = append(out, WasmOpI64ShrU)
	out = append(out, constCode(1)...)
	return append(out, WasmOpI64And)
}
