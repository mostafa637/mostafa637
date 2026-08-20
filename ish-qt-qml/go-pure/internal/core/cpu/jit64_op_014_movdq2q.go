package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVDQ2Q(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 8)
	if err != nil || destination.Kind != operand64MMX {
		return microOp64{}, false, fmt.Errorf("MOVDQ2Q destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || source.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("MOVDQ2Q source: %v", err)
	}
	return makeMOVDQ2Q64(ctx.address, uint8(ctx.inst.Len), destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVDQ2Q] = jit64CompileCaseMOVDQ2Q
}
