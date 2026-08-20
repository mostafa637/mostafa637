package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseCRC32(ctx jit64CompileContext64) (microOp64, bool, error) {
	destination, err := crc32Destination64(ctx)
	if err != nil {
		return microOp64{}, false, err
	}
	sourceWidth, err := crc32SourceWidth64(ctx)
	if err != nil {
		return microOp64{}, false, err
	}
	if err := validateCRC32Width64(destination.Width, sourceWidth); err != nil {
		return microOp64{}, false, err
	}
	source, err := operand64FromArg(ctx.arg(1), sourceWidth)
	if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("CRC32 source: %v", err)
	}
	return makeCRC32C64(ctx.address, uint8(ctx.inst.Len), destination, source), false, nil
}

func crc32Destination64(ctx jit64CompileContext64) (operand64, error) {
	destination, err := operand64FromArg(ctx.arg(0), 0)
	if err != nil || destination.Kind != operand64Reg || (destination.Width != 4 && destination.Width != 8) {
		return operand64{}, fmt.Errorf("CRC32 destination: %v", err)
	}
	return destination, nil
}

func crc32SourceWidth64(ctx jit64CompileContext64) (uint8, error) {
	if reg, ok := ctx.arg(1).(x86asm.Reg); ok {
		_, _, width, valid := reg64FromX86(reg)
		if !valid {
			return 0, fmt.Errorf("CRC32 source register %v", reg)
		}
		return width, nil
	}
	if _, ok := ctx.arg(1).(x86asm.Mem); ok {
		return uint8(ctx.inst.MemBytes), nil
	}
	return 0, fmt.Errorf("CRC32 source is not register or memory")
}

func validateCRC32Width64(destination, source uint8) error {
	if source != 1 && source != 2 && source != 4 && source != 8 {
		return fmt.Errorf("CRC32 source width %d", source)
	}
	if destination == 4 && source == 8 {
		return fmt.Errorf("CRC32 r32 cannot use r/m64")
	}
	if destination == 8 && source != 1 && source != 8 {
		return fmt.Errorf("CRC32 r64 source width %d", source)
	}
	return nil
}

func init() {
	jit64InstructionHandlers[x86asm.CRC32] = jit64CompileCaseCRC32
}
