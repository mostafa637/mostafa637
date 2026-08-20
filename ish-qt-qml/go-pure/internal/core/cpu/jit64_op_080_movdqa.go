package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVDQA(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || (left.Kind != operand64XMM && left.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("SSE move destination: %v", err)
	}
	right, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) || (left.Kind == operand64Mem && right.Kind == operand64Mem) {
		return microOp64{}, false, fmt.Errorf("SSE move source: %v", err)
	}
	return makeSSEMove64(ctx.address, uint8(ctx.inst.Len), left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVDQA] = jit64CompileCaseMOVDQA
	jit64InstructionHandlers[x86asm.MOVDQU] = jit64CompileCaseMOVDQA
	jit64InstructionHandlers[x86asm.MOVAPS] = jit64CompileCaseMOVDQA
	jit64InstructionHandlers[x86asm.MOVUPS] = jit64CompileCaseMOVDQA
	jit64InstructionHandlers[x86asm.MOVAPD] = jit64CompileCaseMOVDQA
	jit64InstructionHandlers[x86asm.MOVUPD] = jit64CompileCaseMOVDQA
}
