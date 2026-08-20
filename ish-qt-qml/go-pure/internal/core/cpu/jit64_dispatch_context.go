package cpu

import "golang.org/x/arch/x86/x86asm"

type jit64CompileContext64 struct {
	inst    x86asm.Inst
	address uint64
	width   uint8
}

func (c jit64CompileContext64) arg(index int) x86asm.Arg {
	if index < 0 || index >= len(c.inst.Args) {
		return nil
	}
	return c.inst.Args[index]
}

func (c jit64CompileContext64) two() (operand64, operand64, error) {
	left, err := operand64FromArg(c.arg(0), c.width)
	if err != nil {
		return operand64{}, operand64{}, err
	}
	right, err := operand64FromArg(c.arg(1), c.width)
	return left, right, err
}

func newJIT64CompileContext64(inst x86asm.Inst, address uint64) jit64CompileContext64 {
	width := instructionWidth64(inst, inst.Args[0], inst.Args[1])
	return jit64CompileContext64{inst: inst, address: address, width: width}
}

const jit64InstructionTableSize = 1 << 16

type jit64InstructionHandler64 func(jit64CompileContext64) (microOp64, bool, error)

var jit64InstructionHandlers [jit64InstructionTableSize]jit64InstructionHandler64

var jit64UnsupportedHandler jit64InstructionHandler64 = func(jit64CompileContext64) (microOp64, bool, error) {
	return microOp64{}, false, ErrUnsupported64
}
