package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePOPF(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makePopFlags64(ctx.address, uint8(ctx.inst.Len)), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.POPF] = jit64CompileCasePOPF
	jit64InstructionHandlers[x86asm.POPFQ] = jit64CompileCasePOPF
}
