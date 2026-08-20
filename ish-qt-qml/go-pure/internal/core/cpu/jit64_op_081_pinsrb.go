package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePINSRB(ctx jit64CompileContext64) (microOp64, bool, error) {
	width, ok := packedInsertWidth64(ctx.inst.Op)
	if !ok {
		return microOp64{}, false, ErrUnsupported64
	}
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	source, err := pinsrbSource64(ctx, width)
	if err != nil {
		return microOp64{}, false, err
	}
	immediate, err := pinsrbImmediate64(ctx)
	if err != nil {
		return microOp64{}, false, err
	}
	return makeSSEInsert64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source, immediate), false, nil
}

func pinsrbSource64(ctx jit64CompileContext64, width uint8) (operand64, error) {
	source, err := operand64FromArg(ctx.arg(1), width)
	if err != nil && ctx.inst.Op != x86asm.PINSRQ {
		source, err = operand64FromArg(ctx.arg(1), 4)
		if err != nil {
			source, err = operand64FromArg(ctx.arg(1), 8)
		}
	}
	if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
		return operand64{}, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	return source, nil
}

func pinsrbImmediate64(ctx jit64CompileContext64) (uint8, error) {
	immediate, ok := ctx.arg(2).(x86asm.Imm)
	if !ok {
		return 0, fmt.Errorf("%s requires an immediate index", ctx.inst.Op)
	}
	return uint8(immediate), nil
}

func init() {
	jit64InstructionHandlers[x86asm.PINSRB] = jit64CompileCasePINSRB
	jit64InstructionHandlers[x86asm.PINSRW] = jit64CompileCasePINSRB
	jit64InstructionHandlers[x86asm.PINSRD] = jit64CompileCasePINSRB
	jit64InstructionHandlers[x86asm.PINSRQ] = jit64CompileCasePINSRB
}
