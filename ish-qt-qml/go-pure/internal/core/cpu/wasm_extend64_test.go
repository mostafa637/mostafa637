package cpu

import (
	"context"
	"testing"
)

func TestWasmExtend64(t *testing.T) {
	memory := NewMemory64()
	start := Address64(0x2000)
	if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	code := []byte{0x48, 0xb8, 0x80, 0, 0, 0, 0, 0, 0, 0, 0x48, 0x0f, 0xb6, 0xc8, 0x48, 0x0f, 0xbe, 0xd0, 0x0f, 0xbe, 0xc8, 0xc3}
	if err := memory.Write(start, code); err != nil {
		t.Fatal(err)
	}
	block := compileWasmTestBlock(t, memory, start, code)
	state := NewMachineState64(memory)
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if state.Get(RCX) != 0xffffff80 || state.Get(RDX) != ^uint64(0x7f) {
		t.Fatalf("RCX=%#x RDX=%#x", state.Get(RCX), state.Get(RDX))
	}
}

func TestWasmExtendMemory64(t *testing.T) {
	memory := NewMemory64()
	start, data := Address64(0x4000), Address64(0x6000)
	if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(data, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	code := []byte{0x48, 0xb8, 0, 0x60, 0, 0, 0, 0, 0, 0, 0x0f, 0xb6, 0x08, 0x0f, 0xbe, 0x10, 0xc3}
	if err := memory.Write(start, code); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(data, []byte{0x80}); err != nil {
		t.Fatal(err)
	}
	block := compileWasmTestBlock(t, memory, start, code)
	state := NewMachineState64(memory)
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if state.Get(RCX) != 0x80 || state.Get(RDX) != 0xffffff80 {
		t.Fatalf("RCX=%#x RDX=%#x", state.Get(RCX), state.Get(RDX))
	}
}

func TestWasmLEA64(t *testing.T) {
	memory := NewMemory64()
	start := Address64(0x3000)
	if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	code := []byte{0x48, 0xb8, 0, 0x10, 0, 0, 0, 0, 0, 0, 0x48, 0x8d, 0x48, 0x10, 0xc3}
	if err := memory.Write(start, code); err != nil {
		t.Fatal(err)
	}
	block := compileWasmTestBlock(t, memory, start, code)
	state := NewMachineState64(memory)
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if state.Get(RCX) != 0x1010 {
		t.Fatalf("RCX=%#x want 0x1010", state.Get(RCX))
	}
}

func compileWasmTestBlock(t *testing.T, memory *Memory64, start Address64, code []byte) *WasmBlock64 {
	t.Helper()
	jit, err := NewWasmJITWithMemory(context.Background(), t.TempDir(), memory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = jit.Close(context.Background()) })
	block, err := jit.CompileBlock64(context.Background(), memory, start, uint64(len(code)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = block.Close(context.Background()) })
	return block
}
