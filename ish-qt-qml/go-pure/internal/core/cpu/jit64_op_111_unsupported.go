package cpu

import (
	"fmt"
)

func jit64CompileCaseUnsupported(ctx jit64CompileContext64) (microOp64, bool, error) {
	if condition, ok := decodeSETcc64(ctx.inst.Op); ok {
		return jit64CompileSETcc(ctx, uint8(condition))
	}
	if condition, ok := decodeCMOVcc64(ctx.inst.Op); ok {
		return jit64CompileCMOVcc(ctx, uint8(condition))
	}
	if condition, ok := decodeCondition64(ctx.inst.Op); ok {
		return jit64CompileJcc(ctx, uint8(condition))
	}
	return microOp64{}, false, fmt.Errorf("%s", ctx.inst.Op)
}

func jit64CompileSETcc(ctx jit64CompileContext64, condition uint8) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), 1)
	if err != nil || (destination.Kind != operand64Reg && destination.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("SETcc destination: %v", err)
	}
	return makeSETcc64(ctx.address, uint8(ctx.inst.Len), conditionCode64(condition), destination), false, nil
}

func jit64CompileCMOVcc(ctx jit64CompileContext64, condition uint8) (microOp64, bool, error) {
	destination, err := operand64FromArg(ctx.arg(0), ctx.width)
	if err != nil || destination.Kind != operand64Reg {
		return microOp64{}, false, fmt.Errorf("CMOVcc destination: %v", err)
	}
	source, err := operand64FromArg(ctx.arg(1), ctx.width)
	if err != nil || (source.Kind != operand64Reg && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("CMOVcc source: %v", err)
	}
	return makeCMOVcc64(ctx.address, uint8(ctx.inst.Len), conditionCode64(condition), destination, source), false, nil
}

func jit64CompileJcc(ctx jit64CompileContext64, condition uint8) (microOp64, bool, error) {
	value, err := operand64FromArg(ctx.arg(0), 8)
	if err != nil {
		return microOp64{}, false, err
	}
	return makeJcc64(ctx.address, uint8(ctx.inst.Len), conditionCode64(condition), value), true, nil
}

func init() {
	jit64UnsupportedHandler = jit64CompileCaseUnsupported
}
