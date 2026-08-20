package wasmjit

import (
	"context"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"testing"
)

func TestCompileBlockRun(t *testing.T) {
	bytes, err := machinecode.EmitX86(testInstructions())
	if err != nil {
		t.Fatal(err)
	}
	host, err := CompileBlock(context.Background(), GuestBlock{Bytes: bytes, Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close(context.Background())
	var regs [16]uint64
	got, err := host.Run(context.Background(), regs)
	if err != nil || got != 9 {
		t.Fatalf("got=%d err=%v", got, err)
	}
}

func TestCompilerSyscall(t *testing.T) {
	var seen uint64
	compiler, err := NewCompilerWithSyscall(context.Background(), t.TempDir(), func(number uint64, _ [6]uint64) uint64 {
		seen = number
		return 77
	})
	if err != nil {
		t.Fatal(err)
	}
	defer compiler.Close(context.Background())
	insts := []machinecode.Instruction{{Op: machinecode.OpMOVImm, Dst: 0, Imm: 39}, {Op: machinecode.OpSyscall}, {Op: machinecode.OpRET}}
	bytes, err := machinecode.EmitX86(insts)
	if err != nil {
		t.Fatal(err)
	}
	host, err := compiler.Compile(context.Background(), GuestBlock{Bytes: bytes, Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close(context.Background())
	var regs [16]uint64
	got, err := host.Run(context.Background(), regs)
	if err != nil || got != 77 || seen != 39 {
		t.Fatalf("got=%d seen=%d err=%v", got, seen, err)
	}
}

func testInstructions() []machinecode.Instruction {
	return []machinecode.Instruction{
		{Op: machinecode.OpMOVImm, Dst: 0, Imm: 7},
		{Op: machinecode.OpADDImm, Dst: 0, Imm: 3},
		{Op: machinecode.OpSUBImm, Dst: 0, Imm: 1},
		{Op: machinecode.OpRET},
	}
}
