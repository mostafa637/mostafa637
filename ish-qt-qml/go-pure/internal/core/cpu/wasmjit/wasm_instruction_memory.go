package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitMemoryLoad(inst machinecode.Instruction) []byte {
	out := emitAddress(inst.Src, inst.Imm)
	return append(out, 0x10, 1, 0x21, byte(inst.Dst))
}

func emitMemoryStore(inst machinecode.Instruction) []byte {
	out := emitAddress(inst.Dst, inst.Imm)
	out = append(out, 0x20, byte(inst.Src), 0x10, 2)
	return out
}

func emitAddress(base int16, disp int64) []byte {
	out := []byte{0x20, byte(base), 0x42}
	out = appendSLEB(out, disp)
	return append(out, 0x7c)
}

func emitSyscall() []byte {
	out := make([]byte, 0, 30)
	for _, index := range []byte{0, 7, 6, 2, 10, 8, 9} {
		out = append(out, 0x20, index)
	}
	return append(out, 0x10, 0, 0x21, 0)
}
