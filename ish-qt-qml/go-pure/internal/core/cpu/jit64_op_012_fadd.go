package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseFADD(ctx jit64CompileContext64) (microOp64, bool, error) {
	if mem, ok := ctx.arg(0).(x86asm.Mem); ok && (ctx.inst.MemBytes == 4 || ctx.inst.MemBytes == 8) {
		operand, err := operand64FromArg(mem, uint8(ctx.inst.MemBytes))
		if err != nil {
			return microOp64{}, false, err
		}
		return makeFPUArithmeticMem64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, operand), false, nil
	}
	left, leftOK := x87Index64(ctx.arg(0))
	right, rightOK := x87Index64(ctx.arg(1))
	if !leftOK || !rightOK {
		return microOp64{}, false, fmt.Errorf("FPU arithmetic operands %v", ctx.inst.Args)
	}
	return makeFPUArithmeticReg64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, left, right), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.FADD] = jit64CompileCaseFADD
	jit64InstructionHandlers[x86asm.FADDP] = jit64CompileCaseFADD
	jit64InstructionHandlers[x86asm.FSUB] = jit64CompileCaseFADD
	jit64InstructionHandlers[x86asm.FSUBP] = jit64CompileCaseFADD
	jit64InstructionHandlers[x86asm.FSUBRP] = jit64CompileCaseFADD
	jit64InstructionHandlers[x86asm.FMUL] = jit64CompileCaseFADD
	jit64InstructionHandlers[x86asm.FMULP] = jit64CompileCaseFADD
	jit64InstructionHandlers[x86asm.FDIV] = jit64CompileCaseFADD
	jit64InstructionHandlers[x86asm.FDIVP] = jit64CompileCaseFADD
	jit64InstructionHandlers[x86asm.FDIVRP] = jit64CompileCaseFADD
}
