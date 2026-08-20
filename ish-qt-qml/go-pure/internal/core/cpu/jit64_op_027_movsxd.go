package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVSXD(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 8)
	if err != nil || destination.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("MOVSXD destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), 4)
	if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("MOVSXD source: %v", err)
	}
	return makeExtend64(ctx.address, uint8(ctx.inst.Len), true, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVSXD] = jit64CompileCaseMOVSXD
}
