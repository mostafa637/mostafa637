package cpu

import (
	"testing"
)

func mappedCode(t *testing.T, code []byte) (*Memory, *MachineState) {
	t.Helper()
	memory := NewMemory()
	if err := memory.Map(1, 1, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(2, 1, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(Address(PageSize), code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState(memory)
	state.EIP = PageSize
	state.Set(ESP, 2*PageSize)
	return memory, state
}

func TestExecutorArithmeticAndConditionalBranch(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0x05, 0x00, 0x00, 0x00, // mov eax, 5
		0x83, 0xC0, 0x03, // add eax, 3
		0x83, 0xF8, 0x08, // cmp eax, 8
		0x74, 0x05, // jz +5, skip mov
		0xB8, 0x63, 0x00, 0x00, 0x00, // mov eax, 99
		0xF4, // hlt
	})
	executor := NewExecutor(nil)
	if err := executor.Run(state, 16); err != nil {
		t.Fatal(err)
	}
	if state.Get(EAX) != 8 {
		t.Fatalf("eax = %d, want 8", state.Get(EAX))
	}
	if !executor.Halted {
		t.Fatal("executor did not halt")
	}
	if state.Cycle != 5 {
		t.Fatalf("cycles = %d, want 5", state.Cycle)
	}
}

func TestExecutorCallRetAndStack(t *testing.T) {
	code := make([]byte, 16)
	copy(code, []byte{
		0xE8, 0x05, 0x00, 0x00, 0x00, // call 0x0a
		0xF4, // return target: hlt
	})
	copy(code[10:], []byte{
		0xB8, 0x2A, 0x00, 0x00, 0x00, // mov eax, 42
		0xC3, // ret
	})
	_, state := mappedCode(t, code)
	executor := NewExecutor(nil)
	if err := executor.Run(state, 16); err != nil {
		t.Fatal(err)
	}
	if state.Get(EAX) != 42 {
		t.Fatalf("eax = %d, want 42", state.Get(EAX))
	}
	if state.Get(ESP) != 2*PageSize {
		t.Fatalf("esp = %#x, want %#x", state.Get(ESP), 2*PageSize)
	}
}

func TestExecutorSyscallTrap(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0x14, 0x00, 0x00, 0x00, // mov eax, getpid
		0xCD, 0x80, // int 0x80
		0xF4,
	})
	called := false
	executor := NewExecutor(func(state *MachineState) int32 {
		called = true
		state.SetEAX(321)
		return 321
	})
	if err := executor.Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if !called || state.Get(EAX) != 321 {
		t.Fatalf("syscall trap: called=%v eax=%d", called, state.Get(EAX))
	}
	if state.TrapNo != 0x80 {
		t.Fatalf("trap number = %#x", state.TrapNo)
	}
}

func TestExecutorPreservesCarryAcrossInc(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0xFF, 0xFF, 0xFF, 0xFF, // mov eax, -1
		0x05, 0x01, 0x00, 0x00, 0x00, // add eax, 1 => CF=1
		0x40, // inc eax; CF must remain 1
		0xF4,
	})
	executor := NewExecutor(nil)
	if err := executor.Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if !state.Flag(FlagCF) {
		t.Fatal("inc changed carry flag")
	}
	if state.Get(EAX) != 1 {
		t.Fatalf("eax = %d", state.Get(EAX))
	}
}
