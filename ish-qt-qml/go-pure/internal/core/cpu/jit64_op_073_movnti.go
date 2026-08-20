package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVNTI(ctx jit64CompileContext64) (microOp64, bool, error) {
	width := uint8(4)
	if ctx.inst.DataSize == 64 {
		width = 8
	} else if ctx.inst.DataSize != 32 {
		return microOp64{}, false, fmt.Errorf("MOVNTI data size %d", ctx.inst.DataSize)
	}
	destination, err := operand64FromArg(ctx.arg(0), width)
	if err != nil || destination.Kind != operand64Mem {
		return microOp64{}, false, fmt.Errorf("MOVNTI destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), width)
	if err != nil || source.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("MOVNTI source: %v", err)
	}
	return makeMove64(ctx.address, uint8(ctx.inst.Len), destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVNTI] = jit64CompileCaseMOVNTI
}
