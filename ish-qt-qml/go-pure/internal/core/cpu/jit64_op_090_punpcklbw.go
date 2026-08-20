package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePUNPCKLBW(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || left.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("PUNPCK destination: %v", err)
	}
	right, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("PUNPCK source: %v", err)
	}
	return makeSSEUnpack64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PUNPCKLBW] = jit64CompileCasePUNPCKLBW
	jit64InstructionHandlers[x86asm.PUNPCKHBW] = jit64CompileCasePUNPCKLBW
	jit64InstructionHandlers[x86asm.PUNPCKLWD] = jit64CompileCasePUNPCKLBW
	jit64InstructionHandlers[x86asm.PUNPCKHWD] = jit64CompileCasePUNPCKLBW
	jit64InstructionHandlers[x86asm.PUNPCKLDQ] = jit64CompileCasePUNPCKLBW
	jit64InstructionHandlers[x86asm.PUNPCKHDQ] = jit64CompileCasePUNPCKLBW
	jit64InstructionHandlers[x86asm.PUNPCKLQDQ] = jit64CompileCasePUNPCKLBW
	jit64InstructionHandlers[x86asm.PUNPCKHQDQ] = jit64CompileCasePUNPCKLBW
}
