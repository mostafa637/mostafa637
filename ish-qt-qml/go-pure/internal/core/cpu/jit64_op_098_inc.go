package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseINC(ctx jit64CompileContext64) (microOp64, bool, error) {
	value, err := operand64FromArg(ctx.arg(0), ctx.width)
	if err != nil || value.Kind == operand64Imm || value.Kind == operand64Rel {
		return microOp64{}, false, err
	}
	return makeUnary64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, value), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.INC] = jit64CompileCaseINC
	jit64InstructionHandlers[x86asm.DEC] = jit64CompileCaseINC
	jit64InstructionHandlers[x86asm.NEG] = jit64CompileCaseINC
	jit64InstructionHandlers[x86asm.NOT] = jit64CompileCaseINC
}
