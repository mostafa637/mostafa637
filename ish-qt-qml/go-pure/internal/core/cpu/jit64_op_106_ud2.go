package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseUD2(ctx jit64CompileContext64) (microOp64, bool, error) {
	return microOp64{Address: ctx.address, Size: uint8(ctx.inst.Len), Run: func(state *MachineState64, next uint64) (Flow64, error) {
		state.RIP = next
		state.TrapNo = Trap64InvalidOpcode
		return Flow64Interrupt, nil
	}}, true, nil
}

func init() {
	jit64InstructionHandlers[x86asm.UD2] = jit64CompileCaseUD2
}
