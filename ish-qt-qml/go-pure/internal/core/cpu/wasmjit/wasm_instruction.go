package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitInstruction(inst machinecode.Instruction) []byte {
	switch inst.Op {
	case machinecode.OpNOP, machinecode.OpRET, machinecode.OpJmp, machinecode.OpJcc, machinecode.OpCall:
		return nil
	case machinecode.OpMOVImm:
		return emitMove(inst)
	case machinecode.OpMOVReg:
		return emitRegMove(inst)
	case machinecode.OpADDImm:
		return emitImmediateArithmetic(inst, 0x7c, emitAddFlags)
	case machinecode.OpSUBImm:
		return emitImmediateArithmetic(inst, 0x7d, emitSubFlags)
	case machinecode.OpADDReg:
		return emitRegisterArithmetic(inst, 0x7c, func(i machinecode.Instruction) []byte { return emitRegisterFlags(i, false) })
	case machinecode.OpSUBReg:
		return emitRegisterArithmetic(inst, 0x7d, func(i machinecode.Instruction) []byte { return emitRegisterFlags(i, true) })
	case machinecode.OpANDImm:
		return emitLogicImmediate(inst, 0x83)
	case machinecode.OpORImm:
		return emitLogicImmediate(inst, 0x84)
	case machinecode.OpXORImm:
		return emitLogicImmediate(inst, 0x85)
	case machinecode.OpCMPImm:
		return emitCompare(inst)
	case machinecode.OpLoad64:
		return emitMemoryLoad(inst)
	case machinecode.OpStore64:
		return emitMemoryStore(inst)
	case machinecode.OpSyscall:
		return emitSyscall()
	default:
		return nil
	}
}

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
