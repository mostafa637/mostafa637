package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseADD(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, right, err := ctx.two()
	if err != nil {
		return microOp64{}, false, err
	}
	return makeBinary64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.ADD] = jit64CompileCaseADD
	jit64InstructionHandlers[x86asm.SUB] = jit64CompileCaseADD
	jit64InstructionHandlers[x86asm.XOR] = jit64CompileCaseADD
	jit64InstructionHandlers[x86asm.AND] = jit64CompileCaseADD
	jit64InstructionHandlers[x86asm.OR] = jit64CompileCaseADD
	jit64InstructionHandlers[x86asm.CMP] = jit64CompileCaseADD
	jit64InstructionHandlers[x86asm.TEST] = jit64CompileCaseADD
}
