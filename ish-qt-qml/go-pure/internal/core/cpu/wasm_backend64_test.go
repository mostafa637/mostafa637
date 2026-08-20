package cpu

import (
	"context"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"testing"
)

func TestWasmCompileBlock64(t *testing.T) {
	code, err := machinecode.EmitX86([]machinecode.Instruction{
		{Op: machinecode.OpMOVImm, Dst: int16(RAX), Imm: 5},
		{Op: machinecode.OpADDImm, Dst: int16(RAX), Imm: 3},
		{Op: machinecode.OpRET},
	})
	if err != nil {
		t.Fatal(err)
	}
	memory := NewMemory64()
	const start Address64 = 0x1000
	if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(start, code); err != nil {
		t.Fatal(err)
	}
	jit, err := NewWasmJIT(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer jit.Close(context.Background())
	block, err := jit.CompileBlock64(context.Background(), memory, start, uint64(len(code)))
	if err != nil {
		t.Fatal(err)
	}
	defer block.Close(context.Background())
	state := NewMachineState64(memory)
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(RAX); got != 8 {
		t.Fatalf("RAX = %d, want 8", got)
	}
}
