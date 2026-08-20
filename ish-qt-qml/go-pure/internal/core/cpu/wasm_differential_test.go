package cpu

import (
	"bytes"
	"context"
	"testing"
)

type differentialCase64 struct {
	name  string
	code  []byte
	setup func(*MachineState64)
}

func TestWasmDifferential64(t *testing.T) {
	cases := []differentialCase64{
		{name: "arithmetic", code: []byte{0x48, 0xc7, 0xc0, 10, 0, 0, 0, 0x48, 0x83, 0xc0, 5, 0x48, 0x83, 0xe8, 2, 0x48, 0x83, 0xe0, 0x0f, 0x48, 0x83, 0xc8, 0x10, 0x48, 0x83, 0xf0, 3, 0x48, 0x83, 0xf8, 0x1a}},
		{name: "memory", code: []byte{0x48, 0x89, 0x58, 0x10, 0x48, 0x8b, 0x48, 0x10}, setup: func(s *MachineState64) { s.Set(RAX, 0x5000); s.Set(RBX, 0x1122334455667788) }},
		{name: "conditional_branch", code: []byte{0x48, 0xc7, 0xc0, 1, 0, 0, 0, 0x48, 0x83, 0xf8, 1, 0x74, 0x00}},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) { runDifferential64(t, item) })
	}
}
func runDifferential64(t *testing.T, item differentialCase64) {
	direct, wasm, directMem, wasmMem := prepareDifferential64(t, item)
	directBlock, err := CompileBlock64(directMem, 0x1000, uint64(len(item.code)))
	if err != nil {
		t.Fatal(err)
	}
	jit, wasmBlock := compileWasm64(t, wasmMem, item.code)
	defer jit.Close(context.Background())
	defer wasmBlock.Close(context.Background())
	leftFlow, leftErr := directBlock.Run(direct)
	rightFlow, rightErr := wasmBlock.Run(context.Background(), wasm)
	if leftErr != nil || rightErr != nil {
		t.Fatalf("direct=%v wasm=%v", leftErr, rightErr)
	}
	if leftFlow != rightFlow {
		t.Fatalf("flows direct=%v wasm=%v", leftFlow, rightFlow)
	}
	compareAll64(t, direct, wasm, directMem, wasmMem)
}
func prepareDifferential64(t *testing.T, item differentialCase64) (*MachineState64, *MachineState64, *Memory64, *Memory64) {
	base := NewMemory64()
	mapExecutable64(t, base, 0x1000, item.code)
	if err := base.Map(0x5000, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	directMem, wasmMem := base.Clone(), base.Clone()
	direct, wasm := NewMachineState64(directMem), NewMachineState64(wasmMem)
	if item.setup != nil {
		item.setup(direct)
		item.setup(wasm)
	}
	return direct, wasm, directMem, wasmMem
}
func compileWasm64(t *testing.T, memory *Memory64, code []byte) (*WasmJIT, *WasmBlock64) {
	jit, err := NewWasmJITWithMemory(context.Background(), t.TempDir(), memory)
	if err != nil {
		t.Fatal(err)
	}
	block, err := jit.CompileBlock64(context.Background(), memory, 0x1000, uint64(len(code)))
	if err != nil {
		jit.Close(context.Background())
		t.Fatal(err)
	}
	return jit, block
}
func compareAll64(t *testing.T, left, right *MachineState64, lm, rm *Memory64) {
	compareRegs64(t, left, right)
	compareFlags64(t, left, right)
	compareMemory64(t, lm, rm)
}
func compareRegs64(t *testing.T, left, right *MachineState64) {
	for reg := Reg64(0); reg < Reg64Count; reg++ {
		if left.Get(reg) != right.Get(reg) {
			t.Fatalf("%s direct=%#x wasm=%#x", reg, left.Get(reg), right.Get(reg))
		}
	}
}
func compareFlags64(t *testing.T, left, right *MachineState64) {
	for _, flag := range []uint64{Flag64CF, Flag64PF, Flag64AF, Flag64ZF, Flag64SF, Flag64OF} {
		if left.Flag(flag) != right.Flag(flag) {
			t.Fatalf("flag %#x differs", flag)
		}
	}
}
func compareMemory64(t *testing.T, left, right *Memory64) {
	ld, rd := make([]byte, 8), make([]byte, 8)
	if err := left.Read(0x5010, ld); err != nil {
		t.Fatal(err)
	}
	if err := right.Read(0x5010, rd); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ld, rd) {
		t.Fatal("memory differs")
	}
}
