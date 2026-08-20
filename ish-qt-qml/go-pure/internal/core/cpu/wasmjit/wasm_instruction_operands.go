package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func saveImmediateOperands(inst machinecode.Instruction) []byte {
	out := append(localCode(inst.Dst), 0x21, 17)
	out = append(out, constCode(inst.Imm)...)
	return append(out, 0x21, 18)
}

func saveRegisterOperands(inst machinecode.Instruction) []byte {
	out := append(localCode(inst.Dst), 0x21, 17)
	out = append(out, localCode(inst.Src)...)
	return append(out, 0x21, 18)
}
