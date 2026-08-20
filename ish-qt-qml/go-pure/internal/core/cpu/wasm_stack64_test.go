package cpu

import (
	"context"
	"testing"
)

func TestWasmPushPop64(t *testing.T) {
	memory := NewMemory64()
	codeAt := Address64(0x1000)
	stackAt := Address64(0x9000)
	mapCodeStack(t, memory, codeAt, stackAt)
	code := []byte{0x6a, 0x7f, 0x58, 0xc3}
	if err := memory.Write(codeAt, code); err != nil {
		t.Fatal(err)
	}
	jit, err := NewWasmJITWithMemory(context.Background(), t.TempDir(), memory)
	if err != nil {
		t.Fatal(err)
	}
	defer jit.Close(context.Background())
	block, err := jit.CompileBlock64(context.Background(), memory, codeAt, uint64(len(code)))
	if err != nil {
		t.Fatal(err)
	}
	defer block.Close(context.Background())
	state := NewMachineState64(memory)
	state.RIP, state.Regs[RSP] = uint64(codeAt), uint64(stackAt+0x800)
	wantSP := state.Regs[RSP]
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if state.Get(RAX) != 0x7f || state.Regs[RSP] != wantSP {
		t.Fatalf("RAX=%#x RSP=%#x want RAX=0x7f RSP=%#x", state.Get(RAX), state.Regs[RSP], wantSP)
	}
}

func TestWasmPopMemory64(t *testing.T) {
	memory := NewMemory64()
	codeAt, stackAt, dataAt := Address64(0x2000), Address64(0x9000), Address64(0xa000)
	mapCodeStack(t, memory, codeAt, stackAt)
	if err := memory.Map(dataAt, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	code := []byte{0x48, 0xb8, 0, 0xa0, 0, 0, 0, 0, 0, 0, 0x6a, 0x7f, 0x48, 0x8f, 0, 0xc3}
	if err := memory.Write(codeAt, code); err != nil {
		t.Fatal(err)
	}
	block := compileWasmTestBlock(t, memory, codeAt, code)
	state := NewMachineState64(memory)
	state.RIP, state.Regs[RSP] = uint64(codeAt), uint64(stackAt+0x800)
	wantSP := state.Regs[RSP]
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	value, err := memory.ReadUint64(dataAt)
	if err != nil {
		t.Fatal(err)
	}
	if value != 0x7f || state.Regs[RSP] != wantSP {
		t.Fatalf("memory=%#x RSP=%#x", value, state.Regs[RSP])
	}
}

func mapCodeStack(t *testing.T, memory *Memory64, code, stack Address64) {
	t.Helper()
	if err := memory.Map(code, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(stack, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
}
