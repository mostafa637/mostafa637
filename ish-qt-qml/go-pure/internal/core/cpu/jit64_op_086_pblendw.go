package cpu

import (
	"errors"
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePBLENDW(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("PBLENDW destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("PBLENDW source: %v", err)
	}
	immediate, ok := ctx.arg(2).(x86asm.Imm)
	if !ok {
		return microOp64{}, false, errors.New("PBLENDW requires an immediate mask")
	}
	return makeSSEBlendW64(ctx.address, uint8(ctx.inst.Len), destination, source, uint8(immediate)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PBLENDW] = jit64CompileCasePBLENDW
}
