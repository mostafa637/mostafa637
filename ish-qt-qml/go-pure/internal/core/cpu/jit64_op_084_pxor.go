package cpu

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCasePXOR(ctx jit64CompileContext64) (microOp64, bool, error) {
	left, err := operand64FromArg(ctx.arg(0), 16)
	if err != nil || left.Kind != operand64XMM {
		return microOp64{}, false, fmt.Errorf("SSE ALU destination: %v", err)
	}
	right, err := operand64FromArg(ctx.arg(1), 16)
	if err != nil || (right.Kind != operand64XMM && right.Kind != operand64Mem) {
		return microOp64{}, false, fmt.Errorf("SSE ALU source: %v", err)
	}
	return makeSSEBinary64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, left, right), false, nil
}

func registerPXOR64() {
	jit64InstructionHandlers[x86asm.PXOR] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PAND] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.POR] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PANDN] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PADDB] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PADDW] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PADDD] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PADDQ] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PSUBB] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PSUBW] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PSUBD] = jit64CompileCasePXOR
}

func registerPXOR64Part2() {
	jit64InstructionHandlers[x86asm.PSUBQ] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PCMPEQB] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PCMPEQW] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PCMPEQD] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PCMPEQQ] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PCMPGTB] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PCMPGTW] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PCMPGTD] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PAVGB] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PAVGW] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMINUB] = jit64CompileCasePXOR
}

func registerPXOR64Part3() {
	jit64InstructionHandlers[x86asm.PMAXUB] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMINUW] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMAXUW] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMINUD] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMAXUD] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMINSB] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMAXSB] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMINSW] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMAXSW] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMINSD] = jit64CompileCasePXOR
	jit64InstructionHandlers[x86asm.PMAXSD] = jit64CompileCasePXOR
}

func init() {
	registerPXOR64()
	registerPXOR64Part2()
	registerPXOR64Part3()
}
