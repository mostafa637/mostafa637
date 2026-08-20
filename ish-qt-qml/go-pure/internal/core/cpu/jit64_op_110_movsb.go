package cpu

import (
	"golang.org/x/arch/x86/x86asm"
)

func jit64CompileCaseMOVSB(ctx jit64CompileContext64) (microOp64, bool, error) {
	stringWidth, ok := stringWidth64(ctx.inst.Op)
	if !ok {
		return microOp64{}, false, ErrUnsupported64
	}
	repeat := stringRepeatMode64(ctx.inst.Prefix)
	return makeString64(ctx.address, uint8(ctx.inst.Len), ctx.inst.Op, stringWidth, uint8(ctx.inst.AddrSize), repeat), false, nil
}

func registerMOVSB64() {
	jit64InstructionHandlers[x86asm.MOVSB] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.MOVSW] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.MOVSD] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.MOVSQ] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.STOSB] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.STOSW] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.STOSD] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.STOSQ] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.LODSB] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.LODSW] = jit64CompileCaseMOVSB
}

func registerMOVSB64Part2() {
	jit64InstructionHandlers[x86asm.LODSD] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.LODSQ] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.CMPSB] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.CMPSW] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.CMPSD] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.CMPSQ] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.SCASB] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.SCASW] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.SCASD] = jit64CompileCaseMOVSB
	jit64InstructionHandlers[x86asm.SCASQ] = jit64CompileCaseMOVSB
}

func init() {
	registerMOVSB64()
	registerMOVSB64Part2()
}
