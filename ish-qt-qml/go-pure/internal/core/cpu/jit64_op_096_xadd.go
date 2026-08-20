package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseXADD(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, right, err := ctx.two()
	if err != nil || left.Kind == operand64Imm || right.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("XADD requires r/m destination and register source: %v", err)
	}
	return makeXadd64(ctx.address, uint8(ctx.inst.Len), left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.XADD] = jit64CompileCaseXADD
}
