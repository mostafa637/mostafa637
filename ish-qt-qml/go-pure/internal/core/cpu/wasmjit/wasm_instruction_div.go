package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitDIV(inst machinecode.Instruction) []byte {
	out := localCode(18)
	out = append(out, WasmOpI64Eqz, WasmOpIf, 0x40)
	out = append(out, emitReturn()...)
	out = append(out, WasmOpElse)
	if inst.Signed {
		out = append(out, emitIDIV(inst)...)
	} else {
		out = append(out, emitUDIV(inst)...)
	}
	return append(out, WasmOpEnd)
}

func emitUDIV(inst machinecode.Instruction) []byte {
	out := localCode(2)
	out = append(out, WasmOpI64Eqz, WasmOpIf, 0x40)
	out = append(out, localCode(0)...)
	out = append(out, localCode(18)...)
	out = append(out, WasmOpI64DivU, WasmOpLocalSet, 19)
	out = append(out, localCode(0)...)
	out = append(out, localCode(18)...)
	out = append(out, WasmOpI64RemU, WasmOpLocalSet, 2)
	out = append(out, localCode(19)...)
	out = append(out, WasmOpLocalSet, 0, WasmOpElse)
	out = append(out, divInit()...)
	out = append(out, divLoop()...)
	out = append(out, localCode(19)...)
	out = append(out, WasmOpLocalSet, 0)
	out = append(out, localCode(18)...)
	out = append(out, WasmOpLocalSet, 2, WasmOpEnd)
	return out
}

func emitIDIV(inst machinecode.Instruction) []byte {
	out := localCode(0)
	out = append(out, constCode(-0x8000000000000000)...)
	out = append(out, WasmOpI64Eq)
	out = append(out, localCode(18)...)
	out = append(out, constCode(-1)...)
	out = append(out, WasmOpI64Eq, WasmOpI32And, WasmOpIf, 0x40)
	out = append(out, emitReturn()...)
	out = append(out, WasmOpElse)
	out = append(out, localCode(0)...)
	out = append(out, localCode(18)...)
	out = append(out, WasmOpI64DivS, WasmOpLocalSet, 19)
	out = append(out, localCode(0)...)
	out = append(out, localCode(18)...)
	out = append(out, WasmOpI64RemS, WasmOpLocalSet, 2)
	out = append(out, localCode(19)...)
	out = append(out, WasmOpLocalSet, 0, WasmOpEnd)
	return out
}

func divInit() []byte {
	out := localCode(18)
	out = append(out, WasmOpLocalSet, 17)
	out = append(out, constCode(0)...)
	out = append(out, WasmOpLocalSet, 19)
	out = append(out, localCode(2)...)
	out = append(out, WasmOpLocalSet, 18)
	out = append(out, constCode(63)...)
	return append(out, WasmOpLocalSet, 20)
}
