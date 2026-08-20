package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseJMP(ctx jit64CompileContext64) (microOp64, bool, error) {
	value, err := operand64FromArg(ctx.arg(0), 8)
	if err != nil {
		return microOp64{}, false, err
	}
	return makeJmp64(ctx.address, uint8(ctx.inst.Len), value), true, nil
}

func init() {
	jit64InstructionHandlers[x86asm.JMP] = jit64CompileCaseJMP
}
