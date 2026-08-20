package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePUSHF(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makePushFlags64(ctx.address, uint8(ctx.inst.Len)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PUSHF] = jit64CompileCasePUSHF
	jit64InstructionHandlers[x86asm.PUSHFQ] = jit64CompileCasePUSHF
}
