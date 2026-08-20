package cpu

import (
	"context"
	"testing"
)

func TestWasmMemoryBaseIndexScale(t *testing.T) {
	code := []byte{0x48, 0x8b, 0x44, 0x8b, 0x08, 0xc3}
	memory := mappedMemory(t, 0x5000, 0x6000)
	const value Address64 = 0x1122334455667788
	if err := memory.Write(0x6010, u64Bytes(uint64(value))); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(0x5000, code); err != nil {
		t.Fatal(err)
	}
	jit, block := compileRawBlock(t, memory, 0x5000, uint64(len(code)))
	defer jit.Close(context.Background())
	defer block.Close(context.Background())
	state := NewMachineState64(memory)
	state.Set(RBX, 0x6000)
	state.Set(RCX, 2)
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if state.Get(RAX) != uint64(value) {
		t.Fatalf("rax=%#x want %#x", state.Get(RAX), value)
	}
}

func TestWasmMemoryRIPRelative(t *testing.T) {
	code := []byte{0x48, 0x8b, 0x05, 0x20, 0x00, 0x00, 0x00, 0xc3}
	memory := mappedMemory(t, 0x7000)
	const value Address64 = 0xaabbccddeeff0011
	if err := memory.Write(0x7027, u64Bytes(uint64(value))); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(0x7000, code); err != nil {
		t.Fatal(err)
	}
	jit, block := compileRawBlock(t, memory, 0x7000, uint64(len(code)))
	defer jit.Close(context.Background())
	defer block.Close(context.Background())
	state := NewMachineState64(memory)
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if state.Get(RAX) != uint64(value) {
		t.Fatalf("rax=%#x want %#x", state.Get(RAX), value)
	}
}

func mappedMemory(t *testing.T, starts ...Address64) *Memory64 {
	t.Helper()
	memory := NewMemory64()
	for _, start := range starts {
		if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
	}
	return memory
}

func compileRawBlock(t *testing.T, memory *Memory64, start Address64, size uint64) (*WasmJIT, *WasmBlock64) {
	t.Helper()
	jit, err := NewWasmJITWithMemory(context.Background(), t.TempDir(), memory)
	if err != nil {
		t.Fatal(err)
	}
	block, err := jit.CompileBlock64(context.Background(), memory, start, size)
	if err != nil {
		jit.Close(context.Background())
		t.Fatal(err)
	}
	return jit, block
}

func u64Bytes(value uint64) []byte {
	return []byte{byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24), byte(value >> 32), byte(value >> 40), byte(value >> 48), byte(value >> 56)}
}
