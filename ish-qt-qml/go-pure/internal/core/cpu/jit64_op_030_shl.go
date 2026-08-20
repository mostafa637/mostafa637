package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseSHL(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), ctx.width)
	if err != nil || (destination.Kind != operand64Reg && destination.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	count, err := operand64FromArg(ctx.arg(1), 1)
	if err != nil || (count.Kind != operand64Reg && count.Kind != operand64Imm) {
		return microOp64{}, false, fmt.Errorf("%s count: %v", ctx.inst.Op, err)
	}
	return makeShift64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, count), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.SHL] = jit64CompileCaseSHL
	jit64InstructionHandlers[x86asm.SHR] = jit64CompileCaseSHL
	jit64InstructionHandlers[x86asm.SAR] = jit64CompileCaseSHL
	jit64InstructionHandlers[x86asm.ROL] = jit64CompileCaseSHL
	jit64InstructionHandlers[x86asm.ROR] = jit64CompileCaseSHL
	jit64InstructionHandlers[x86asm.RCL] = jit64CompileCaseSHL
	jit64InstructionHandlers[x86asm.RCR] = jit64CompileCaseSHL
}
