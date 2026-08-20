package cpu

import (
	"context"
	"testing"
)

func TestWasmChain64JumpsAndInvalidation(t *testing.T) {
	memory := NewMemory64()
	const start Address64 = 0x3000
	if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	first := []byte{0x48, 0xc7, 0xc0, 1, 0, 0, 0, 0xeb, 7}
	second := []byte{0x48, 0x83, 0xc0, 2, 0xc3}
	if err := memory.Write(start, first); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(0x3010, second); err != nil {
		t.Fatal(err)
	}
	jit, err := NewWasmJITWithMemory(context.Background(), t.TempDir(), memory)
	if err != nil {
		t.Fatal(err)
	}
	defer jit.Close(context.Background())
	chain := NewWasmChain64(jit, memory, 32)
	defer chain.Close(context.Background())
	state := NewMachineState64(memory)
	state.RIP = uint64(start)
	flow, err := chain.Run(context.Background(), state, 4)
	if err != nil || flow != Flow64Stop || state.Get(RAX) != 3 {
		t.Fatalf("flow=%v rax=%d err=%v", flow, state.Get(RAX), err)
	}
	if _, ok := chain.Get(uint64(start)); !ok {
		t.Fatal("compiled jump block missing")
	}
	if err := memory.Write(start, []byte{0x90}); err != nil {
		t.Fatal(err)
	}
	if _, ok := chain.Get(uint64(start)); ok {
		t.Fatal("invalidated block remained cached")
	}
}
