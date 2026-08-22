package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

const (
	flagCF uint64 = 1 << 0
	flagPF uint64 = 1 << 2
	flagAF uint64 = 1 << 4
	flagZF uint64 = 1 << 6
	flagSF uint64 = 1 << 7
	flagOF uint64 = 1 << 11
)

const arithmeticMask = flagCF | flagPF | flagAF | flagZF | flagSF | flagOF
const logicMask = flagPF | flagZF | flagSF

func emitAddFlags(inst machinecode.Instruction) []byte {
	return emitArithmeticFlags(inst, false, false)
}

func emitSubFlags(inst machinecode.Instruction) []byte {
	return emitArithmeticFlags(inst, true, false)
}

func emitCompareFlags(inst machinecode.Instruction) []byte {
	return emitArithmeticFlags(inst, true, true)
}

func emitArithmeticFlags(inst machinecode.Instruction, sub, compare bool) []byte {
	result := localCode(inst.Dst)
	if compare {
		result = localCode(19)
	}
	return emitPackedFlags(localCode(17), localCode(18), result, sub, arithmeticMask)
}

func emitRegisterFlags(inst machinecode.Instruction, sub bool) []byte {
	return emitPackedFlags(localCode(17), localCode(18), localCode(inst.Dst), sub, arithmeticMask)
}

func emitLogicFlags(inst machinecode.Instruction) []byte {
	return emitLogicResultFlags(inst.Dst)
}

func emitLogicResultFlags(result int16) []byte {
	out := clearFlags(nil, arithmeticMask)
	out = appendFlag(out, flagZero(localCode(result)), 6)
	out = appendFlag(out, flagSign(localCode(result)), 7)
	return appendFlag(out, flagParity(localCode(result)), 2)
}

func emitPackedFlags(op1, op2, result []byte, sub bool, mask uint64) []byte {
	out := clearFlags(nil, mask)
	out = appendFlag(out, flagCarry(op1, op2, result, sub), 0)
	out = appendFlag(out, flagParity(result), 2)
	out = appendFlag(out, flagAux(op1, op2, result), 4)
	out = appendFlag(out, flagZero(result), 6)
	out = appendFlag(out, flagSign(result), 7)
	return appendFlag(out, flagOverflow(op1, op2, result, sub), 11)
}

func localCode(index int16) []byte    { return []byte{WasmOpLocalGet, byte(index)} }
func localSetCode(index int16) []byte { return []byte{WasmOpLocalSet, byte(index)} }

func constCode(value int64) []byte    { return appendSLEB([]byte{WasmOpI64Const}, value) }
func i32ConstCode(value int32) []byte { return appendSLEB([]byte{WasmOpI32Const}, int64(value)) }

func clearFlags(out []byte, mask uint64) []byte {
	out = append(out, WasmOpLocalGet, 16, WasmOpI64Const)
	out = appendSLEB(out, int64(^mask))
	return append(out, WasmOpI64And, WasmOpLocalSet, 16)
}

func appendFlag(out, expr []byte, bit byte) []byte {
	out = append(out, expr...)
	out = append(out, constCode(int64(bit))...)
	out = append(out, WasmOpI64Shl, WasmOpLocalGet, 16, WasmOpI64Or, WasmOpLocalSet, 16)
	return out
}

func updateCommonFlags(reg int16, width uint8) []byte {
	val := localCode(reg)
	out := clearFlags(nil, arithmeticMask)
	out = appendFlag(out, flagZero(val), 6)
	out = appendFlag(out, flagSign(val), 7)
	out = appendFlag(out, flagParity(val), 2)
	return out
}
