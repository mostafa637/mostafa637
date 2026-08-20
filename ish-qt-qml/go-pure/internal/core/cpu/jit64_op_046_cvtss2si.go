package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCVTSS2SI(ctx jit64CompileContext64) (microOp64, bool, error) {
	destinationWidth := instructionDataWidth64(ctx.inst)
	destination, err := operand64FromArg(ctx.arg(0), destinationWidth)
	if err != nil || destination.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	sourceWidth := uint8(4)
	if ctx.inst.Op == x86asm.CVTSD2SI || ctx.inst.Op == x86asm.CVTTSD2SI {
		sourceWidth = 8
	}
	source, sourceErr := operand64ScalarSSEFromArg(ctx.arg(1), sourceWidth)
	if sourceErr != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, sourceErr)
	}
	return makeCVTScalar64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source, sourceWidth), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CVTSS2SI] = jit64CompileCaseCVTSS2SI
	jit64InstructionHandlers[x86asm.CVTSD2SI] = jit64CompileCaseCVTSS2SI
	jit64InstructionHandlers[x86asm.CVTTSS2SI] = jit64CompileCaseCVTSS2SI
	jit64InstructionHandlers[x86asm.CVTTSD2SI] = jit64CompileCaseCVTSS2SI
}
