package wasmjit

func flagCarry(op1, op2, result []byte, sub bool) []byte {
	if sub {
		return extendBool(append(append([]byte{}, op1...), append(op2, WasmOpI64LtU)...))
	}
	return extendBool(append(append([]byte{}, result...), append(op1, WasmOpI64LtU)...))
}

func flagZero(result []byte) []byte { return extendBool(append(result, WasmOpI64Eqz)) }

func flagSign(result []byte) []byte {
	out := append(append([]byte{}, result...), WasmOpI64Const, 63, WasmOpI64ShrU, WasmOpI64Const, 1, WasmOpI64And)
	return out
}

func flagParity(result []byte) []byte {
	out := append(outLocal(result), WasmOpLocalGet, 25, WasmOpLocalGet, 25, WasmOpI64Const, 4, WasmOpI64ShrU, WasmOpI64Xor, WasmOpLocalSet, 25)
	out = append(out, WasmOpLocalGet, 25, WasmOpLocalGet, 25, WasmOpI64Const, 2, WasmOpI64ShrU, WasmOpI64Xor, WasmOpLocalSet, 25)
	out = append(out, WasmOpLocalGet, 25, WasmOpLocalGet, 25, WasmOpI64Const, 1, WasmOpI64ShrU, WasmOpI64Xor, WasmOpLocalSet, 25)
	out = append(out, WasmOpLocalGet, 25, WasmOpI64Const, 1, WasmOpI64And, WasmOpI64Eqz)
	return extendBool(out)
}

func outLocal(result []byte) []byte {
	out := append([]byte{}, result...)
	out = append(out, constCode(255)...)
	return append(out, WasmOpI64And, WasmOpLocalSet, 25)
}

func flagAux(op1, op2, result []byte) []byte {
	out := append(append([]byte{}, op1...), op2...)
	out = append(out, WasmOpI64Xor)
	out = append(out, result...)
	return append(out, WasmOpI64Xor, WasmOpI64Const, 4, WasmOpI64ShrU, WasmOpI64Const, 1, WasmOpI64And)
}

func flagOverflow(op1, op2, result []byte, sub bool) []byte {
	out := append(append([]byte{}, op1...), op2...)
	out = append(out, WasmOpI64Xor)
	if !sub {
		out = append(out, WasmOpI64Const)
		out = appendSLEB(out, -1)
		out = append(out, WasmOpI64Xor)
	}
	out = append(out, op1...)
	out = append(out, result...)
	return append(out, WasmOpI64Xor, WasmOpI64And, WasmOpI64Const, 63, WasmOpI64ShrU, WasmOpI64Const, 1, WasmOpI64And)
}

func extendBool(out []byte) []byte { return append(out, WasmOpI64ExtendI32U) }
