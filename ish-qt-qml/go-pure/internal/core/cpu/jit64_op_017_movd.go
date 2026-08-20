package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVD(ctx jit64CompileContext64) (microOp64, bool, error) {
	scalarWidth := movdScalarWidth64(ctx)
	leftWidth, rightWidth := movdOperandWidths64(ctx, scalarWidth)
	left, err := movdOperand64(ctx, 0, leftWidth, "destination")
	if err != nil {
		return microOp64{}, false, err
	}
	right, err := movdOperand64(ctx, 1, rightWidth, "source")
	if err != nil {
		return microOp64{}, false, err
	}
	if err := validateMOVDOperands64(ctx.inst.Op, left, right); err != nil {
		return microOp64{}, false, err
	}
	return makeMOVScalar64(ctx.address, uint8(ctx.inst.Len), scalarWidth, left, right), false, nil
}

func movdScalarWidth64(ctx jit64CompileContext64) uint8 {
	if ctx.inst.Op == x86asm.MOVQ {
		return 8
	}
	return 4
}

func movdOperandWidths64(ctx jit64CompileContext64, scalar uint8) (uint8, uint8) {
	left, right := scalar, scalar
	if reg, ok := ctx.arg(0).(x86asm.Reg); ok && reg >= x86asm.X0 && reg <= x86asm.X15 {
		left = 16
	}
	if reg, ok := ctx.arg(1).(x86asm.Reg); ok && reg >= x86asm.X0 && reg <= x86asm.X15 {
		right = 16
	}
	return left, right
}

func movdOperand64(ctx jit64CompileContext64, index int, width uint8, label string) (operand64, error) {
	operand, err := operand64FromArg(ctx.arg(index), width)
	if err != nil {
		return operand64{}, fmt.Errorf("%s %s: %v", ctx.inst.Op, label, err)
	}
	return operand, nil
}

func validateMOVDOperands64(op x86asm.Op, left, right operand64) error {
	validLeft := left.Kind == operand64XMM || left.Kind == operand64Reg || left.Kind == operand64Mem
	validRight := right.Kind == operand64XMM || right.Kind == operand64Reg || right.Kind == operand64Mem
	if !validLeft || !validRight || (left.Kind == operand64Mem && right.Kind == operand64Mem) {
		return fmt.Errorf("%s unsupported operands", op)
	}
	if left.Kind != operand64XMM && right.Kind != operand64XMM {
		return fmt.Errorf("%s requires an XMM operand", op)
	}
	return nil
}

func init() {
	jit64InstructionHandlers[x86asm.MOVD] = jit64CompileCaseMOVD
	jit64InstructionHandlers[x86asm.MOVQ] = jit64CompileCaseMOVD
}
