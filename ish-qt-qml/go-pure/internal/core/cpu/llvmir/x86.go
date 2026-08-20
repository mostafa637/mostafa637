package llvmir

import (
	"fmt"

	"golang.org/x/arch/x86/x86asm"
)

var ErrUnsupportedX86 = fmt.Errorf("llvmir: unsupported x86 instruction")

func FromX86(inst x86asm.Inst) (Program, error) {
	value, ok := x86Immediate(inst)
	if !ok {
		return Program{}, ErrUnsupportedX86
	}
	kind, ok := x86OpKind(inst.Op)
	if !ok {
		return Program{}, ErrUnsupportedX86
	}
	return Program{Name: "x86_block", Ops: []Op{{Kind: kind, Value: value}}}, nil
}

func x86OpKind(op x86asm.Op) (OpKind, bool) {
	switch op {
	case x86asm.ADD:
		return OpAdd, true
	case x86asm.SUB:
		return OpSub, true
	case x86asm.IMUL:
		return OpMul, true
	default:
		return 0, false
	}
}

func x86Immediate(inst x86asm.Inst) (uint64, bool) {
	for _, arg := range inst.Args {
		if imm, ok := arg.(x86asm.Imm); ok {
			return uint64(int64(imm)), true
		}
	}
	return 0, false
}
