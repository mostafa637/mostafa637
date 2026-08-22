package wasmjit

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
)

func emitRET(inst machinecode.Instruction) []byte {
	return emitReturn()
}

func emitReturn() []byte {
	return []byte{WasmOpBr, 0}
}

func emitJmp(inst machinecode.Instruction) []byte {
	out := constCode(int64(inst.Target))
	out = append(out, WasmOpLocalSet, 15)
	return append(out, emitReturn()...)
}

func emitCall(inst machinecode.Instruction) []byte {
	out := constCode(int64(inst.Fallthrough))
	out = append(out, WasmOpLocalSet, 15)
	return append(out, emitReturn()...)
}

func emitSyscallReturn() []byte {
	return emitReturn()
}

func emitJcc(inst machinecode.Instruction) []byte {
	out := emitCondition(inst.Cond)
	out = append(out, WasmOpIf, 0x40)
	out = append(out, emitJmp(machinecode.Instruction{Target: inst.Target})...)
	out = append(out, WasmOpElse)
	out = append(out, emitJmp(machinecode.Instruction{Target: inst.Fallthrough})...)
	out = append(out, WasmOpEnd)
	return out
}

func emitCondition(cond uint8) []byte {
	var out []byte
	switch cond {
	case 4:
		out = condBit(localCode(16), 6)
	case 5:
		out = notBit(condBit(localCode(16), 6))
	case 2:
		out = condBit(localCode(16), 0)
	case 3:
		out = notBit(condBit(localCode(16), 0))
	case 8:
		out = condBit(localCode(16), 7)
	case 9:
		out = notBit(condBit(localCode(16), 7))
	case 0:
		out = condBit(localCode(16), 11)
	case 1:
		out = notBit(condBit(localCode(16), 11))
	case 6:
		out = orBits64(condBit(localCode(16), 0), condBit(localCode(16), 6))
	case 7:
		out = notBit(orBits64(condBit(localCode(16), 0), condBit(localCode(16), 6)))
	case 10:
		out = condBit(localCode(16), 2)
	case 11:
		out = notBit(condBit(localCode(16), 2))
	case 12:
		out = xorBits64(condBit(localCode(16), 7), condBit(localCode(16), 11))
	case 13:
		out = notBit(xorBits64(condBit(localCode(16), 7), condBit(localCode(16), 11)))
	case 14:
		out = orBits64(xorBits64(condBit(localCode(16), 7), condBit(localCode(16), 11)), condBit(localCode(16), 6))
	case 15:
		out = notBit(orBits64(xorBits64(condBit(localCode(16), 7), condBit(localCode(16), 11)), condBit(localCode(16), 6)))
	default:
		out = constCode(0)
	}
	return append(out, WasmOpI32WrapI64)
}

func condBit(val []byte, bit byte) []byte {
	out := append(val, constCode(int64(bit))...)
	out = append(out, WasmOpI64ShrU)
	out = append(out, constCode(1)...)
	return append(out, WasmOpI64And)
}

func notBit(val []byte) []byte {
	return append(val, WasmOpI64Eqz, WasmOpI64ExtendI32U)
}

func orBits64(a, b []byte) []byte {
	return append(append(a, b...), WasmOpI64Or)
}

func xorBits64(a, b []byte) []byte {
	return append(append(a, b...), WasmOpI64Xor)
}
