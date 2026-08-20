package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitInstruction(inst machinecode.Instruction) []byte {
	if controlOp(inst.Op) {
		return nil
	}
	return emitInstructionOp(inst)
}

func emitInstructionOp(inst machinecode.Instruction) []byte {
	switch inst.Op {
	case machinecode.OpMOVImm, machinecode.OpMOVReg:
		return emitMoveOp(inst)
	case machinecode.OpADDImm, machinecode.OpSUBImm, machinecode.OpADDReg, machinecode.OpSUBReg:
		return emitArithmeticOp(inst)
	case machinecode.OpANDImm, machinecode.OpORImm, machinecode.OpXORImm:
		return emitLogicOp(inst)
	case machinecode.OpCMPImm, machinecode.OpTestImm, machinecode.OpTestReg:
		return emitCompareOp(inst)
	case machinecode.OpLoad64, machinecode.OpStore64, machinecode.OpPush64, machinecode.OpPop64:
		return emitMemoryOp(inst)
	case machinecode.OpExtend64, machinecode.OpLEA64, machinecode.OpShift64:
		return emitExtendedOp(inst)
	case machinecode.OpSyscall:
		return emitSyscall()
	default:
		return nil
	}
}

func controlOp(op machinecode.Op) bool {
	return op == machinecode.OpNOP || op == machinecode.OpRET || op == machinecode.OpJmp ||
		op == machinecode.OpJcc || op == machinecode.OpCall
}

func emitMoveOp(inst machinecode.Instruction) []byte {
	if inst.Op == machinecode.OpMOVImm {
		return emitMove(inst)
	}
	return emitRegMove(inst)
}

func emitArithmeticOp(inst machinecode.Instruction) []byte {
	if inst.Op == machinecode.OpADDImm {
		return emitImmediateArithmetic(inst, 0x7c, emitAddFlags)
	}
	if inst.Op == machinecode.OpSUBImm {
		return emitImmediateArithmetic(inst, 0x7d, emitSubFlags)
	}
	if inst.Op == machinecode.OpADDReg {
		return emitRegisterArithmetic(inst, 0x7c, addRegisterFlags)
	}
	return emitRegisterArithmetic(inst, 0x7d, subRegisterFlags)
}

func addRegisterFlags(inst machinecode.Instruction) []byte {
	return emitRegisterFlags(inst, false)
}

func subRegisterFlags(inst machinecode.Instruction) []byte {
	return emitRegisterFlags(inst, true)
}

func emitLogicOp(inst machinecode.Instruction) []byte {
	op := byte(0x83)
	if inst.Op == machinecode.OpORImm {
		op = 0x84
	}
	if inst.Op == machinecode.OpXORImm {
		op = 0x85
	}
	return emitLogicImmediate(inst, op)
}

func emitCompareOp(inst machinecode.Instruction) []byte {
	if inst.Op == machinecode.OpCMPImm {
		return emitCompare(inst)
	}
	if inst.Op == machinecode.OpTestImm {
		return emitTestImmediate(inst)
	}
	return emitTestRegister(inst)
}
