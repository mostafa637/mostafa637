package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseFLD1(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makeFPUConst64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.FLD1] = jit64CompileCaseFLD1
	jit64InstructionHandlers[x86asm.FLDZ] = jit64CompileCaseFLD1
}
