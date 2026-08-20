package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVHPD(ctx jit64CompileContext64) (microOp64, bool, error) {
	if _, ok := ctx.arg(0).(x86asm.Mem); ok {
		destination, err := operand64FromArg(ctx.arg(0), 8)
		if err != nil || destination.Kind != operand64Mem {
			return microOp64{}, false, fmt.Errorf("%s memory destination: %v", ctx.inst.Op, err)
		}
		source, err := operand64FromArg(ctx.arg(1), 16)
		if err != nil || source.Kind != operand64XMM {
			return microOp64{}, false, fmt.Errorf("%s requires an XMM source: %v", ctx.inst.Op, err)
		}
		return makeSSEHalfMove64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source), false, nil
	}
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s XMM destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 8)
	if err != nil || source.Kind != operand64Mem {
		return microOp64{}, false, fmt.Errorf("%s requires a memory source: %v", ctx.inst.Op, err)
	}
	return makeSSEHalfMove64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVHPD] = jit64CompileCaseMOVHPD
	jit64InstructionHandlers[x86asm.MOVHPS] = jit64CompileCaseMOVHPD
	jit64InstructionHandlers[x86asm.MOVLPD] = jit64CompileCaseMOVHPD
	jit64InstructionHandlers[x86asm.MOVLPS] = jit64CompileCaseMOVHPD
}
