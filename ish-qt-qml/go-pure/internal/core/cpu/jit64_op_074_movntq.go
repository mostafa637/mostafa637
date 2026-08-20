package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVNTQ(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 8)
	if err != nil || destination.Kind != operand64Mem {
		return microOp64{}, false, fmt.Errorf("MOVNTQ destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), 8)
	if err != nil || source.Kind != operand64MMX {
		return microOp64{}, false, fmt.Errorf("MOVNTQ source: %v", err)
	}
	return makeMOVNTQ64(ctx.address, uint8(ctx.inst.Len), destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVNTQ] = jit64CompileCaseMOVNTQ
}
