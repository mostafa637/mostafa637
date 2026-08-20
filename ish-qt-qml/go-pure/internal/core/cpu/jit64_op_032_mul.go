package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMUL(ctx jit64CompileContext64) (microOp64, bool, error) {
	if ctx.arg(0) == nil || ctx.arg(1) != nil {
		return microOp64{}, false, fmt.Errorf("%s requires one operand", ctx.inst.Op)
	}
	source, err := operand64FromArg(ctx.arg(0), ctx.width)
	if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	return makeImplicitArithmetic64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, source, ctx.width), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MUL] = jit64CompileCaseMUL
	jit64InstructionHandlers[x86asm.DIV] = jit64CompileCaseMUL
	jit64InstructionHandlers[x86asm.IDIV] = jit64CompileCaseMUL
}
