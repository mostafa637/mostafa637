package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCMPXCHG8B(ctx jit64CompileContext64) (microOp64, bool, error) {
	memoryArg, ok := ctx.arg(0).(x86asm.Mem)
	if !ok {
		return microOp64{}, false, fmt.Errorf("%s requires a memory operand", ctx.inst.Op)
	}
	memoryWidth := uint8(8)
	if ctx.inst.Op == x86asm.CMPXCHG16B {
		memoryWidth = 16
	}
	destination, err := operand64FromArg(memoryArg, memoryWidth)
	if err != nil {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	return makeCmpxchgB64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CMPXCHG8B] = jit64CompileCaseCMPXCHG8B
	jit64InstructionHandlers[x86asm.CMPXCHG16B] = jit64CompileCaseCMPXCHG8B
}
