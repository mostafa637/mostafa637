package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitArithmeticOp(inst machinecode.Instruction) []byte {
	op := WasmOpI64Add
	if inst.Op == machinecode.OpSUBImm || inst.Op == machinecode.OpSUBReg {
		op = WasmOpI64Sub
	}
	if inst.Op == machinecode.OpADDImm || inst.Op == machinecode.OpSUBImm {
		return emitImmediateArithmetic(inst, op)
	}
	return emitRegisterArithmetic(inst, op)
}

func emitImmediateArithmetic(inst machinecode.Instruction, op byte) []byte {
	out := saveImmediateOperands(inst)
	out = append(out, WasmOpLocalGet, 17, WasmOpLocalGet, 18, op, WasmOpLocalSet, byte(inst.Dst))
	if op == WasmOpI64Add {
		return append(out, emitAddFlags(inst)...)
	}
	return append(out, emitSubFlags(inst)...)
}

func emitRegisterArithmetic(inst machinecode.Instruction, op byte) []byte {
	out := saveRegisterOperands(inst)
	out = append(out, WasmOpLocalGet, 17, WasmOpLocalGet, 18, op, WasmOpLocalSet, byte(inst.Dst))
	return append(out, emitRegisterFlags(inst, op == WasmOpI64Sub)...)
}

func emitLogicOp(inst machinecode.Instruction) []byte {
	op := WasmOpI64And
	if inst.Op == machinecode.OpORImm {
		op = WasmOpI64Or
	}
	if inst.Op == machinecode.OpXORImm {
		op = WasmOpI64Xor
	}
	out := saveImmediateOperands(inst)
	out = append(out, WasmOpLocalGet, 17, WasmOpLocalGet, 18, op, WasmOpLocalSet, byte(inst.Dst))
	return append(out, emitLogicFlags(inst)...)
}

func emitCompareOp(inst machinecode.Instruction) []byte {
	out := compareOperands(inst)
	if inst.Op == machinecode.OpCMPImm {
		out = append(out, WasmOpLocalGet, 17, WasmOpLocalGet, 18, WasmOpI64Sub, WasmOpLocalSet, 19)
		return append(out, emitCompareFlags(inst)...)
	}
	out = append(out, WasmOpLocalGet, 17, WasmOpLocalGet, 18, WasmOpI64And, WasmOpLocalSet, 19)
	return append(out, emitLogicResultFlags(19)...)
}

func compareOperands(inst machinecode.Instruction) []byte {
	if inst.Op == machinecode.OpCMPImm || inst.Op == machinecode.OpTestImm {
		return saveImmediateOperands(inst)
	}
	return saveRegisterOperands(inst)
}

func emitMoveOp(inst machinecode.Instruction) []byte {
	if inst.Op == machinecode.OpMOVImm {
		return append(constCode(inst.Imm), WasmOpLocalSet, byte(inst.Dst))
	}
	return append(localCode(inst.Src), WasmOpLocalSet, byte(inst.Dst))
}

func saveImmediateOperands(inst machinecode.Instruction) []byte {
	out := localCode(inst.Dst)
	out = append(out, WasmOpLocalSet, 17)
	out = append(out, constCode(inst.Imm)...)
	return append(out, WasmOpLocalSet, 18)
}

func saveRegisterOperands(inst machinecode.Instruction) []byte {
	out := localCode(inst.Dst)
	out = append(out, WasmOpLocalSet, 17)
	out = append(out, localCode(inst.Src)...)
	return append(out, WasmOpLocalSet, 18)
}
