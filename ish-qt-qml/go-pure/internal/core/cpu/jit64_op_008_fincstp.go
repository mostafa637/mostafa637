package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseFINCSTP(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makeFPUTopMove64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.FINCSTP] = jit64CompileCaseFINCSTP
	jit64InstructionHandlers[x86asm.FDECSTP] = jit64CompileCaseFINCSTP
}
