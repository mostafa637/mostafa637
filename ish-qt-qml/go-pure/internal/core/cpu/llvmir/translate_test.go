package llvmir

import (
	"strings"
	"testing"

	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/x86/x86asm"
)

func TestFromARM64Add(t *testing.T) {
	inst := arm64asm.Inst{Op: arm64asm.ADD, Enc: 1 << 10, Args: arm64asm.Args{nil, nil, arm64asm.ImmShift{}}}
	program, err := FromARM64(inst)
	if err != nil || len(program.Ops) != 1 || program.Ops[0].Value != 1 {
		t.Fatalf("unexpected ARM64 program: %+v, %v", program, err)
	}
}

func TestFromX86Add(t *testing.T) {
	inst := x86asm.Inst{Op: x86asm.ADD, Args: x86asm.Args{x86asm.Imm(9)}}
	program, err := FromX86(inst)
	if err != nil || len(program.Ops) != 1 || program.Ops[0].Value != 9 {
		t.Fatalf("unexpected x86 program: %+v, %v", program, err)
	}
}

func TestBuildARM64Block(t *testing.T) {
	module, err := BuildARM64Block([]byte{0x00, 0x04, 0x00, 0x91})
	if err != nil || !strings.Contains(module.String(), "add i64") {
		t.Fatalf("unexpected ARM64 IR: %v", err)
	}
}

func TestBuildX86Block(t *testing.T) {
	module, err := BuildX86Block([]byte{0x48, 0x83, 0xc0, 0x01})
	if err != nil || !strings.Contains(module.String(), "add i64") {
		t.Fatalf("unexpected x86 IR: %v", err)
	}
}
