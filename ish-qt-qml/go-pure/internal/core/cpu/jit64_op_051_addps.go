package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseADDPS(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	return makeSSEPackedFloatBinary64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.ADDPS] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.SUBPS] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.MULPS] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.DIVPS] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.ADDPD] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.SUBPD] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.MULPD] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.DIVPD] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.MINPS] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.MAXPS] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.MINPD] = jit64CompileCaseADDPS
	jit64InstructionHandlers[x86asm.MAXPD] = jit64CompileCaseADDPS
}
