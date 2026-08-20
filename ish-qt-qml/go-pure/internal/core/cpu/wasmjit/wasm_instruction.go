package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitMove(inst machinecode.Instruction) []byte {
	out := constCode(inst.Imm)
	return append(out, 0x21, byte(inst.Dst))
}

func emitRegMove(inst machinecode.Instruction) []byte {
	return []byte{0x20, byte(inst.Src), 0x21, byte(inst.Dst)}
}

func emitImmediateArithmetic(inst machinecode.Instruction, op byte, flags func(machinecode.Instruction) []byte) []byte {
	out := saveImmediateOperands(inst)
	out = append(out, localCode(17)...)
	out = append(out, localCode(18)...)
	out = append(out, op, 0x21, byte(inst.Dst))
	return append(out, flags(inst)...)
}

func emitRegisterArithmetic(inst machinecode.Instruction, op byte, flags func(machinecode.Instruction) []byte) []byte {
	out := saveRegisterOperands(inst)
	out = append(out, localCode(17)...)
	out = append(out, localCode(18)...)
	out = append(out, op, 0x21, byte(inst.Dst))
	return append(out, flags(inst)...)
}

func emitLogicImmediate(inst machinecode.Instruction, op byte) []byte {
	out := append(localCode(inst.Dst), constCode(inst.Imm)...)
	out = append(out, op, 0x21, byte(inst.Dst))
	return append(out, emitLogicFlags(inst)...)
}

func emitCompare(inst machinecode.Instruction) []byte {
	out := saveImmediateOperands(inst)
	out = append(out, localCode(17)...)
	out = append(out, localCode(18)...)
	out = append(out, 0x7d, 0x21, 19)
	return append(out, emitCompareFlags(inst)...)
}
