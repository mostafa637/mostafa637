package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCVTSI2SS(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	sourceWidth := instructionDataWidth64(ctx.inst)
	source, sourceErr := operand64FromArg(ctx.arg(1), sourceWidth)
	if sourceErr != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, sourceErr)
	}
	return makeCVTScalar64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source, sourceWidth), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CVTSI2SS] = jit64CompileCaseCVTSI2SS
	jit64InstructionHandlers[x86asm.CVTSI2SD] = jit64CompileCaseCVTSI2SS
}
