package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseSHA256RNDS2(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	implicit, ok := ctx.arg(2).(x86asm.Reg)
	if !ok || implicit != x86asm.X0 {
		return microOp64{}, false, fmt.Errorf("%s requires implicit XMM0", ctx.inst.Op)
	}
	return makeSHA256Rounds2_64(ctx.address, uint8(ctx.inst.Len), destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.SHA256RNDS2] = jit64CompileCaseSHA256RNDS2
}
