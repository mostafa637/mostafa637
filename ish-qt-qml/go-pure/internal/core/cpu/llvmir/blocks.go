package llvmir

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/x86/x86asm"
)

var ErrTruncatedBlock = fmt.Errorf("llvmir: truncated instruction block")

func BuildARM64Block(code []byte) (*ir.Module, error) {
	program, err := ProgramARM64(code)
	if err != nil {
		return nil, err
	}
	return Build(program)
}

func ProgramARM64(code []byte) (Program, error) {
	if len(code)%4 != 0 {
		return Program{}, ErrTruncatedBlock
	}
	program := Program{Name: "arm64_block"}
	for offset := 0; offset < len(code); offset += 4 {
		inst, err := arm64asm.Decode(code[offset : offset+4])
		if err != nil {
			return Program{}, err
		}
		part, err := FromARM64(inst)
		if err != nil {
			return Program{}, err
		}
		program.Ops = append(program.Ops, part.Ops...)
	}
	return program, nil
}

func BuildX86Block(code []byte) (*ir.Module, error) {
	program, err := ProgramX86(code)
	if err != nil {
		return nil, err
	}
	return Build(program)
}

func ProgramX86(code []byte) (Program, error) {
	program := Program{Name: "x86_block"}
	for len(code) > 0 {
		inst, err := x86asm.Decode(code, 64)
		if err != nil {
			return Program{}, err
		}
		part, err := FromX86(inst)
		if err != nil {
			return Program{}, err
		}
		program.Ops = append(program.Ops, part.Ops...)
		if inst.Len <= 0 || inst.Len > len(code) {
			return Program{}, ErrTruncatedBlock
		}
		code = code[inst.Len:]
	}
	return program, nil
}
