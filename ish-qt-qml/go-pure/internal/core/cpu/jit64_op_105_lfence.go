package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseLFENCE(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makeFence64(ctx.address, uint8(ctx.inst.Len)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.LFENCE] = jit64CompileCaseLFENCE
	jit64InstructionHandlers[x86asm.MFENCE] = jit64CompileCaseLFENCE
	jit64InstructionHandlers[x86asm.SFENCE] = jit64CompileCaseLFENCE
}
