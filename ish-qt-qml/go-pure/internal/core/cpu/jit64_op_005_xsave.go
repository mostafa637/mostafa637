package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseXSAVE(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 1)
	if err != nil || destination.Kind != operand64Mem {
		return microOp64{}, false, fmt.Errorf("%s requires a memory destination: %v", ctx.inst.Op, err)
	}
	return makeXSAVE64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.XSAVE] = jit64CompileCaseXSAVE
	jit64InstructionHandlers[x86asm.XSAVE64] = jit64CompileCaseXSAVE
	jit64InstructionHandlers[x86asm.XSAVEOPT] = jit64CompileCaseXSAVE
	jit64InstructionHandlers[x86asm.XSAVEOPT64] = jit64CompileCaseXSAVE
	jit64InstructionHandlers[x86asm.XSAVEC] = jit64CompileCaseXSAVE
	jit64InstructionHandlers[x86asm.XSAVEC64] = jit64CompileCaseXSAVE
	jit64InstructionHandlers[x86asm.XSAVES] = jit64CompileCaseXSAVE
	jit64InstructionHandlers[x86asm.XSAVES64] = jit64CompileCaseXSAVE
}
