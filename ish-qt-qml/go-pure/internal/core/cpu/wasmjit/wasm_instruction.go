package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitInstruction(inst machinecode.Instruction) []byte {
	switch inst.Op {
	case machinecode.OpNOP, machinecode.OpRET:
		return nil
	case machinecode.OpMOVImm:
		return emitMove(inst)
	case machinecode.OpMOVReg:
		return emitRegMove(inst)
	case machinecode.OpADDImm:
		return emitArithmetic(inst, 0x7c)
	case machinecode.OpSUBImm:
		return emitArithmetic(inst, 0x7d)
	case machinecode.OpADDReg:
		return emitRegArithmetic(inst, 0x7c)
	case machinecode.OpSUBReg:
		return emitRegArithmetic(inst, 0x7d)
	case machinecode.OpANDImm:
		return emitArithmetic(inst, 0x83)
	case machinecode.OpORImm:
		return emitArithmetic(inst, 0x84)
	case machinecode.OpXORImm:
		return emitArithmetic(inst, 0x85)
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
	out := []byte{0x42}
	out = appendSLEB(out, inst.Imm)
	return append(out, 0x21, byte(inst.Dst))
}

func emitRegMove(inst machinecode.Instruction) []byte {
	return []byte{0x20, byte(inst.Src), 0x21, byte(inst.Dst)}
}

func emitRegArithmetic(inst machinecode.Instruction, op byte) []byte {
	return []byte{0x20, byte(inst.Dst), 0x20, byte(inst.Src), op, 0x21, byte(inst.Dst)}
}

func emitCompare(inst machinecode.Instruction) []byte {
	out := []byte{0x20, byte(inst.Dst), 0x42}
	out = appendSLEB(out, inst.Imm)
	return append(out, 0x51, 0xad, 0x21, 16)
}

func emitArithmetic(inst machinecode.Instruction, op byte) []byte {
	out := []byte{0x20, byte(inst.Dst), 0x42}
	out = appendSLEB(out, inst.Imm)
	return append(out, op, 0x21, byte(inst.Dst))
}
