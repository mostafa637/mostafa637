package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseADDSS(ctx jit64CompileContext64) (microOp64, bool, error) {
	width := scalarSSEWidth64(ctx.inst.Op)
	destination, err := operand64ScalarSSEFromArg(ctx.arg(0), width)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("scalar SSE destination: %v", err)
	}
	if ctx.inst.Op == x86asm.SQRTSS || ctx.inst.Op == x86asm.SQRTSD {
		source, sourceErr := operand64ScalarSSEFromArg(ctx.arg(1), width)

		if sourceErr != nil {
			return microOp64{}, false, fmt.Errorf("scalar SSE sqrt source: %v", sourceErr)
		}
		return makeSSEScalarUnary64(ctx.address, uint8(ctx.inst.Len), width, ctx.inst.Op, destination, source), false, nil
	}
	source, sourceErr := operand64ScalarSSEFromArg(ctx.arg(1), width)
	if sourceErr != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("scalar SSE source: %v", sourceErr)
	}
	return makeSSEScalarBinary64(ctx.address, uint8(ctx.inst.Len), width, ctx.inst.Op, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.ADDSS] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.ADDSD] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.SUBSS] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.SUBSD] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.MULSS] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.MULSD] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.DIVSS] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.DIVSD] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.MINSS] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.MINSD] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.MAXSS] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.MAXSD] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.SQRTSS] = jit64CompileCaseADDSS
	jit64InstructionHandlers[x86asm.SQRTSD] = jit64CompileCaseADDSS
}
