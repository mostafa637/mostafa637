package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseFCHS(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makeFPUUnary64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.FCHS] = jit64CompileCaseFCHS
	jit64InstructionHandlers[x86asm.FABS] = jit64CompileCaseFCHS
}
