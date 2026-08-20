package cpu

import "golang.org/x/arch/x86/x86asm"

func compileInstruction64(inst x86asm.Inst, address uint64) (microOp64, bool, error) {
	ctx := newJIT64CompileContext64(inst, address)
	index := uint64(inst.Op)
	if index >= uint64(len(jit64InstructionHandlers)) {
		return jit64UnsupportedHandler(ctx)
	}
	handler := jit64InstructionHandlers[index]
	if handler == nil {
		return jit64UnsupportedHandler(ctx)
	}
	return handler(ctx)
}
