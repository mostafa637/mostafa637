package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCMPXCHG(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, right, err := ctx.two()
	if err != nil || left.Kind == operand64Imm || right.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("CMPXCHG requires r/m destination and register source: %v", err)
	}
	return makeCmpxchg64(ctx.address, uint8(ctx.inst.Len), left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CMPXCHG] = jit64CompileCaseCMPXCHG
}
