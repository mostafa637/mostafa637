package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseAESENC(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s round key: %v", ctx.inst.Op, err)
	}
	return makeAESRound64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.AESENC] = jit64CompileCaseAESENC
	jit64InstructionHandlers[x86asm.AESENCLAST] = jit64CompileCaseAESENC
	jit64InstructionHandlers[x86asm.AESDEC] = jit64CompileCaseAESENC
	jit64InstructionHandlers[x86asm.AESDECLAST] = jit64CompileCaseAESENC
}
