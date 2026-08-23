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
