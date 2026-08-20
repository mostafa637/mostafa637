package cpu

import (
	"errors"
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePSLLW(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("SSE shift destination: %v", err)
	}
	immediate, ok := ctx.arg(1).(x86asm.Imm)
	if !ok {
		return microOp64{}, false, errors.New("SSE shift requires an immediate count")
	}
	return makeSSEShift64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, uint8(immediate)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PSLLW] = jit64CompileCasePSLLW
	jit64InstructionHandlers[x86asm.PSLLD] = jit64CompileCasePSLLW
	jit64InstructionHandlers[x86asm.PSLLQ] = jit64CompileCasePSLLW
	jit64InstructionHandlers[x86asm.PSRLW] = jit64CompileCasePSLLW
	jit64InstructionHandlers[x86asm.PSRLD] = jit64CompileCasePSLLW
	jit64InstructionHandlers[x86asm.PSRLQ] = jit64CompileCasePSLLW
	jit64InstructionHandlers[x86asm.PSRAW] = jit64CompileCasePSLLW
	jit64InstructionHandlers[x86asm.PSRAD] = jit64CompileCasePSLLW
	jit64InstructionHandlers[x86asm.PSLLDQ] = jit64CompileCasePSLLW
	jit64InstructionHandlers[x86asm.PSRLDQ] = jit64CompileCasePSLLW
}
