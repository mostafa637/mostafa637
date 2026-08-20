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
const logicMask = flagCF | flagPF | flagZF | flagSF | flagOF

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
	out := clearFlags(nil, logicMask)
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

func localCode(index int16) []byte { return []byte{0x20, byte(index)} }

func constCode(value int64) []byte { return appendSLEB([]byte{0x42}, value) }

func clearFlags(out []byte, mask uint64) []byte {
	out = append(out, 0x20, 16, 0x42)
	out = appendSLEB(out, int64(^mask))
	return append(out, 0x83, 0x21, 16)
}

func appendFlag(out, expr []byte, bit byte) []byte {
	out = append(out, expr...)
	out = append(out, 0x42, bit, 0x86, 0x20, 16, 0x84, 0x21, 16)
	return out
}
