package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCOMISS(ctx jit64CompileContext64) (microOp64, bool, error) {
	width := scalarSSEWidth64(ctx.inst.Op)
	left, err := operand64ScalarSSEFromArg(ctx.arg(0), width)
	if err != nil || (left.Kind != operand64XMM && left.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("scalar SSE compare left: %v", err)
	}
	right, rightErr := operand64ScalarSSEFromArg(ctx.arg(1), width)
	if rightErr != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("scalar SSE compare right: %v", rightErr)
	}
	return makeSSECompare64(ctx.address, uint8(ctx.inst.Len), width, left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.COMISS] = jit64CompileCaseCOMISS
	jit64InstructionHandlers[x86asm.COMISD] = jit64CompileCaseCOMISS
	jit64InstructionHandlers[x86asm.UCOMISS] = jit64CompileCaseCOMISS
	jit64InstructionHandlers[x86asm.UCOMISD] = jit64CompileCaseCOMISS
}
