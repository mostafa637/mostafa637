package cpu

import (
	"context"
	"testing"
)

func TestWasmTestRegister(t *testing.T) {
	state, block, jit := testOpBlock(t, []byte{0x48, 0x85, 0xc0, 0xc3})
	defer jit.Close(context.Background())
	defer block.Close(context.Background())
	state.Set(RAX, 0xf0)
	state.RFLAGS = Flag64CF | Flag64OF
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if state.Get(RAX) != 0xf0 {
		t.Fatalf("rax=%#x", state.Get(RAX))
	}
	want := Flag64PF
	if state.RFLAGS&wasmArithmeticFlags != want {
		t.Fatalf("flags=%#x want %#x", state.RFLAGS, want)
	}
}

func TestWasmTestImmediate(t *testing.T) {
	state, block, jit := testOpBlock(t, []byte{0x48, 0xa9, 1, 0, 0, 0, 0xc3})
	defer jit.Close(context.Background())
	defer block.Close(context.Background())
	state.Set(RAX, 2)
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if state.Get(RAX) != 2 {
		t.Fatalf("rax=%#x", state.Get(RAX))
	}
	want := Flag64PF | Flag64ZF
	if state.RFLAGS&wasmArithmeticFlags != want {
		t.Fatalf("flags=%#x want %#x", state.RFLAGS, want)
	}
}

func testOpBlock(t *testing.T, code []byte) (*MachineState64, *WasmBlock64, *WasmJIT) {
	t.Helper()
	memory := NewMemory64()
	const start Address64 = 0x6100
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
	block, err := jit.CompileBlock64(context.Background(), memory, start, uint64(len(code)))
	if err != nil {
		jit.Close(context.Background())
		t.Fatal(err)
	}
	return NewMachineState64(memory), block, jit
}
