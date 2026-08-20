package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseFLD(ctx jit64CompileContext64) (microOp64, bool, error) {
	if index, ok := x87Index64(ctx.arg(0)); ok {
		return makeFPULoadReg64(ctx.address, uint8(ctx.inst.Len), index), false, nil
	}
	if mem, ok := ctx.arg(0).(x86asm.Mem); ok && (ctx.inst.MemBytes == 4 || ctx.inst.MemBytes == 8) {
		operand, err := operand64FromArg(mem, uint8(ctx.inst.MemBytes))
		if err != nil {
			return microOp64{}, false, err
		}
		return makeFPULoadMem64(ctx.address, uint8(ctx.inst.Len), operand), false, nil
	}
	return microOp64{}, false, fmt.Errorf("FLD operand %v", ctx.arg(0))
}

func init() {
	jit64InstructionHandlers[x86asm.FLD] = jit64CompileCaseFLD
}
