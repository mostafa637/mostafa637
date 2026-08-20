package cpu

import (
	"context"
	"testing"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
)

const wasmArithmeticFlags = Flag64CF | Flag64PF | Flag64AF | Flag64ZF | Flag64SF | Flag64OF

type wasmFlagCase struct {
	name       string
	op         machinecode.Op
	input, imm uint64
	want       uint64
}

func TestWasmArithmeticFlags(t *testing.T) {
	cases := []wasmFlagCase{
		{"add_signed_overflow", machinecode.OpADDImm, 0x7fffffffffffffff, 1, Flag64PF | Flag64AF | Flag64SF | Flag64OF},
		{"add_carry_zero", machinecode.OpADDImm, ^uint64(0), 1, Flag64CF | Flag64PF | Flag64AF | Flag64ZF},
		{"sub_borrow_negative", machinecode.OpSUBImm, 0, 1, Flag64CF | Flag64PF | Flag64AF | Flag64SF},
		{"sub_signed_overflow", machinecode.OpSUBImm, 0x8000000000000000, 1, Flag64PF | Flag64AF | Flag64OF},
		{"cmp_equal", machinecode.OpCMPImm, 5, 5, Flag64PF | Flag64ZF},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { checkWasmFlags(t, tc) })
	}
}

func checkWasmFlags(t *testing.T, tc wasmFlagCase) {
	jit, block, memory := makeFlagBlock(t, tc)
	defer jit.Close(context.Background())
	defer block.Close(context.Background())
	state := NewMachineState64(memory)
	state.Set(RAX, tc.input)
	_, packed, rawErr := block.Host.RunRegsFlags(context.Background(), state.Regs, state.RFLAGS)
	if rawErr != nil {
		t.Fatal(rawErr)
	}
	if _, err := block.Run(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	got := state.RFLAGS & wasmArithmeticFlags
	if got != tc.want {
		t.Fatalf("flags=%#x want %#x packed=%#x", got, tc.want, packed)
	}
}

func makeFlagBlock(t *testing.T, tc wasmFlagCase) (*WasmJIT, *WasmBlock64, *Memory64) {
	t.Helper()
	code, err := machinecode.EmitX86([]machinecode.Instruction{{Op: tc.op, Dst: int16(RAX), Imm: int64(tc.imm)}, {Op: machinecode.OpRET}})
	if err != nil {
		t.Fatal(err)
	}
	memory := NewMemory64()
	const start Address64 = 0x5000
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
	return jit, block, memory
}
