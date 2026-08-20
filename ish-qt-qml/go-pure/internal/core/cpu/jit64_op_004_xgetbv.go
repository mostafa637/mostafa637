package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseXGETBV(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makeXCR64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.XGETBV] = jit64CompileCaseXGETBV
	jit64InstructionHandlers[x86asm.XSETBV] = jit64CompileCaseXGETBV
}
