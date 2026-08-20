package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVZX(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := movzxDestination64(ctx)
	if err != nil {
		return microOp64{}, false, err
	}
	sourceWidth, err := movzxSourceWidth64(ctx)
	if err != nil {
		return microOp64{}, false, err
	}
	if err := validateMOVZXWidth64(sourceWidth); err != nil {
		return microOp64{}, false, err
	}
	source, err := operand64FromArg(ctx.arg(1), sourceWidth)
	if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("extend source: %v", err)
	}
	return makeExtend64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op == x86asm.MOVSX, destination, source), false, nil
}

func movzxDestination64(ctx jit64CompileContext64) (operand64, error) {
	destination, err := operand64FromArg(ctx.arg(0), 0)
	if err != nil || destination.Kind != operand64Reg {
		return operand64{}, fmt.Errorf("extend destination: %v", err)
	}
	return destination, nil
}

func movzxSourceWidth64(ctx jit64CompileContext64) (uint8, error) {
	if reg, ok := ctx.arg(1).(x86asm.Reg); ok {
		_, _, width, valid := reg64FromX86(reg)
		if !valid {
			return 0, fmt.Errorf("extend source register %v", reg)
		}
		return width, nil
	}
	if _, ok := ctx.arg(1).(x86asm.Mem); ok {
		return uint8(ctx.inst.MemBytes), nil
	}
	return 0, fmt.Errorf("extend source is not register or memory")
}

func validateMOVZXWidth64(width uint8) error {
	if width != 1 && width != 2 && width != 4 {
		return fmt.Errorf("extend source width %d", width)
	}
	return nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVZX] = jit64CompileCaseMOVZX
	jit64InstructionHandlers[x86asm.MOVSX] = jit64CompileCaseMOVZX
}
