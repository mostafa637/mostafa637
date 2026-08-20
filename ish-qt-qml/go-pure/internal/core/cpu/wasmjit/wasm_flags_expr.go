package wasmjit

func flagCarry(op1, op2, result []byte, sub bool) []byte {
	if sub {
		return extendBool(append(append([]byte{}, op1...), append(op2, 0x54)...))
	}
	return extendBool(append(append([]byte{}, result...), append(op1, 0x54)...))
}

func flagZero(result []byte) []byte { return extendBool(append(result, 0x50)) }

func flagSign(result []byte) []byte {
	out := append(append([]byte{}, result...), 0x42, 63, 0x88, 0x42, 1, 0x83)
	return out
}

func flagParity(result []byte) []byte {
	out := append(append([]byte{}, result...), 0x42)
	out = appendSLEB(out, 255)
	out = append(out, 0x83, 0x7b, 0x42, 1, 0x83, 0x50)
	return extendBool(out)
}

func flagAux(op1, op2, result []byte) []byte {
	out := append(append([]byte{}, op1...), op2...)
	out = append(out, 0x85)
	out = append(out, result...)
	return append(out, 0x85, 0x42, 4, 0x88, 0x42, 1, 0x83)
}

func flagOverflow(op1, op2, result []byte, sub bool) []byte {
	out := append(append([]byte{}, op1...), op2...)
	out = append(out, 0x85)
	if !sub {
		out = append(out, 0x42)
		out = appendSLEB(out, -1)
		out = append(out, 0x85)
	}
	out = append(out, op1...)
	out = append(out, result...)
	return append(out, 0x85, 0x83, 0x42, 63, 0x88, 0x42, 1, 0x83)
}

func extendBool(out []byte) []byte { return append(out, 0xad) }
