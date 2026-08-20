package cpu

import (
	"context"
	"testing"
)

func TestWasmRunner64SyscallLifecycle(t *testing.T) {
	memory := NewMemory64()
	start := Address64(0x1000)
	if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	code := []byte{0x48, 0xc7, 0xc0, 39, 0, 0, 0, 0x0f, 0x05, 0xc3}
	if err := memory.Write(start, code); err != nil {
		t.Fatal(err)
	}
	handler := func(_ uint64, _ [6]uint64) uint64 { return 123 }
	jit, err := NewWasmJITWithSyscallAndMemory(context.Background(), t.TempDir(), handler, memory)
	if err != nil {
		t.Fatal(err)
	}
	defer jit.Close(context.Background())
	runner := NewWasmRunner64(jit, memory, 32)
	runner.SetSyscall(func(state *MachineState64) (bool, error) {
		return true, nil
	})
	state := NewMachineState64(memory)
	state.RIP = uint64(start)
	if trap := runner.RunToInterrupt(context.Background(), state); trap != Trap64Exit {
		t.Fatalf("trap = %#x, want exit", trap)
	}
	if state.Get(RAX) != 123 {
		t.Fatalf("RAX = %d, want 123", state.Get(RAX))
	}
}
