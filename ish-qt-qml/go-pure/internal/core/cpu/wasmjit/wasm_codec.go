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
	out := []byte{1, 4, 0x7e}
	for _, inst := range insts {
		out = append(out, emitInstruction(inst)...)
	}
	out = appendRegisterResults(out)
	return append(out, 0x0b)
}

func appendRegisterResults(out []byte) []byte {
	for i := byte(0); i < 17; i++ {
		out = append(out, 0x20, i)
	}
	return out
}
