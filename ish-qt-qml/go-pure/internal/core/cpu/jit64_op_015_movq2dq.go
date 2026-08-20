package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVQ2DQ(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("MOVQ2DQ destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), 8)
	if err != nil || source.Kind != operand64MMX {
		return microOp64{}, false, fmt.Errorf("MOVQ2DQ source: %v", err)
	}
	return makeMOVQ2DQ64(ctx.address, uint8(ctx.inst.Len), destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVQ2DQ] = jit64CompileCaseMOVQ2DQ
}
