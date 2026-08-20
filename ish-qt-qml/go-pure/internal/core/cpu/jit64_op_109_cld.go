package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCLD(ctx jit64CompileContext64) (microOp64, bool, error) {
	set := ctx.inst.Op == x86asm.STD
	return microOp64{Address: ctx.address, Size: uint8(ctx.inst.Len), Run: func(state *MachineState64, next uint64) (Flow64, error) {
		if set {
			state.RFLAGS |= Flag64DF
		} else {
			state.RFLAGS &^= Flag64DF
		}
		state.RIP = next
		return Flow64Continue, nil
	}}, false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.CLD] = jit64CompileCaseCLD
	jit64InstructionHandlers[x86asm.STD] = jit64CompileCaseCLD
}
