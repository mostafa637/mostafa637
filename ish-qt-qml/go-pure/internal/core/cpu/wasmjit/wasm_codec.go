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
	// Locals: 0-16 are params, 17-26 are temps.
	out := appendULEB(nil, 1) // 1 group of locals
	out = append(out, 10)     // 10 locals in this group
	out = append(out, 0x7e)   // type i64

	// Wrap instructions in a block so we can branch to the end
	out = append(out, WasmOpBlock, 0x40)

	for _, inst := range insts {
		out = append(out, emitInstruction(inst)...)
	}

	out = append(out, WasmOpEnd) // end of block

	// We always append results once at the end.
	out = appendRegisterResults(out)
	return append(out, WasmOpEnd) // end of function
}

func appendRegisterResults(out []byte) []byte {
	for i := byte(0); i < 27; i++ {
		out = append(out, WasmOpLocalGet, i)
	}
	return out
}

func decodeGuest(block GuestBlock) ([]machinecode.Instruction, error) {
	return decodeX86(block.Bytes, block.PC)
}

func lastFlow(insts []machinecode.Instruction) (machinecode.Instruction, bool) {
	var res machinecode.Instruction
	var found bool
	// Preference: last semantic op (MUL/DIV) then last flow op (RET/JMP)
	for _, inst := range insts {
		if isSemanticOp(inst.Op) {
			res = inst
			found = true
		}
	}
	if found {
		return res, true
	}
	for _, inst := range insts {
		if isFlow(inst.Op) {
			res = inst
			found = true
		}
	}
	return res, found
}

func isSemanticOp(op machinecode.Op) bool {
	return op == machinecode.OpMUL64 || op == machinecode.OpIMUL64 || op == machinecode.OpDIV64
}

func isFlow(op machinecode.Op) bool {
	return op == machinecode.OpJmp || op == machinecode.OpJcc || op == machinecode.OpCall || op == machinecode.OpRET || op == machinecode.OpSyscall
}
