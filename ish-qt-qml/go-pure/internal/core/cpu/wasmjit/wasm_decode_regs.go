package wasmjit

import "golang.org/x/arch/x86/x86asm"

func decodeReg(arg x86asm.Arg) (int16, bool) {
	reg, ok := arg.(x86asm.Reg)
	if !ok || reg < x86asm.RAX || reg > x86asm.R15 {
		return 0, false
	}
	return int16(reg - x86asm.RAX), true
}

func decodeImm(arg x86asm.Arg) (int64, bool) {
	imm, ok := arg.(x86asm.Imm)
	return int64(imm), ok
}
