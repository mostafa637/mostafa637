package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseLEAVE(ctx jit64CompileContext64) (microOp64, bool, error) {
	stackWidth := uint8(8)
	if ctx.inst.DataSize == 16 {
		// In long mode 0x66 selects the 32-bit LEAVE form.
		stackWidth = 4
	}
	return makeLeave64(ctx.address, uint8(ctx.inst.Len), stackWidth), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.LEAVE] = jit64CompileCaseLEAVE
}
