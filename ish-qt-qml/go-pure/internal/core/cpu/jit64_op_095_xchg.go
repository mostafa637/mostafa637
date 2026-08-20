package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseXCHG(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, right, err := ctx.two()
	if err != nil || (left.Kind == operand64Mem && right.Kind == operand64Mem) || left.Kind == operand64Imm || right.Kind == operand64Imm {
		return microOp64{}, false, fmt.Errorf("XCHG requires register and register/memory operands: %v", err)
	}
	return makeXchg64(ctx.address, uint8(ctx.inst.Len), left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.XCHG] = jit64CompileCaseXCHG
}
