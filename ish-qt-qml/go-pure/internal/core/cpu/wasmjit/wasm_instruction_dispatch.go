package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitInstruction(inst machinecode.Instruction) []byte {
	if isFlow(inst.Op) {
		return emitControlOp(inst)
	}
	switch inst.Op {
	case machinecode.OpMOVImm, machinecode.OpMOVReg:
		return emitMoveOp(inst)
	case machinecode.OpADDImm, machinecode.OpSUBImm, machinecode.OpADDReg, machinecode.OpSUBReg:
		return emitArithmeticOp(inst)
	case machinecode.OpANDImm, machinecode.OpORImm, machinecode.OpXORImm:
		return emitLogicOp(inst)
	case machinecode.OpCMPImm, machinecode.OpTestImm, machinecode.OpTestReg:
		return emitCompareOp(inst)
	case machinecode.OpMUL64, machinecode.OpIMUL64:
		return emitMulOrImul(inst)
	case machinecode.OpDIV64:
		return append(saveRegisterOperands(inst), emitDIV(inst)...)
	case machinecode.OpShift64:
		return emitShift(inst)
	case machinecode.OpLoad64, machinecode.OpStore64, machinecode.OpPush64, machinecode.OpPop64:
		return emitMemoryOp(inst)
	case machinecode.OpLEA64:
		return append(emitAddress(inst), WasmOpLocalSet, byte(inst.Dst))
	case machinecode.OpSyscall:
		return emitSyscall(inst)
	case machinecode.OpExtend64:
		return emitExtend(inst)
	default:
		return nil
	}
}

func emitMulOrImul(inst machinecode.Instruction) []byte {
	if inst.Op == machinecode.OpIMUL64 && inst.Src < 0 {
		return emitIMUL3(inst)
	}
	return append(saveRegisterOperands(inst), emitMUL(inst)...)
}

func emitControlOp(inst machinecode.Instruction) []byte {
	switch inst.Op {
	case machinecode.OpRET:
		return emitRET(inst)
	case machinecode.OpJmp:
		return emitJmp(inst)
	case machinecode.OpJcc:
		return emitJcc(inst)
	case machinecode.OpCall:
		return emitCall(inst)
	case machinecode.OpSyscall:
		return emitSyscall(inst)
	default:
		return nil
	}
}

func emitArithmeticOp(inst machinecode.Instruction) []byte {
	op := WasmOpI64Add
	if inst.Op == machinecode.OpSUBImm || inst.Op == machinecode.OpSUBReg {
		op = WasmOpI64Sub
	}
	if inst.Op == machinecode.OpADDImm || inst.Op == machinecode.OpSUBImm {
		out := saveImmediateOperands(inst)
		out = append(out, WasmOpLocalGet, 17, WasmOpLocalGet, 18, op, WasmOpLocalSet, byte(inst.Dst))
		if op == WasmOpI64Add {
			return append(out, emitAddFlags(inst)...)
		}
		return append(out, emitSubFlags(inst)...)
	}
	out := saveRegisterOperands(inst)
	out = append(out, WasmOpLocalGet, 17, WasmOpLocalGet, 18, op, WasmOpLocalSet, byte(inst.Dst))
	if op == WasmOpI64Add {
		return append(out, emitRegisterFlags(inst, false)...)
	}
	return append(out, emitRegisterFlags(inst, true)...)
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
	var out []byte
	if inst.Op == machinecode.OpCMPImm || inst.Op == machinecode.OpTestImm {
		out = saveImmediateOperands(inst)
	} else {
		out = saveRegisterOperands(inst)
	}
	if inst.Op == machinecode.OpCMPImm {
		out = append(out, WasmOpLocalGet, 17, WasmOpLocalGet, 18, WasmOpI64Sub, WasmOpLocalSet, 19)
		return append(out, emitCompareFlags(inst)...)
	}
	out = append(out, WasmOpLocalGet, 17, WasmOpLocalGet, 18, WasmOpI64And, WasmOpLocalSet, 19)
	return append(out, emitLogicResultFlags(19)...)
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
