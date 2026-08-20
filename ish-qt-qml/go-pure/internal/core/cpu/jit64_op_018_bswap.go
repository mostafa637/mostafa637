package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseBSWAP(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 0)
	if err != nil || destination.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("BSWAP destination: %v", err)
	}
	return makeBSWAP64(ctx.address, uint8(ctx.inst.Len), destination), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.BSWAP] = jit64CompileCaseBSWAP
}
