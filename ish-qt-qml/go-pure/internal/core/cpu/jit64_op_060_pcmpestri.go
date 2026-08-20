package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePCMPESTRI(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || left.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s first string operand: %v", ctx.inst.Op, err)
	}
	right, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s second string operand: %v", ctx.inst.Op, err)
	}
	immediate, ok := ctx.arg(2).(x86asm.Imm)
	if !ok {
		return microOp64{}, false, fmt.Errorf("%s requires an immediate control byte", ctx.inst.Op)
	}
	return makeSSEStringCompare64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, left, right, uint8(immediate), instructionDataWidth64(ctx.inst)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PCMPESTRI] = jit64CompileCasePCMPESTRI
	jit64InstructionHandlers[x86asm.PCMPESTRM] = jit64CompileCasePCMPESTRI
	jit64InstructionHandlers[x86asm.PCMPISTRI] = jit64CompileCasePCMPESTRI
	jit64InstructionHandlers[x86asm.PCMPISTRM] = jit64CompileCasePCMPESTRI
}
