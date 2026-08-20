package wasmjit

import (
	"fmt"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"golang.org/x/arch/x86/x86asm"
)

func decodeX86(src []byte) ([]machinecode.Instruction, error) {
	var out []machinecode.Instruction
	for len(src) > 0 {
		inst, err := x86asm.Decode(src, 64)
		if err != nil {
			return nil, err
		}
		item, err := decodeX86Inst(inst)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
		src = src[inst.Len:]
	}
	return out, nil
}

func decodeX86Inst(inst x86asm.Inst) (machinecode.Instruction, error) {
	switch inst.Op {
	case x86asm.NOP:
		return machinecode.Instruction{Op: machinecode.OpNOP}, nil
	case x86asm.RET:
		return machinecode.Instruction{Op: machinecode.OpRET}, nil
	case x86asm.SYSCALL:
		return machinecode.Instruction{Op: machinecode.OpSyscall}, nil
	case x86asm.MOV:
		return decodeMove(inst)
	case x86asm.ADD:
		return decodeArithmetic(inst, machinecode.OpADDImm)
	case x86asm.SUB:
		return decodeArithmetic(inst, machinecode.OpSUBImm)
	default:
		return machinecode.Instruction{}, fmt.Errorf("wasmjit: unsupported x86 op %s", inst.Op)
	}
}

func decodeMove(inst x86asm.Inst) (machinecode.Instruction, error) {
	dst, ok := decodeReg(inst.Args[0])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	imm, ok := decodeImm(inst.Args[1])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	return machinecode.Instruction{Op: machinecode.OpMOVImm, Dst: dst, Imm: imm}, nil
}

func decodeArithmetic(inst x86asm.Inst, op machinecode.Op) (machinecode.Instruction, error) {
	dst, ok := decodeReg(inst.Args[0])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	imm, ok := decodeImm(inst.Args[1])
	if !ok {
		return machinecode.Instruction{}, ErrUnsupported
	}
	return machinecode.Instruction{Op: op, Dst: dst, Imm: imm}, nil
}

func decodeReg(arg x86asm.Arg) (int16, bool) {
	reg, ok := arg.(x86asm.Reg)
	if !ok || reg < x86asm.RAX || reg > x86asm.R15 {
		return 0, false
	}
	return int16(reg - x86asm.RAX), true
}

func decodeImm(arg x86asm.Arg) (int64, bool) {
	imm, ok := arg.(x86asm.Imm)
	return int64(imm), ok
}
