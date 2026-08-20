package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePSHUFW(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 8)
	if err != nil || destination.Kind != operand64MMX {
		return microOp64{}, false, fmt.Errorf("PSHUFW destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), 8)
	if err != nil || (source.Kind != operand64MMX && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("PSHUFW source: %v", err)
	}
	order, ok := ctx.arg(2).(x86asm.Imm)
	if !ok {
		return microOp64{}, false, fmt.Errorf("PSHUFW immediate: %v", ctx.arg(2))
	}
	return makePSHUFW64(ctx.address, uint8(ctx.inst.Len), uint8(order), destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PSHUFW] = jit64CompileCasePSHUFW
}
