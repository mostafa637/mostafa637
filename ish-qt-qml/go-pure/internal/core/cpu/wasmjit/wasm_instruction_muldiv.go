package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func emitMUL(inst machinecode.Instruction) []byte {
	if inst.Signed {
		return emitIMUL(inst)
	}
	out := wideMulU(inst)
	return append(out, mulCarryFlag()...)
}

func emitIMUL(inst machinecode.Instruction) []byte {
	out := localCode(0)
	out = append(out, WasmOpLocalSet, 17)
	out = append(out, wideMulU(inst)...)
	out = append(out, localCode(2)...)
	out = append(out, localCode(17)...)
	out = append(out, constCode(63)...)
	out = append(out, WasmOpI64ShrS)
	out = append(out, localCode(18)...)
	out = append(out, WasmOpI64Mul, WasmOpI64Sub)
	out = append(out, localCode(18)...)
	out = append(out, constCode(63)...)
	out = append(out, WasmOpI64ShrS)
	out = append(out, localCode(17)...)
	out = append(out, WasmOpI64Mul, WasmOpI64Sub, WasmOpLocalSet, 2)
	return append(out, imulCarryFlag()...)
}

func mulCarryFlag() []byte {
	out := localCode(2)
	out = append(out, WasmOpI64Eqz, WasmOpIf, 0x40)
	out = append(out, clearFlags(nil, flagCF|flagOF)...)
	out = append(out, WasmOpElse)
	out = append(out, clearFlags(nil, flagCF|flagOF)...)
	out = append(out, appendFlag(nil, constCode(1), 0)...)
	out = append(out, appendFlag(nil, constCode(1), 11)...)
	return append(out, WasmOpEnd)
}

func imulCarryFlag() []byte {
	out := localCode(0)
	out = append(out, constCode(63)...)
	out = append(out, WasmOpI64ShrS, WasmOpLocalSet, 19)
	out = append(out, localCode(2)...)
	out = append(out, localCode(19)...)
	out = append(out, WasmOpI64Eq, WasmOpIf, 0x40)
	out = append(out, clearFlags(nil, flagCF|flagOF)...)
	out = append(out, WasmOpElse)
	out = append(out, clearFlags(nil, flagCF|flagOF)...)
	out = append(out, appendFlag(nil, constCode(1), 0)...)
	out = append(out, appendFlag(nil, constCode(1), 11)...)
	return append(out, WasmOpEnd)
}

func emitIMUL3(inst machinecode.Instruction) []byte {
	out := localCode(inst.MulSource)
	out = append(out, constCode(inst.Imm)...)
	out = append(out, WasmOpI64Mul, WasmOpLocalSet, byte(inst.Dst))
	return out
}
