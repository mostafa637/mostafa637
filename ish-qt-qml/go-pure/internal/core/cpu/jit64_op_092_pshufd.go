package cpu

import (
	"errors"
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePSHUFD(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("PSHUFD destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("PSHUFD source: %v", err)
	}
	immediate, ok := ctx.arg(2).(x86asm.Imm)
	if !ok {
		return microOp64{}, false, errors.New("PSHUFD requires an immediate selector")
	}
	if ctx.inst.Op == x86asm.PSHUFD {
		return makeSSEShuffleD64(ctx.address, uint8(ctx.inst.Len), destination, source, uint8(immediate)), false, nil
	}
	return makeSSEShuffleW64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source, uint8(immediate)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PSHUFD] = jit64CompileCasePSHUFD
	jit64InstructionHandlers[x86asm.PSHUFLW] = jit64CompileCasePSHUFD
	jit64InstructionHandlers[x86asm.PSHUFHW] = jit64CompileCasePSHUFD
}
