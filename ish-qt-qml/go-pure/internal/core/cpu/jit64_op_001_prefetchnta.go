package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePREFETCHNTA(ctx jit64CompileContext64) (microOp64, bool, error) {
	source, err := operand64FromArg(ctx.arg(0), 1)
	if err != nil || source.Kind != operand64Mem {
		return microOp64{}, false, fmt.Errorf("%s requires a memory operand: %v", ctx.inst.Op, err)
	}
	return makeNoOp64(ctx.address, uint8(ctx.inst.Len)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PREFETCHNTA] = jit64CompileCasePREFETCHNTA
	jit64InstructionHandlers[x86asm.PREFETCHT0] = jit64CompileCasePREFETCHNTA
	jit64InstructionHandlers[x86asm.PREFETCHT1] = jit64CompileCasePREFETCHNTA
	jit64InstructionHandlers[x86asm.PREFETCHT2] = jit64CompileCasePREFETCHNTA
	jit64InstructionHandlers[x86asm.PREFETCHW] = jit64CompileCasePREFETCHNTA
}
