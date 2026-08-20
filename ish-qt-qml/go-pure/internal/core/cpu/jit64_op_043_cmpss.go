package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCMPSS(ctx jit64CompileContext64) (microOp64, bool, error) {
	width := uint8(4)
	if ctx.inst.Op == x86asm.CMPSD_XMM {
		width = 8
	}
	destination, err := operand64ScalarSSEFromArg(ctx.arg(0), width)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, sourceErr := operand64ScalarSSEFromArg(ctx.arg(1), width)
	if sourceErr != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, sourceErr)
	}
	immediate, ok := ctx.arg(2).(x86asm.Imm)
	if !ok {
		return microOp64{}, false, fmt.Errorf("%s requires an immediate predicate", ctx.inst.Op)
	}
	return makeSSEComparePredicate64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source, uint8(immediate)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CMPSS] = jit64CompileCaseCMPSS
	jit64InstructionHandlers[x86asm.CMPSD_XMM] = jit64CompileCaseCMPSS
}
