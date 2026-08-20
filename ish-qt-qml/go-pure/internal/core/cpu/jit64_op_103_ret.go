package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseRET(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makeRet64(ctx.address, uint8(ctx.inst.Len)), true, nil
}

func init() {
	jit64InstructionHandlers[x86asm.RET] = jit64CompileCaseRET
}
