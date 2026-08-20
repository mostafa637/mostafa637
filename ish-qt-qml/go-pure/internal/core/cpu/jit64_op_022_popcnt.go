package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePOPCNT(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 0)
	if err != nil || destination.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("POPCNT destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), destination.Width)
	if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("POPCNT source: %v", err)
	}
	return makePOPCNT64(ctx.address, uint8(ctx.inst.Len), destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.POPCNT] = jit64CompileCasePOPCNT
}
