package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseRDFSBASE(ctx jit64CompileContext64) (microOp64, bool, error) {
	baseOperand, err := operand64FromArg(ctx.arg(0), ctx.width)
	if err != nil || baseOperand.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("%s operand: %v", ctx.inst.Op, err)
	}
	return makeFSBase64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, baseOperand), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.RDFSBASE] = jit64CompileCaseRDFSBASE
	jit64InstructionHandlers[x86asm.RDGSBASE] = jit64CompileCaseRDFSBASE
	jit64InstructionHandlers[x86asm.WRFSBASE] = jit64CompileCaseRDFSBASE
	jit64InstructionHandlers[x86asm.WRGSBASE] = jit64CompileCaseRDFSBASE
}
