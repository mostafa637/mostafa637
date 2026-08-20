package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseSHA1RNDS4(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	immediate, ok := ctx.arg(2).(x86asm.Imm)
	if !ok {
		return microOp64{}, false, fmt.Errorf("%s requires an immediate round selector", ctx.inst.Op)
	}
	return makeSHA1Rounds4_64(ctx.address, uint8(ctx.inst.Len), destination, source, uint8(immediate)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.SHA1RNDS4] = jit64CompileCaseSHA1RNDS4
}
