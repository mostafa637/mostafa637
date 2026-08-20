package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePEXTRB(ctx jit64CompileContext64) (microOp64, bool, error) {
	width, ok := packedExtractWidth64(ctx.inst.Op)
	if !ok {
		return microOp64{}, false, ErrUnsupported64
	}
	destination, err := pextrDestination64(ctx, width)
	if err != nil {
		return microOp64{}, false, err
	}
	source, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || source.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	immediate, err := pextrImmediate64(ctx)
	if err != nil {
		return microOp64{}, false, err
	}
	return makeSSEExtract64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source, immediate), false, nil
}

func pextrDestination64(ctx jit64CompileContext64, width uint8) (operand64, error) {
	destination, err := operand64FromArg(ctx.arg(0), width)
	if err != nil && ctx.inst.Op != x86asm.PEXTRQ {
		destination, err = operand64FromArg(ctx.arg(0), 4)
		if err != nil {
			destination, err = operand64FromArg(ctx.arg(0), 8)
		}
	}
	if err != nil || (destination.Kind != operand64Reg && destination.Kind != operand64Mem) {
		return operand64{}, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	return destination, nil
}

func pextrImmediate64(ctx jit64CompileContext64) (uint8, error) {
	immediate, ok := ctx.arg(2).(x86asm.Imm)
	if !ok {
		return 0, fmt.Errorf("%s requires an immediate index", ctx.inst.Op)
	}
	return uint8(immediate), nil
}

func init() {
	jit64InstructionHandlers[x86asm.PEXTRB] = jit64CompileCasePEXTRB
	jit64InstructionHandlers[x86asm.PEXTRW] = jit64CompileCasePEXTRB
	jit64InstructionHandlers[x86asm.PEXTRD] = jit64CompileCasePEXTRB
	jit64InstructionHandlers[x86asm.PEXTRQ] = jit64CompileCasePEXTRB
}
