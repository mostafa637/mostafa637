package cpu

import (
	"errors"
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseIMUL(ctx jit64CompileContext64) (microOp64, bool, error) {
	if ctx.arg(0) == nil {
		return microOp64{}, false, errors.New("IMUL requires an operand")
	}
	if ctx.arg(1) == nil {
		return jit64CompileImplicitIMUL(ctx)
	}
	return jit64CompileExplicitIMUL(ctx)
}

func jit64CompileImplicitIMUL(ctx jit64CompileContext64) (microOp64, bool, error) {
	source, err := operand64FromArg(ctx.arg(0), ctx.width)
	if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("IMUL source: %v", err)
	}
	return makeImplicitArithmetic64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, source, ctx.width), false, nil
}

func jit64CompileExplicitIMUL(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), ctx.width)
	if err != nil || destination.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("IMUL destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), ctx.width)
	if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("IMUL source: %v", err)
	}
	if ctx.arg(2) == nil {
		return makeExplicitIMul64(ctx.address, uint8(ctx.inst.Len), destination, destination, source, ctx.width), false, nil
	}
	return jit64CompileImmediateIMUL(ctx, destination, source)
}

func jit64CompileImmediateIMUL(ctx jit64CompileContext64, destination, source operand64) (microOp64, bool, error) {
	immediate, ok := ctx.arg(2).(x86asm.Imm)
	if !ok {
		return microOp64{}, false, errors.New("IMUL third operand must be immediate")
	}
	multiplier := operand64{Kind: operand64Imm, Imm: uint64(immediate), Width: ctx.width}
	return makeExplicitIMul64(ctx.address, uint8(ctx.inst.Len), destination, source, multiplier, ctx.width), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.IMUL] = jit64CompileCaseIMUL
}
