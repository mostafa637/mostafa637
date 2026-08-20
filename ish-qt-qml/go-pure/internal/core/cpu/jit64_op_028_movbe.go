package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVBE(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, err := operand64FromArg(ctx.arg(0), ctx.width)
	if err != nil {
		return microOp64{}, false, err
	}
	right, err := operand64FromArg(ctx.arg(1), ctx.width)
	if err != nil || (left.Kind == operand64Mem && right.Kind == operand64Mem) || (left.Kind != operand64Mem && right.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("MOVBE requires one register and one memory operand: %v", err)
	}
	return makeMOVBE64(ctx.address, uint8(ctx.inst.Len), left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVBE] = jit64CompileCaseMOVBE
}
