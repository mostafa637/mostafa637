package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePMULUDQ(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || left.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("SSE special destination: %v", err)
	}
	right, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("SSE special source: %v", err)
	}
	return makeSSESpecialBinary64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, left, right), false, nil
}

func registerPMULUDQ64() {
	jit64InstructionHandlers[x86asm.PMULUDQ] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PMULLD] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PMULDQ] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PMULHUW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PMULLW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PMULHW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PSADBW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PMADDWD] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PMADDUBSW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PMULHRSW] = jit64CompileCasePMULUDQ
}

func registerPMULUDQ64Part2() {
	jit64InstructionHandlers[x86asm.PHADDW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PHADDSW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PHADDD] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PHSUBW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PHSUBSW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PHSUBD] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PACKSSWB] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PACKSSDW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PACKUSWB] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PADDUSB] = jit64CompileCasePMULUDQ
}

func registerPMULUDQ64Part3() {
	jit64InstructionHandlers[x86asm.PADDUSW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PSUBUSB] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PSUBUSW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PADDSB] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PADDSW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PSUBSB] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PSUBSW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PSIGNB] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PSIGNW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PSIGND] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PABSB] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PABSW] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PABSD] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PCMPGTQ] = jit64CompileCasePMULUDQ
	jit64InstructionHandlers[x86asm.PACKUSDW] = jit64CompileCasePMULUDQ
}

func init() {
	registerPMULUDQ64()
	registerPMULUDQ64Part2()
	registerPMULUDQ64Part3()
}
