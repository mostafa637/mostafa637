package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVMSKPS(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 0)
	if err != nil || destination.Kind != operand64Reg || (destination.Width != 4 && destination.Width != 8) {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || source.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	return makeSSEFloatMovemask64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVMSKPS] = jit64CompileCaseMOVMSKPS
	jit64InstructionHandlers[x86asm.MOVMSKPD] = jit64CompileCaseMOVMSKPS
}
