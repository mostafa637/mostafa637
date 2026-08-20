package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseLAHF(ctx jit64CompileContext64) (microOp64, bool, error) {
	return makeLAHFSAHF64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.LAHF] = jit64CompileCaseLAHF
	jit64InstructionHandlers[x86asm.SAHF] = jit64CompileCaseLAHF
}
