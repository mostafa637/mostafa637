package cpu

import (
	"context"
	"testing"
)

func TestWasmChain64ConditionalBranch(t *testing.T) {
	memory := NewMemory64()
	const start Address64 = 0x4000
	if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	first := []byte{0x48, 0xc7, 0xc0, 5, 0, 0, 0, 0x48, 0x83, 0xf8, 5, 0x74, 3, 0xc3}
	target := []byte{0x48, 0xc7, 0xc0, 7, 0, 0, 0, 0xc3}
	if err := memory.Write(start, first); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(0x4010, target); err != nil {
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
	if err != nil || flow != Flow64Stop || state.Get(RAX) != 7 {
		t.Fatalf("flow=%v rax=%d rip=%#x err=%v", flow, state.Get(RAX), state.RIP, err)
	}
}
