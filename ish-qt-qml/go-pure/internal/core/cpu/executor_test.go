package cpu

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/emu/fpu"
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

func TestExecutorLogicalInstructionsDecodedByX86ASM(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xB8, 0xF0, 0x00, 0x00, 0x00, // mov eax, 0xf0
		0x25, 0x0F, 0x00, 0x00, 0x00, // and eax, 0x0f => 0
		0x0D, 0x03, 0x00, 0x00, 0x00, // or eax, 3 => 3
		0xA9, 0x01, 0x00, 0x00, 0x00, // test eax, 1 => ZF=0
		0xF4,
	})
	andInst, err := Decode(memory, PageSize+5)
	if err != nil {
		t.Fatal(err)
	}
	if andInst.Op != OpLogicalImm || andInst.Group != 1 {
		t.Fatalf("AND decode = %+v", andInst)
	}
	orInst, err := Decode(memory, PageSize+10)
	if err != nil {
		t.Fatal(err)
	}
	if orInst.Op != OpLogicalImm || orInst.Group != 0 {
		t.Fatalf("OR decode = %+v", orInst)
	}
	testInst, err := Decode(memory, PageSize+15)
	if err != nil {
		t.Fatal(err)
	}
	if testInst.Op != OpTestImm {
		t.Fatalf("TEST decode = %+v", testInst)
	}
	executor := NewExecutor(nil)
	if err := executor.Run(state, 16); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 3 {
		t.Fatalf("eax = %#x, want 3", got)
	}
	if state.Flag(FlagZF) {
		t.Fatal("TEST incorrectly set ZF")
	}
}

func TestExecutorLogicalMemoryOperandDecodedByX86ASM(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0xC7, 0x03, 0xF0, 0x00, 0x00, 0x00, // mov dword [ebx], 0xf0
		0x81, 0x23, 0x0F, 0x00, 0x00, 0x00, // and dword [ebx], 0x0f
		0xF4,
	})
	inst, err := Decode(memory, PageSize+11)
	if err != nil {
		t.Fatal(err)
	}
	if inst.Op != OpLogicalImm || !inst.Dst.IsMem || inst.Group != 1 {
		t.Fatalf("memory AND decode = %+v", inst)
	}
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	var raw [4]byte
	if err := memory.Read(0x2000, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 0 {
		t.Fatalf("memory = %#x, want 0", got)
	}
}

func TestExecutorShiftInstructionsDecodedByX86ASM(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xB8, 0x01, 0x00, 0x00, 0x00, // mov eax, 1
		0xC1, 0xE0, 0x02, // shl eax, 2 => 4
		0xD1, 0xE8, // shr eax, 1 => 2
		0xD1, 0xF8, // sar eax, 1 => 1
		0xB9, 0x01, 0x00, 0x00, 0x00, // mov ecx, 1
		0xD3, 0xE0, // shl eax, cl => 2
		0xF4,
	})
	shift, err := Decode(memory, PageSize+5)
	if err != nil {
		t.Fatal(err)
	}
	if shift.Op != OpShift || shift.Group != 0 || shift.Imm != 2 || shift.Dst.Reg != EAX {
		t.Fatalf("SHL decode = %+v", shift)
	}
	shr, err := Decode(memory, PageSize+8)
	if err != nil {
		t.Fatal(err)
	}
	if shr.Op != OpShift || shr.Group != 1 || shr.Imm != 1 {
		t.Fatalf("SHR decode = %+v", shr)
	}
	clShift, err := Decode(memory, PageSize+17)
	if err != nil {
		t.Fatal(err)
	}
	if clShift.Op != OpShift || clShift.Src.Width != 1 || clShift.Src.Reg != ECX {
		t.Fatalf("CL shift decode = %+v", clShift)
	}
	if err := NewExecutor(nil).Run(state, 16); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 2 {
		t.Fatalf("eax = %#x, want 2", got)
	}
}

func TestExecutorShiftFlagsDecodedByX86ASM(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0x00, 0x00, 0x00, 0x80, // mov eax, 0x80000000
		0xC1, 0xE0, 0x01, // shl eax, 1 => 0, CF=1, OF=1
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 4); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 0 {
		t.Fatalf("eax = %#x, want 0", got)
	}
	if !state.Flag(FlagCF) || !state.Flag(FlagOF) || !state.Flag(FlagZF) {
		t.Fatalf("shift flags: cf=%v of=%v zf=%v", state.Flag(FlagCF), state.Flag(FlagOF), state.Flag(FlagZF))
	}
}

func TestExecutorShiftCountZeroPreservesFlags(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0x01, 0x00, 0x00, 0x00, // mov eax, 1
		0xD3, 0xE0, // shl eax, cl; ECX=0, flags unchanged
		0xF4,
	})
	state.Set(ECX, 0)
	state.EFlags |= FlagCF
	state.ExpandFlags()
	if err := NewExecutor(nil).Run(state, 3); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 1 {
		t.Fatalf("eax = %#x, want 1", got)
	}
	if !state.Flag(FlagCF) {
		t.Fatal("count-zero shift changed CF")
	}
}

func TestExecutorUnaryInstructionsDecodedByX86ASM(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xB8, 0x0F, 0x00, 0x00, 0x00, // mov eax, 0xf
		0xF7, 0xD0, // not eax
		0xF4,
	})
	unary, err := Decode(memory, PageSize+5)
	if err != nil {
		t.Fatal(err)
	}
	if unary.Op != OpUnary || unary.Group != 0 || unary.Dst.Reg != EAX {
		t.Fatalf("NOT decode = %+v", unary)
	}
	state.EFlags |= FlagCF
	state.ExpandFlags()
	if err := NewExecutor(nil).Run(state, 3); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 0xfffffff0 {
		t.Fatalf("not eax = %#x, want %#x", got, uint32(0xfffffff0))
	}
	if !state.Flag(FlagCF) {
		t.Fatal("NOT changed CF")
	}
}

func TestExecutorNegFlagsDecodedByX86ASM(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xB8, 0x01, 0x00, 0x00, 0x00, // mov eax, 1
		0xF7, 0xD8, // neg eax
		0xF4,
	})
	unary, err := Decode(memory, PageSize+5)
	if err != nil {
		t.Fatal(err)
	}
	if unary.Op != OpUnary || unary.Group != 1 || unary.Dst.Reg != EAX {
		t.Fatalf("NEG decode = %+v", unary)
	}
	if err := NewExecutor(nil).Run(state, 3); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 0xffffffff {
		t.Fatalf("neg eax = %#x, want %#x", got, uint32(0xffffffff))
	}
	if !state.Flag(FlagCF) || state.Flag(FlagOF) || state.Flag(FlagZF) {
		t.Fatalf("NEG flags: cf=%v of=%v zf=%v", state.Flag(FlagCF), state.Flag(FlagOF), state.Flag(FlagZF))
	}
}

func TestExecutorSetccByteRegister(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0x31, 0xC0, // xor eax, eax: ZF=1
		0x0F, 0x94, 0xC0, // sete al
		0xF4,
	})
	setcc, err := Decode(memory, PageSize+2)
	if err != nil {
		t.Fatal(err)
	}
	if setcc.Op != OpSetcc || setcc.Dst.Width != 1 || setcc.Dst.Reg != EAX || setcc.Dst.ByteOffset != 0 {
		t.Fatalf("SETE AL decode = %+v", setcc)
	}
	if err := NewExecutor(nil).Run(state, 3); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 1 {
		t.Fatalf("SETE AL changed EAX to %#x, want 1", got)
	}
}

func TestExecutorSetccByteMemory(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0x31, 0xC0, // xor eax, eax: ZF=1
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0xC7, 0x03, 0x00, 0x00, 0x00, 0x00, // mov dword ptr [ebx], 0
		0x0F, 0x95, 0x03, // setne byte ptr [ebx] -> 0
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 5); err != nil {
		t.Fatal(err)
	}
	var raw [4]byte
	if err := memory.Read(Address(2*PageSize), raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 0 {
		t.Fatalf("SETNE memory = %#x, want 0", got)
	}
}

func TestExecutorCMOVccRegister(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0x31, 0xC0, // xor eax, eax: ZF=1
		0xBB, 0x2A, 0x00, 0x00, 0x00, // mov ebx, 42
		0x0F, 0x44, 0xCB, // cmove ecx, ebx
		0x0F, 0x45, 0xD3, // cmovne edx, ebx (must not move)
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 5); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(ECX); got != 42 {
		t.Fatalf("CMOVE ECX, EBX = %d, want 42", got)
	}
	if got := state.Get(EDX); got != 0 {
		t.Fatalf("CMOVNE EDX, EBX = %d, want 0", got)
	}
}

func TestExecutorCMOVccMemory(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0x31, 0xC0, // xor eax, eax: ZF=1
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0xC7, 0x03, 0x2A, 0x00, 0x00, 0x00, // mov dword ptr [ebx], 42
		0x0F, 0x44, 0x0B, // cmove ecx, dword ptr [ebx]
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 5); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(ECX); got != 42 {
		t.Fatalf("CMOVE ECX, [EBX] = %d, want 42", got)
	}
}

func TestExecutorXchgRegisters(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0x07, 0x00, 0x00, 0x00, // mov eax, 7
		0xBB, 0x2A, 0x00, 0x00, 0x00, // mov ebx, 42
		0x87, 0xC3, // xchg eax, ebx
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 4); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 42 {
		t.Fatalf("EAX after XCHG = %d, want 42", got)
	}
	if got := state.Get(EBX); got != 7 {
		t.Fatalf("EBX after XCHG = %d, want 7", got)
	}
}

func TestExecutorXchgMemory(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0xB8, 0x07, 0x00, 0x00, 0x00, // mov eax, 7
		0xC7, 0x03, 0x2A, 0x00, 0x00, 0x00, // mov dword ptr [ebx], 42
		0x87, 0x03, // xchg dword ptr [ebx], eax
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 5); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 42 {
		t.Fatalf("EAX after memory XCHG = %d, want 42", got)
	}
	var raw [4]byte
	if err := memory.Read(Address(0x2000), raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 7 {
		t.Fatalf("memory after XCHG = %d, want 7", got)
	}
}

func TestExecutorAddSubMemoryDestination(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0xB8, 0x07, 0x00, 0x00, 0x00, // mov eax, 7
		0xC7, 0x03, 0x2A, 0x00, 0x00, 0x00, // mov dword ptr [ebx], 42
		0x01, 0x03, // add dword ptr [ebx], eax
		0x29, 0x03, // sub dword ptr [ebx], eax
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 6); err != nil {
		t.Fatal(err)
	}
	var raw [4]byte
	if err := memory.Read(Address(0x2000), raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 42 {
		t.Fatalf("memory after ADD/SUB = %d, want 42", got)
	}
	if got := state.Get(EAX); got != 7 {
		t.Fatalf("EAX after ADD/SUB = %d, want 7", got)
	}
	if state.Flag(FlagZF) {
		t.Fatal("ZF unexpectedly set after final SUB")
	}
}

func TestExecutorAddRegisterFromMemory(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0xC7, 0x03, 0x2A, 0x00, 0x00, 0x00, // mov dword ptr [ebx], 42
		0xB8, 0x07, 0x00, 0x00, 0x00, // mov eax, 7
		0x03, 0x03, // add eax, dword ptr [ebx]
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 5); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 49 {
		t.Fatalf("EAX after ADD EAX, [EBX] = %d, want 49", got)
	}
}

func TestExecutorAddSubMemoryImmediate(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0xC7, 0x03, 0x2A, 0x00, 0x00, 0x00, // mov dword ptr [ebx], 42
		0x83, 0x03, 0x05, // add dword ptr [ebx], 5
		0x83, 0x2B, 0x02, // sub dword ptr [ebx], 2
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 5); err != nil {
		t.Fatal(err)
	}
	var raw [4]byte
	if err := memory.Read(Address(0x2000), raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 45 {
		t.Fatalf("memory after immediate ADD/SUB = %d, want 45", got)
	}
}

func TestExecutorMulImplicit(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0x00, 0x00, 0x00, 0x80, // mov eax, 0x80000000
		0xBB, 0x02, 0x00, 0x00, 0x00, // mov ebx, 2
		0xF7, 0xE3, // mul ebx
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 4); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 0 {
		t.Fatalf("EAX after MUL = %#x, want 0", got)
	}
	if got := state.Get(EDX); got != 1 {
		t.Fatalf("EDX after MUL = %#x, want 1", got)
	}
	if !state.Flag(FlagCF) || !state.Flag(FlagOF) {
		t.Fatal("MUL did not set CF and OF for non-zero high half")
	}
}

func TestExecutorIMulForms(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0xFE, 0xFF, 0xFF, 0xFF, // mov eax, -2
		0xBB, 0x03, 0x00, 0x00, 0x00, // mov ebx, 3
		0x0F, 0xAF, 0xC3, // imul eax, ebx
		0x6B, 0xC3, 0x04, // imul eax, ebx, 4
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 5); err != nil {
		t.Fatal(err)
	}
	if got := int32(state.Get(EAX)); got != 12 {
		t.Fatalf("EAX after IMUL forms = %d, want 12", got)
	}
	if got := state.Get(EDX); got != 0 {
		t.Fatalf("EDX changed by two/three operand IMUL = %#x, want 0", got)
	}
	if state.Flag(FlagCF) || state.Flag(FlagOF) {
		t.Fatal("IMUL unexpectedly set CF or OF for representable result")
	}
}

func TestExecutorIMulImplicit(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0xF6, 0xFF, 0xFF, 0xFF, // mov eax, -10
		0xBB, 0x03, 0x00, 0x00, 0x00, // mov ebx, 3
		0xF7, 0xEB, // imul ebx
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 4); err != nil {
		t.Fatal(err)
	}
	if got := int32(state.Get(EAX)); got != -30 {
		t.Fatalf("EAX after implicit IMUL = %d, want -30", got)
	}
	if got := int32(state.Get(EDX)); got != -1 {
		t.Fatalf("EDX after implicit IMUL = %d, want -1", got)
	}
}

func TestExecutorDivImplicit(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xBA, 0x02, 0x00, 0x00, 0x00, // mov edx, 2
		0xB8, 0x1A, 0x00, 0x00, 0x00, // mov eax, 26
		0xBB, 0x08, 0x00, 0x00, 0x00, // mov ebx, 8
		0xF7, 0xF3, // div ebx
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 5); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 0x40000000+3 { // (2<<32 + 26) / 8
		t.Fatalf("EAX after DIV = %#x, want %#x", got, uint32(0x40000000+3))
	}
	if got := state.Get(EDX); got != 2 {
		t.Fatalf("EDX after DIV = %#x, want 2", got)
	}
}

func TestExecutorIDivImplicit(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xBA, 0xFF, 0xFF, 0xFF, 0xFF, // mov edx, -1
		0xB8, 0xF6, 0xFF, 0xFF, 0xFF, // mov eax, -10
		0xBB, 0x03, 0x00, 0x00, 0x00, // mov ebx, 3
		0xF7, 0xFB, // idiv ebx
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 5); err != nil {
		t.Fatal(err)
	}
	if got := int32(state.Get(EAX)); got != -3 {
		t.Fatalf("EAX after IDIV = %d, want -3", got)
	}
	if got := int32(state.Get(EDX)); got != -1 {
		t.Fatalf("EDX after IDIV = %d, want -1", got)
	}
}

func TestExecutorDivisionErrors(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		want error
	}{
		{
			name: "zero divisor",
			code: []byte{
				0xB8, 0x01, 0x00, 0x00, 0x00, // mov eax, 1
				0xBB, 0x00, 0x00, 0x00, 0x00, // mov ebx, 0
				0xF7, 0xF3, // div ebx
			},
			want: ErrDivisionByZero,
		},
		{
			name: "unsigned quotient overflow",
			code: []byte{
				0xBA, 0x01, 0x00, 0x00, 0x00, // mov edx, 1
				0xB8, 0x00, 0x00, 0x00, 0x00, // mov eax, 0
				0xBB, 0x01, 0x00, 0x00, 0x00, // mov ebx, 1
				0xF7, 0xF3, // div ebx
			},
			want: ErrDivisionOverflow,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, state := mappedCode(t, test.code)
			if err := NewExecutor(nil).Run(state, 5); !errors.Is(err, test.want) {
				t.Fatalf("Run error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestExecutorStringInstructions(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xF3, 0xA4, // rep movsb
		0xF3, 0xAA, // rep stosb
		0xF3, 0xAC, // rep lodsb
		0xF4,
	})
	if err := state.Memory.Write(0x2000, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	state.Set(ESI, 0x2000)
	state.Set(EDI, 0x2010)
	state.Set(ECX, 3)
	if _, err := NewExecutor(nil).Step(state); err != nil {
		t.Fatal(err)
	}
	var copied [3]byte
	if err := state.Memory.Read(0x2010, copied[:]); err != nil {
		t.Fatal(err)
	}
	if copied != [3]byte{1, 2, 3} {
		t.Fatalf("REP MOVSB destination = %v, want [1 2 3]", copied)
	}
	if state.Get(ECX) != 0 || state.Get(ESI) != 0x2003 || state.Get(EDI) != 0x2013 {
		t.Fatalf("REP MOVSB indices/count = ecx %#x esi %#x edi %#x", state.Get(ECX), state.Get(ESI), state.Get(EDI))
	}

	state.Set(EAX, 0xAA)
	state.Set(ECX, 3)
	if _, err := NewExecutor(nil).Step(state); err != nil {
		t.Fatal(err)
	}
	var filled [3]byte
	if err := state.Memory.Read(0x2013, filled[:]); err != nil {
		t.Fatal(err)
	}
	if filled != [3]byte{0xAA, 0xAA, 0xAA} {
		t.Fatalf("REP STOSB destination = %v, want [170 170 170]", filled)
	}
	if state.Get(ECX) != 0 || state.Get(EDI) != 0x2016 {
		t.Fatalf("REP STOSB indices/count = ecx %#x edi %#x", state.Get(ECX), state.Get(EDI))
	}

	state.Set(ESI, 0x2001)
	state.Set(ECX, 1)
	if _, err := NewExecutor(nil).Step(state); err != nil {
		t.Fatal(err)
	}
	if state.EAXValue()&0xff != 2 || state.Get(ESI) != 0x2002 {
		t.Fatalf("REP LODSB = eax %#x esi %#x", state.EAXValue(), state.Get(ESI))
	}
}

func TestExecutorStringWordInstructions(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0x66, 0xA5, // movsw
		0x66, 0xAB, // stosw
		0x66, 0xAD, // lodsw
		0xF4,
	})
	if err := memory.Write(0x2100, []byte{0x34, 0x12}); err != nil {
		t.Fatal(err)
	}
	state.Set(ESI, 0x2100)
	state.Set(EDI, 0x2200)
	if _, err := NewExecutor(nil).Step(state); err != nil {
		t.Fatal(err)
	}
	var moved [2]byte
	if err := memory.Read(0x2200, moved[:]); err != nil {
		t.Fatal(err)
	}
	if moved != [2]byte{0x34, 0x12} || state.Get(ESI) != 0x2102 || state.Get(EDI) != 0x2202 {
		t.Fatalf("MOVSW result=%v esi=%#x edi=%#x", moved, state.Get(ESI), state.Get(EDI))
	}

	state.Set(EAX, 0xAABBCCDD)
	state.Set(EDI, 0x2204)
	if _, err := NewExecutor(nil).Step(state); err != nil {
		t.Fatal(err)
	}
	var stored [2]byte
	if err := memory.Read(0x2204, stored[:]); err != nil {
		t.Fatal(err)
	}
	if stored != [2]byte{0xDD, 0xCC} || state.Get(EDI) != 0x2206 {
		t.Fatalf("STOSW result=%v edi=%#x", stored, state.Get(EDI))
	}

	state.Set(EAX, 0xABCD0000)
	state.Set(ESI, 0x2100)
	if _, err := NewExecutor(nil).Step(state); err != nil {
		t.Fatal(err)
	}
	if state.EAXValue() != 0xABCD1234 || state.Get(ESI) != 0x2102 {
		t.Fatalf("LODSW eax=%#x esi=%#x", state.EAXValue(), state.Get(ESI))
	}

	_, state = mappedCode(t, []byte{0xF3, 0x66, 0xA5, 0xF4})
	if err := state.Memory.Write(0x2100, []byte{0x34, 0x12, 0x78, 0x56}); err != nil {
		t.Fatal(err)
	}
	state.Set(ESI, 0x2100)
	state.Set(EDI, 0x2200)
	state.Set(ECX, 2)
	if _, err := NewExecutor(nil).Step(state); err != nil {
		t.Fatal(err)
	}
	var repeated [4]byte
	if err := state.Memory.Read(0x2200, repeated[:]); err != nil {
		t.Fatal(err)
	}
	if repeated != [4]byte{0x34, 0x12, 0x78, 0x56} || state.Get(ECX) != 0 || state.Get(ESI) != 0x2104 || state.Get(EDI) != 0x2204 {
		t.Fatalf("REP MOVSW result=%v ecx=%#x esi=%#x edi=%#x", repeated, state.Get(ECX), state.Get(ESI), state.Get(EDI))
	}

	_, state = mappedCode(t, []byte{0xF3, 0x66, 0xAB, 0xF4})
	state.Set(EAX, 0xAABBCCDD)
	state.Set(EDI, 0x2200)
	state.Set(ECX, 2)
	if err := NewExecutor(nil).Run(state, 2); err != nil {
		t.Fatal(err)
	}
	if err := state.Memory.Read(0x2200, repeated[:]); err != nil {
		t.Fatal(err)
	}
	if repeated != [4]byte{0xDD, 0xCC, 0xDD, 0xCC} || state.Get(ECX) != 0 || state.Get(EDI) != 0x2204 {
		t.Fatalf("REP STOSW result=%v ecx=%#x edi=%#x", repeated, state.Get(ECX), state.Get(EDI))
	}

	_, state = mappedCode(t, []byte{0xF2, 0x66, 0xAF, 0xF4})
	if err := state.Memory.Write(0x2200, []byte{0x11, 0x11, 0x34, 0x12, 0x77, 0x77}); err != nil {
		t.Fatal(err)
	}
	state.Set(EAX, 0xABCD1234)
	state.Set(EDI, 0x2200)
	state.Set(ECX, 3)
	if err := NewExecutor(nil).Run(state, 2); err != nil {
		t.Fatal(err)
	}
	if state.Get(ECX) != 1 || state.Get(EDI) != 0x2204 || !state.Flag(FlagZF) {
		t.Fatalf("REPNE SCASW ecx=%#x edi=%#x zf=%v", state.Get(ECX), state.Get(EDI), state.Flag(FlagZF))
	}

	_, state = mappedCode(t, []byte{0xF3, 0x66, 0xA7, 0xF4})
	if err := state.Memory.Write(0x2100, []byte{1, 0, 2, 0, 3, 0}); err != nil {
		t.Fatal(err)
	}
	if err := state.Memory.Write(0x2200, []byte{1, 0, 4, 0, 3, 0}); err != nil {
		t.Fatal(err)
	}
	state.Set(ESI, 0x2100)
	state.Set(EDI, 0x2200)
	state.Set(ECX, 3)
	if err := NewExecutor(nil).Run(state, 2); err != nil {
		t.Fatal(err)
	}
	if state.Get(ECX) != 1 || state.Get(ESI) != 0x2104 || state.Get(EDI) != 0x2204 || state.Flag(FlagZF) {
		t.Fatalf("REPE CMPSW ecx=%#x esi=%#x edi=%#x zf=%v", state.Get(ECX), state.Get(ESI), state.Get(EDI), state.Flag(FlagZF))
	}

	_, state = mappedCode(t, []byte{0xFD, 0xF3, 0x66, 0xA5, 0xF4})
	if err := state.Memory.Write(0x2200, []byte{0x11, 0x22, 0x33, 0x44}); err != nil {
		t.Fatal(err)
	}
	state.Set(ESI, 0x2202)
	state.Set(EDI, 0x2302)
	state.Set(ECX, 2)
	if err := NewExecutor(nil).Run(state, 3); err != nil {
		t.Fatal(err)
	}
	if err := state.Memory.Read(0x2300, repeated[:]); err != nil {
		t.Fatal(err)
	}
	if repeated != [4]byte{0x11, 0x22, 0x33, 0x44} || state.Get(ESI) != 0x21FE || state.Get(EDI) != 0x22FE {
		t.Fatalf("DF REP MOVSW result=%v esi=%#x edi=%#x", repeated, state.Get(ESI), state.Get(EDI))
	}
}

func TestExecutorRepneScasStopsOnMatch(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xF2, 0xAE, // repne scasb
		0xF4,
	})
	if err := state.Memory.Write(0x2010, []byte{1, 9, 3}); err != nil {
		t.Fatal(err)
	}
	state.Set(EAX, 9)
	state.Set(EDI, 0x2010)
	state.Set(ECX, 3)
	if err := NewExecutor(nil).Run(state, 2); err != nil {
		t.Fatal(err)
	}
	if state.Get(ECX) != 1 || state.Get(EDI) != 0x2012 {
		t.Fatalf("REPNE SCASB indices/count = ecx %#x edi %#x", state.Get(ECX), state.Get(EDI))
	}
	if !state.Flag(FlagZF) {
		t.Fatal("REPNE SCASB did not leave ZF set on match")
	}
}

func TestExecutorRepezCmpsStopsOnMismatch(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xF3, 0xA6, // repe cmpsb
		0xF4,
	})
	if err := state.Memory.Write(0x2000, []byte{4, 5, 6}); err != nil {
		t.Fatal(err)
	}
	if err := state.Memory.Write(0x2010, []byte{4, 9, 6}); err != nil {
		t.Fatal(err)
	}
	state.Set(ESI, 0x2000)
	state.Set(EDI, 0x2010)
	state.Set(ECX, 3)
	if err := NewExecutor(nil).Run(state, 2); err != nil {
		t.Fatal(err)
	}
	if state.Get(ECX) != 1 || state.Get(ESI) != 0x2002 || state.Get(EDI) != 0x2012 {
		t.Fatalf("REPE CMPSB indices/count = ecx %#x esi %#x edi %#x", state.Get(ECX), state.Get(ESI), state.Get(EDI))
	}
	if state.Flag(FlagZF) {
		t.Fatal("REPE CMPSB left ZF set after mismatch")
	}
}

func TestExecutorStringDirectionFlag(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xFD,       // std
		0xF3, 0xA4, // rep movsb
		0xF4,
	})
	if err := state.Memory.Write(0x2000, []byte{'a', 'b', 'c'}); err != nil {
		t.Fatal(err)
	}
	state.Set(ESI, 0x2002)
	state.Set(EDI, 0x2012)
	state.Set(ECX, 3)
	if err := NewExecutor(nil).Run(state, 3); err != nil {
		t.Fatal(err)
	}
	var copied [3]byte
	if err := state.Memory.Read(0x2010, copied[:]); err != nil {
		t.Fatal(err)
	}
	if copied != [3]byte{'a', 'b', 'c'} {
		t.Fatalf("DF REP MOVSB destination = %q, want %q", copied, [3]byte{'a', 'b', 'c'})
	}
	if state.Get(ESI) != 0x1fff || state.Get(EDI) != 0x200f {
		t.Fatalf("DF REP MOVSB indices = esi %#x edi %#x", state.Get(ESI), state.Get(EDI))
	}
}

func TestExecutorPushPopMemory(t *testing.T) {
	code := []byte{
		0xBB, 0x00, 0x11, 0x00, 0x00, // mov ebx, 0x1100
		0xB8, 0x78, 0x56, 0x34, 0x12, // mov eax, 0x12345678
		0x50,             // push eax
		0x8F, 0x43, 0x04, // pop dword ptr [ebx+4]
		0xF4, // hlt
	}
	memory, state := mappedCode(t, code)
	if err := memory.Map(3, 1, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := NewExecutor(nil).Run(state, 16); err != nil {
		t.Fatal(err)
	}
	var raw [4]byte
	if err := memory.Read(Address(0x1104), raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 0x12345678 {
		t.Fatalf("memory value = %#x, want %#x", got, uint32(0x12345678))
	}
	if got := state.Get(ESP); got != 2*PageSize {
		t.Fatalf("esp = %#x, want %#x", got, 2*PageSize)
	}
}

func TestExecutorCallRegisterAndRetImm(t *testing.T) {
	code := make([]byte, 32)
	copy(code, []byte{
		0xB8, 0x0C, 0x10, 0x00, 0x00, // mov eax, 0x100c
		0xFF, 0xD0, // call eax
		0xF4, // return target: hlt
	})
	copy(code[12:], []byte{
		0xB8, 0x2A, 0x00, 0x00, 0x00, // mov eax, 42
		0xC2, 0x08, 0x00, // ret 8
	})
	_, state := mappedCode(t, code)
	if err := NewExecutor(nil).Run(state, 16); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 42 {
		t.Fatalf("eax = %d, want 42", got)
	}
	if got := state.Get(ESP); got != 2*PageSize+8 {
		t.Fatalf("esp = %#x, want %#x", got, 2*PageSize+8)
	}
}

func TestExecutorJccTakenAndNotTaken(t *testing.T) {
	code := []byte{
		0x76, 0x05, // jbe +5: target offset 7
		0xB8, 0x63, 0x00, 0x00, 0x00, // mov eax, 99 (not-taken path)
		0xF4, // hlt
	}
	memory, taken := mappedCode(t, code)
	taken.SetEFlags(FlagCF)
	beforeFlags := taken.EFlags
	if err := NewExecutor(nil).Run(taken, 8); err != nil {
		t.Fatal(err)
	}
	if got := taken.Get(EAX); got != 0 {
		t.Fatalf("taken JBE eax = %d, want 0", got)
	}
	if taken.EFlags != beforeFlags {
		t.Fatalf("taken JBE changed EFLAGS: before %#x after %#x", beforeFlags, taken.EFlags)
	}

	_, notTaken := mappedCode(t, code)
	notTaken.SetEFlags(0)
	beforeFlags = notTaken.EFlags
	if err := NewExecutor(nil).Run(notTaken, 8); err != nil {
		t.Fatal(err)
	}
	if got := notTaken.Get(EAX); got != 99 {
		t.Fatalf("not-taken JBE eax = %d, want 99", got)
	}
	if notTaken.EFlags != beforeFlags {
		t.Fatalf("not-taken JBE changed EFLAGS: before %#x after %#x", beforeFlags, notTaken.EFlags)
	}
	_ = memory
}

func TestExecutorJccSignedCondition(t *testing.T) {
	code := []byte{
		0x7F, 0x05, // jg +5: target offset 7
		0xB8, 0x01, 0x00, 0x00, 0x00, // mov eax, 1 (not-taken path)
		0xF4,
	}
	_, state := mappedCode(t, code)
	state.SetEFlags(FlagZF)
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 1 {
		t.Fatalf("not-taken JG eax = %d, want 1", got)
	}
}

func TestExecutorLoopFamily(t *testing.T) {
	tests := []struct {
		name string
		code []byte
	}{
		{
			name: "loop",
			code: []byte{
				0xB9, 0x02, 0x00, 0x00, 0x00, // mov ecx, 2
				0x31, 0xC0, // xor eax, eax: ZF=1
				0xE2, 0xFE, // loop -2
				0xF4,
			},
		},
		{
			name: "loope",
			code: []byte{
				0xB9, 0x02, 0x00, 0x00, 0x00,
				0x31, 0xC0, // keep ZF=1
				0xE1, 0xFE, // loope -2
				0xF4,
			},
		},
		{
			name: "loopne",
			code: []byte{
				0xB9, 0x02, 0x00, 0x00, 0x00,
				0xB8, 0x01, 0x00, 0x00, 0x00,
				0x3D, 0x00, 0x00, 0x00, 0x00, // cmp eax, 0: ZF=0
				0xE0, 0xFE, // loopne -2
				0xF4,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, state := mappedCode(t, test.code)
			if err := NewExecutor(nil).Run(state, 32); err != nil {
				t.Fatal(err)
			}
			if got := state.Get(ECX); got != 0 {
				t.Fatalf("%s left ECX=%#x, want 0", test.name, got)
			}
		})
	}
}

func TestExecutorLoopDoesNotBranchWhenCounterReachesZero(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB9, 0x01, 0x00, 0x00, 0x00, // mov ecx, 1
		0xE2, 0xFE, // loop -2: decrement to zero, then fall through
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if state.Get(ECX) != 0 || state.EIP != PageSize+8 {
		t.Fatalf("LOOP termination: ecx=%#x eip=%#x", state.Get(ECX), state.EIP)
	}
}

func TestExecutorJECXZ(t *testing.T) {
	code := []byte{
		0xE3, 0x05, // jecxz +5: target offset 7
		0xB8, 0x01, 0x00, 0x00, 0x00, // mov eax, 1 (not-taken path)
		0xF4,
	}
	_, taken := mappedCode(t, code)
	taken.Set(ECX, 0)
	if err := NewExecutor(nil).Run(taken, 8); err != nil {
		t.Fatal(err)
	}
	if taken.Get(EAX) != 0 || taken.Get(ECX) != 0 {
		t.Fatalf("taken JECXZ: eax=%d ecx=%#x", taken.Get(EAX), taken.Get(ECX))
	}

	_, notTaken := mappedCode(t, code)
	notTaken.Set(ECX, 1)
	if err := NewExecutor(nil).Run(notTaken, 8); err != nil {
		t.Fatal(err)
	}
	if notTaken.Get(EAX) != 1 || notTaken.Get(ECX) != 1 {
		t.Fatalf("not-taken JECXZ: eax=%d ecx=%#x", notTaken.Get(EAX), notTaken.Get(ECX))
	}
}

func TestExecutorJCXZUsesLowSixteenBits(t *testing.T) {
	code := []byte{
		0x67, 0xE3, 0x05, // jcxz +5: target offset 8
		0xB8, 0x01, 0x00, 0x00, 0x00, // mov eax, 1 (not-taken path)
		0xF4,
	}
	_, taken := mappedCode(t, code)
	taken.Set(ECX, 0x00010000) // CX == 0, ECX != 0.
	if err := NewExecutor(nil).Run(taken, 8); err != nil {
		t.Fatal(err)
	}
	if taken.Get(EAX) != 0 || taken.Get(ECX) != 0x00010000 {
		t.Fatalf("taken JCXZ: eax=%d ecx=%#x", taken.Get(EAX), taken.Get(ECX))
	}

	_, notTaken := mappedCode(t, code)
	notTaken.Set(ECX, 1)
	if err := NewExecutor(nil).Run(notTaken, 8); err != nil {
		t.Fatal(err)
	}
	if notTaken.Get(EAX) != 1 || notTaken.Get(ECX) != 1 {
		t.Fatalf("not-taken JCXZ: eax=%d ecx=%#x", notTaken.Get(EAX), notTaken.Get(ECX))
	}
}

func TestExecutorMovsxAndMovzxMemory(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0x0F, 0xBE, 0x43, 0x00, // movsx eax, byte [ebx]
		0x0F, 0xBF, 0x4B, 0x02, // movsx ecx, word [ebx+2]
		0x0F, 0xB6, 0x53, 0x00, // movzx edx, byte [ebx]
		0xF4,
	})
	base := Address(2 * PageSize)
	if err := memory.Write(base, []byte{0x80, 0x00, 0x80, 0xFF}); err != nil {
		t.Fatal(err)
	}
	state.Set(EBX, uint32(base))
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 0xffffff80 {
		t.Fatalf("MOVSX byte eax=%#x, want 0xffffff80", got)
	}
	if got := state.Get(ECX); got != 0xffffff80 {
		t.Fatalf("MOVSX word ecx=%#x, want 0xffffff80", got)
	}
	if got := state.Get(EDX); got != 0x80 {
		t.Fatalf("MOVZX byte edx=%#x, want 0x80", got)
	}
}

func TestExecutorCmpxchgMemorySuccess(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // ebx = 0x2000
		0xC7, 0x03, 0x05, 0x00, 0x00, 0x00, // [ebx] = 5
		0xB8, 0x05, 0x00, 0x00, 0x00, // eax = 5 (accumulator)
		0xB9, 0x07, 0x00, 0x00, 0x00, // ecx = 7
		0x0F, 0xB1, 0x0B, // cmpxchg [ebx], ecx
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 16); err != nil {
		t.Fatal(err)
	}
	var raw [4]byte
	if err := memory.Read(0x2000, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 7 {
		t.Fatalf("cmpxchg success memory = %#x, want 7", got)
	}
	if state.Get(EAX) != 5 || !state.Flag(FlagZF) || state.Flag(FlagCF) {
		t.Fatalf("cmpxchg success eax=%#x zf=%v cf=%v", state.Get(EAX), state.Flag(FlagZF), state.Flag(FlagCF))
	}
}

func TestExecutorCmpxchgMemoryFailure(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // ebx = 0x2000
		0xC7, 0x03, 0x03, 0x00, 0x00, 0x00, // [ebx] = 3
		0xB8, 0x05, 0x00, 0x00, 0x00, // eax = 5 (wrong accumulator)
		0xB9, 0x07, 0x00, 0x00, 0x00, // ecx = 7
		0x0F, 0xB1, 0x0B, // cmpxchg [ebx], ecx
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 16); err != nil {
		t.Fatal(err)
	}
	var raw [4]byte
	if err := memory.Read(0x2000, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 3 {
		t.Fatalf("cmpxchg failure memory = %#x, want 3", got)
	}
	if state.Get(EAX) != 3 || state.Flag(FlagZF) {
		t.Fatalf("cmpxchg failure eax=%#x zf=%v", state.Get(EAX), state.Flag(FlagZF))
	}
}

func TestExecutorXaddMemoryAndByte(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // ebx = 0x2000
		0xC7, 0x03, 0x05, 0x00, 0x00, 0x00, // [ebx] = 5
		0xB9, 0x03, 0x00, 0x00, 0x00, // ecx = 3
		0x0F, 0xC1, 0x0B, // xadd [ebx], ecx
		0xB8, 0x05, 0x33, 0x22, 0x11, // eax = 0x11223305
		0xB9, 0x03, 0x00, 0x00, 0x00, // ecx = 3 (low byte)
		0x0F, 0xC0, 0xC8, // xadd al, cl
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 24); err != nil {
		t.Fatal(err)
	}
	var raw [4]byte
	if err := memory.Read(0x2000, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 8 {
		t.Fatalf("xadd memory = %#x, want 8", got)
	}
	if state.Get(ECX) != 5 {
		t.Fatalf("xadd byte source = %#x, want 5", state.Get(ECX))
	}
	if state.Get(EAX) != 0x11223308 {
		t.Fatalf("xadd byte destination = %#x, want 0x11223308", state.Get(EAX))
	}
}

func TestExecutorXaddByteFlags(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0xFF, 0x00, 0x00, 0x00, // al = 0xff
		0xB9, 0x01, 0x00, 0x00, 0x00, // cl = 1
		0x0F, 0xC0, 0xC8, // xadd al, cl => al=0, cl=0xff, CF=1
		0xF4,
	})
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if state.Get(EAX)&0xff != 0 || state.Get(ECX)&0xff != 0xff || !state.Flag(FlagCF) || state.Flag(FlagOF) || !state.Flag(FlagZF) {
		t.Fatalf("xadd byte flags eax=%#x ecx=%#x cf=%v of=%v zf=%v", state.Get(EAX), state.Get(ECX), state.Flag(FlagCF), state.Flag(FlagOF), state.Flag(FlagZF))
	}
}

func TestExecutorADCAndSBB(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xB8, 0x05, 0x00, 0x00, 0x00, // mov eax, 5
		0xBB, 0x02, 0x00, 0x00, 0x00, // mov ebx, 2
		0x11, 0xD8, // adc eax, ebx => 8 with initial CF
		0x83, 0xD8, 0x01, // sbb eax, 1 => 7
		0xF4,
	})
	state.EFlags |= FlagCF
	state.ExpandFlags()
	if err := NewExecutor(nil).Run(state, 16); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX); got != 7 {
		t.Fatalf("eax = %d, want 7", got)
	}
	if state.Flag(FlagCF) || state.Flag(FlagOF) {
		t.Fatalf("unexpected carry/overflow after ADC/SBB: cf=%v of=%v", state.Flag(FlagCF), state.Flag(FlagOF))
	}
}

func TestExecutorADCByteAndMemory(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0x14, 0x01, // adc al, 1
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0x89, 0x03, // mov [ebx], eax
		0x11, 0x03, // adc [ebx], eax
		0xF4,
	})
	state.Set(EAX, 0x0000000f)
	state.EFlags |= FlagCF
	state.ExpandFlags()
	executor := NewExecutor(nil)
	if _, err := executor.Step(state); err != nil {
		t.Fatal(err)
	}
	if got := state.Get(EAX) & 0xff; got != 0x11 {
		t.Fatalf("al = %#x, want 0x11", got)
	}
	if !state.Flag(FlagAF) {
		t.Fatal("ADC byte did not set AF across nibble carry")
	}
	if err := executor.Run(state, 16); err != nil {
		t.Fatal(err)
	}
	var raw [4]byte
	if err := memory.Read(0x2000, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 0x22 {
		t.Fatalf("memory = %#x, want 0x22", got)
	}
}

func TestExecutorLAHFSAHF(t *testing.T) {
	_, lahfState := mappedCode(t, []byte{
		0x31, 0xC0, // xor eax, eax => PF=ZF=1
		0x9F, // lahf
		0xF4,
	})
	if err := NewExecutor(nil).Run(lahfState, 8); err != nil {
		t.Fatal(err)
	}
	if got := (lahfState.Get(EAX) >> 8) & 0xff; got != 0x46 {
		t.Fatalf("AH after LAHF = %#x, want 0x46", got)
	}

	_, sahfState := mappedCode(t, []byte{0x9E, 0xF4}) // sahf; hlt
	sahfState.Set(EAX, 0x0000d700)
	sahfState.EFlags |= FlagOF
	sahfState.ExpandFlags()
	if err := NewExecutor(nil).Run(sahfState, 4); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []uint32{FlagCF, FlagPF, FlagAF, FlagZF, FlagSF, FlagOF} {
		if !sahfState.Flag(flag) {
			t.Fatalf("SAHF did not set flag %#x", flag)
		}
	}
}

func TestExecutorBitOperations(t *testing.T) {
	t.Run("bswap", func(t *testing.T) {
		_, state := mappedCode(t, []byte{0x0F, 0xC8, 0xF4}) // bswap eax; hlt
		state.Set(EAX, 0x11223344)
		if err := NewExecutor(nil).Run(state, 4); err != nil {
			t.Fatal(err)
		}
		if got := state.Get(EAX); got != 0x44332211 {
			t.Fatalf("bswap eax = %#x, want 0x44332211", got)
		}
	})

	for _, test := range []struct {
		name string
		code []byte
		want uint32
		cf   bool
	}{
		{name: "bt set", code: []byte{0x0F, 0xA3, 0xC8, 0xF4}, want: 8, cf: true},  // bt eax, ecx
		{name: "bts set", code: []byte{0x0F, 0xAB, 0xC8, 0xF4}, want: 8, cf: true}, // bts eax, ecx
		{name: "btr set", code: []byte{0x0F, 0xB3, 0xC8, 0xF4}, want: 0, cf: true}, // btr eax, ecx
		{name: "btc set", code: []byte{0x0F, 0xBB, 0xC8, 0xF4}, want: 0, cf: true}, // btc eax, ecx
	} {
		t.Run(test.name, func(t *testing.T) {
			_, state := mappedCode(t, test.code)
			state.Set(EAX, 8)
			state.Set(ECX, 3)
			if err := NewExecutor(nil).Run(state, 4); err != nil {
				t.Fatal(err)
			}
			if got := state.Get(EAX); got != test.want {
				t.Fatalf("eax = %#x, want %#x", got, test.want)
			}
			if state.Flag(FlagCF) != test.cf {
				t.Fatalf("cf = %v, want %v", state.Flag(FlagCF), test.cf)
			}
		})
	}
}

func TestExecutorBitOperationsMemoryIndex(t *testing.T) {
	memory, state := mappedCode(t, []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // ebx = 0x2000
		0xB9, 0x20, 0x00, 0x00, 0x00, // ecx = 32
		0x0F, 0xAB, 0x0B, // bts dword ptr [ebx], ecx => [ebx+4], bit 0
		0xF4,
	})
	var initial [4]byte
	binary.LittleEndian.PutUint32(initial[:], 0x80000000)
	if err := memory.Write(0x2004, initial[:]); err != nil {
		t.Fatal(err)
	}
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	var raw [4]byte
	if err := memory.Read(0x2000, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 0 {
		t.Fatalf("word before indexed bit = %#x, want 0", got)
	}
	if err := memory.Read(0x2004, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(raw[:]); got != 0x80000001 {
		t.Fatalf("word after indexed BTS = %#x, want 0x80000001", got)
	}
	if state.Flag(FlagCF) {
		t.Fatal("BTS unexpectedly reported the previously clear bit as set")
	}
}

func TestExecutorBitScan(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0x0F, 0xBC, 0xC3, // bsf eax, ebx
		0x0F, 0xBD, 0xCB, // bsr ecx, ebx
		0xF4,
	})
	state.Set(EBX, 0x80100020)
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if state.Get(EAX) != 5 || state.Get(ECX) != 31 || state.Flag(FlagZF) {
		t.Fatalf("bit scan eax=%d ecx=%d zf=%v, want 5/31/false", state.Get(EAX), state.Get(ECX), state.Flag(FlagZF))
	}

	_, zeroState := mappedCode(t, []byte{0x0F, 0xBC, 0xC3, 0xF4}) // bsf eax, ebx; hlt
	zeroState.Set(EAX, 0xDEADBEEF)
	zeroState.Set(EBX, 0)
	if err := NewExecutor(nil).Run(zeroState, 4); err != nil {
		t.Fatal(err)
	}
	if zeroState.Get(EAX) != 0xDEADBEEF || !zeroState.Flag(FlagZF) {
		t.Fatalf("zero BSF eax=%#x zf=%v, want destination unchanged and zf=true", zeroState.Get(EAX), zeroState.Flag(FlagZF))
	}
}

func TestExecutorPopcnt(t *testing.T) {
	_, state := mappedCode(t, []byte{0xF3, 0x0F, 0xB8, 0xC3, 0xF4}) // popcnt eax, ebx; hlt
	state.Set(EBX, 0xF0F0F0F0)
	state.EFlags |= FlagCF | FlagPF | FlagAF | FlagSF | FlagOF | FlagZF
	state.ExpandFlags()
	if err := NewExecutor(nil).Run(state, 4); err != nil {
		t.Fatal(err)
	}
	if state.Get(EAX) != 16 || state.Flag(FlagCF) || state.Flag(FlagPF) || state.Flag(FlagAF) || state.Flag(FlagSF) || state.Flag(FlagOF) || state.Flag(FlagZF) {
		t.Fatalf("popcnt eax=%d flags cf=%v pf=%v af=%v sf=%v of=%v zf=%v", state.Get(EAX), state.Flag(FlagCF), state.Flag(FlagPF), state.Flag(FlagAF), state.Flag(FlagSF), state.Flag(FlagOF), state.Flag(FlagZF))
	}

	_, zeroState := mappedCode(t, []byte{0xF3, 0x0F, 0xB8, 0xC3, 0xF4})
	zeroState.Set(EBX, 0)
	if err := NewExecutor(nil).Run(zeroState, 4); err != nil {
		t.Fatal(err)
	}
	if zeroState.Get(EAX) != 0 || !zeroState.Flag(FlagZF) {
		t.Fatalf("zero popcnt eax=%d zf=%v, want 0/true", zeroState.Get(EAX), zeroState.Flag(FlagZF))
	}
}

func TestExecutorRotate(t *testing.T) {
	t.Run("rol immediate", func(t *testing.T) {
		_, state := mappedCode(t, []byte{0xD1, 0xC0, 0xF4}) // rol eax, 1; hlt
		state.Set(EAX, 0x80000001)
		if err := NewExecutor(nil).Run(state, 4); err != nil {
			t.Fatal(err)
		}
		if state.Get(EAX) != 3 || !state.Flag(FlagCF) || !state.Flag(FlagOF) {
			t.Fatalf("rol eax=%#x cf=%v of=%v, want 3/true/true", state.Get(EAX), state.Flag(FlagCF), state.Flag(FlagOF))
		}
	})

	t.Run("ror immediate", func(t *testing.T) {
		_, state := mappedCode(t, []byte{0xC1, 0xC8, 0x01, 0xF4}) // ror eax, 1; hlt
		state.Set(EAX, 0x80000001)
		if err := NewExecutor(nil).Run(state, 4); err != nil {
			t.Fatal(err)
		}
		if state.Get(EAX) != 0xC0000000 || !state.Flag(FlagCF) || state.Flag(FlagOF) {
			t.Fatalf("ror eax=%#x cf=%v of=%v, want 0xc0000000/true/false", state.Get(EAX), state.Flag(FlagCF), state.Flag(FlagOF))
		}
	})

	t.Run("rcl through carry", func(t *testing.T) {
		_, state := mappedCode(t, []byte{0xD1, 0xD0, 0xF4}) // rcl eax, 1; hlt
		state.Set(EAX, 0x80000000)
		state.EFlags |= FlagCF
		state.ExpandFlags()
		if err := NewExecutor(nil).Run(state, 4); err != nil {
			t.Fatal(err)
		}
		if state.Get(EAX) != 1 || !state.Flag(FlagCF) || !state.Flag(FlagOF) {
			t.Fatalf("rcl eax=%#x cf=%v of=%v, want 1/true/true", state.Get(EAX), state.Flag(FlagCF), state.Flag(FlagOF))
		}
	})

	t.Run("rcr memory", func(t *testing.T) {
		memory, state := mappedCode(t, []byte{
			0xB8, 0x00, 0x20, 0x00, 0x00, // eax = 0x2000
			0xC1, 0x18, 0x01, // rcr dword ptr [eax], 1
			0xF4,
		})
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], 1)
		if err := memory.Write(0x2000, raw[:]); err != nil {
			t.Fatal(err)
		}
		state.EFlags |= FlagCF
		state.ExpandFlags()
		if err := NewExecutor(nil).Run(state, 4); err != nil {
			t.Fatal(err)
		}
		if err := memory.Read(0x2000, raw[:]); err != nil {
			t.Fatal(err)
		}
		if got := binary.LittleEndian.Uint32(raw[:]); got != 0x80000000 || !state.Flag(FlagCF) || !state.Flag(FlagOF) {
			t.Fatalf("rcr memory=%#x cf=%v of=%v, want 0x80000000/true/true", got, state.Flag(FlagCF), state.Flag(FlagOF))
		}
	})

	t.Run("cl count", func(t *testing.T) {
		_, state := mappedCode(t, []byte{0xD3, 0xC0, 0xF4}) // rol eax, cl; hlt
		state.Set(EAX, 1)
		state.Set(ECX, 4)
		if err := NewExecutor(nil).Run(state, 4); err != nil {
			t.Fatal(err)
		}
		if state.Get(EAX) != 16 || state.Flag(FlagCF) {
			t.Fatalf("rol cl eax=%#x cf=%v, want 16/false", state.Get(EAX), state.Flag(FlagCF))
		}
	})

	t.Run("count zero preserves flags", func(t *testing.T) {
		_, state := mappedCode(t, []byte{0xC1, 0xC0, 0x20, 0xF4}) // rol eax, 32 -> masked zero; hlt
		state.Set(EAX, 0x12345678)
		state.EFlags |= FlagCF | FlagOF
		state.ExpandFlags()
		if err := NewExecutor(nil).Run(state, 4); err != nil {
			t.Fatal(err)
		}
		if state.Get(EAX) != 0x12345678 || !state.Flag(FlagCF) || !state.Flag(FlagOF) {
			t.Fatalf("zero rotate eax=%#x cf=%v of=%v, want unchanged", state.Get(EAX), state.Flag(FlagCF), state.Flag(FlagOF))
		}
	})

	t.Run("count greater than one preserves undefined OF", func(t *testing.T) {
		_, state := mappedCode(t, []byte{0xC1, 0xC0, 0x04, 0xF4}) // rol eax, 4; hlt
		state.Set(EAX, 1)
		state.EFlags |= FlagOF
		state.ExpandFlags()
		if err := NewExecutor(nil).Run(state, 4); err != nil {
			t.Fatal(err)
		}
		if state.Get(EAX) != 16 || !state.Flag(FlagOF) {
			t.Fatalf("multi-bit rotate eax=%#x of=%v, want 16/unchanged true", state.Get(EAX), state.Flag(FlagOF))
		}
	})
}

func TestExecutorMovByteImmediate(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		reg  Reg32
		want uint32
	}{
		{name: "al", code: []byte{0xB0, 0x12, 0xF4}, reg: EAX, want: 0xA1B2C312},
		{name: "cl", code: []byte{0xB1, 0x34, 0xF4}, reg: ECX, want: 0x11223334},
		{name: "dl", code: []byte{0xB2, 0x56, 0xF4}, reg: EDX, want: 0x55667756},
		{name: "bl", code: []byte{0xB3, 0x78, 0xF4}, reg: EBX, want: 0x99AABB78},
		{name: "ah", code: []byte{0xB4, 0x9A, 0xF4}, reg: EAX, want: 0xA1B29AD4},
		{name: "ch", code: []byte{0xB5, 0xBC, 0xF4}, reg: ECX, want: 0x1122BC44},
		{name: "dh", code: []byte{0xB6, 0xDE, 0xF4}, reg: EDX, want: 0x5566DE88},
		{name: "bh", code: []byte{0xB7, 0xF0, 0xF4}, reg: EBX, want: 0x99AAF0CC},
	}
	initial := []uint32{0xA1B2C3D4, 0x11223344, 0x55667788, 0x99AABBCC}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, state := mappedCode(t, test.code)
			state.Set(EAX, initial[0])
			state.Set(ECX, initial[1])
			state.Set(EDX, initial[2])
			state.Set(EBX, initial[3])
			state.EFlags |= FlagCF | FlagPF | FlagAF | FlagZF | FlagSF | FlagOF
			state.ExpandFlags()
			if err := NewExecutor(nil).Run(state, 4); err != nil {
				t.Fatal(err)
			}
			if got := state.Get(test.reg); got != test.want {
				t.Fatalf("%s result=%#x want=%#x", test.name, got, test.want)
			}
			for _, flag := range []uint32{FlagCF, FlagPF, FlagAF, FlagZF, FlagSF, FlagOF} {
				if !state.Flag(flag) {
					t.Fatalf("MOV byte changed flag %#x", flag)
				}
			}
		})
	}
}

func TestExecutorBound(t *testing.T) {
	const boundsAddress = Address(0x2000)
	writeBounds := func(t *testing.T, memory *Memory, lower, upper int32) {
		t.Helper()
		var raw [8]byte
		binary.LittleEndian.PutUint32(raw[:4], uint32(lower))
		binary.LittleEndian.PutUint32(raw[4:], uint32(upper))
		if err := memory.Write(boundsAddress, raw[:]); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("signed index inside range", func(t *testing.T) {
		memory, state := mappedCode(t, []byte{
			0x62, 0x03, // bound eax, [ebx]
			0xF4,
		})
		state.Set(EBX, uint32(boundsAddress))
		negativeIndex := int32(-2)
		state.Set(EAX, uint32(negativeIndex))
		writeBounds(t, memory, -5, -1)
		state.EFlags |= FlagCF | FlagZF
		state.ExpandFlags()
		if _, err := NewExecutor(nil).Step(state); err != nil {
			t.Fatal(err)
		}
		if state.EIP != PageSize+2 {
			t.Fatalf("EIP after BOUND=%#x, want %#x", state.EIP, PageSize+2)
		}
		if !state.Flag(FlagCF) || !state.Flag(FlagZF) {
			t.Fatalf("BOUND changed flags: cf=%v zf=%v", state.Flag(FlagCF), state.Flag(FlagZF))
		}
	})

	t.Run("out of range traps before advancing", func(t *testing.T) {
		memory, state := mappedCode(t, []byte{
			0x62, 0x03, // bound eax, [ebx]
			0xF4,
		})
		state.Set(EBX, uint32(boundsAddress))
		state.Set(EAX, 8)
		writeBounds(t, memory, -5, 7)
		state.EFlags |= FlagCF | FlagZF
		state.ExpandFlags()
		beforeEIP := state.EIP
		if _, err := NewExecutor(nil).Step(state); !errors.Is(err, ErrBoundRange) {
			t.Fatalf("BOUND error=%v, want %v", err, ErrBoundRange)
		}
		if state.EIP != beforeEIP {
			t.Fatalf("EIP after BOUND trap=%#x, want %#x", state.EIP, beforeEIP)
		}
		if !state.Flag(FlagCF) || !state.Flag(FlagZF) {
			t.Fatalf("BOUND trap changed flags: cf=%v zf=%v", state.Flag(FlagCF), state.Flag(FlagZF))
		}
	})
}

func TestExecutorEnter(t *testing.T) {
	t.Run("simple frame and leave compatibility", func(t *testing.T) {
		memory, state := mappedCode(t, []byte{
			0xC8, 0x08, 0x00, 0x00, // enter 8, 0
			0xC9, // leave
			0xF4,
		})
		state.Set(ESP, 0x2000)
		state.Set(EBP, 0x12345678)
		state.EFlags |= FlagCF | FlagPF | FlagAF | FlagZF | FlagSF | FlagOF
		state.ExpandFlags()
		executor := NewExecutor(nil)
		if _, err := executor.Step(state); err != nil {
			t.Fatal(err)
		}
		if state.Get(EBP) != 0x1ffc || state.Get(ESP) != 0x1ff4 {
			t.Fatalf("ENTER frame ebp=%#x esp=%#x, want ebp=0x1ffc esp=0x1ff4", state.Get(EBP), state.Get(ESP))
		}
		var saved [4]byte
		if err := memory.Read(0x1ffc, saved[:]); err != nil {
			t.Fatal(err)
		}
		if got := binary.LittleEndian.Uint32(saved[:]); got != 0x12345678 {
			t.Fatalf("saved EBP=%#x, want 0x12345678", got)
		}
		for _, flag := range []uint32{FlagCF, FlagPF, FlagAF, FlagZF, FlagSF, FlagOF} {
			if !state.Flag(flag) {
				t.Fatalf("ENTER changed flag %#x", flag)
			}
		}
		if _, err := executor.Step(state); err != nil {
			t.Fatal(err)
		}
		if state.Get(EBP) != 0x12345678 || state.Get(ESP) != 0x2000 {
			t.Fatalf("LEAVE after ENTER ebp=%#x esp=%#x, want ebp=0x12345678 esp=0x2000", state.Get(EBP), state.Get(ESP))
		}
	})

	t.Run("nested display area", func(t *testing.T) {
		memory, state := mappedCode(t, []byte{
			0xC8, 0x04, 0x00, 0x02, // enter 4, 2
			0xF4,
		})
		state.Set(ESP, 0x2000)
		state.Set(EBP, 0x1f00)
		if err := memory.Write(0x1efc, uint32Bytes(0xaaaabbbb)); err != nil {
			t.Fatal(err)
		}
		if _, err := NewExecutor(nil).Step(state); err != nil {
			t.Fatal(err)
		}
		if state.Get(EBP) != 0x1ffc || state.Get(ESP) != 0x1ff0 {
			t.Fatalf("nested ENTER ebp=%#x esp=%#x, want ebp=0x1ffc esp=0x1ff0", state.Get(EBP), state.Get(ESP))
		}
		words := make([]byte, 12)
		if err := memory.Read(0x1ff4, words); err != nil {
			t.Fatal(err)
		}
		want := []uint32{0x1ffc, 0xaaaabbbb, 0x1f00}
		for i, expected := range want {
			if got := binary.LittleEndian.Uint32(words[i*4:]); got != expected {
				t.Fatalf("nested frame word %d=%#x, want %#x", i, got, expected)
			}
		}
	})
}

func TestExecutorMovbe(t *testing.T) {
	t.Run("load register from memory", func(t *testing.T) {
		memory, state := mappedCode(t, []byte{
			0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
			0x0F, 0x38, 0xF0, 0x03, // movbe eax, [ebx]
			0xF4,
		})
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], 0x11223344)
		if err := memory.Write(0x2000, raw[:]); err != nil {
			t.Fatal(err)
		}
		state.EFlags |= FlagCF | FlagPF | FlagAF | FlagZF | FlagSF | FlagOF
		state.ExpandFlags()
		if err := NewExecutor(nil).Run(state, 4); err != nil {
			t.Fatal(err)
		}
		if state.Get(EAX) != 0x44332211 {
			t.Fatalf("MOVBE load eax=%#x, want 0x44332211", state.Get(EAX))
		}
		for _, flag := range []uint32{FlagCF, FlagPF, FlagAF, FlagZF, FlagSF, FlagOF} {
			if !state.Flag(flag) {
				t.Fatalf("MOVBE load changed flag %#x", flag)
			}
		}
	})

	t.Run("store memory with SIB", func(t *testing.T) {
		memory, state := mappedCode(t, []byte{
			0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
			0xBE, 0x02, 0x00, 0x00, 0x00, // mov esi, 2
			0xBD, 0x78, 0x56, 0x34, 0x12, // mov ebp, 0x12345678
			0x0F, 0x38, 0xF1, 0x6C, 0xB3, 0x04, // movbe [ebx+esi*4+4], ebp
			0xF4,
		})
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], 0)
		if err := memory.Write(0x200C, raw[:]); err != nil {
			t.Fatal(err)
		}
		state.EFlags |= FlagCF | FlagPF | FlagAF | FlagZF | FlagSF | FlagOF
		state.ExpandFlags()
		if err := NewExecutor(nil).Run(state, 8); err != nil {
			t.Fatal(err)
		}
		if err := memory.Read(0x200C, raw[:]); err != nil {
			t.Fatal(err)
		}
		want := [4]byte{0x12, 0x34, 0x56, 0x78}
		if raw != want {
			t.Fatalf("MOVBE store bytes=%#v want %#v", raw, want)
		}
		for _, flag := range []uint32{FlagCF, FlagPF, FlagAF, FlagZF, FlagSF, FlagOF} {
			if !state.Flag(flag) {
				t.Fatalf("MOVBE store changed flag %#x", flag)
			}
		}
	})
}

func TestExecutorFPUStackAndUnary(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xD9, 0xE8, // fld1
		0xD9, 0xE0, // fchs
		0xD9, 0xE1, // fabs
		0xD9, 0xEE, // fldz
		0xD9, 0xC9, // fxch st1
		0xD9, 0xC1, // fld st1
		0xF4,
	})
	beforeFlags := state.EFlags
	if err := NewExecutor(nil).Run(state, 16); err != nil {
		t.Fatal(err)
	}
	if got := state.FPUTop(); got != 5 {
		t.Fatalf("FPU TOP = %d, want 5 after three pushes", got)
	}
	if got := state.FPAt(0).ToFloat64(); math.Abs(got) > 1e-12 {
		t.Fatalf("ST0 = %v, want 0 after FLD ST1", got)
	}
	if got := state.FPAt(1).ToFloat64(); math.Abs(got-1) > 1e-12 {
		t.Fatalf("ST1 = %v, want 1 after FXCH", got)
	}
	if state.EFlags != beforeFlags {
		t.Fatalf("x87 stack/unary changed EFLAGS: before %#x after %#x", beforeFlags, state.EFlags)
	}
}

func TestExecutorFPUArithmetic(t *testing.T) {
	tests := []struct {
		name string
		code []byte
		want float64
	}{
		{name: "fadd", code: []byte{0xD8, 0xC1}, want: 5},
		{name: "fsub", code: []byte{0xD8, 0xE1}, want: -1},
		{name: "fmul", code: []byte{0xD8, 0xC9}, want: 6},
		{name: "fdiv", code: []byte{0xD8, 0xF1}, want: 2.0 / 3.0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, state := mappedCode(t, append(append([]byte{}, test.code...), 0xF4))
			state.FP[0] = fpu.FromFloat64(2)
			state.FP[1] = fpu.FromFloat64(3)
			if err := NewExecutor(nil).Run(state, 4); err != nil {
				t.Fatal(err)
			}
			if got := state.FPAt(0).ToFloat64(); math.Abs(got-test.want) > 1e-12 {
				t.Fatalf("ST0 = %.17g, want %.17g", got, test.want)
			}
		})
	}

	_, state := mappedCode(t, []byte{0xDE, 0xC1, 0xF4}) // faddp st1, st0
	state.FP[0] = fpu.FromFloat64(2)
	state.FP[1] = fpu.FromFloat64(3)
	if err := NewExecutor(nil).Run(state, 4); err != nil {
		t.Fatal(err)
	}
	if state.FPUTop() != 1 || math.Abs(state.FPAt(0).ToFloat64()-5) > 1e-12 {
		t.Fatalf("FADDP result/TOP = %.17g/%d, want 5/1", state.FPAt(0).ToFloat64(), state.FPUTop())
	}
}

func TestExecutorFPUStorePopAndTopRotation(t *testing.T) {
	_, state := mappedCode(t, []byte{
		0xDD, 0xD9, // fstp st1
		0xD9, 0xF7, // fincstp
		0xD9, 0xF6, // fdecstp
		0xF4,
	})
	state.FP[0] = fpu.FromFloat64(42)
	state.FP[1] = fpu.FromFloat64(-7)
	beforeFlags := state.EFlags
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	if state.FPUTop() != 1 {
		t.Fatalf("FPU TOP = %d, want 1 after FSTP followed by INCSTP/DECSTP", state.FPUTop())
	}
	if got := state.FP[1].ToFloat64(); math.Abs(got-42) > 1e-12 {
		t.Fatalf("physical FP1 = %v, want 42 after FSTP ST1", got)
	}
	if state.EFlags != beforeFlags {
		t.Fatalf("x87 store/top changed EFLAGS: before %#x after %#x", beforeFlags, state.EFlags)
	}
}

func TestExecutorFPUMemoryFloat32AndFloat64(t *testing.T) {
	code := []byte{
		0xDD, 0x05, 0x00, 0x20, 0x00, 0x00, // fld qword ptr [0x2000]
		0xD9, 0x15, 0x08, 0x20, 0x00, 0x00, // fst dword ptr [0x2008]
		0xDD, 0x1D, 0x10, 0x20, 0x00, 0x00, // fstp qword ptr [0x2010]
		0xF4,
	}
	memory, state := mappedCode(t, code)
	var input [8]byte
	binary.LittleEndian.PutUint64(input[:], math.Float64bits(3.5))
	if err := memory.Write(0x2000, input[:]); err != nil {
		t.Fatal(err)
	}
	beforeFlags := state.EFlags
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	var output32 [4]byte
	if err := memory.Read(0x2008, output32[:]); err != nil {
		t.Fatal(err)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(output32[:])); got != float32(3.5) {
		t.Fatalf("m32 store = %v, want 3.5", got)
	}
	var output64 [8]byte
	if err := memory.Read(0x2010, output64[:]); err != nil {
		t.Fatal(err)
	}
	if got := math.Float64frombits(binary.LittleEndian.Uint64(output64[:])); got != 3.5 {
		t.Fatalf("m64 store = %v, want 3.5", got)
	}
	if state.FPUTop() != 0 {
		t.Fatalf("FPU TOP = %d, want 0 after FLD/FST/FSTP", state.FPUTop())
	}
	if state.EFlags != beforeFlags {
		t.Fatalf("x87 memory changed EFLAGS: before %#x after %#x", beforeFlags, state.EFlags)
	}
}

func TestExecutorFPUMemorySIB(t *testing.T) {
	code := []byte{
		0xBB, 0x00, 0x20, 0x00, 0x00, // mov ebx, 0x2000
		0xBE, 0x10, 0x00, 0x00, 0x00, // mov esi, 0x10
		0xD9, 0x44, 0xB3, 0x04, // fld dword ptr [ebx+esi*4+4]
		0xD9, 0x5C, 0xB3, 0x08, // fstp dword ptr [ebx+esi*4+8]
		0xF4,
	}
	memory, state := mappedCode(t, code)
	var input [4]byte
	binary.LittleEndian.PutUint32(input[:], math.Float32bits(1.25))
	if err := memory.Write(0x2044, input[:]); err != nil {
		t.Fatal(err)
	}
	if err := NewExecutor(nil).Run(state, 12); err != nil {
		t.Fatal(err)
	}
	var output [4]byte
	if err := memory.Read(0x2048, output[:]); err != nil {
		t.Fatal(err)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(output[:])); got != float32(1.25) {
		t.Fatalf("SIB m32 store = %v, want 1.25", got)
	}
	if state.FPUTop() != 0 {
		t.Fatalf("FPU TOP = %d, want 0 after SIB FLD/FSTP", state.FPUTop())
	}
}

func TestExecutorX87IntegerMemory(t *testing.T) {
	code := []byte{
		0xDF, 0x05, 0x00, 0x21, 0x00, 0x00, // fild word ptr [0x2100]
		0xDF, 0x15, 0x08, 0x21, 0x00, 0x00, // fist word ptr [0x2108]
		0xDF, 0x1D, 0x0A, 0x21, 0x00, 0x00, // fistp word ptr [0x210a]
		0xDD, 0x05, 0x00, 0x22, 0x00, 0x00, // fld qword ptr [0x2200]
		0xDF, 0x0D, 0x08, 0x22, 0x00, 0x00, // fisttp word ptr [0x2208]
		0xDD, 0x05, 0x10, 0x22, 0x00, 0x00, // fld qword ptr [0x2210]
		0xDF, 0x1D, 0x18, 0x22, 0x00, 0x00, // fistp word ptr [0x2218]
		0xF4,
	}
	memory, state := mappedCode(t, code)
	var word [2]byte
	negativeWord := int16(-123)
	binary.LittleEndian.PutUint16(word[:], uint16(negativeWord))
	if err := memory.Write(0x2100, word[:]); err != nil {
		t.Fatal(err)
	}
	var value [8]byte
	binary.LittleEndian.PutUint64(value[:], math.Float64bits(3.5))
	if err := memory.Write(0x2200, value[:]); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint64(value[:], math.Float64bits(40000))
	if err := memory.Write(0x2210, value[:]); err != nil {
		t.Fatal(err)
	}
	if err := NewExecutor(nil).Run(state, 8); err != nil {
		t.Fatal(err)
	}
	var output [2]byte
	if err := memory.Read(0x2108, output[:]); err != nil {
		t.Fatal(err)
	}
	if got := int16(binary.LittleEndian.Uint16(output[:])); got != -123 {
		t.Fatalf("FIST m16 = %d, want -123", got)
	}
	if err := memory.Read(0x210A, output[:]); err != nil {
		t.Fatal(err)
	}
	if got := int16(binary.LittleEndian.Uint16(output[:])); got != -123 {
		t.Fatalf("FISTP m16 = %d, want -123", got)
	}
	if err := memory.Read(0x2208, output[:]); err != nil {
		t.Fatal(err)
	}
	if got := int16(binary.LittleEndian.Uint16(output[:])); got != 3 {
		t.Fatalf("FISTTP m16 = %d, want 3", got)
	}
	if err := memory.Read(0x2218, output[:]); err != nil {
		t.Fatal(err)
	}
	if got := int16(binary.LittleEndian.Uint16(output[:])); got != -32768 {
		t.Fatalf("out-of-range FISTP m16 = %d, want -32768", got)
	}
	if state.FPUTop() != 0 {
		t.Fatalf("FPU TOP = %d, want 0 after integer stores", state.FPUTop())
	}
}

func TestExecutorX87IntegerMemorySIBAndM64(t *testing.T) {
	code := []byte{
		0xBB, 0x00, 0x24, 0x00, 0x00, // mov ebx, 0x2400
		0xBE, 0x10, 0x00, 0x00, 0x00, // mov esi, 0x10
		0xDB, 0x44, 0xB3, 0x04, // fild dword ptr [ebx+esi*4+4]
		0xDF, 0x2D, 0x20, 0x24, 0x00, 0x00, // fild qword ptr [0x2420]
		0xDF, 0x3D, 0x28, 0x24, 0x00, 0x00, // fistp qword ptr [0x2428]
		0xDB, 0x1D, 0x30, 0x24, 0x00, 0x00, // fistp dword ptr [0x2430]
		0xF4,
	}
	memory, state := mappedCode(t, code)
	var input32 [4]byte
	negativeDword := int32(-16)
	binary.LittleEndian.PutUint32(input32[:], uint32(negativeDword))
	if err := memory.Write(0x2444, input32[:]); err != nil {
		t.Fatal(err)
	}
	var input64 [8]byte
	negativeQword := int64(-1 << 40)
	binary.LittleEndian.PutUint64(input64[:], uint64(negativeQword))
	if err := memory.Write(0x2420, input64[:]); err != nil {
		t.Fatal(err)
	}
	if err := NewExecutor(nil).Run(state, 10); err != nil {
		t.Fatal(err)
	}
	if err := memory.Read(0x2428, input64[:]); err != nil {
		t.Fatal(err)
	}
	if got := int64(binary.LittleEndian.Uint64(input64[:])); got != negativeQword {
		t.Fatalf("FISTP m64 = %d, want %d", got, -1<<40)
	}
	if err := memory.Read(0x2430, input32[:]); err != nil {
		t.Fatal(err)
	}
	if got := int32(binary.LittleEndian.Uint32(input32[:])); got != -16 {
		t.Fatalf("SIB FISTP m32 = %d, want -16", got)
	}
	if state.FPUTop() != 0 {
		t.Fatalf("FPU TOP = %d, want 0 after SIB and m64 operations", state.FPUTop())
	}
}
