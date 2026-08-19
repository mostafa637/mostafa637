package cpu

import (
	"encoding/binary"
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

func TestExecutorMemoryOperandsAndLEA(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0xB8, 0x44, 0x33, 0x22, 0x11, // mov eax, 0x11223344
		0x89, 0x43, 0x04, // mov [ebx+4], eax
		0x8B, 0x4B, 0x04, // mov ecx, [ebx+4]
		0x89, 0x4B, 0x08, // mov [ebx+8], ecx
		0x8D, 0x53, 0x08, // lea edx, [ebx+8]
		0x8B, 0x02, // mov eax, [edx]
		0xC7, 0x43, 0x0C, 0x78, 0x56, 0x34, 0x12, // mov dword [ebx+12], 0x12345678
		0xF4,
	})
	executor := NewExecutor(nil)
	if err := executor.Run(state, 32); err != nil {
		t.Fatal(err)
	}
	if state.Get(EAX) != 0x11223344 {
		t.Fatalf("eax = %#x", state.Get(EAX))
	}
	if state.Get(ECX) != 0x11223344 {
		t.Fatalf("ecx = %#x", state.Get(ECX))
	}
	if state.Get(EDX) != 0x2008 {
		t.Fatalf("edx = %#x", state.Get(EDX))
	}
	var raw [4]byte
	if err := state.Memory.Read(0x200c, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 0x12345678 {
		t.Fatalf("stored immediate = %#x", got)
	}
}

func TestExecutorSIBAddressing(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xBF, 0x00, 0x20, 0x00, 0x00, // mov edi, 0x2000
		0xBE, 0x02, 0x00, 0x00, 0x00, // mov esi, 2
		0xB8, 0xEF, 0xBE, 0xAD, 0xDE, // mov eax, 0xdeadbeef
		0x89, 0x84, 0xB7, 0x10, 0x00, 0x00, 0x00, // mov [edi+esi*4+0x10], eax
		0x8B, 0x8C, 0xB7, 0x10, 0x00, 0x00, 0x00, // mov ecx, [edi+esi*4+0x10]
		0xF4,
	})
	executor := NewExecutor(nil)
	if err := executor.Run(state, 32); err != nil {
		t.Fatal(err)
	}
	if state.Get(ECX) != 0xDEADBEEF {
		t.Fatalf("ecx = %#x", state.Get(ECX))
	}
	var raw [4]byte
	if err := state.Memory.Read(0x2018, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 0xDEADBEEF {
		t.Fatalf("SIB stored value = %#x", got)
	}
}

func TestExecutorExtendedMemoryAndFrameInstructions(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0xC7, 0x03, 0x05, 0x00, 0x00, 0x00, // mov [ebx], 5
		0x83, 0x03, 0x03, // add dword [ebx], 3
		0x0F, 0xB6, 0x03, // movzx eax, byte [ebx]
		0xB9, 0x04, 0x00, 0x00, 0x00, // mov ecx, 4
		0x0F, 0xAF, 0xC1, // imul eax, ecx
		0xF4,
	})
	executor := NewExecutor(nil)
	if err := executor.Run(state, 32); err != nil {
		t.Fatal(err)
	}
	if state.Get(EAX) != 32 {
		t.Fatalf("eax = %#x, want 32", state.Get(EAX))
	}
	var raw [4]byte
	if err := state.Memory.Read(0x2000, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 8 {
		t.Fatalf("memory = %#x, want 8", got)
	}
}

func TestExecutorPushPopAll(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0x11, 0x00, 0x00, 0x00, // eax = 0x11
		0xBB, 0x22, 0x00, 0x00, 0x00, // ebx = 0x22
		0x60,       // pushad
		0x31, 0xC0, // xor eax, eax
		0x61, // popad
		0xF4,
	})
	executor := NewExecutor(nil)
	if err := executor.Run(state, 32); err != nil {
		t.Fatal(err)
	}
	if state.Get(EAX) != 0x11 || state.Get(EBX) != 0x22 {
		t.Fatalf("popad registers eax=%#x ebx=%#x", state.Get(EAX), state.Get(EBX))
	}
}

func TestExecutorRetImm(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xC2, 0x04, 0x00, // ret 4
		0xF4, // return target
	})
	returnAddress := state.EIP + 3
	if err := state.Memory.Write(Address(state.Get(ESP)-4), uint32Bytes(returnAddress)); err != nil {
		t.Fatal(err)
	}
	state.Set(ESP, state.Get(ESP)-4)
	before := state.Get(ESP)
	executor := NewExecutor(nil)
	if err := executor.Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if state.Get(ESP) != before+8 {
		t.Fatalf("ret imm esp=%#x, want %#x", state.Get(ESP), before+8)
	}
}

func TestExecutorCWDEAndCDQ(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0x00, 0x80, 0x00, 0x00, // eax = 0x8000
		0x98, // cwde => 0xffff8000
		0x99, // cdq => edx = 0xffffffff
		0xF4,
	})
	executor := NewExecutor(nil)
	if err := executor.Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if state.Get(EAX) != 0xffff8000 || state.Get(EDX) != 0xffffffff {
		t.Fatalf("cwde/cdq eax=%#x edx=%#x", state.Get(EAX), state.Get(EDX))
	}
}

func TestExecutorGSRelativeMemoryOperand(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0x65, 0x8B, 0x03, // mov eax, gs:[ebx]
		0xF4,
	})
	if err := memory.Map(3, 1, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(3*PageSize, uint32Bytes(0xA1B2C3D4)); err != nil {
		t.Fatal(err)
	}
	state.GSBase = 3 * PageSize
	state.TLS = state.GSBase
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 0xA1B2C3D4 {
		t.Fatalf("GS-relative load = %#x, want %#x", got, uint32(0xA1B2C3D4))
	}
}

func TestDisassemble32GSInstruction(t *testing.T) {
	memory, _ := mappedCode(t, []byte{
		0x65, 0x8B, 0x03, // mov eax, gs:[ebx]
	})
	instruction, err := Disassemble32(memory, PageSize)
	if err != nil {
		t.Fatal(err)
	}
	if instruction.Len != 3 {
		t.Fatalf("x86asm length = %d, want 3", instruction.Len)
	}
}
