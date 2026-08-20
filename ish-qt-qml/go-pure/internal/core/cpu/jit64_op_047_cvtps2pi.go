package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCVTPS2PI(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 8)
	if err != nil || destination.Kind != operand64MMX {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 0)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	if source.Kind == operand64Mem {
		source.Width = 8
	}
	return makeMMXFromPackedFloat64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CVTPS2PI] = jit64CompileCaseCVTPS2PI
	jit64InstructionHandlers[x86asm.CVTTPS2PI] = jit64CompileCaseCVTPS2PI
}
