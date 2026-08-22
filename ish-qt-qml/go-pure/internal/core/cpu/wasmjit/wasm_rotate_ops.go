package wasmjit

func rotateLeft() []byte {
	out := []byte{WasmOpLocalSet, 22}
	out = append(out, localCode(19)...)
	out = append(out, localCode(22)...)
	out = append(out, WasmOpI64Shl)
	out = append(out, localCode(19)...)
	out = append(out, constCode(64)...)
	out = append(out, localCode(22)...)
	return append(out, WasmOpI64Sub, WasmOpI64ShrU, WasmOpI64Or)
}

func rotateRight() []byte {
	out := []byte{WasmOpLocalSet, 22}
	out = append(out, localCode(19)...)
	out = append(out, localCode(22)...)
	out = append(out, WasmOpI64ShrU)
	out = append(out, localCode(19)...)
	out = append(out, constCode(64)...)
	out = append(out, localCode(22)...)
	return append(out, WasmOpI64Sub, WasmOpI64Shl, WasmOpI64Or)
}

func rotateCarryLeft() []byte {
	out := localCode(19)
	out = append(out, constCode(63)...)
	out = append(out, WasmOpI64ShrU, WasmOpLocalSet, 24)
	out = append(out, localCode(19)...)
	out = append(out, constCode(1)...)
	out = append(out, WasmOpI64Shl)
	out = append(out, localCode(22)...)
	out = append(out, WasmOpI64Or, WasmOpLocalSet, 19)
	out = append(out, localCode(24)...)
	out = append(out, WasmOpLocalSet, 22)
	out = append(out, localCode(19)...)
	return append(out, WasmOpLocalSet, 20)
}

func rotateCarryRight() []byte {
	out := localCode(19)
	out = append(out, constCode(1)...)
	out = append(out, WasmOpI64And, WasmOpLocalSet, 24)
	out = append(out, localCode(19)...)
	out = append(out, constCode(1)...)
	out = append(out, WasmOpI64ShrU)
	out = append(out, localCode(22)...)
	out = append(out, constCode(63)...)
	out = append(out, WasmOpI64Shl)
	out = append(out, WasmOpI64Or, WasmOpLocalSet, 19)
	out = append(out, localCode(24)...)
	out = append(out, WasmOpLocalSet, 22)
	out = append(out, localCode(19)...)
	return append(out, WasmOpLocalSet, 20)
}
