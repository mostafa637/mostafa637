package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseADDSUBPS(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	return makeSSEPackedHorizontal64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.ADDSUBPS] = jit64CompileCaseADDSUBPS
	jit64InstructionHandlers[x86asm.ADDSUBPD] = jit64CompileCaseADDSUBPS
	jit64InstructionHandlers[x86asm.HADDPS] = jit64CompileCaseADDSUBPS
	jit64InstructionHandlers[x86asm.HADDPD] = jit64CompileCaseADDSUBPS
	jit64InstructionHandlers[x86asm.HSUBPS] = jit64CompileCaseADDSUBPS
	jit64InstructionHandlers[x86asm.HSUBPD] = jit64CompileCaseADDSUBPS
}
