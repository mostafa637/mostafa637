package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

const stackReg64 = 4

func emitStackPush(inst machinecode.Instruction) []byte {
	value := []byte{}
	if inst.Src >= 0 {
		value = append(value, localCode(inst.Src)...)
	} else if hasMemory(inst) {
		value = emitMemoryLoad(stackLoad(inst, 17))
	} else {
		value = constCode(inst.Imm)
	}
	value = append(value, 0x21, 17)
	value = append(value, stackAdjust(-8)...)
	return append(value, emitMemoryStore(stackStore(17))...)
}

func emitStackPop(inst machinecode.Instruction) []byte {
	value := emitMemoryLoad(stackLoad(inst, 17))
	value = append(value, stackAdjust(8)...)
	if inst.Dst >= 0 {
		return append(value, 0x20, 17, 0x21, byte(inst.Dst))
	}
	return append(value, emitMemoryStore(stackStore(17, inst))...)
}

func stackAdjust(delta int64) []byte {
	out := append(localCode(stackReg64), constCode(delta)...)
	return append(out, 0x7c, 0x21, stackReg64)
}

func stackLoad(inst machinecode.Instruction, dst int16) machinecode.Instruction {
	if inst.Op == machinecode.OpPop64 {
		return machinecode.Instruction{Op: machinecode.OpLoad64, Dst: dst, MemBase: stackReg64, MemIndex: -1}
	}
	return machinecode.Instruction{Op: machinecode.OpLoad64, Dst: dst, Imm: inst.Imm,
		MemBase: inst.MemBase, MemIndex: inst.MemIndex, MemScale: inst.MemScale,
		MemRIP: inst.MemRIP, NextPC: inst.NextPC}
}

func stackStore(src int16, extra ...machinecode.Instruction) machinecode.Instruction {
	inst := machinecode.Instruction{Op: machinecode.OpStore64, Src: src, MemBase: stackReg64, MemIndex: -1}
	if len(extra) > 0 {
		ref := extra[0]
		inst.Imm, inst.MemBase, inst.MemIndex, inst.MemScale = ref.Imm, ref.MemBase, ref.MemIndex, ref.MemScale
		inst.MemRIP, inst.NextPC = ref.MemRIP, ref.NextPC
	}
	return inst
}

func hasMemory(inst machinecode.Instruction) bool {
	return inst.MemBase >= 0 || inst.MemIndex >= 0 || inst.MemRIP
}
