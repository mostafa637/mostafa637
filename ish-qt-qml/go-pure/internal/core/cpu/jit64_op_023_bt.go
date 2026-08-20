package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseBT(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), ctx.width)
	if err != nil || (destination.Kind != operand64Reg && destination.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	index, err := operand64FromArg(ctx.arg(1), ctx.width)
	if err != nil || (index.Kind != operand64Reg && index.Kind != operand64Imm) {
		return microOp64{}, false, fmt.Errorf("%s bit index: %v", ctx.inst.Op, err)
	}
	return makeBitTest64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, index), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.BT] = jit64CompileCaseBT
	jit64InstructionHandlers[x86asm.BTS] = jit64CompileCaseBT
	jit64InstructionHandlers[x86asm.BTR] = jit64CompileCaseBT
	jit64InstructionHandlers[x86asm.BTC] = jit64CompileCaseBT
}
