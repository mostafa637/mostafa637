package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseBLENDPS(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	immediate, ok := ctx.arg(2).(x86asm.Imm)
	if !ok {
		return microOp64{}, false, fmt.Errorf("%s requires an immediate mask", ctx.inst.Op)
	}
	return makeSSEBlend64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source, uint8(immediate), false), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.BLENDPS] = jit64CompileCaseBLENDPS
	jit64InstructionHandlers[x86asm.BLENDPD] = jit64CompileCaseBLENDPS
}
