package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseADC(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, right, err := ctx.two()
	if err != nil || left.Kind == operand64Imm {
		return microOp64{}, false, fmt.Errorf("%s operands: %v", ctx.inst.Op, err)
	}
	return makeCarryBinary64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.ADC] = jit64CompileCaseADC
	jit64InstructionHandlers[x86asm.SBB] = jit64CompileCaseADC
}
