package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVNTSD(ctx jit64CompileContext64) (microOp64, bool, error) {
	width := uint8(8)
	if ctx.inst.Op == x86asm.MOVNTSS {
		width = 4
	}
	destination, err := operand64FromArg(ctx.arg(0), width)
	if err != nil || destination.Kind != operand64Mem {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || source.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	return makeMOVScalar64(ctx.address, uint8(ctx.inst.Len), width, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVNTSD] = jit64CompileCaseMOVNTSD
	jit64InstructionHandlers[x86asm.MOVNTSS] = jit64CompileCaseMOVNTSD
}
