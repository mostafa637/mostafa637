package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePMOVMSKB(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 4)
	if err != nil || destination.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("PMOVMSKB destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || source.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("PMOVMSKB source: %v", err)
	}
	return makeSSEMovemask64(ctx.address, uint8(ctx.inst.Len), destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PMOVMSKB] = jit64CompileCasePMOVMSKB
}
