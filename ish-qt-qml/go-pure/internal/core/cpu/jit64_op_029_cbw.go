package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCBW(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makeConvertAccumulator64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CBW] = jit64CompileCaseCBW
	jit64InstructionHandlers[x86asm.CWDE] = jit64CompileCaseCBW
	jit64InstructionHandlers[x86asm.CDQ] = jit64CompileCaseCBW
	jit64InstructionHandlers[x86asm.CDQE] = jit64CompileCaseCBW
	jit64InstructionHandlers[x86asm.CWD] = jit64CompileCaseCBW
	jit64InstructionHandlers[x86asm.CQO] = jit64CompileCaseCBW
}
