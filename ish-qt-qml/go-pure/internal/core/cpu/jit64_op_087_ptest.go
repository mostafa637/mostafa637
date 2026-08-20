package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePTEST(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("PTEST destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("PTEST source: %v", err)
	}
	return makeSSETest64(ctx.address, uint8(ctx.inst.Len), destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PTEST] = jit64CompileCasePTEST
}
