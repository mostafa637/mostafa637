package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseRDRAND(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := rdrandDestination64(ctx.inst, ctx.arg(0))
	if err != nil || destination.Kind != operand64Reg || (destination.Width != 2 && destination.Width != 4 && destination.Width != 8) {
		return microOp64{}, false, fmt.Errorf("RDRAND destination: %v", err)
	}
	return makeRDRAND64(ctx.address, uint8(ctx.inst.Len), destination), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.RDRAND] = jit64CompileCaseRDRAND
}
