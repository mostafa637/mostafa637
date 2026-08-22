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
	case machinecode.OpPop64:
		return emitStackPop(inst)
	default:
		return nil
	}
}

func emitMemoryLoad(inst machinecode.Instruction) []byte {
	out := emitAddress(inst)
	return append(out, 0x10, 1, WasmOpLocalSet, byte(inst.Dst))
}

func emitMemoryStore(inst machinecode.Instruction) []byte {
	out := emitAddress(inst)
	out = append(out, WasmOpLocalGet, byte(inst.Src), 0x10, 2)
	return out
}

func emitAddress(inst machinecode.Instruction) []byte {
	out := []byte{}
	base := memoryBase(inst)
	if base >= 0 {
		out = append(out, WasmOpLocalGet, byte(base))
	}
	if inst.MemRIP {
		out = append(out, WasmOpLocalGet, 15)
		if base >= 0 {
			out = append(out, WasmOpI64Add)
		}
	}
	if inst.MemIndex >= 0 {
		out = append(out, WasmOpLocalGet, byte(inst.MemIndex), WasmOpI64Const)
		out = appendSLEB(out, int64(memoryScale(inst.MemScale)))
		out = append(out, WasmOpI64Mul)
		if base >= 0 || inst.MemRIP {
			out = append(out, WasmOpI64Add)
		}
	}
	out = append(out, WasmOpI64Const)
	out = appendSLEB(out, inst.Imm)
	if base >= 0 || inst.MemRIP || inst.MemIndex >= 0 {
		out = append(out, WasmOpI64Add)
	}
	return out
}

func memoryBase(inst machinecode.Instruction) int16 {
	if inst.MemBase >= 0 {
		return inst.MemBase
	}
	return -1
}

func memoryScale(scale uint8) uint8 {
	if scale == 0 {
		return 1
	}
	return scale
}

func emitSyscall(inst machinecode.Instruction) []byte {
	var out []byte
	out = append(out, localCode(0)...)   // RAX
	out = append(out, localCode(7)...)   // RDI
	out = append(out, localCode(6)...)   // RSI
	out = append(out, localCode(2)...)   // RDX
	out = append(out, localCode(10)...)  // R10
	out = append(out, localCode(8)...)   // R8
	out = append(out, localCode(9)...)   // R9
	out = append(out, 0x10, 0)           // call syscall64
	out = append(out, WasmOpLocalSet, 0) // Store result back to RAX
	out = append(out, constCode(int64(inst.NextPC))...)
	out = append(out, WasmOpLocalSet, 15) // Update RIP
	return out
}
