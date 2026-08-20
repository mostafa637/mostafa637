package cpu

import (
	"context"
	"testing"
)

func TestWasmCallRetStack(t *testing.T) {
	memory := NewMemory64()
	const start Address64 = 0x4000
	if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	call := []byte{0xe8, 0x0b, 0, 0, 0, 0x48, 0x83, 0xc0, 5, 0xc3}
	callee := []byte{0x48, 0xc7, 0xc0, 7, 0, 0, 0, 0xc2, 8, 0}
	if err := memory.Write(start, call); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(0x4010, callee); err != nil {
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
	state.Regs[RSP] = 0x4ff0
	flow, err := chain.Run(context.Background(), state, 3)
	if err != nil || flow != Flow64Stop {
		t.Fatalf("flow=%v err=%v", flow, err)
	}
	if state.Get(RAX) != 12 || state.Regs[RSP] != 0x4ff8 {
		t.Fatalf("rax=%d rsp=%#x", state.Get(RAX), state.Regs[RSP])
	}
}
