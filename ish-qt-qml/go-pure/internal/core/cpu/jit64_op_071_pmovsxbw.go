package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePMOVSXBW(ctx jit64CompileContext64) (microOp64, bool, error) {
	sourceWidth, _, _, _, ok := packedWidenSpec64(ctx.inst.Op)
	if !ok {
		return microOp64{}, false, ErrUnsupported64
	}
	destination, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || destination.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("%s destination: %v", ctx.inst.Op, err)
	}
	var source operand64
	if _, isRegister := ctx.arg(1).(x86asm.Reg); isRegister {
		source, err = operand64FromArg(ctx.arg(1), 16)
	} else {
		source, err = operand64FromArg(ctx.arg(1), sourceWidth)
	}
	if err != nil || (source.Kind != operand64XMM && source.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("%s source: %v", ctx.inst.Op, err)
	}
	return makeSSEWiden64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, destination, source), false, nil
}

func init() {
	jit64InstructionHandlers[x86asm.PMOVSXBW] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVSXBD] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVSXBQ] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVSXWD] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVSXWQ] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVSXDQ] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVZXBW] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVZXBD] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVZXBQ] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVZXWD] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVZXWQ] = jit64CompileCasePMOVSXBW
	jit64InstructionHandlers[x86asm.PMOVZXDQ] = jit64CompileCasePMOVSXBW
}
