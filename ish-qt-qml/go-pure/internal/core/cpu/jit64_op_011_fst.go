package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseFST(ctx jit64CompileContext64) (microOp64, bool, error) {
	if index, ok := x87Index64(ctx.arg(0)); ok {
		return makeFPUStoreReg64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, index), false, nil
	}
	if mem, ok := ctx.arg(0).(x86asm.Mem); ok && (ctx.inst.MemBytes == 4 || ctx.inst.MemBytes == 8) {
		operand, err := operand64FromArg(mem, uint8(ctx.inst.MemBytes))
		if err != nil {
			return microOp64{}, false, err
		}
		return makeFPUStoreMem64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, operand), false, nil
	}
	return microOp64{}, false, fmt.Errorf("FST operand %v", ctx.arg(0))
}

func init() {
	jit64InstructionHandlers[x86asm.FST] = jit64CompileCaseFST
	jit64InstructionHandlers[x86asm.FSTP] = jit64CompileCaseFST
}
