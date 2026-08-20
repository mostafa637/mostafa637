package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOV(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, right, err := ctx.two()
	if err != nil || left.Kind == operand64Imm || left.Kind == operand64Rel {
		return microOp64{}, false, fmt.Errorf("MOV destination: %v", err)
	}
	return makeMove64(ctx.address, uint8(ctx.inst.Len), left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOV] = jit64CompileCaseMOV
}
