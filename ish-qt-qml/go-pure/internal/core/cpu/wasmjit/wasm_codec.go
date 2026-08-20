package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func wasmSection(id byte, payload []byte) []byte {
	out := []byte{id}
	out = append(out, appendULEB(nil, uint32(len(payload)))...)
	return append(out, payload...)
}

func appendULEB(out []byte, value uint32) []byte {
	for value >= 0x80 {
		out = append(out, byte(value)|0x80)
		value >>= 7
	}
	return append(out, byte(value))
}

func appendSLEB(out []byte, value int64) []byte {
	more := true
	for more {
		b := byte(value) & 0x7f
		value >>= 7
		more = !((value == 0 && b&0x40 == 0) || (value == -1 && b&0x40 != 0))
		if more {
			b |= 0x80
		}
		out = append(out, b)
	}
	return out
}

func emitBody(insts []machinecode.Instruction) []byte {
	out := []byte{0}
	for _, inst := range insts {
		out = append(out, emitInstruction(inst)...)
	}
	out = appendRegisterResults(out)
	return append(out, 0x0b)
}

func appendRegisterResults(out []byte) []byte {
	for i := byte(0); i < 16; i++ {
		out = append(out, 0x20, i)
	}
	return out
}

func emitInstruction(inst machinecode.Instruction) []byte {
	switch inst.Op {
	case machinecode.OpNOP, machinecode.OpRET:
		return nil
	case machinecode.OpMOVImm:
		return emitMove(inst)
	case machinecode.OpADDImm:
		return emitArithmetic(inst, 0x7c)
	case machinecode.OpSUBImm:
		return emitArithmetic(inst, 0x7d)
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

func emitSyscall() []byte {
	out := make([]byte, 0, 30)
	for _, index := range []byte{0, 7, 6, 2, 10, 8, 9} {
		out = append(out, 0x20, index)
	}
	return append(out, 0x10, 0, 0x21, 0)
}

func emitArithmetic(inst machinecode.Instruction, op byte) []byte {
	out := []byte{0x20, byte(inst.Dst), 0x42}
	out = appendSLEB(out, inst.Imm)
	return append(out, op, 0x21, byte(inst.Dst))
}
