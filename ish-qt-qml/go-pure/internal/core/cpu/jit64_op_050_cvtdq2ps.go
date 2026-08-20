package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCVTDQ2PS(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 0)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	if source.Kind == operand64Mem {
		source.Width = packedConversionSourceWidth64(ctx.inst.Op)
	}
	return makeSSEPackedConvert64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CVTDQ2PS] = jit64CompileCaseCVTDQ2PS
	jit64InstructionHandlers[x86asm.CVTPS2DQ] = jit64CompileCaseCVTDQ2PS
	jit64InstructionHandlers[x86asm.CVTTPS2DQ] = jit64CompileCaseCVTDQ2PS
	jit64InstructionHandlers[x86asm.CVTDQ2PD] = jit64CompileCaseCVTDQ2PS
	jit64InstructionHandlers[x86asm.CVTPD2DQ] = jit64CompileCaseCVTDQ2PS
	jit64InstructionHandlers[x86asm.CVTPS2PD] = jit64CompileCaseCVTDQ2PS
	jit64InstructionHandlers[x86asm.CVTPD2PS] = jit64CompileCaseCVTDQ2PS
	jit64InstructionHandlers[x86asm.CVTTPD2DQ] = jit64CompileCaseCVTDQ2PS
}
