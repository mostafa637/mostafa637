package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitTestImmediate(inst machinecode.Instruction) []byte {
	out := append(localCode(inst.Dst), constCode(inst.Imm)...)
	out = append(out, 0x83, 0x21, 19)
	return append(out, emitLogicResultFlags(19)...)
}

func emitTestRegister(inst machinecode.Instruction) []byte {
	out := append(localCode(inst.Dst), localCode(inst.Src)...)
	out = append(out, 0x83, 0x21, 19)
	return append(out, emitLogicResultFlags(19)...)
}
