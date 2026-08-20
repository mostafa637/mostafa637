package cpu

import (
	"github.com/llir/llvm/ir"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/llvmir"
	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/x86/x86asm"
)

func CompileLLVMARM64(inst arm64asm.Inst) (*ir.Module, error) {
	program, err := llvmir.FromARM64(inst)
	if err != nil {
		return nil, err
	}
	return llvmir.Build(program)
}

func CompileLLVMX86(inst x86asm.Inst) (*ir.Module, error) {
	program, err := llvmir.FromX86(inst)
	if err != nil {
		return nil, err
	}
	return llvmir.Build(program)
}
