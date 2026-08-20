package cpu

import (
	"context"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"testing"
)

func TestWasmRegisterLogicAndCompare(t *testing.T) {
	code, err := machinecode.EmitX86([]machinecode.Instruction{
		{Op: machinecode.OpMOVReg, Dst: int16(RAX), Src: int16(RBX)},
		{Op: machinecode.OpADDReg, Dst: int16(RAX), Src: int16(RBX)},
		{Op: machinecode.OpANDImm, Dst: int16(RAX), Imm: 7},
		{Op: machinecode.OpORImm, Dst: int16(RAX), Imm: 8},
		{Op: machinecode.OpXORImm, Dst: int16(RAX), Imm: 1},
		{Op: machinecode.OpCMPImm, Dst: int16(RAX), Imm: 15},
		{Op: machinecode.OpRET},
	})
	if err != nil {
		t.Fatal(err)
	}
	memory := NewMemory64()
	const start Address64 = 0x2000
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
	defer jit.Close(context.Background())
	block, err := jit.CompileBlock64(context.Background(), memory, start, uint64(len(code)))
	if err != nil {
		t.Fatal(err)
	}
	defer block.Close(context.Background())
	state := NewMachineState64(memory)
	state.Set(RBX, 3)
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	if state.Get(RAX) != 15 || state.RFLAGS&Flag64ZF == 0 {
		t.Fatalf("rax=%d rflags=%#x", state.Get(RAX), state.RFLAGS)
	}
}
