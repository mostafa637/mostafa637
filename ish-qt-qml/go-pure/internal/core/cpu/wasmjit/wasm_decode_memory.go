package wasmjit

import (
	"golang.org/x/arch/x86/x86asm"
)

func decodeMem(arg x86asm.Arg) (int16, int64, bool) {
	mem, ok := arg.(x86asm.Mem)
	if !ok || mem.Base == 0 || mem.Index != 0 || mem.Segment != 0 {
		return 0, 0, false
	}
	base, ok := decodeReg(mem.Base)
	return base, mem.Disp, ok
}
