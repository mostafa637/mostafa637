package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitRotate(inst machinecode.Instruction) []byte {
	out := rotatePrepare(inst)
	if inst.ShiftKind == machinecode.ShiftRCL || inst.ShiftKind == machinecode.ShiftRCR {
		out = append(out, rotateThroughCarry(inst)...)
	} else {
		out = append(out, rotateOp(inst)...)
	}
	out = append(out, rotateWrite(inst)...)
	return append(out, rotateFlags(inst)...)
}

func rotatePrepare(inst machinecode.Instruction) []byte {
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

func rotateOp(inst machinecode.Instruction) []byte {
	out := localCode(19)
	out = append(out, localCode(21)...)
	if inst.ShiftKind == machinecode.ShiftROL {
		out = append(out, rotateLeft()...)
	} else {
		out = append(out, rotateRight()...)
	}
	return append(out, WasmOpLocalSet, 20)
}

func rotateThroughCarry(inst machinecode.Instruction) []byte {
	out := localCode(16)
	out = append(out, constCode(1)...)
	out = append(out, WasmOpI64And, WasmOpLocalSet, 22)
	if inst.ShiftKind == machinecode.ShiftRCL {
		return append(out, rotateCarryLeft()...)
	}
	return append(out, rotateCarryRight()...)
}

func rotateWrite(inst machinecode.Instruction) []byte {
	return append(localCode(20), WasmOpLocalSet, byte(inst.Dst))
}
