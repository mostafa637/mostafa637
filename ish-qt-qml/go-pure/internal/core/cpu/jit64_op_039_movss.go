package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVSS(ctx jit64CompileContext64) (microOp64, bool, error) {
	width := uint8(4)
	if ctx.inst.Op == x86asm.MOVSD_XMM {
		width = 8
	}

	left, err := operand64ScalarSSEFromArg(ctx.arg(0), width)
	if err != nil || (left.Kind != operand64XMM && left.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("scalar SSE move destination: %v", err)
	}
	right, err := operand64ScalarSSEFromArg(ctx.arg(1), width)
	if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) || (left.Kind == operand64Mem && right.Kind == operand64Mem) {
		return microOp64{}, false, fmt.Errorf("scalar SSE move source: %v", err)
	}
	return makeSSEScalarMove64(ctx.address, uint8(ctx.inst.Len), width, left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVSS] = jit64CompileCaseMOVSS
	jit64InstructionHandlers[x86asm.MOVSD_XMM] = jit64CompileCaseMOVSS
}
