package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitMemoryLoad(inst machinecode.Instruction) []byte {
	out := emitAddress(inst)
	return append(out, 0x10, 1, 0x21, byte(inst.Dst))
}

func emitMemoryStore(inst machinecode.Instruction) []byte {
	out := emitAddress(inst)
	out = append(out, 0x20, byte(inst.Src), 0x10, 2)
	return out
}

func emitAddress(inst machinecode.Instruction) []byte {
	out := []byte{}
	base := memoryBase(inst)
	if inst.MemRIP {
		out = append(out, 0x42)
		out = appendSLEB(out, int64(inst.NextPC))
	} else if base >= 0 {
		out = append(out, 0x20, byte(base))
	}
	if inst.MemIndex >= 0 {
		out = append(out, 0x20, byte(inst.MemIndex))
		out = append(out, 0x42)
		out = appendSLEB(out, int64(memoryScale(inst.MemScale)))
		out = append(out, 0x7e)
		if base >= 0 || inst.MemRIP {
			out = append(out, 0x7c)
		}
	}
	out = append(out, 0x42)
	out = appendSLEB(out, inst.Imm)
	return append(out, 0x7c)
}

func memoryBase(inst machinecode.Instruction) int16 {
	if inst.MemBase < 0 && inst.MemIndex < 0 && !inst.MemRIP && inst.Src >= 0 {
		return inst.Src
	}
	return inst.MemBase
}

func memoryScale(scale uint8) uint64 {
	if scale == 0 {
		return 1
	}
	return uint64(scale)
}

func emitSyscall() []byte {
	out := make([]byte, 0, 30)
	for _, index := range []byte{0, 7, 6, 2, 10, 8, 9} {
		out = append(out, 0x20, index)
	}
	return append(out, 0x10, 0, 0x21, 0)
}
