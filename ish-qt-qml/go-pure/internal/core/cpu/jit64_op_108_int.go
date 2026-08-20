package cpu

import (
	"errors"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseINT(ctx jit64CompileContext64) (microOp64, bool, error) {
	vector, ok := ctx.arg(0).(x86asm.Imm)
	if !ok {
		return microOp64{}, false, errors.New("INT requires immediate vector")
	}
	return microOp64{Address: ctx.address, Size: uint8(ctx.inst.Len), Run: func(state *MachineState64, next uint64) (Flow64, error) {
		state.RIP = next
		state.TrapNo = uint64(uint8(vector))
		return Flow64Interrupt, nil
	}}, true, nil
}

func init() {
	jit64InstructionHandlers[x86asm.INT] = jit64CompileCaseINT
}
