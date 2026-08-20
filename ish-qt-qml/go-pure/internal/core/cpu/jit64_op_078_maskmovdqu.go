package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMASKMOVDQU(ctx jit64CompileContext64) (microOp64, bool, error) {
	source, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || source.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	mask, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || mask.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s mask: %v", ctx.inst.Op, err)
	}
	return makeSSEMaskMoveDQU64(ctx.address, uint8(ctx.inst.Len), source, mask), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MASKMOVDQU] = jit64CompileCaseMASKMOVDQU
}
