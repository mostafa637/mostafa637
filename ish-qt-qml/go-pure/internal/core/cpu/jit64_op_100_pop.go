package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePOP(ctx jit64CompileContext64) (microOp64, bool, error) {
	value, err := operand64FromArg(ctx.arg(0), 8)
	if err != nil || value.Kind == operand64Imm || value.Kind == operand64Rel {
		return microOp64{}, false, err
	}
	return makePop64(ctx.address, uint8(ctx.inst.Len), value), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.POP] = jit64CompileCasePOP
}
