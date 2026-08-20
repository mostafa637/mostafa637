package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitMemoryOp(inst machinecode.Instruction) []byte {
	switch inst.Op {
	case machinecode.OpLoad64:
		return emitMemoryLoad(inst)
	case machinecode.OpStore64:
		return emitMemoryStore(inst)
	case machinecode.OpPush64:
		return emitStackPush(inst)
	default:
		return emitStackPop(inst)
	}
}

func emitExtendedOp(inst machinecode.Instruction) []byte {
	if inst.Op == machinecode.OpExtend64 {
		return emitExtend(inst)
	}
	if inst.Op == machinecode.OpLEA64 {
		return emitLEA(inst)
	}
	return emitShift(inst)
}
