package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseNOP(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makeNoOp64(ctx.address, uint8(ctx.inst.Len)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.NOP] = jit64CompileCaseNOP
	jit64InstructionHandlers[x86asm.PAUSE] = jit64CompileCaseNOP
}
