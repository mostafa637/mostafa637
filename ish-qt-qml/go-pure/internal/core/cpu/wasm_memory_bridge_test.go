package cpu

import (
	"context"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"testing"
)

func TestWasmMemoryBridge(t *testing.T) {
	code, err := machinecode.EmitX86([]machinecode.Instruction{
		{Op: machinecode.OpStore64, Dst: int16(RAX), Src: int16(RBX), Imm: 16},
		{Op: machinecode.OpLoad64, Dst: int16(RCX), Src: int16(RAX), Imm: 16},
		{Op: machinecode.OpRET},
	})
	if err != nil {
		t.Fatal(err)
	}
	memory := NewMemory64()
	const start Address64 = 0x3000
	const data Address64 = 0x4000
	if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(data, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(start, code); err != nil {
		t.Fatal(err)
	}
	jit, err := NewWasmJITWithMemory(context.Background(), t.TempDir(), memory)
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
	state.Set(RAX, uint64(data))
	state.Set(RBX, 0x1122334455667788)
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if state.Get(RCX) != 0x1122334455667788 {
		t.Fatalf("RCX = %#x", state.Get(RCX))
	}
}
