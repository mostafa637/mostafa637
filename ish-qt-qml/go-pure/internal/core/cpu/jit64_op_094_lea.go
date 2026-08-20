package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseLEA(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, err := operand64FromArg(ctx.arg(0), ctx.width)
	if err != nil || left.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("LEA destination: %v", err)
	}
	right, err := operand64FromArg(ctx.arg(1), 8)
	if err != nil || right.Kind != operand64Mem {
		return microOp64{}, false, fmt.Errorf("LEA source: %v", err)
	}
	return microOp64{Address: ctx.address, Size: uint8(ctx.inst.Len), Run: func(state *MachineState64, next uint64) (Flow64, error) {
		value, err := effectiveAddress64(state, right.Mem, next)
		if err != nil {
			return Flow64Stop, err
		}
		writeReg64(state, left, uint64(value))
		state.RIP = next
		return Flow64Continue, nil
	}}, false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.LEA] = jit64CompileCaseLEA
}
