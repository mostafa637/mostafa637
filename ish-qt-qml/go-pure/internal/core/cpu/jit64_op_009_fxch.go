package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseFXCH(ctx jit64CompileContext64) (microOp64, bool, error) {
	index, ok := x87Index64(ctx.arg(0))
	if !ok {
		return microOp64{}, false, fmt.Errorf("FXCH operand %v", ctx.arg(0))
	}
	return makeFPUFXCH64(ctx.address, uint8(ctx.inst.Len), index), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.FXCH] = jit64CompileCaseFXCH
}
