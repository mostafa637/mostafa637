package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCVTPI2PS(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 0)
	if err != nil || (source.Kind != operand64MMX && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	if source.Kind == operand64Mem {
		source.Width = 8
	}
	return makePackedIntToFloat64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CVTPI2PS] = jit64CompileCaseCVTPI2PS
	jit64InstructionHandlers[x86asm.CVTPI2PD] = jit64CompileCaseCVTPI2PS
}
