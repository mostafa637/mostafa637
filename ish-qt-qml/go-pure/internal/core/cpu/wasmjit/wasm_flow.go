package wasmjit

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"

func decodeGuest(block GuestBlock) ([]machinecode.Instruction, error) {
	return decodeX86(block.Bytes, block.PC)
}

func isFlow(op machinecode.Op) bool {
	return op == machinecode.OpRET || op == machinecode.OpJmp || op == machinecode.OpJcc
}

func lastFlow(insts []machinecode.Instruction) (machinecode.Instruction, bool) {
	for i := len(insts) - 1; i >= 0; i-- {
		if insts[i].Op == machinecode.OpJmp || insts[i].Op == machinecode.OpJcc {
			return insts[i], true
		}
	}
	return machinecode.Instruction{}, false
}
