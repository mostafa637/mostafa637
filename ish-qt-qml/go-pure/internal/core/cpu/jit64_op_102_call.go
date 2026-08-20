package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCALL(ctx jit64CompileContext64) (microOp64, bool, error) {
	value, err := operand64FromArg(ctx.arg(0), 8)
	if err != nil {
		return microOp64{}, false, err
	}
	return makeCall64(ctx.address, uint8(ctx.inst.Len), value), true, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CALL] = jit64CompileCaseCALL
}
