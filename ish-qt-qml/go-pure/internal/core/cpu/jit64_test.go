package cpu

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/bits"
	"testing"
)

func mapExecutable64(t *testing.T, memory *Memory64, start Address64, code []byte) {
	t.Helper()
	if err := memory.Map(start, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(start, code); err != nil {
		t.Fatal(err)
	}
}

func TestCompileBlock64AndRunBranch(t *testing.T) {
	memory := NewMemory64()
	const start Address64 = 0x1000
	// mov rax,5; add rax,3; cmp rax,8; je +2; mov ebx,1; mov ebx,2; syscall
	code := []byte{
		0x48, 0xc7, 0xc0, 0x05, 0, 0, 0, // mov rax,5
		0x48, 0x83, 0xc0, 0x03, // add rax,3
		0x48, 0x83, 0xf8, 0x08, // cmp rax,8
		0x74, 0x05, // je to the second mov
		0xbb, 0x01, 0, 0, 0, // mov ebx,1
		0xbb, 0x02, 0, 0, 0, // mov ebx,2
		0x0f, 0x05, // syscall
	}
	mapExecutable64(t, memory, start, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(start)
	state.Regs[RSP] = 0x8000
	if err := memory.Map(Address64(0x8000-Page64Size), Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}

	block, err := CompileBlock64(memory, start, Page64Size)
	if err != nil {
		t.Fatal(err)
	}
	if len(block.Ops) != 4 {
		t.Fatalf("compiled %d operations, want 4 before conditional branch", len(block.Ops))
	}
	flow, err := block.Run(state)
	if err != nil {
		t.Fatal(err)
	}
	if flow != Flow64Branch || state.RIP != uint64(start)+22 {
		t.Fatalf("flow=%v rip=%#x, want branch rip=%#x", flow, state.RIP, uint64(start)+22)
	}
	if state.Get(RAX) != 8 || !state.Flag(Flag64ZF) {
		t.Fatalf("state after branch: rax=%d zf=%v", state.Get(RAX), state.Flag(Flag64ZF))
	}

	target, err := CompileBlock64(memory, Address64(state.RIP), Page64Size)
	if err != nil {
		t.Fatal(err)
	}
	flow, err = target.Run(state)
	if err != nil {
		t.Fatal(err)
	}
	if flow != Flow64Interrupt || state.Get(RBX) != 2 || state.TrapNo != 0x80 {
		t.Fatalf("target result: flow=%v rbx=%d trap=%#x", flow, state.Get(RBX), state.TrapNo)
	}
}

func TestBlockCache64GenerationInvalidation(t *testing.T) {
	memory := NewMemory64()
	const start Address64 = 0x2000
	mapExecutable64(t, memory, start, []byte{0x90, 0x0f, 0x05})
	cache := NewBlockCache64(memory)
	block, err := CompileBlock64(memory, start, Page64Size)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Put(block); err != nil {
		t.Fatal(err)
	}
	if got, ok := cache.Get(uint64(start)); !ok || got != block {
		t.Fatal("expected cache hit")
	}
	if err := memory.Write(start, []byte{0x90}); err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.Get(uint64(start)); ok {
		t.Fatal("stale block survived code-page write")
	}
	if cache.Len() != 0 {
		t.Fatalf("cache length=%d, want 0", cache.Len())
	}
}

func TestJIT64AtomicXADDAndCMPXCHG(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x1000
	const dataAddress Address64 = 0x4000
	code := []byte{
		0x48, 0xc7, 0x07, 0x05, 0x00, 0x00, 0x00, // mov qword ptr [rdi], 5
		0x48, 0xc7, 0xc1, 0x03, 0x00, 0x00, 0x00, // mov rcx, 3
		0xf0, 0x48, 0x0f, 0xc1, 0x0f, // lock xadd [rdi], rcx
		0x48, 0xc7, 0xc0, 0x05, 0x00, 0x00, 0x00, // mov rax, 5
		0x48, 0xc7, 0xc2, 0x09, 0x00, 0x00, 0x00, // mov rdx, 9
		0xf0, 0x48, 0x0f, 0xb1, 0x17, // lock cmpxchg [rdi], rdx (fails)
		0x48, 0xc7, 0xc0, 0x08, 0x00, 0x00, 0x00, // mov rax, 8
		0xf0, 0x48, 0x0f, 0xb1, 0x17, // lock cmpxchg [rdi], rdx (succeeds)
		0xf4, // hlt
	}
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	jit := NewJIT64(memory)
	trap := jit.RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	value, err := memory.ReadUint64(dataAddress)
	if err != nil {
		t.Fatal(err)
	}
	if value != 9 {
		t.Fatalf("atomic memory=%d, want 9", value)
	}
	if state.Get(RCX) != 5 {
		t.Fatalf("xadd source=%d, want old value 5", state.Get(RCX))
	}
	if state.Get(RAX) != 8 {
		t.Fatalf("cmpxchg accumulator=%d, want 8", state.Get(RAX))
	}
	if !state.Flag(Flag64ZF) {
		t.Fatal("successful cmpxchg did not set ZF")
	}
}

func TestJIT64SSE2MovesAndLogicalOps(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x2000
	const dataAddress Address64 = 0x5000
	code := []byte{
		0x66, 0x0f, 0x6f, 0x07, // movdqa xmm0, [rdi]
		0x66, 0x0f, 0xef, 0xc1, // pxor xmm0, xmm1
		0xf3, 0x0f, 0x7f, 0x47, 0x10, // movdqu [rdi+16], xmm0
		0xf4, // hlt
	}
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	input := [16]byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80, 0x90, 0xa0, 0xb0, 0xc0, 0xd0, 0xe0, 0xf0, 0xff}
	if err := memory.Write(dataAddress, input[:]); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	for i := range state.XMM[1] {
		state.XMM[1][i] = 0xff
	}
	jit := NewJIT64(memory)
	trap := jit.RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v", trap, state.Halted)
	}
	var output [16]byte
	if err := memory.Read(dataAddress+16, output[:]); err != nil {
		t.Fatal(err)
	}
	var want [16]byte
	for i := range want {
		want[i] = input[i] ^ 0xff
	}
	if output != want || state.XMM[0] != want {
		t.Fatalf("SSE result output=%x xmm0=%x want=%x", output, state.XMM[0], want)
	}
}

func TestJIT64X87StackAndMemory(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x3000
	const dataAddress Address64 = 0x6000
	code := []byte{
		0xd9, 0xe8, // fld1
		0xd9, 0xee, // fldz
		0xd9, 0xe0, // fchs
		0xd8, 0xc1, // fadd st0, st1
		0xdd, 0x1f, // fstp qword ptr [rdi]
		0xf4, // hlt
	}
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	jit := NewJIT64(memory)
	trap := jit.RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v", trap, state.Halted)
	}
	value, err := memory.ReadUint64(dataAddress)
	if err != nil {
		t.Fatal(err)
	}
	if got := math.Float64frombits(value); got != 1 {
		t.Fatalf("stored x87 value=%v, want 1", got)
	}
	if state.FPUTop() != 7 {
		t.Fatalf("FPU top=%d, want 7 after one pop", state.FPUTop())
	}
}

func TestJIT64SSE2PackedArithmetic(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x7000
	const dataAddress Address64 = 0x9000
	code := []byte{
		0x66, 0x0f, 0x6f, 0x07, // movdqa xmm0, [rdi]
		0x66, 0x0f, 0x6f, 0x4f, 0x10, // movdqa xmm1, [rdi+16]
		0x66, 0x0f, 0xd4, 0xc1, // paddq xmm0, xmm1
		0x66, 0x0f, 0x76, 0xc1, // pcmpeqd xmm0, xmm1
		0x66, 0x0f, 0xdf, 0xc1, // pandn xmm0, xmm1
		0xf3, 0x0f, 0x7f, 0x47, 0x20, // movdqu [rdi+32], xmm0
		0xf4, // hlt
	}
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	left := [16]byte{1, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0}
	right := [16]byte{3, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0}
	if err := memory.Write(dataAddress, left[:]); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(dataAddress+16, right[:]); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v", trap, state.Halted)
	}
	var output [16]byte
	if err := memory.Read(dataAddress+32, output[:]); err != nil {
		t.Fatal(err)
	}
	var want [16]byte
	// After PADDQ, the 32-bit lanes are [4,0,6,0], while xmm1 is
	// [3,0,4,0]. No lane is equal, so PCMPEQD produces zero and PANDN
	// with xmm1 leaves xmm1 unchanged.
	want = right
	if output != want {
		t.Fatalf("packed output=%x want=%x", output, want)
	}
}

func TestJIT64ScalarExtensions(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0xa000
	code := []byte{
		0xb0, 0x80, // mov al, 0x80
		0x48, 0x0f, 0xb6, 0xc8, // movzx rcx, al
		0x48, 0x0f, 0xbe, 0xd0, // movsx rdx, al
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v", trap, state.Halted)
	}
	if got := state.Get(RCX); got != 0x80 {
		t.Fatalf("movzx rcx=%#x, want 0x80", got)
	}
	if got := state.Get(RDX); got != ^uint64(0x7f) {
		t.Fatalf("movsx rdx=%#x, want %#x", got, ^uint64(0x7f))
	}
}

func TestJIT64CarryAndFlagTransfers(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0xb000
	code := []byte{
		0xb8, 0xff, 0xff, 0xff, 0xff, // mov eax, -1
		0x83, 0xc0, 0x01, // add eax, 1: CF=1, result=0
		0xbb, 0x01, 0x00, 0x00, 0x00, // mov ebx, 1
		0x83, 0xd3, 0x02, // adc ebx, 2: 1+2+CF=4
		0x31, 0xc9, // xor ecx, ecx
		0x83, 0xe9, 0x01, // sub ecx, 1: CF=1
		0xba, 0x05, 0x00, 0x00, 0x00, // mov edx, 5
		0x83, 0xda, 0x01, // sbb edx, 1: 5-1-CF=3
		0x31, 0xc0, // xor eax, eax: ZF=1, PF=1
		0x9f,       // lahf
		0x88, 0xe1, // mov cl, ah: preserve LAHF result
		0xb4, 0xff, // mov ah, 0xff

		0x9e, // sahf: set CF/PF/AF/ZF/SF
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v", trap, state.Halted)
	}
	if got := state.Get(RBX); got != 4 {
		t.Fatalf("adc ebx=%d, want 4", got)
	}
	if got := state.Get(RDX); got != 3 {
		t.Fatalf("sbb edx=%d, want 3", got)
	}
	if got := state.Get(RCX) & 0xff; got != 0x46 {
		t.Fatalf("lahf byte=%#x, want 0x46", got)
	}
	if !state.Flag(Flag64CF) || !state.Flag(Flag64PF) || !state.Flag(Flag64AF) || !state.Flag(Flag64ZF) || !state.Flag(Flag64SF) {
		t.Fatalf("SAHF flags: rflags=%#x", state.RFLAGS)
	}
}

func TestJIT64ShiftsSetccAndCMOVcc(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0xc000
	code := []byte{
		0xb8, 0x01, 0x00, 0x00, 0x80, // mov eax, 0x80000001
		0xd1, 0xe0, // shl eax, 1 => 2, CF=1, OF=1
		0x0f, 0x92, 0xc3, // setb bl
		0x0f, 0x90, 0xc1, // seto cl
		0x41, 0x0f, 0x95, 0xc0, // setne r8b
		0xd1, 0xe8, // shr eax, 1 => 1
		0xd1, 0xf8, // sar eax, 1 => 0
		0xb8, 0xff, 0xff, 0xff, 0xff, // mov eax, -1
		0x83, 0xf8, 0x00, // cmp eax, 0: SF=1, OF=0
		0x41, 0x0f, 0x9c, 0xc1, // setl r9b
		0x41, 0x0f, 0x9d, 0xc2, // setge r10b
		0xbe, 0x07, 0x00, 0x00, 0x00, // mov esi, 7
		0xbf, 0x09, 0x00, 0x00, 0x00, // mov edi, 9
		0xba, 0x01, 0x00, 0x00, 0x00, // mov edx, 1
		0x0f, 0x4c, 0xd6, // cmovl edx, esi => 7
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if got := state.Get(RBX) & 0xff; got != 1 {
		t.Fatalf("setb bl=%d, want 1", got)
	}
	if got := state.Get(RCX) & 0xff; got != 1 {
		t.Fatalf("seto cl=%d, want 1", got)
	}
	if got := state.Get(R8) & 0xff; got != 1 {
		t.Fatalf("setne r8b=%d, want 1", got)
	}
	if got := state.Get(R9) & 0xff; got != 1 {
		t.Fatalf("setl r9b=%d, want 1", got)
	}
	if got := state.Get(R10) & 0xff; got != 0 {
		t.Fatalf("setge r10b=%d, want 0", got)
	}
	if got := state.Get(RDX); got != 7 {
		t.Fatalf("cmovl edx=%d, want 7", got)
	}
}

func TestJIT64MultiplyAndDivide(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0xd000
	code := []byte{
		0x48, 0xc7, 0xc0, 0x03, 0x00, 0x00, 0x00, // mov rax, 3
		0x48, 0xc7, 0xc3, 0x04, 0x00, 0x00, 0x00, // mov rbx, 4
		0x48, 0xf7, 0xe3, // mul rbx => rdx:rax = 12
		0x48, 0xc7, 0xc1, 0x02, 0x00, 0x00, 0x00, // mov rcx, 2
		0x48, 0xf7, 0xf1, // div rcx => rax=6, rdx=0
		0xb8, 0x06, 0x00, 0x00, 0x00, // mov eax, 6
		0xb9, 0x07, 0x00, 0x00, 0x00, // mov ecx, 7
		0x0f, 0xaf, 0xc1, // imul eax, ecx => 42
		0xba, 0x06, 0x00, 0x00, 0x00, // mov edx, 6
		0xbb, 0x00, 0x00, 0x00, 0x00, // clear ebx
		0x6b, 0xda, 0x07, // imul ebx, edx, 7 => 42
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if got := state.Get(RAX); got != 42 {
		t.Fatalf("imul/div rax=%d, want 42", got)
	}
	if got := state.Get(RBX); got != 42 {
		t.Fatalf("imul immediate ebx=%d, want 42", got)
	}
	if got := state.Get(RDX); got != 6 {
		t.Fatalf("source edx=%d, want 6", got)
	}
}

func TestJIT64SignedDivide(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0xe000
	code := []byte{
		0x48, 0xc7, 0xc0, 0xf4, 0xff, 0xff, 0xff, // mov rax, -12
		0x48, 0xc7, 0xc2, 0xff, 0xff, 0xff, 0xff, // mov rdx, -1 (sign-extended high half)
		0x48, 0xc7, 0xc1, 0x03, 0x00, 0x00, 0x00, // mov rcx, 3
		0x48, 0xf7, 0xf9, // idiv rcx => rax=-4, rdx=0
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if got := state.Get(RAX); got != ^uint64(3) {
		t.Fatalf("idiv rax=%#x, want %#x", got, ^uint64(3))
	}
	if got := state.Get(RDX); got != 0 {
		t.Fatalf("idiv rdx=%#x, want 0", got)
	}
}

func TestJIT64BitOperationsAndBSWAP(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0xf000
	code := []byte{
		0xb8, 0x11, 0x00, 0x00, 0x80, // mov eax, 0x80000011
		0x0f, 0xbc, 0xc8, // bsf ecx, eax => 0
		0x0f, 0xbd, 0xd0, // bsr edx, eax => 31
		0xf3, 0x0f, 0xb8, 0xf0, // popcnt esi, eax => 3
		0x0f, 0xba, 0xe0, 0x04, // bt eax, 4 => CF=1
		0x0f, 0xba, 0xf0, 0x04, // btr eax, 4 => clear bit 4
		0x41, 0x89, 0xc0, // mov r8d, eax
		0x0f, 0xba, 0xe8, 0x01, // bts eax, 1 => set bit 1
		0x41, 0x89, 0xc1, // mov r9d, eax
		0x0f, 0xba, 0xf8, 0x00, // btc eax, 0 => clear bit 0
		0x41, 0x89, 0xc2, // mov r10d, eax
		0x0f, 0xbc, 0xf8, // bsf edi, eax => 1
		0x0f, 0xc8, // bswap eax => 0x02000080
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if got := state.Get(RCX); got != 0 {
		t.Fatalf("bsf ecx=%d, want 0", got)
	}
	if got := state.Get(RDX); got != 31 {
		t.Fatalf("bsr edx=%d, want 31", got)
	}
	if got := state.Get(RSI); got != 3 {
		t.Fatalf("popcnt esi=%d, want 3", got)
	}
	if got := state.Get(R8); got != 0x80000001 {
		t.Fatalf("btr snapshot r8=%#x, want %#x", got, uint64(0x80000001))
	}
	if got := state.Get(R9); got != 0x80000003 {
		t.Fatalf("bts snapshot r9=%#x, want %#x", got, uint64(0x80000003))
	}
	if got := state.Get(R10); got != 0x80000002 {
		t.Fatalf("btc snapshot r10=%#x, want %#x", got, uint64(0x80000002))
	}
	if got := state.Get(RDI); got != 1 {
		t.Fatalf("bsf after bit ops edi=%d, want 1", got)
	}
	if got := state.Get(RAX); got != 0x02000080 {
		t.Fatalf("bswap eax=%#x, want %#x", got, uint64(0x02000080))
	}
	if !state.Flag(Flag64CF) {
		t.Fatal("BTC did not preserve the selected old bit in CF")
	}
}

func TestJIT64MOVDAndMOVQ(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x11000
	code := []byte{
		0xb8, 0x44, 0x33, 0x22, 0x11, // mov eax, 0x11223344
		0x66, 0x0f, 0x6e, 0xc0, // movd xmm0, eax
		0x66, 0x0f, 0x7e, 0xc1, // movd ecx, xmm0
		0xf3, 0x0f, 0x7e, 0xc8, // movq xmm1, xmm0 (copy low qword)
		0x66, 0x48, 0x0f, 0x7e, 0xca, // movq rdx, xmm1
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if got := state.Get(RCX); got != 0x11223344 {
		t.Fatalf("movd ecx=%#x, want %#x", got, uint64(0x11223344))
	}
	if got := state.Get(RDX); got != 0x11223344 {
		t.Fatalf("movq rdx=%#x, want %#x", got, uint64(0x11223344))
	}
	for i, value := range state.XMM[1][4:] {
		if value != 0 {
			t.Fatalf("movd/movq xmm1 upper byte %d=%#x, want 0", i+4, value)
		}
	}
}

func TestJIT64PushAndPopFlags(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x12000
	const stackAddress Address64 = 0x18000
	code := []byte{
		0x9c,                                     // pushfq
		0x58,                                     // pop rax
		0x48, 0xc7, 0xc0, 0x01, 0x02, 0x00, 0x00, // mov rax, 0x201 (CF|IF)
		0x50,             // push rax
		0x9d,             // popfq
		0x0f, 0x92, 0xc1, // setb cl
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	if err := memory.Map(stackAddress-Address64(Page64Size), Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Regs[RSP] = uint64(stackAddress)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if got := state.Get(RAX); got&Flag64IF == 0 {
		t.Fatalf("pushfq/pop rax=%#x does not contain IF", got)
	}
	if got := state.Get(RCX) & 0xff; got != 1 {
		t.Fatalf("popfq/setb cl=%d, want 1", got)
	}
}

func TestJIT64StringOperations(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x13000
	const sourceAddress Address64 = 0x20000
	const copyAddress Address64 = 0x20100
	const fillAddress Address64 = 0x20200
	code := []byte{
		0x48, 0xbe, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // mov rsi, source
		0x48, 0xbf, 0x00, 0x01, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // mov rdi, copy
		0x48, 0xc7, 0xc1, 0x03, 0x00, 0x00, 0x00, // mov rcx, 3
		0xf3, 0xa4, // rep movsb
		0xb0, 0x7f, // mov al, 0x7f
		0x48, 0xbf, 0x00, 0x02, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // mov rdi, fill
		0x48, 0xc7, 0xc1, 0x04, 0x00, 0x00, 0x00, // mov rcx, 4
		0xf3, 0xaa, // rep stosb
		0x48, 0xbe, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // mov rsi, source
		0xac,                                                       // lodsb
		0x48, 0xbe, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // mov rsi, source
		0x48, 0xbf, 0x00, 0x01, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // mov rdi, copy
		0x48, 0xc7, 0xc1, 0x03, 0x00, 0x00, 0x00, // mov rcx, 3
		0xf3, 0xa6, // repe cmpsb
		0xb0, 0x03, // mov al, 3
		0x48, 0xbf, 0x00, 0x01, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // mov rdi, copy
		0x48, 0xc7, 0xc1, 0x03, 0x00, 0x00, 0x00, // mov rcx, 3
		0xf2, 0xae, // repne scasb
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	if err := memory.Map(sourceAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(fillAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(sourceAddress, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("string trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	var copied [3]byte
	if err := memory.Read(copyAddress, copied[:]); err != nil {
		t.Fatal(err)
	}
	if copied != [3]byte{1, 2, 3} {
		t.Fatalf("rep movsb copy=%v, want [1 2 3]", copied)
	}
	var filled [4]byte
	if err := memory.Read(fillAddress, filled[:]); err != nil {
		t.Fatal(err)
	}
	if filled != [4]byte{0x7f, 0x7f, 0x7f, 0x7f} {
		t.Fatalf("rep stosb fill=%v, want four 0x7f bytes", filled)
	}
	if got := state.Get(RAX) & 0xff; got != 3 {
		t.Fatalf("lodsb/scasb al=%d, want 3", got)
	}
	if got := state.Get(RCX); got != 0 {
		t.Fatalf("repne scasb rcx=%d, want 0", got)
	}
	if got := state.Get(RDI); got != uint64(copyAddress+3) {
		t.Fatalf("repne scasb rdi=%#x, want %#x", got, uint64(copyAddress+3))
	}
	if !state.Flag(Flag64ZF) {
		t.Fatalf("repne scasb did not leave ZF set: rflags=%#x", state.RFLAGS)
	}
}

func TestJIT64StringWidthsDirectionAndAddressSize(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x14000
	const wordSource Address64 = 0x21000
	const wordDest Address64 = 0x21100
	const byteSource Address64 = 0x22000
	const byteDest Address64 = 0x22100
	code := []byte{
		0x48, 0xbe, 0x02, 0x10, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // mov rsi, wordSource+2
		0x48, 0xbf, 0x02, 0x11, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, // mov rdi, wordDest+2
		0x48, 0xc7, 0xc1, 0x02, 0x00, 0x00, 0x00, // mov rcx, 2
		0xfd,             // std
		0x66, 0xf3, 0xa5, // rep movsw backwards
		0xfc,                                                       // cld
		0x48, 0xbe, 0x00, 0x20, 0x02, 0x00, 0x01, 0x00, 0x00, 0x00, // rsi=1<<32|byteSource
		0x48, 0xbf, 0x00, 0x21, 0x02, 0x00, 0x01, 0x00, 0x00, 0x00, // rdi=1<<32|byteDest
		0x48, 0xb9, 0x03, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, // rcx=1<<32|3
		0x67, 0xf3, 0xa4, // addr32 rep movsb
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	for _, address := range []Address64{wordSource, wordDest, byteSource, byteDest} {
		if err := memory.Map(address, Page64Size, PRead|PWrite); err != nil {
			t.Fatal(err)
		}
	}
	if err := memory.Write(wordSource, []byte{1, 2, 3, 4}); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(byteSource, []byte{5, 6, 7}); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("string widths trap=%#x halted=%v", trap, state.Halted)
	}
	var words [4]byte
	if err := memory.Read(wordDest, words[:]); err != nil {
		t.Fatal(err)
	}
	if words != [4]byte{1, 2, 3, 4} {
		t.Fatalf("DF rep movsw=%v, want [1 2 3 4]", words)
	}
	var bytes [3]byte
	if err := memory.Read(byteDest, bytes[:]); err != nil {
		t.Fatal(err)
	}
	if bytes != [3]byte{5, 6, 7} {
		t.Fatalf("addr32 rep movsb=%v, want [5 6 7]", bytes)
	}
	if got := state.Get(RSI); got != uint64(byteSource+3) || state.Get(RDI) != uint64(byteDest+3) || state.Get(RCX) != 0 {
		t.Fatalf("addr32 final indices rsi=%#x rdi=%#x rcx=%#x", got, state.Get(RDI), state.Get(RCX))
	}
	if state.RFLAGS&Flag64DF != 0 {
		t.Fatal("cld did not clear direction flag")
	}
}

func TestJIT64MOVBEAndAccumulatorConversions(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x15000
	const dataAddress Address64 = 0x23000
	code := []byte{
		0xb8, 0x44, 0x33, 0x22, 0x11, // mov eax, 0x11223344
		0x0f, 0x38, 0xf1, 0x07, // movbe [rdi], eax
		0x0f, 0x38, 0xf0, 0x0f, // movbe ecx, [rdi]
		0x48, 0xb8, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // mov rax, 0x0102030405060708
		0x48, 0x0f, 0x38, 0xf1, 0x47, 0x08, // movbe [rdi+8], rax
		0x48, 0x0f, 0x38, 0xf0, 0x77, 0x08, // movbe rsi, [rdi+8]
		0xb8, 0x44, 0x33, 0x22, 0x11, // mov eax, 0x11223344
		0x48, 0x63, 0xd0, // movsxd rdx, eax
		0x49, 0x89, 0xd5, // mov r13, rdx
		0xb0, 0x80, // mov al, 0x80
		0x66, 0x98, // cbw
		0x0f, 0xb7, 0xc8, // movzx ecx, ax
		0xb8, 0x00, 0x80, 0x00, 0x00, // mov eax, 0x8000
		0x98,             // cwde
		0x41, 0x89, 0xc0, // mov r8d, eax
		0xb8, 0x00, 0x00, 0x00, 0x80, // mov eax, 0x80000000
		0x48, 0x98, // cdqe
		0x49, 0x89, 0xc1, // mov r9, rax
		0xb8, 0x00, 0x80, 0x00, 0x00, // mov eax, 0x8000
		0x66, 0x99, // cwd
		0x41, 0x89, 0xd2, // mov r10d, edx
		0xb8, 0x00, 0x00, 0x00, 0x80, // mov eax, 0x80000000
		0x99,             // cdq
		0x41, 0x89, 0xd3, // mov r11d, edx
		0x48, 0xc7, 0xc0, 0xff, 0xff, 0xff, 0xff, // mov rax, -1
		0x48, 0x99, // cqo
		0x49, 0x89, 0xd4, // mov r12, rdx
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("conversion trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	var first [4]byte
	if err := memory.Read(dataAddress, first[:]); err != nil {
		t.Fatal(err)
	}
	if first != [4]byte{0x11, 0x22, 0x33, 0x44} {
		t.Fatalf("movbe dword bytes=%x, want 11223344", first)
	}
	var second [8]byte
	if err := memory.Read(dataAddress+8, second[:]); err != nil {
		t.Fatal(err)
	}
	if second != [8]byte{1, 2, 3, 4, 5, 6, 7, 8} {
		t.Fatalf("movbe qword bytes=%x, want 0102030405060708", second)
	}
	if state.Get(RCX) != 0xff80 {
		t.Fatalf("cbw/movzx ecx=%#x, want 0xff80", state.Get(RCX))
	}
	if state.Get(R8) != 0xffff8000 {
		t.Fatalf("cwde r8=%#x, want 0xffff8000", state.Get(R8))
	}
	if state.Get(R9) != 0xffffffff80000000 {
		t.Fatalf("cdqe r9=%#x, want 0xffffffff80000000", state.Get(R9))
	}
	if state.Get(R10) != 0x1122ffff {
		t.Fatalf("cwd r10=%#x, want 0x1122ffff (DX sign extension with preserved upper EDX)", state.Get(R10))
	}
	if state.Get(R11) != 0xffffffff {
		t.Fatalf("cdq r11=%#x, want 0xffffffff", state.Get(R11))
	}
	if state.Get(R12) != ^uint64(0) {
		t.Fatalf("cqo r12=%#x, want %#x", state.Get(R12), ^uint64(0))
	}
	if state.Get(R13) != 0x11223344 {
		t.Fatalf("movsxd r13=%#x, want 0x11223344", state.Get(R13))
	}
	if state.Get(RSI) != 0x0102030405060708 {
		t.Fatalf("movbe qword rsi=%#x, want 0x0102030405060708", state.Get(RSI))
	}
}

func TestJIT64LeaveFencesAndUD2(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x16000
	const frameAddress Address64 = 0x25000
	const savedBase Address64 = 0x2a000
	code := []byte{
		0xc9,             // leave
		0x0f, 0xae, 0xe8, // lfence
		0x0f, 0xae, 0xf0, // mfence
		0x0f, 0xae, 0xf8, // sfence
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	if err := memory.Map(frameAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteUint64(frameAddress, uint64(savedBase)); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RBP, uint64(frameAddress))
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("leave/fence trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.Get(RBP) != uint64(savedBase) || state.Get(RSP) != uint64(frameAddress+8) {
		t.Fatalf("leave rbp=%#x rsp=%#x, want rbp=%#x rsp=%#x", state.Get(RBP), state.Get(RSP), savedBase, frameAddress+8)
	}

	invalidMemory := NewMemory64()
	const invalidAddress Address64 = 0x17000
	mapExecutable64(t, invalidMemory, invalidAddress, []byte{0x0f, 0x0b}) // ud2
	invalidState := NewMachineState64(invalidMemory)
	invalidState.RIP = uint64(invalidAddress)
	if trap := NewJIT64(invalidMemory).RunToInterrupt(invalidState); trap != Trap64InvalidOpcode || invalidState.TrapNo != Trap64InvalidOpcode {
		t.Fatalf("ud2 trap=%#x trapNo=%#x, want %#x", trap, invalidState.TrapNo, Trap64InvalidOpcode)
	}
}

func TestJIT64FSBaseGSBaseAndXCR(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x18000
	code := []byte{
		0xf3, 0x48, 0x0f, 0xae, 0xc0, // rdfsbase rax
		0x49, 0x89, 0xc0, // mov r8, rax
		0xf3, 0x48, 0x0f, 0xae, 0xc9, // rdgsbase rcx
		0x49, 0x89, 0xc9, // mov r9, rcx
		0x48, 0xc7, 0xc0, 0x00, 0x50, 0x34, 0x12, // mov rax, 0x12345000
		0xf3, 0x48, 0x0f, 0xae, 0xd0, // wrfsbase rax
		0x48, 0xc7, 0xc1, 0x00, 0x60, 0x45, 0x23, // mov rcx, 0x23456000
		0xf3, 0x48, 0x0f, 0xae, 0xd9, // wrgsbase rcx
		0x31, 0xc9, // xor ecx, ecx: select XCR0
		0x0f, 0x01, 0xd0, // xgetbv
		0x49, 0x89, 0xc2, // mov r10, rax
		0x49, 0x89, 0xd3, // mov r11, rdx
		0xb8, 0x03, 0x00, 0x00, 0x00, // mov eax, 3
		0x31, 0xd2, // xor edx, edx
		0x31, 0xc9, // xor ecx, ecx
		0x0f, 0x01, 0xd1, // xsetbv
		0xf3, 0x49, 0x0f, 0xae, 0xc4, // rdfsbase r12
		0xf3, 0x49, 0x0f, 0xae, 0xcd, // rdgsbase r13
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.FSBase = 0x1000
	state.GSBase = 0x2000
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("tls/xcr trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.Get(R8) != 0x1000 || state.Get(R9) != 0x2000 {
		t.Fatalf("initial bases r8=%#x r9=%#x", state.Get(R8), state.Get(R9))
	}
	if state.Get(R10) != 3 || state.Get(R11) != 0 {
		t.Fatalf("xgetbv r10=%#x r11=%#x, want 3 and 0", state.Get(R10), state.Get(R11))
	}
	if state.FSBase != 0x12345000 || state.GSBase != 0x23456000 || state.XCR0 != 3 {
		t.Fatalf("final fs=%#x gs=%#x xcr0=%#x", state.FSBase, state.GSBase, state.XCR0)
	}
	if state.Get(R12) != state.FSBase || state.Get(R13) != state.GSBase {
		t.Fatalf("final base reads r12=%#x r13=%#x", state.Get(R12), state.Get(R13))
	}

	invalidMemory := NewMemory64()
	const invalidAddress Address64 = 0x19000
	mapExecutable64(t, invalidMemory, invalidAddress, []byte{
		0xb9, 0x01, 0x00, 0x00, 0x00, // mov ecx, 1
		0x0f, 0x01, 0xd0, // xgetbv: unsupported XCR index
	})
	invalidState := NewMachineState64(invalidMemory)
	invalidState.RIP = uint64(invalidAddress)
	if trap := NewJIT64(invalidMemory).RunToInterrupt(invalidState); trap != Trap64GeneralFault || invalidState.TrapNo != Trap64GeneralFault {
		t.Fatalf("invalid xgetbv trap=%#x trapNo=%#x, want %#x", trap, invalidState.TrapNo, Trap64GeneralFault)
	}
}

func TestJIT64Cmpxchg8BAnd16B(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x1a000
	const dataAddress Address64 = 0x2a000
	code := []byte{
		0xb8, 0x88, 0x77, 0x66, 0x55, // mov eax, 0x55667788
		0xba, 0x11, 0x22, 0x33, 0x44, // mov edx, 0x44332211
		0xbb, 0xef, 0xbe, 0xad, 0xde, // mov ebx, 0xdeadbeef
		0xb9, 0xbe, 0xba, 0xfe, 0xca, // mov ecx, 0xcafebabe
		0x0f, 0xc7, 0x0f, // cmpxchg8b [rdi]
		0x48, 0xb8, 0xef, 0xbe, 0xad, 0xde, 0xbe, 0xba, 0xfe, 0xca, // mov rax, 0xcafebabedeadbeef
		0x48, 0xba, 0x18, 0x17, 0x16, 0x15, 0x14, 0x13, 0x12, 0x11, // mov rdx, 0x1112131415161718
		0x48, 0xbb, 0x11, 0x00, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, // mov rbx, 0xaabbccddeeff0011
		0x48, 0xb9, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, // mov rcx, 0x2233445566778899
		0x48, 0x0f, 0xc7, 0x0f, // cmpxchg16b [rdi]
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteUint64(dataAddress, 0x4433221155667788); err != nil {
		t.Fatal(err)
	}
	if err := memory.WriteUint64(dataAddress+8, 0x1112131415161718); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("cmpxchg success trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if got, err := memory.ReadUint64(dataAddress); err != nil || got != 0xaabbccddeeff0011 {
		t.Fatalf("cmpxchg16b low=%#x err=%v", got, err)
	}
	if got, err := memory.ReadUint64(dataAddress + 8); err != nil || got != 0x2233445566778899 {
		t.Fatalf("cmpxchg16b high=%#x err=%v", got, err)
	}
	if !state.Flag(Flag64ZF) {
		t.Fatal("cmpxchg16b success did not set ZF")
	}

	mismatchMemory := NewMemory64()
	const mismatchCode Address64 = 0x1b000
	const mismatchData Address64 = 0x2b000
	mapExecutable64(t, mismatchMemory, mismatchCode, []byte{
		0xb8, 0x11, 0x11, 0x11, 0x11, // mov eax, 0x11111111
		0xba, 0x22, 0x22, 0x22, 0x22, // mov edx, 0x22222222
		0x0f, 0xc7, 0x0f, // cmpxchg8b [rdi]
		0xf4, // hlt
	})
	if err := mismatchMemory.Map(mismatchData, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := mismatchMemory.WriteUint64(mismatchData, 0x88776655ccbbaa99); err != nil {
		t.Fatal(err)
	}
	mismatchState := NewMachineState64(mismatchMemory)
	mismatchState.RIP = uint64(mismatchCode)
	mismatchState.Set(RDI, uint64(mismatchData))
	if trap := NewJIT64(mismatchMemory).RunToInterrupt(mismatchState); trap != Trap64Timer || !mismatchState.Halted {
		t.Fatalf("cmpxchg mismatch trap=%#x halted=%v", trap, mismatchState.Halted)
	}
	if mismatchState.Get(RAX) != 0xccbbaa99 || mismatchState.Get(RDX) != 0x88776655 || mismatchState.Flag(Flag64ZF) {
		t.Fatalf("cmpxchg8b mismatch rax=%#x rdx=%#x zf=%v", mismatchState.Get(RAX), mismatchState.Get(RDX), mismatchState.Flag(Flag64ZF))
	}
}

func TestJIT64SSE2ShiftsShuffleAndMovemask(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x1c000
	code := []byte{
		0x66, 0x0f, 0x71, 0xf0, 0x01, // psllw xmm0, 1
		0x66, 0x0f, 0x72, 0xd1, 0x04, // psrld xmm1, 4
		0x66, 0x0f, 0x71, 0xe2, 0x02, // psraw xmm2, 2
		0x66, 0x0f, 0x73, 0xfb, 0x02, // pslldq xmm3, 2
		0x66, 0x0f, 0x73, 0xdc, 0x03, // psrldq xmm4, 3
		0x66, 0x0f, 0x70, 0xee, 0x1b, // pshufd xmm5, xmm6, 0x1b
		0x66, 0x0f, 0xd7, 0xc7, // pmovmskb eax, xmm7
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	for i, value := range []uint16{1, 2, 3, 4, 5, 6, 7, 8} {
		binary.LittleEndian.PutUint16(state.XMM[0][i*2:], value)
	}
	for i, value := range []uint32{0x12345000, 0x80000000, 0xffffffff, 0x00000010} {
		binary.LittleEndian.PutUint32(state.XMM[1][i*4:], value)
	}
	for i, value := range []uint32{0xfffffff0, 0x80000000, 0x7fffffff, 0x00000010} {
		binary.LittleEndian.PutUint32(state.XMM[2][i*4:], value)
	}
	for i := range state.XMM[3] {
		state.XMM[3][i] = byte(i)
		state.XMM[4][i] = byte(i + 0x10)
	}
	for i, value := range []uint32{10, 20, 30, 40} {
		binary.LittleEndian.PutUint32(state.XMM[6][i*4:], value)
	}
	for _, index := range []int{0, 2, 4, 7, 15} {
		state.XMM[7][index] = 0x80
	}
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("SSE shift/shuffle trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	for i, value := range []uint16{2, 4, 6, 8, 10, 12, 14, 16} {
		if got := binary.LittleEndian.Uint16(state.XMM[0][i*2:]); got != value {
			t.Fatalf("PSLLW lane %d = %#x, want %#x", i, got, value)
		}
	}
	for i, value := range []uint32{0x01234500, 0x08000000, 0x0fffffff, 0x00000001} {
		if got := binary.LittleEndian.Uint32(state.XMM[1][i*4:]); got != value {
			t.Fatalf("PSRLD lane %d = %#x, want %#x", i, got, value)
		}
	}
	for i, value := range []uint32{0xfffffffc, 0xe0000000, 0x1fffffff, 0x00000004} {
		if got := binary.LittleEndian.Uint32(state.XMM[2][i*4:]); got != value {
			t.Fatalf("PSRAD lane %d = %#x, want %#x", i, got, value)
		}
	}
	for i := 0; i < 2; i++ {
		if state.XMM[3][i] != 0 {
			t.Fatalf("PSLLDQ low byte %d = %#x, want 0", i, state.XMM[3][i])
		}
	}
	for i := 2; i < 16; i++ {
		if state.XMM[3][i] != byte(i-2) {
			t.Fatalf("PSLLDQ byte %d = %#x, want %#x", i, state.XMM[3][i], byte(i-2))
		}
	}
	for i := 0; i < 13; i++ {
		if state.XMM[4][i] != byte(i+3+0x10) {
			t.Fatalf("PSRLDQ byte %d = %#x, want %#x", i, state.XMM[4][i], byte(i+3+0x10))
		}
	}
	for i := 13; i < 16; i++ {
		if state.XMM[4][i] != 0 {
			t.Fatalf("PSRLDQ high byte %d = %#x, want 0", i, state.XMM[4][i])
		}
	}
	for i, value := range []uint32{40, 30, 20, 10} {
		if got := binary.LittleEndian.Uint32(state.XMM[5][i*4:]); got != value {
			t.Fatalf("PSHUFD lane %d = %#x, want %#x", i, got, value)
		}
	}
	if got := state.Get(RAX); got != 0x8095 {
		t.Fatalf("PMOVMSKB result=%#x, want %#x", got, uint64(0x8095))
	}
}

func TestJIT64SSE2UnpackAndWordShuffle(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x1d000
	code := []byte{
		0x66, 0x0f, 0x60, 0xc1, // punpcklbw xmm0, xmm1
		0x66, 0x0f, 0x68, 0xd3, // punpckhbw xmm2, xmm3
		0x66, 0x0f, 0x61, 0xe5, // punpcklwd xmm4, xmm5
		0x66, 0x0f, 0x69, 0xf7, // punpckhwd xmm6, xmm7
		0x66, 0x45, 0x0f, 0x62, 0xc1, // punpckldq xmm8, xmm9
		0x66, 0x45, 0x0f, 0x6a, 0xd3, // punpckhdq xmm10, xmm11
		0x66, 0x45, 0x0f, 0x6c, 0xe5, // punpcklqdq xmm12, xmm13
		0x66, 0x45, 0x0f, 0x6d, 0xf7, // punpckhqdq xmm14, xmm15
		0xf2, 0x0f, 0x70, 0xc8, 0x1b, // pshuflw xmm1, xmm0, 0x1b
		0xf3, 0x0f, 0x70, 0xda, 0x1b, // pshufhw xmm3, xmm2, 0x1b
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	for i := 0; i < 16; i++ {
		state.XMM[0][i] = byte(i)
		state.XMM[1][i] = byte(0x10 + i)
		state.XMM[2][i] = byte(i)
		state.XMM[3][i] = byte(0x20 + i)
	}
	for i, value := range []uint16{0, 1, 2, 3, 4, 5, 6, 7} {
		binary.LittleEndian.PutUint16(state.XMM[4][i*2:], value)
		binary.LittleEndian.PutUint16(state.XMM[6][i*2:], value)
		binary.LittleEndian.PutUint16(state.XMM[5][i*2:], 0x10+value)
		binary.LittleEndian.PutUint16(state.XMM[7][i*2:], 0x20+value)
	}
	for i, value := range []uint32{1, 2, 3, 4} {
		binary.LittleEndian.PutUint32(state.XMM[8][i*4:], value)
		binary.LittleEndian.PutUint32(state.XMM[10][i*4:], value)
		binary.LittleEndian.PutUint32(state.XMM[9][i*4:], 0x11+value-1)
		binary.LittleEndian.PutUint32(state.XMM[11][i*4:], 0x21+value-1)
	}
	binary.LittleEndian.PutUint64(state.XMM[12][0:], 1)
	binary.LittleEndian.PutUint64(state.XMM[12][8:], 2)
	binary.LittleEndian.PutUint64(state.XMM[13][0:], 0x11)
	binary.LittleEndian.PutUint64(state.XMM[13][8:], 0x12)
	binary.LittleEndian.PutUint64(state.XMM[14][0:], 1)
	binary.LittleEndian.PutUint64(state.XMM[14][8:], 2)
	binary.LittleEndian.PutUint64(state.XMM[15][0:], 0x21)
	binary.LittleEndian.PutUint64(state.XMM[15][8:], 0x22)
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("SSE unpack trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	wantBytes := func(reg int, want []byte) {
		t.Helper()
		if got := state.XMM[reg][:]; string(got) != string(want) {
			t.Fatalf("XMM%d = % x, want % x", reg, got, want)
		}
	}
	wantBytes(0, []byte{0, 0x10, 1, 0x11, 2, 0x12, 3, 0x13, 4, 0x14, 5, 0x15, 6, 0x16, 7, 0x17})
	wantBytes(1, []byte{3, 0x13, 2, 0x12, 1, 0x11, 0, 0x10, 4, 0x14, 5, 0x15, 6, 0x16, 7, 0x17})
	wantBytes(2, []byte{8, 0x28, 9, 0x29, 10, 0x2a, 11, 0x2b, 12, 0x2c, 13, 0x2d, 14, 0x2e, 15, 0x2f})
	wantBytes(3, []byte{8, 0x28, 9, 0x29, 10, 0x2a, 11, 0x2b, 15, 0x2f, 14, 0x2e, 13, 0x2d, 12, 0x2c})
	for i, want := range []uint16{0, 0x10, 1, 0x11, 2, 0x12, 3, 0x13} {
		if got := binary.LittleEndian.Uint16(state.XMM[4][i*2:]); got != want {
			t.Fatalf("PUNPCKLWD lane %d = %#x, want %#x", i, got, want)
		}
	}
	for i, want := range []uint16{4, 0x24, 5, 0x25, 6, 0x26, 7, 0x27} {
		if got := binary.LittleEndian.Uint16(state.XMM[6][i*2:]); got != want {
			t.Fatalf("PUNPCKHWD lane %d = %#x, want %#x", i, got, want)
		}
	}
	for i, want := range []uint32{1, 0x11, 2, 0x12} {
		if got := binary.LittleEndian.Uint32(state.XMM[8][i*4:]); got != want {
			t.Fatalf("PUNPCKLDQ lane %d = %#x, want %#x", i, got, want)
		}
	}
	for i, want := range []uint32{3, 0x23, 4, 0x24} {
		if got := binary.LittleEndian.Uint32(state.XMM[10][i*4:]); got != want {
			t.Fatalf("PUNPCKHDQ lane %d = %#x, want %#x", i, got, want)
		}
	}
	if got := binary.LittleEndian.Uint64(state.XMM[12][0:]); got != 1 || binary.LittleEndian.Uint64(state.XMM[12][8:]) != 0x11 {
		t.Fatalf("PUNPCKLQDQ = % x", state.XMM[12])
	}
	if got := binary.LittleEndian.Uint64(state.XMM[14][0:]); got != 2 || binary.LittleEndian.Uint64(state.XMM[14][8:]) != 0x22 {
		t.Fatalf("PUNPCKHQDQ = % x", state.XMM[14])
	}
}

func TestJIT64SSE2ShuffleAverageMinMax(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x1e000
	code := []byte{
		0x66, 0x0f, 0x38, 0x00, 0xc1, // pshufb xmm0, xmm1
		0x66, 0x0f, 0xe0, 0xd3, // pavgb xmm2, xmm3
		0x66, 0x0f, 0xe3, 0xe5, // pavgw xmm4, xmm5
		0x66, 0x0f, 0xda, 0xf7, // pminub xmm6, xmm7
		0x66, 0x45, 0x0f, 0x38, 0x38, 0xc1, // pminsb xmm8, xmm9
		0x66, 0x45, 0x0f, 0x38, 0x3f, 0xd3, // pmaxud xmm10, xmm11
		0xf4,
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	for i := 0; i < 16; i++ {
		state.XMM[0][i] = byte(i)
	}
	copy(state.XMM[1][:], []byte{15, 14, 13, 12, 0x80, 4, 6, 0x8f, 8, 9, 10, 11, 12, 13, 14, 15})
	copy(state.XMM[2][:], []byte{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150})
	copy(state.XMM[3][:], []byte{1, 11, 21, 31, 41, 51, 61, 71, 81, 91, 101, 111, 121, 131, 141, 151})
	for i, value := range []uint16{0, 100, 1000, 30000, 40000, 50000, 60000, 65535} {
		binary.LittleEndian.PutUint16(state.XMM[4][i*2:], value)
		binary.LittleEndian.PutUint16(state.XMM[5][i*2:], value+1)
	}
	copy(state.XMM[6][:], []byte{0, 255, 10, 240, 20, 230, 30, 220, 40, 210, 50, 200, 60, 190, 70, 180})
	copy(state.XMM[7][:], []byte{1, 1, 11, 239, 19, 231, 31, 219, 41, 211, 49, 201, 61, 189, 69, 181})
	copy(state.XMM[8][:], []byte{0x80, 0x7f, 0xfe, 0x02, 0x10, 0xf0, 0x7e, 0x81, 1, 2, 3, 4, 0xfc, 0xfd, 0x7d, 0x7c})
	copy(state.XMM[9][:], []byte{0x7f, 0x80, 0xfd, 0x03, 0x0f, 0x10, 0x7f, 0x80, 2, 1, 4, 3, 0xfb, 0xfe, 0x7e, 0x7b})
	for i, value := range []uint32{1, 0xffffffff, 100, 0x80000000} {
		binary.LittleEndian.PutUint32(state.XMM[10][i*4:], value)
	}
	for i, value := range []uint32{2, 3, 50, 0x7fffffff} {
		binary.LittleEndian.PutUint32(state.XMM[11][i*4:], value)
	}
	if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
		t.Fatalf("SIMD trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if got, want := state.XMM[0][:], []byte{15, 14, 13, 12, 0, 4, 6, 0, 8, 9, 10, 11, 12, 13, 14, 15}; string(got) != string(want) {
		t.Fatalf("PSHUFB = % x, want % x", got, want)
	}
	if got, want := state.XMM[2][:], []byte{1, 11, 21, 31, 41, 51, 61, 71, 81, 91, 101, 111, 121, 131, 141, 151}; string(got) != string(want) {
		t.Fatalf("PAVGB = % x, want % x", got, want)
	}
	for i, want := range []uint16{1, 101, 1001, 30001, 40001, 50001, 60001, 0x8000} {
		if got := binary.LittleEndian.Uint16(state.XMM[4][i*2:]); got != want {
			t.Fatalf("PAVGW lane %d = %#x, want %#x", i, got, want)
		}
	}
	if got, want := state.XMM[6][:], []byte{0, 1, 10, 239, 19, 230, 30, 219, 40, 210, 49, 200, 60, 189, 69, 180}; string(got) != string(want) {
		t.Fatalf("PMINUB = % x, want % x", got, want)
	}
	if got, want := state.XMM[8][:], []byte{0x80, 0x80, 0xfd, 0x02, 0x0f, 0xf0, 0x7e, 0x80, 1, 1, 3, 3, 0xfb, 0xfd, 0x7d, 0x7b}; string(got) != string(want) {
		t.Fatalf("PMINSB = % x, want % x", got, want)
	}
	for i, want := range []uint32{2, 0xffffffff, 100, 0x80000000} {
		if got := binary.LittleEndian.Uint32(state.XMM[10][i*4:]); got != want {
			t.Fatalf("PMAXUD lane %d = %#x, want %#x", i, got, want)
		}
	}
}

func TestJIT64ScalarAndFullSSEMoves(t *testing.T) {
	memory := NewMemory64()
	const start Address64 = 0x7000
	const data Address64 = 0x20000
	mapExecutable64(t, memory, start, []byte{
		0xf3, 0x0f, 0x10, 0xc1, // movss xmm0,xmm1
		0xf2, 0x0f, 0x10, 0xd3, // movsd xmm2,xmm3
		0x0f, 0x11, 0x07, // movups [rdi],xmm0
		0x0f, 0x28, 0x27, // movaps xmm4,[rdi]
		0xf2, 0x0f, 0x11, 0x57, 0x08, // movsd [rdi+8],xmm2
		0xf3, 0x0f, 0x10, 0x6f, 0x08, // movss xmm5,[rdi+8]
		0x0f, 0x05, // syscall
	})
	if err := memory.Map(data, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(start)
	state.Set(RDI, uint64(data))
	state.XMM[0] = [16]byte{0xf0, 0xf1, 0xf2, 0xf3, 0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xab}
	state.XMM[1] = [16]byte{0x01, 0x02, 0x03, 0x04, 0xb0, 0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7, 0xb8, 0xb9, 0xba, 0xbb}
	state.XMM[2] = [16]byte{0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xc0, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7}
	state.XMM[3] = [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0xd0, 0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7}
	state.XMM[5] = [16]byte{0x99, 0x99, 0x99, 0x99, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97, 0x98, 0x99, 0x9a, 0x9b, 0x9c}

	jit := NewJIT64(memory)
	if trap := jit.RunToInterrupt(state); trap != Trap64Syscall {
		t.Fatalf("trap=%#x, want syscall rip=%#x fault=%#x write=%v", trap, state.RIP, state.FaultAt, state.FaultWrite)
	}
	want0 := [16]byte{0x01, 0x02, 0x03, 0x04, 0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xab}
	if state.XMM[0] != want0 {
		t.Fatalf("MOVSS xmm0=%x, want %x", state.XMM[0], want0)
	}
	want2 := [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0xc0, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7}
	if state.XMM[2] != want2 {
		t.Fatalf("MOVSD xmm2=%x, want %x", state.XMM[2], want2)
	}
	if state.XMM[4] != want0 {
		t.Fatalf("MOVAPS xmm4=%x, want %x", state.XMM[4], want0)
	}
	want5 := state.XMM[5]
	want5[0], want5[1], want5[2], want5[3] = 0x11, 0x12, 0x13, 0x14
	if state.XMM[5] != want5 {
		t.Fatalf("MOVSS load xmm5=%x, want %x", state.XMM[5], want5)
	}
	var got [16]byte
	if err := memory.Read(data, got[:]); err != nil {
		t.Fatal(err)
	}
	wantMemory := [16]byte{0x01, 0x02, 0x03, 0x04, 0xa0, 0xa1, 0xa2, 0xa3, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}
	if got != wantMemory {
		t.Fatalf("SSE move memory=%x, want %x", got, wantMemory)
	}
}

func TestJIT64ScalarSSEArithmeticAndCompare(t *testing.T) {
	setFloat32 := func(state *MachineState64, xmm uint8, value float32) {
		binary.LittleEndian.PutUint32(state.XMM[xmm][0:4], math.Float32bits(value))
	}
	setFloat64 := func(state *MachineState64, xmm uint8, value float64) {
		binary.LittleEndian.PutUint64(state.XMM[xmm][0:8], math.Float64bits(value))
	}
	checkRun := func(t *testing.T, code []byte, state *MachineState64) {
		t.Helper()
		const codeAddress Address64 = 0x9000
		if err := state.Memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
		if err := state.Memory.Write(codeAddress, code); err != nil {
			t.Fatal(err)
		}
		state.RIP = uint64(codeAddress)
		trap := NewJIT64(state.Memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		setFloat32(state, 0, 1.5)
		setFloat32(state, 1, 2.5)
		setFloat32(state, 2, 2)
		setFloat32(state, 3, 3)
		checkRun(t, []byte{
			0xf3, 0x0f, 0x58, 0xc1, // addss xmm0, xmm1 = 4
			0xf3, 0x0f, 0x59, 0xc2, // mulss xmm0, xmm2 = 8
			0xf3, 0x0f, 0x5d, 0xc3, // minss xmm0, xmm3 = 3
			0xf3, 0x0f, 0x51, 0xc0, // sqrtss xmm0, xmm0
			0xf4,
		}, state)
		got := math.Float32frombits(binary.LittleEndian.Uint32(state.XMM[0][0:4]))
		if math.Abs(float64(got-float32(math.Sqrt(3)))) > 1e-6 {
			t.Fatalf("scalar float32 result=%v, want sqrt(3)", got)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		setFloat64(state, 0, 1.5)
		setFloat64(state, 1, 2.5)
		setFloat64(state, 2, 2)
		setFloat64(state, 3, 9)
		checkRun(t, []byte{
			0xf2, 0x0f, 0x58, 0xc1, // addsd xmm0, xmm1 = 4
			0xf2, 0x0f, 0x59, 0xc2, // mulsd xmm0, xmm2 = 8
			0xf2, 0x0f, 0x5f, 0xc3, // maxsd xmm0, xmm3 = 9
			0xf2, 0x0f, 0x51, 0xc0, // sqrtsd xmm0, xmm0
			0xf4,
		}, state)
		got := math.Float64frombits(binary.LittleEndian.Uint64(state.XMM[0][0:8]))
		if math.Abs(got-3) > 1e-12 {
			t.Fatalf("scalar float64 result=%v, want 3", got)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		setFloat32(state, 0, 1)
		setFloat32(state, 1, 2)
		checkRun(t, []byte{0x0f, 0x2f, 0xc1, 0xf4}, state) // comiss xmm0, xmm1; hlt
		if !state.Flag(Flag64CF) || state.Flag(Flag64ZF) || state.Flag(Flag64PF) {
			t.Fatalf("COMISS flags cf=%v zf=%v pf=%v", state.Flag(Flag64CF), state.Flag(Flag64ZF), state.Flag(Flag64PF))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		setFloat64(state, 0, math.NaN())
		setFloat64(state, 1, 2)
		checkRun(t, []byte{0x66, 0x0f, 0x2e, 0xc1, 0xf4}, state) // ucomisd xmm0, xmm1; hlt
		if !state.Flag(Flag64CF) || !state.Flag(Flag64ZF) || !state.Flag(Flag64PF) {
			t.Fatalf("UCOMISD unordered flags cf=%v zf=%v pf=%v", state.Flag(Flag64CF), state.Flag(Flag64ZF), state.Flag(Flag64PF))
		}
	}
}

func TestJIT64ScalarSSEConversions(t *testing.T) {
	setFloat32 := func(state *MachineState64, xmm uint8, value float32) {
		binary.LittleEndian.PutUint32(state.XMM[xmm][0:4], math.Float32bits(value))
	}
	setFloat64 := func(state *MachineState64, xmm uint8, value float64) {
		binary.LittleEndian.PutUint64(state.XMM[xmm][0:8], math.Float64bits(value))
	}
	checkRun := func(t *testing.T, code []byte, state *MachineState64) {
		t.Helper()
		const codeAddress Address64 = 0x1a000
		if err := state.Memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
		if err := state.Memory.Write(codeAddress, code); err != nil {
			t.Fatal(err)
		}
		state.RIP = uint64(codeAddress)
		trap := NewJIT64(state.Memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.Set(RAX, ^uint64(5))
		checkRun(t, []byte{
			0xf3, 0x48, 0x0f, 0x2a, 0xc0, // cvtsi2ss xmm0, rax = -6
			0xf2, 0x48, 0x0f, 0x2a, 0xd8, // cvtsi2sd xmm3, rax = -6
			0xf3, 0x0f, 0x5a, 0xc8, // cvtss2sd xmm1, xmm0 = -6
			0xf2, 0x0f, 0x5a, 0xd1, // cvtsd2ss xmm2, xmm1 = -6
			0xf3, 0x0f, 0x2c, 0xca, // cvttss2si ecx, xmm2 = -6
			0xf4,
		}, state)
		if got := math.Float32frombits(binary.LittleEndian.Uint32(state.XMM[0][0:4])); got != -6 {
			t.Fatalf("cvtsi2ss=%v, want -6", got)
		}
		if got := math.Float64frombits(binary.LittleEndian.Uint64(state.XMM[1][0:8])); got != -6 {
			t.Fatalf("cvtss2sd=%v, want -6", got)
		}
		if got := math.Float32frombits(binary.LittleEndian.Uint32(state.XMM[2][0:4])); got != -6 {
			t.Fatalf("cvtsd2ss=%v, want -6", got)
		}
		if got := math.Float64frombits(binary.LittleEndian.Uint64(state.XMM[3][0:8])); got != -6 {
			t.Fatalf("cvtsi2sd=%v, want -6", got)
		}
		if got := state.Get(RCX); got != 0xfffffffa {
			t.Fatalf("cvttss2si ecx=%#x, want 0xfffffffa", got)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		setFloat32(state, 0, 3.5)
		setFloat64(state, 1, -2.5)
		checkRun(t, []byte{
			0xf3, 0x0f, 0x2d, 0xc8, // cvtss2si ecx, xmm0: round-to-even 4
			0xf3, 0x0f, 0x2c, 0xd0, // cvttss2si edx, xmm0: truncate 3
			0xf2, 0x0f, 0x2d, 0xd1, // cvtsd2si edx, xmm1: round-to-even -2
			0xf2, 0x0f, 0x2c, 0xd9, // cvttsd2si ebx, xmm1: truncate -2
			0xf4,
		}, state)
		if got := state.Get(RCX); got != 4 {
			t.Fatalf("cvtss2si=%d, want 4", got)
		}
		if got := state.Get(RDX); got != 0xfffffffe {
			t.Fatalf("cvtsd2si=%#x, want 0xfffffffe", got)
		}
		if got := state.Get(RBX); got != 0xfffffffe {
			t.Fatalf("cvttsd2si=%#x, want 0xfffffffe", got)
		}
	}

	{
		memory := NewMemory64()
		const dataAddress Address64 = 0x1b000
		if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
			t.Fatal(err)
		}
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], math.Float32bits(3.5))
		if err := memory.Write(dataAddress, raw[:]); err != nil {
			t.Fatal(err)
		}
		state := NewMachineState64(memory)
		state.Set(RDI, uint64(dataAddress))
		checkRun(t, []byte{
			0xf3, 0x0f, 0x2d, 0x0f, // cvtss2si ecx, dword ptr [rdi] = 4
			0xf4,
		}, state)
		if got := state.Get(RCX); got != 4 {
			t.Fatalf("memory cvtss2si=%d, want 4", got)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		setFloat64(state, 0, 3.5)
		checkRun(t, []byte{
			0xf2, 0x48, 0x0f, 0x2d, 0xc0, // cvtsd2si rax, xmm0 = 4
			0xf2, 0x48, 0x0f, 0x2c, 0xc8, // cvttsd2si rcx, xmm0 = 3
			0xf4,
		}, state)
		if got := state.Get(RAX); got != 4 {
			t.Fatalf("64-bit cvtsd2si=%d, want 4", got)
		}
		if got := state.Get(RCX); got != 3 {
			t.Fatalf("64-bit cvttsd2si=%d, want 3", got)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		setFloat64(state, 0, math.NaN())
		checkRun(t, []byte{
			0xf2, 0x0f, 0x2d, 0xc8, // cvtsd2si ecx, xmm0 => indefinite integer
			0xf4,
		}, state)
		if got := state.Get(RCX); got != 0x80000000 {
			t.Fatalf("NaN cvtsd2si=%#x, want 0x80000000", got)
		}
	}
}

func TestJIT64SSE2AndBitCountExtensions(t *testing.T) {
	setVector := func(state *MachineState64, xmm uint8, value []byte) {
		t.Helper()
		if len(value) != 16 {
			t.Fatalf("vector length=%d, want 16", len(value))
		}
		copy(state.XMM[xmm][:], value)
	}
	checkRun := func(t *testing.T, code []byte, state *MachineState64) {
		t.Helper()
		const codeAddress Address64 = 0x1c000
		if err := state.Memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
		if err := state.Memory.Write(codeAddress, code); err != nil {
			t.Fatal(err)
		}
		state.RIP = uint64(codeAddress)
		trap := NewJIT64(state.Memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		left := make([]byte, 16)
		right := make([]byte, 16)
		for i := range left {
			left[i] = byte(i)
			right[i] = byte(0x10 + i)
		}
		setVector(state, 0, left)
		setVector(state, 1, right)
		checkRun(t, []byte{
			0x66, 0x0f, 0x3a, 0x0f, 0xc1, 0x08, // palignr xmm0, xmm1, 8
			0xf4,
		}, state)
		want := append(append([]byte{}, right[8:]...), left[:8]...)
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("palignr=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		var left, right [16]byte
		for lane := 0; lane < 8; lane++ {
			binary.LittleEndian.PutUint16(left[lane*2:], uint16(0x1000+lane))
			binary.LittleEndian.PutUint16(right[lane*2:], uint16(0x2000+lane))
		}
		setVector(state, 0, left[:])
		setVector(state, 1, right[:])
		checkRun(t, []byte{
			0x66, 0x0f, 0x3a, 0x0e, 0xc1, 0x55, // pblendw xmm0, xmm1, 0x55
			0xf4,
		}, state)
		for lane := 0; lane < 8; lane++ {
			got := binary.LittleEndian.Uint16(state.XMM[0][lane*2:])
			want := binary.LittleEndian.Uint16(left[lane*2:])
			if lane%2 == 0 {
				want = binary.LittleEndian.Uint16(right[lane*2:])
			}
			if got != want {
				t.Fatalf("pblendw lane %d=%#x, want %#x", lane, got, want)
			}
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		setVector(state, 0, []byte{0x7f, 0x80, 0x01, 0xff, 0x10, 0xf0, 0x00, 0x80, 0x7f, 0x01, 0x80, 0x02, 0x03, 0x04, 0xfb, 0xfc})
		setVector(state, 1, []byte{0x01, 0x7f, 0x00, 0x01, 0x20, 0x0f, 0x00, 0x7f, 0x80, 0x02, 0x7f, 0x01, 0x04, 0x03, 0xfa, 0xfd})
		want := [16]byte{}
		for i := range want {
			if int8(state.XMM[0][i]) > int8(state.XMM[1][i]) {
				want[i] = 0xff
			}
		}
		checkRun(t, []byte{0x66, 0x0f, 0x64, 0xc1, 0xf4}, state) // pcmpgtb xmm0, xmm1
		if !bytes.Equal(state.XMM[0][:], want[:]) {
			t.Fatalf("pcmpgtb=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		var left, right [16]byte
		for lane := 0; lane < 8; lane++ {
			binary.LittleEndian.PutUint16(left[lane*2:], uint16(int16(lane)-3))
			binary.LittleEndian.PutUint16(right[lane*2:], uint16(int16(lane)-4))
		}
		setVector(state, 0, left[:])
		setVector(state, 1, right[:])
		checkRun(t, []byte{0x66, 0x0f, 0x65, 0xc1, 0xf4}, state) // pcmpgtw xmm0, xmm1
		for lane := 0; lane < 8; lane++ {
			if binary.LittleEndian.Uint16(state.XMM[0][lane*2:]) != 0xffff {
				t.Fatalf("pcmpgtw lane %d=%#x, want 0xffff", lane, binary.LittleEndian.Uint16(state.XMM[0][lane*2:]))
			}
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		var left, right [16]byte
		binary.LittleEndian.PutUint32(left[0:], uint32(0xffffffff))
		binary.LittleEndian.PutUint32(right[0:], uint32(0xfffffffe))
		binary.LittleEndian.PutUint32(left[4:], 5)
		binary.LittleEndian.PutUint32(right[4:], 5)
		binary.LittleEndian.PutUint32(left[8:], 7)
		binary.LittleEndian.PutUint32(right[8:], 8)
		binary.LittleEndian.PutUint32(left[12:], uint32(0xfffffff8))
		binary.LittleEndian.PutUint32(right[12:], uint32(int32(7)))
		setVector(state, 0, left[:])
		setVector(state, 1, right[:])
		checkRun(t, []byte{0x66, 0x0f, 0x66, 0xc1, 0xf4}, state) // pcmpgtd xmm0, xmm1
		if got := binary.LittleEndian.Uint32(state.XMM[0][0:4]); got != 0xffffffff {
			t.Fatalf("pcmpgtd lane0=%#x, want 0xffffffff", got)
		}
		if got := binary.LittleEndian.Uint32(state.XMM[0][4:8]); got != 0 {
			t.Fatalf("pcmpgtd lane1=%#x, want 0", got)
		}
		if got := binary.LittleEndian.Uint32(state.XMM[0][8:12]); got != 0 {
			t.Fatalf("pcmpgtd lane2=%#x, want 0", got)
		}
		if got := binary.LittleEndian.Uint32(state.XMM[0][12:16]); got != 0 {
			t.Fatalf("pcmpgtd lane3=%#x, want 0", got)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		setVector(state, 0, []byte{0x0f, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		setVector(state, 1, []byte{0x03, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		checkRun(t, []byte{0x66, 0x0f, 0x38, 0x17, 0xc1, 0xf4}, state) // ptest xmm0, xmm1
		if state.Flag(Flag64ZF) || !state.Flag(Flag64CF) {
			t.Fatalf("ptest flags zf=%v cf=%v, want false/true", state.Flag(Flag64ZF), state.Flag(Flag64CF))
		}
	}
	{
		state := NewMachineState64(NewMemory64())
		setVector(state, 0, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		setVector(state, 1, []byte{0x03, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		checkRun(t, []byte{0x66, 0x0f, 0x38, 0x17, 0xc1, 0xf4}, state)
		if !state.Flag(Flag64ZF) || state.Flag(Flag64CF) {
			t.Fatalf("ptest zero flags zf=%v cf=%v, want true/false", state.Flag(Flag64ZF), state.Flag(Flag64CF))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		var source [16]byte
		for i := range source {
			source[i] = byte(0xa0 + i)
		}
		setVector(state, 1, source[:])
		checkRun(t, []byte{0xf2, 0x0f, 0x12, 0xc1, 0xf4}, state) // movddup xmm0, xmm1
		if !bytes.Equal(state.XMM[0][0:8], source[0:8]) || !bytes.Equal(state.XMM[0][8:16], source[0:8]) {
			t.Fatalf("movddup=%x, want %x duplicated", state.XMM[0], source[0:8])
		}
	}

	{
		memory := NewMemory64()
		const dataAddress Address64 = 0x1d000
		if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
			t.Fatal(err)
		}
		var source [8]byte
		for i := range source {
			source[i] = byte(0xb0 + i)
		}
		if err := memory.Write(dataAddress, source[:]); err != nil {
			t.Fatal(err)
		}
		state := NewMachineState64(memory)
		state.Set(RDI, uint64(dataAddress))
		checkRun(t, []byte{0xf2, 0x0f, 0x12, 0x07, 0xf4}, state) // movddup xmm0, qword ptr [rdi]
		if !bytes.Equal(state.XMM[0][0:8], source[:]) || !bytes.Equal(state.XMM[0][8:16], source[:]) {
			t.Fatalf("memory movddup=%x, want %x duplicated", state.XMM[0], source)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.Set(RCX, 0x00100000)
		checkRun(t, []byte{0xf3, 0x0f, 0xbd, 0xc1, 0xf4}, state) // lzcnt eax, ecx = 11
		if got := state.Get(RAX); got != 11 || state.Flag(Flag64CF) || state.Flag(Flag64ZF) {
			t.Fatalf("lzcnt value=%d cf=%v zf=%v, want 11/false/false", got, state.Flag(Flag64CF), state.Flag(Flag64ZF))
		}
	}
	{
		state := NewMachineState64(NewMemory64())
		state.Set(RCX, 0)
		checkRun(t, []byte{0xf3, 0x0f, 0xbd, 0xc1, 0xf4}, state) // lzcnt eax, ecx = 32
		if got := state.Get(RAX); got != 32 || !state.Flag(Flag64CF) || state.Flag(Flag64ZF) {
			t.Fatalf("lzcnt zero value=%d cf=%v zf=%v, want 32/true/false", got, state.Flag(Flag64CF), state.Flag(Flag64ZF))
		}
	}
	{
		state := NewMachineState64(NewMemory64())
		state.Set(RCX, 0x10)
		checkRun(t, []byte{0xf3, 0x0f, 0xbc, 0xc1, 0xf4}, state) // tzcnt eax, ecx = 4
		if got := state.Get(RAX); got != 4 || state.Flag(Flag64CF) || state.Flag(Flag64ZF) {
			t.Fatalf("tzcnt value=%d cf=%v zf=%v, want 4/false/false", got, state.Flag(Flag64CF), state.Flag(Flag64ZF))
		}
	}
	{
		state := NewMachineState64(NewMemory64())
		state.Set(RCX, 1)
		checkRun(t, []byte{0xf3, 0x48, 0x0f, 0xbd, 0xc1, 0xf4}, state) // lzcnt rax, rcx = 63
		if got := state.Get(RAX); got != 63 || state.Flag(Flag64CF) || state.Flag(Flag64ZF) {
			t.Fatalf("64-bit lzcnt value=%d cf=%v zf=%v, want 63/false/false", got, state.Flag(Flag64CF), state.Flag(Flag64ZF))
		}
	}
	{
		state := NewMachineState64(NewMemory64())
		state.Set(RCX, 1)
		checkRun(t, []byte{0xf3, 0x0f, 0xbc, 0xc1, 0xf4}, state) // tzcnt eax, ecx = 0
		if got := state.Get(RAX); got != 0 || state.Flag(Flag64CF) || !state.Flag(Flag64ZF) {
			t.Fatalf("tzcnt low-bit value=%d cf=%v zf=%v, want 0/false/true", got, state.Flag(Flag64CF), state.Flag(Flag64ZF))
		}
	}
}

func TestJIT64PackedSSEConversions(t *testing.T) {
	setVector := func(state *MachineState64, xmm uint8, value []byte) {
		t.Helper()
		if len(value) != 16 {
			t.Fatalf("vector length=%d, want 16", len(value))
		}
		copy(state.XMM[xmm][:], value)
	}
	checkRun := func(t *testing.T, code []byte, state *MachineState64) {
		t.Helper()
		const codeAddress Address64 = 0x1e000
		if err := state.Memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
		if err := state.Memory.Write(codeAddress, code); err != nil {
			t.Fatal(err)
		}
		state.RIP = uint64(codeAddress)
		trap := NewJIT64(state.Memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}
	putInt32 := func(vector []byte, lane int, value int32) {
		binary.LittleEndian.PutUint32(vector[lane*4:], uint32(value))
	}
	putFloat32 := func(vector []byte, lane int, value float32) {
		binary.LittleEndian.PutUint32(vector[lane*4:], math.Float32bits(value))
	}
	putFloat64 := func(vector []byte, lane int, value float64) {
		binary.LittleEndian.PutUint64(vector[lane*8:], math.Float64bits(value))
	}

	{
		state := NewMachineState64(NewMemory64())
		source := make([]byte, 16)
		putInt32(source, 0, -3)
		putInt32(source, 1, 0)
		putInt32(source, 2, 2)
		putInt32(source, 3, 16777217)
		setVector(state, 1, source)
		checkRun(t, []byte{0x0f, 0x5b, 0xc1, 0xf4}, state) // cvtdq2ps xmm0, xmm1
		want := make([]byte, 16)
		putFloat32(want, 0, -3)
		putFloat32(want, 1, 0)
		putFloat32(want, 2, 2)
		putFloat32(want, 3, 16777216) // rounded by float32 representation
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("cvtdq2ps=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		source := make([]byte, 16)
		putFloat32(source, 0, 3.5)
		putFloat32(source, 1, -2.5)
		putFloat32(source, 2, 2.6)
		putFloat32(source, 3, float32(math.NaN()))
		setVector(state, 1, source)
		checkRun(t, []byte{0x66, 0x0f, 0x5b, 0xc1, 0xf4}, state) // cvtps2dq xmm0, xmm1
		want := make([]byte, 16)
		putInt32(want, 0, 4)
		putInt32(want, 1, -2)
		putInt32(want, 2, 3)
		putInt32(want, 3, math.MinInt32)
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("cvtps2dq=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		source := make([]byte, 16)
		putFloat32(source, 0, 3.9)
		putFloat32(source, 1, -2.9)
		putFloat32(source, 2, float32(math.Inf(1)))
		putFloat32(source, 3, float32(math.NaN()))
		setVector(state, 1, source)
		checkRun(t, []byte{0xf3, 0x0f, 0x5b, 0xc1, 0xf4}, state) // cvttps2dq xmm0, xmm1
		want := make([]byte, 16)
		putInt32(want, 0, 3)
		putInt32(want, 1, -2)
		putInt32(want, 2, math.MinInt32)
		putInt32(want, 3, math.MinInt32)
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("cvttps2dq=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		source := make([]byte, 16)
		putInt32(source, 0, -7)
		putInt32(source, 1, 11)
		putInt32(source, 2, 123)
		putInt32(source, 3, -456)
		setVector(state, 1, source)
		checkRun(t, []byte{0xf3, 0x0f, 0xe6, 0xc1, 0xf4}, state) // cvtdq2pd xmm0, xmm1
		want := make([]byte, 16)
		putFloat64(want, 0, -7)
		putFloat64(want, 1, 11)
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("cvtdq2pd=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		source := make([]byte, 16)
		putFloat64(source, 0, 3.5)
		putFloat64(source, 1, -2.5)
		setVector(state, 1, source)
		for i := 8; i < 16; i++ {
			state.XMM[0][i] = 0xaa
		}
		checkRun(t, []byte{0xf2, 0x0f, 0xe6, 0xc1, 0xf4}, state) // cvtpd2dq xmm0, xmm1
		want := make([]byte, 16)
		putInt32(want, 0, 4)
		putInt32(want, 1, -2)
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("cvtpd2dq=%x, want %x", state.XMM[0], want)
		}
	}

	{
		memory := NewMemory64()
		const dataAddress Address64 = 0x1f000
		if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
			t.Fatal(err)
		}
		source := make([]byte, 8)
		putInt32(source, 0, -12)
		putInt32(source, 1, 34)
		if err := memory.Write(dataAddress, source); err != nil {
			t.Fatal(err)
		}
		state := NewMachineState64(memory)
		state.Set(RDI, uint64(dataAddress))
		checkRun(t, []byte{0xf3, 0x0f, 0xe6, 0x07, 0xf4}, state) // cvtdq2pd xmm0, qword ptr [rdi]
		want := make([]byte, 16)
		putFloat64(want, 0, -12)
		putFloat64(want, 1, 34)
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("memory cvtdq2pd=%x, want %x", state.XMM[0], want)
		}
	}
}

func TestJIT64PackedArithmeticAndSaturation(t *testing.T) {
	run := func(t *testing.T, code []byte, state *MachineState64) {
		t.Helper()
		const address Address64 = 0x22000
		mapExecutable64(t, state.Memory, address, code)
		state.RIP = uint64(address)
		trap := NewJIT64(state.Memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}
	setVector := func(state *MachineState64, xmm uint8, value []byte) {
		t.Helper()
		if len(value) != 16 {
			t.Fatalf("vector length=%d, want 16", len(value))
		}
		copy(state.XMM[xmm][:], value)
	}
	putU16 := func(value []byte, lane int, raw uint16) {
		binary.LittleEndian.PutUint16(value[lane*2:], raw)
	}
	putU32 := func(value []byte, lane int, raw uint32) {
		binary.LittleEndian.PutUint32(value[lane*4:], raw)
	}
	putU64 := func(value []byte, lane int, raw uint64) {
		binary.LittleEndian.PutUint64(value[lane*8:], raw)
	}

	{
		state := NewMachineState64(NewMemory64())
		left := make([]byte, 16)
		right := make([]byte, 16)
		putU32(left, 0, 0xffffffff)
		putU32(left, 1, 0x00000002)
		putU32(left, 2, 0x12345678)
		putU32(left, 3, 0x00000003)
		putU32(right, 0, 2)
		putU32(right, 1, 3)
		putU32(right, 2, 0x10)
		putU32(right, 3, 7)
		setVector(state, 0, left)
		setVector(state, 1, right)
		run(t, []byte{0x66, 0x0f, 0xf4, 0xc1, 0xf4}, state) // pmuludq xmm0, xmm1
		want := make([]byte, 16)
		putU64(want, 0, uint64(0xffffffff)*2)
		putU64(want, 1, uint64(0x12345678)*0x10)
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("pmuludq=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		left := make([]byte, 16)
		right := make([]byte, 16)
		for i, value := range []int16{-300, -2, 3, 200, 7, -8, 11, 12} {
			putU16(left, i, uint16(value))
		}
		for i, value := range []int16{2, -3, 4, -5, 6, 7, -9, 10} {
			putU16(right, i, uint16(value))
		}
		setVector(state, 0, left)
		setVector(state, 1, right)
		run(t, []byte{0x66, 0x0f, 0xd5, 0xc1, 0xf4}, state) // pmullw xmm0, xmm1
		want := make([]byte, 16)
		for i, value := range []int16{-600, 6, 12, -1000, 42, -56, -99, 120} {
			putU16(want, i, uint16(value))
		}
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("pmullw=%x, want %x", state.XMM[0], want)
		}

		setVector(state, 0, left)
		run(t, []byte{0x66, 0x0f, 0xe5, 0xc1, 0xf4}, state) // pmulhw xmm0, xmm1
		want = make([]byte, 16)
		for i, value := range []int16{-1, 0, 0, -1, 0, -1, -1, 0} {
			putU16(want, i, uint16(value))
		}
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("pmulhw=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		left := make([]byte, 16)
		right := make([]byte, 16)
		for i := 0; i < 8; i++ {
			left[i] = byte(i)
			right[i] = byte(8 - i)
			left[i+8] = byte(i + 8)
			right[i+8] = byte(i)
		}
		setVector(state, 0, left)
		setVector(state, 1, right)
		run(t, []byte{0x66, 0x0f, 0xf6, 0xc1, 0xf4}, state) // psadbw xmm0, xmm1
		want := make([]byte, 16)
		putU64(want, 0, 32)
		putU64(want, 1, 64)
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("psadbw=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		left := make([]byte, 16)
		right := make([]byte, 16)
		for i, value := range []int16{-200, -129, -128, 0, 127, 128, 300, -1} {
			putU16(left, i, uint16(value))
		}
		for i, value := range []int16{200, 129, -127, 1, -127, -128, -300, 127} {
			putU16(right, i, uint16(value))
		}
		setVector(state, 0, left)
		setVector(state, 1, right)
		run(t, []byte{0x66, 0x0f, 0x63, 0xc1, 0xf4}, state) // packsswb xmm0, xmm1
		want := []byte{0x80, 0x80, 0x80, 0x00, 0x7f, 0x7f, 0x7f, 0xff, 0x7f, 0x7f, 0x81, 0x01, 0x81, 0x80, 0x80, 0x7f}
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("packsswb=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		left := make([]byte, 16)
		right := make([]byte, 16)
		for i, value := range []int32{-40000, -32768, 32767, 40000} {
			putU32(left, i, uint32(value))
		}
		for i, value := range []int32{-1, 0, 1, 100000} {
			putU32(right, i, uint32(value))
		}
		setVector(state, 0, left)
		setVector(state, 1, right)
		run(t, []byte{0x66, 0x0f, 0x6b, 0xc1, 0xf4}, state) // packssdw xmm0, xmm1
		want := make([]byte, 16)
		for i, value := range []int16{-32768, -32768, 32767, 32767, -1, 0, 1, 32767} {
			putU16(want, i, uint16(value))
		}
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("packssdw=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		left := make([]byte, 16)
		right := make([]byte, 16)
		for i, value := range []int16{-1, 0, 1, 255, 256, 32767, -32768, 7} {
			putU16(left, i, uint16(value))
		}
		for i, value := range []int16{300, -3, 2, 254, -200, 256, 8, 9} {
			putU16(right, i, uint16(value))
		}
		setVector(state, 0, left)
		setVector(state, 1, right)
		run(t, []byte{0x66, 0x0f, 0x67, 0xc1, 0xf4}, state) // packuswb xmm0, xmm1
		want := []byte{0, 0, 1, 255, 255, 255, 0, 7, 255, 0, 2, 254, 0, 255, 8, 9}
		if !bytes.Equal(state.XMM[0][:], want) {
			t.Fatalf("packuswb=%x, want %x", state.XMM[0], want)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		left := make([]byte, 16)
		right := make([]byte, 16)
		for i := range left {
			left[i] = 250
			right[i] = 10
		}
		setVector(state, 0, left)
		setVector(state, 1, right)
		run(t, []byte{0x66, 0x0f, 0xdc, 0xc1, 0xf4}, state) // paddusb xmm0, xmm1
		for i, value := range state.XMM[0] {
			if value != 255 {
				t.Fatalf("paddusb lane %d=%d, want 255", i, value)
			}
		}

		for i := range left {
			left[i] = 3
			right[i] = 5
		}
		setVector(state, 0, left)
		setVector(state, 1, right)
		run(t, []byte{0x66, 0x0f, 0xd8, 0xc1, 0xf4}, state) // psubusb xmm0, xmm1
		for i, value := range state.XMM[0] {
			if value != 0 {
				t.Fatalf("psubusb lane %d=%d, want 0", i, value)
			}
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		left := make([]byte, 16)
		right := make([]byte, 16)
		for i := 0; i < 8; i++ {
			putU16(left, i, 65000)
			putU16(right, i, 1000)
		}
		setVector(state, 0, left)
		setVector(state, 1, right)
		run(t, []byte{0x66, 0x0f, 0xdd, 0xc1, 0xf4}, state) // paddusw xmm0, xmm1
		for i := 0; i < 8; i++ {
			if binary.LittleEndian.Uint16(state.XMM[0][i*2:]) != 65535 {
				t.Fatalf("paddusw lane %d=%d, want 65535", i, binary.LittleEndian.Uint16(state.XMM[0][i*2:]))
			}
		}

		for i := 0; i < 8; i++ {
			putU16(left, i, 1000)
			putU16(right, i, 2000)
		}
		setVector(state, 0, left)
		setVector(state, 1, right)
		run(t, []byte{0x66, 0x0f, 0xd9, 0xc1, 0xf4}, state) // psubusw xmm0, xmm1
		for i := 0; i < 8; i++ {
			if binary.LittleEndian.Uint16(state.XMM[0][i*2:]) != 0 {
				t.Fatalf("psubusw lane %d=%d, want 0", i, binary.LittleEndian.Uint16(state.XMM[0][i*2:]))
			}
		}
	}

	{
		memory := NewMemory64()
		const dataAddress Address64 = 0x23000
		if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
			t.Fatal(err)
		}
		source := make([]byte, 16)
		for i := range source {
			source[i] = 1
		}
		if err := memory.Write(dataAddress, source); err != nil {
			t.Fatal(err)
		}
		state := NewMachineState64(memory)
		state.Set(RDI, uint64(dataAddress))
		left := make([]byte, 16)
		for i := range left {
			left[i] = 2
		}
		setVector(state, 0, left)
		run(t, []byte{0x66, 0x0f, 0xdc, 0x07, 0xf4}, state) // paddusb xmm0, [rdi]
		for i, value := range state.XMM[0] {
			if value != 3 {
				t.Fatalf("memory paddusb lane %d=%d, want 3", i, value)
			}
		}
	}
}

func TestJIT64PackedInsertExtract(t *testing.T) {
	run := func(t *testing.T, code []byte, state *MachineState64) {
		t.Helper()
		const address Address64 = 0x24000
		mapExecutable64(t, state.Memory, address, code)
		state.RIP = uint64(address)
		trap := NewJIT64(state.Memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}
	mapData := func(state *MachineState64, address Address64, data []byte) {
		t.Helper()
		if err := state.Memory.Map(address, Page64Size, PRead|PWrite); err != nil {
			t.Fatal(err)
		}
		if err := state.Memory.Write(address, data); err != nil {
			t.Fatal(err)
		}
	}
	checkWord := func(vector [16]byte, lane int, want uint16) {
		t.Helper()
		got := binary.LittleEndian.Uint16(vector[lane*2:])
		if got != want {
			t.Fatalf("word lane %d=%#x, want %#x; vector=%x", lane, got, want, vector)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		for i := range state.XMM[0] {
			state.XMM[0][i] = 0xaa
		}
		state.Set(RCX, 0x1234)
		run(t, []byte{0x66, 0x0f, 0xc4, 0xc1, 0x02, 0xf4}, state) // pinsrw xmm0, ecx, 2
		checkWord(state.XMM[0], 2, 0x1234)
		if state.XMM[0][0] != 0xaa || state.XMM[0][15] != 0xaa {
			t.Fatalf("pinsrw changed unrelated lanes: %x", state.XMM[0])
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.Set(RCX, 0xabcdef12)
		run(t, []byte{0x66, 0x0f, 0x3a, 0x20, 0xc1, 0x05, 0xf4}, state) // pinsrb xmm0, ecx, 5
		if state.XMM[0][5] != 0x12 {
			t.Fatalf("pinsrb byte=%#x, want %#x", state.XMM[0][5], 0x12)
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.Set(RCX, 0xdeadbeef)
		run(t, []byte{0x66, 0x0f, 0x3a, 0x22, 0xc1, 0x01, 0xf4}, state) // pinsrd xmm0, ecx, 1
		if got := binary.LittleEndian.Uint32(state.XMM[0][4:]); got != 0xdeadbeef {
			t.Fatalf("pinsrd=%#x, want %#x", got, uint32(0xdeadbeef))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.Set(RCX, 0x1122334455667788)
		run(t, []byte{0x66, 0x48, 0x0f, 0x3a, 0x22, 0xc1, 0x01, 0xf4}, state) // pinsrq xmm0, rcx, 1
		if got := binary.LittleEndian.Uint64(state.XMM[0][8:]); got != 0x1122334455667788 {
			t.Fatalf("pinsrq=%#x, want %#x", got, uint64(0x1122334455667788))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.XMM[0] = [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
		run(t, []byte{0x66, 0x0f, 0xc5, 0xc0, 0x02, 0xf4}, state) // pextrw eax, xmm0, 2
		if got := state.Get(RAX); got != 0x0000000000005544 {
			t.Fatalf("pextrw RAX=%#x, want %#x", got, uint64(0x5544))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.XMM[0] = [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
		run(t, []byte{0x66, 0x0f, 0x3a, 0x14, 0xc0, 0x0b, 0xf4}, state) // pextrb eax, xmm0, 11
		if got := state.Get(RAX); got != 0x00000000000000bb {
			t.Fatalf("pextrb RAX=%#x, want %#x", got, uint64(0xbb))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.XMM[0] = [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
		run(t, []byte{0x66, 0x48, 0x0f, 0x3a, 0x14, 0xc0, 0x0b, 0xf4}, state) // pextrb rax, xmm0, 11 (REX.W)
		if got := state.Get(RAX); got != 0x00000000000000bb {
			t.Fatalf("pextrb rex.w RAX=%#x, want %#x", got, uint64(0xbb))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.Set(RCX, 0x1122334455667788)
		run(t, []byte{0x66, 0x48, 0x0f, 0x3a, 0x20, 0xc1, 0x05, 0xf4}, state) // pinsrb xmm0, rcx, 5 (REX.W ignored for byte source)
		if got := state.XMM[0][5]; got != 0x88 {
			t.Fatalf("pinsrb rex.w byte=%#x, want %#x", got, byte(0x88))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.XMM[0] = [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
		run(t, []byte{0x66, 0x0f, 0x3a, 0x16, 0xc0, 0x01, 0xf4}, state) // pextrd eax, xmm0, 1
		if got := state.Get(RAX); got != 0x0000000077665544 {
			t.Fatalf("pextrd RAX=%#x, want %#x", got, uint64(0x77665544))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		state.XMM[0] = [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
		run(t, []byte{0x66, 0x48, 0x0f, 0x3a, 0x16, 0xc0, 0x01, 0xf4}, state) // pextrq rax, xmm0, 1
		if got := state.Get(RAX); got != 0xffeeddccbbaa9988 {
			t.Fatalf("pextrq RAX=%#x, want %#x", got, uint64(0xffeeddccbbaa9988))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		const dataAddress Address64 = 0x25000
		mapData(state, dataAddress, []byte{0, 0, 0, 0, 0, 0, 0, 0})
		state.Set(RDI, uint64(dataAddress))
		state.XMM[0] = [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
		run(t, []byte{0x66, 0x0f, 0x3a, 0x16, 0x07, 0x01, 0xf4}, state) // pextrd [rdi], xmm0, 1
		stored := make([]byte, 4)
		if err := state.Memory.Read(dataAddress, stored); err != nil {
			t.Fatal(err)
		}
		if got := binary.LittleEndian.Uint32(stored); got != 0x77665544 {
			t.Fatalf("memory pextrd=%#x, want %#x", got, uint32(0x77665544))
		}
	}

	{
		state := NewMachineState64(NewMemory64())
		const dataAddress Address64 = 0x26000
		mapData(state, dataAddress, []byte{0x78, 0x56, 0x34, 0x12, 0, 0, 0, 0})
		state.Set(RDI, uint64(dataAddress))
		run(t, []byte{0x66, 0x0f, 0x3a, 0x22, 0x07, 0x02, 0xf4}, state) // pinsrd xmm0, [rdi], 2
		if got := binary.LittleEndian.Uint32(state.XMM[0][8:]); got != 0x12345678 {
			t.Fatalf("memory pinsrd=%#x, want %#x", got, uint32(0x12345678))
		}
	}
}

func TestJIT64SignedSaturatingPackedArithmetic(t *testing.T) {
	putWords := func(vector *[16]byte, values ...int16) {
		t.Helper()
		if len(values) != 8 {
			t.Fatalf("word count=%d, want 8", len(values))
		}
		for i, value := range values {
			binary.LittleEndian.PutUint16(vector[i*2:], uint16(value))
		}
	}
	checkRun := func(t *testing.T, memory *Memory64, codeAddress Address64, code []byte, state *MachineState64) {
		t.Helper()
		if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
		if err := memory.Write(codeAddress, code); err != nil {
			t.Fatal(err)
		}
		state.RIP = uint64(codeAddress)
		trap := NewJIT64(memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}

	state := NewMachineState64(NewMemory64())
	state.XMM[0] = [16]byte{120, 127, 136, 128, 10, 246, 0, 1, 0x40, 0xc0, 0x7f, 0x81, 0, 0, 0x7f, 0x80}
	state.XMM[1] = [16]byte{20, 1, 236, 255, 30, 30, 1, 2, 0x40, 0x40, 1, 0xff, 1, 1, 1, 1}
	putWords(&state.XMM[2], 32000, 32767, -32000, -32768, 100, -100, 0, 1)
	putWords(&state.XMM[3], 2000, 1, -2000, -1, -200, 200, -1, -2)
	state.XMM[4] = [16]byte{136, 128, 120, 127, 10, 246, 0, 1, 0x40, 0xc0, 0x7f, 0x81, 0, 0, 0x7f, 0x80}
	state.XMM[5] = [16]byte{20, 1, 236, 255, 30, 226, 1, 2, 0x40, 0x40, 1, 0xff, 1, 1, 1, 1}
	putWords(&state.XMM[6], -32000, -32768, 32000, 32767, 100, -100, 0, 1)
	putWords(&state.XMM[7], 2000, 1, -2000, -1, -200, 200, 1, 2)
	checkRun(t, state.Memory, 0x2b000, []byte{
		0x66, 0x0f, 0xec, 0xc1, // paddsb xmm0, xmm1
		0x66, 0x0f, 0xed, 0xd3, // paddsw xmm2, xmm3
		0x66, 0x0f, 0xe8, 0xe5, // psubsb xmm4, xmm5
		0x66, 0x0f, 0xe9, 0xf7, // psubsw xmm6, xmm7
		0xf4,
	}, state)
	if got, want := state.XMM[0][:], []byte{127, 127, 0x80, 0x80, 40, 20, 1, 3, 0x7f, 0x00, 0x7f, 0x80, 1, 1, 0x7f, 0x81}; string(got) != string(want) {
		t.Fatalf("PADDSB=% x, want % x", got, want)
	}
	for i, want := range []int16{32767, 32767, -32768, -32768, -100, 100, -1, -1} {
		if got := int16(binary.LittleEndian.Uint16(state.XMM[2][i*2:])); got != want {
			t.Fatalf("PADDSW lane %d=%d, want %d", i, got, want)
		}
	}
	if got, want := state.XMM[4][:], []byte{0x80, 0x80, 0x7f, 0x7f, 0xec, 20, 0xff, 0xff, 0, 0x80, 0x7e, 0x82, 0xff, 0xff, 0x7e, 0x80}; string(got) != string(want) {
		t.Fatalf("PSUBSB=% x, want % x", got, want)
	}
	for i, want := range []int16{-32768, -32768, 32767, 32767, 300, -300, -1, -1} {
		if got := int16(binary.LittleEndian.Uint16(state.XMM[6][i*2:])); got != want {
			t.Fatalf("PSUBSW lane %d=%d, want %d", i, got, want)
		}
	}

	memory := NewMemory64()
	const dataAddress Address64 = 0x2c000
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	data := []byte{20, 20, 20, 20, 236, 236, 236, 236, 1, 2, 3, 4, 5, 6, 7, 8}
	if err := memory.Write(dataAddress, data); err != nil {
		t.Fatal(err)
	}
	memoryState := NewMachineState64(memory)
	memoryState.Set(RDI, uint64(dataAddress))
	memoryState.XMM[0] = [16]byte{120, 120, 120, 120, 136, 136, 136, 136, 0, 0, 0, 0, 0, 0, 0, 0}
	checkRun(t, memory, 0x2d000, []byte{
		0x66, 0x0f, 0xec, 0x07, // paddsb xmm0, [rdi]
		0xf4,
	}, memoryState)
	if got, want := memoryState.XMM[0][:8], []byte{127, 127, 127, 127, 0x80, 0x80, 0x80, 0x80}; string(got) != string(want) {
		t.Fatalf("memory PADDSB=% x, want % x", got, want)
	}
}

func TestJIT64PackedMultiplyAccumulateAndQwordArithmetic(t *testing.T) {
	checkRun := func(t *testing.T, memory *Memory64, codeAddress Address64, code []byte, state *MachineState64) {
		t.Helper()
		if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
		if err := memory.Write(codeAddress, code); err != nil {
			t.Fatal(err)
		}
		state.RIP = uint64(codeAddress)
		trap := NewJIT64(memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}
	putWords := func(vector *[16]byte, values ...uint16) {
		t.Helper()
		if len(values) != 8 {
			t.Fatalf("word count=%d, want 8", len(values))
		}
		for i, value := range values {
			binary.LittleEndian.PutUint16(vector[i*2:], value)
		}
	}
	checkWords := func(vector [16]byte, want ...uint16) {
		t.Helper()
		if len(want) != 8 {
			t.Fatalf("want word count=%d, want 8", len(want))
		}
		for i, value := range want {
			if got := binary.LittleEndian.Uint16(vector[i*2:]); got != value {
				t.Fatalf("word lane %d=%#x, want %#x; vector=%x", i, got, value, vector)
			}
		}
	}

	state := NewMachineState64(NewMemory64())
	putWords(&state.XMM[0], 0xffff, 0x8000, 0x1234, 1, 0, 0x7fff, 0xffff, 0x8000)
	putWords(&state.XMM[1], 2, 2, 0x10, 0xffff, 0xffff, 2, 0xffff, 0x8000)
	putWords(&state.XMM[2], 1, 2, 0xfffd, 4, 0x7fff, 0xffff, 0x8000, 0x8000)
	putWords(&state.XMM[3], 10, 20, 30, 0xfffb, 2, 2, 0x8000, 0x8000)
	state.XMM[4] = [16]byte{255, 255, 128, 128, 200, 200, 1, 2, 0, 0, 100, 100, 255, 255, 128, 128}
	state.XMM[5] = [16]byte{127, 127, 127, 127, 127, 127, 0x80, 0x80, 0x80, 0x80, 127, 127, 0xff, 0xff, 0x80, 0x80}
	binary.LittleEndian.PutUint64(state.XMM[6][0:], ^uint64(0))
	binary.LittleEndian.PutUint64(state.XMM[6][8:], 5)
	binary.LittleEndian.PutUint64(state.XMM[7][0:], 1)
	binary.LittleEndian.PutUint64(state.XMM[7][8:], 10)
	checkRun(t, state.Memory, 0x2e000, []byte{
		0x66, 0x0f, 0xe4, 0xc1, // pmulhuw xmm0, xmm1
		0x66, 0x0f, 0xf5, 0xd3, // pmaddwd xmm2, xmm3
		0x66, 0x0f, 0xd4, 0xf7, // paddq xmm6, xmm7
		0x66, 0x0f, 0xfb, 0xf7, // psubq xmm6, xmm7
		0xf4,
	}, state)
	checkWords(state.XMM[0], 1, 1, 1, 0, 0, 0, 0xfffe, 0x4000)
	for i, want := range []uint32{50, 0xffffff92, 0x0000fffc, 0x80000000} {
		if got := binary.LittleEndian.Uint32(state.XMM[2][i*4:]); got != want {
			t.Fatalf("PMADDWD dword lane %d=%#x, want %#x", i, got, want)
		}
	}
	if got := binary.LittleEndian.Uint64(state.XMM[6][0:]); got != ^uint64(0) {
		t.Fatalf("PADDQ/PSUBQ low qword=%#x, want %#x", got, ^uint64(0))
	}
	if got := binary.LittleEndian.Uint64(state.XMM[6][8:]); got != 5 {
		t.Fatalf("PADDQ/PSUBQ high qword=%#x, want %#x", got, uint64(5))
	}

	memory := NewMemory64()
	const dataAddress Address64 = 0x2f000
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 16)
	putWords((*[16]byte)(data), 0xffff, 0x8000, 0, 0, 0, 0, 0, 0)
	if err := memory.Write(dataAddress, data); err != nil {
		t.Fatal(err)
	}
	memoryState := NewMachineState64(memory)
	memoryState.Set(RDI, uint64(dataAddress))
	putWords(&memoryState.XMM[0], 2, 2, 0, 0, 0, 0, 0, 0)
	checkRun(t, memory, 0x30000, []byte{
		0x66, 0x0f, 0xe4, 0x07, // pmulhuw xmm0, [rdi]
		0xf4,
	}, memoryState)
	checkWords(memoryState.XMM[0], 1, 1, 0, 0, 0, 0, 0, 0)
}

func TestJIT64SSSE3HorizontalArithmetic(t *testing.T) {
	checkRun := func(t *testing.T, memory *Memory64, codeAddress Address64, code []byte, state *MachineState64) {
		t.Helper()
		if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
		if err := memory.Write(codeAddress, code); err != nil {
			t.Fatal(err)
		}
		state.RIP = uint64(codeAddress)
		trap := NewJIT64(memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}
	putWords := func(vector *[16]byte, values ...int16) {
		t.Helper()
		if len(values) != 8 {
			t.Fatalf("word count=%d, want 8", len(values))
		}
		for i, value := range values {
			binary.LittleEndian.PutUint16(vector[i*2:], uint16(value))
		}
	}
	checkWords := func(vector [16]byte, want ...int16) {
		t.Helper()
		if len(want) != 8 {
			t.Fatalf("want word count=%d, want 8", len(want))
		}
		for i, value := range want {
			if got := int16(binary.LittleEndian.Uint16(vector[i*2:])); got != value {
				t.Fatalf("word lane %d=%d, want %d; vector=%x", i, got, value, vector)
			}
		}
	}
	putDwords := func(vector *[16]byte, values ...int32) {
		t.Helper()
		if len(values) != 4 {
			t.Fatalf("dword count=%d, want 4", len(values))
		}
		for i, value := range values {
			binary.LittleEndian.PutUint32(vector[i*4:], uint32(value))
		}
	}
	checkDwords := func(vector [16]byte, want ...int32) {
		t.Helper()
		if len(want) != 4 {
			t.Fatalf("want dword count=%d, want 4", len(want))
		}
		for i, value := range want {
			if got := int32(binary.LittleEndian.Uint32(vector[i*4:])); got != value {
				t.Fatalf("dword lane %d=%d, want %d; vector=%x", i, got, value, vector)
			}
		}
	}

	state := NewMachineState64(NewMemory64())
	putWords(&state.XMM[0], 1000, -2000, 30000, 10000, -32768, -1, 100, 200)
	putWords(&state.XMM[1], 1, 2, -1000, -2000, 32767, -1, -100, -200)
	putWords(&state.XMM[2], 30000, 10000, -30000, -10000, 1, 2, 3, 4)
	putWords(&state.XMM[3], 30000, 10000, -2, -3, 32767, 1, -32768, -1)
	putWords(&state.XMM[4], 10, 3, -10, 20, 1000, -2000, -3000, 4000)
	putWords(&state.XMM[5], 1, 4, 20, -10, 100, 50, -100, -200)
	putWords(&state.XMM[6], -32768, 1, 32767, -1, -100, 100, -200, 200)
	putWords(&state.XMM[7], 32767, -1, -32768, 1, 32767, -1, -32768, 1)
	putDwords(&state.XMM[8], 100000, -40000, 0x7fffffff, 1)
	putDwords(&state.XMM[9], 5, 6, -7, -8)
	putDwords(&state.XMM[10], 100, 50, -100, 100)
	putDwords(&state.XMM[11], 7, 10, -200, 300)
	putWords(&state.XMM[12], 16384, -16384, -32768, 1000, 1, -1, 1234, -1234)
	putWords(&state.XMM[13], 16384, 16384, -32768, 1000, 1, -1, -1234, 1234)
	checkRun(t, state.Memory, 0x31000, []byte{
		0x66, 0x0f, 0x38, 0x01, 0xc1, // phaddw xmm0, xmm1
		0x66, 0x0f, 0x38, 0x03, 0xd3, // phaddsw xmm2, xmm3
		0x66, 0x0f, 0x38, 0x05, 0xe5, // phsubw xmm4, xmm5
		0x66, 0x0f, 0x38, 0x07, 0xf7, // phsubsw xmm6, xmm7
		0x66, 0x45, 0x0f, 0x38, 0x02, 0xc1, // phaddd xmm8, xmm9
		0x66, 0x45, 0x0f, 0x38, 0x06, 0xd3, // phsubd xmm10, xmm11
		0x66, 0x0f, 0x38, 0x0b, 0xe5, // pmulhrsw xmm4, xmm5
		0xf4,
	}, state)
	checkWords(state.XMM[0], -1000, -25536, 32767, 300, 3, -3000, 32766, -300)
	checkWords(state.XMM[2], 32767, -32768, 3, 7, 32767, -5, 32767, -32768)
	checkWords(state.XMM[4], 0, 0, 2, 2, 0, 0, 0, -1)
	checkWords(state.XMM[6], -32768, 32767, -200, -400, 32767, -32768, 32767, -32768)
	checkDwords(state.XMM[8], 60000, -2147483648, 11, -15)
	checkDwords(state.XMM[10], 50, -200, -3, -500)

	memory := NewMemory64()
	const dataAddress Address64 = 0x32000
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 16)
	putWords((*[16]byte)(data), 1, 2, 3, 4, 5, 6, 7, 8)
	if err := memory.Write(dataAddress, data); err != nil {
		t.Fatal(err)
	}
	memoryState := NewMachineState64(memory)
	memoryState.Set(RDI, uint64(dataAddress))
	putWords(&memoryState.XMM[0], 10, 20, 30, 40, 50, 60, 70, 80)
	checkRun(t, memory, 0x33000, []byte{
		0x66, 0x0f, 0x38, 0x01, 0x07, // phaddw xmm0, [rdi]
		0xf4,
	}, memoryState)
	checkWords(memoryState.XMM[0], 30, 70, 110, 150, 3, 7, 11, 15)
}

func TestJIT64SSSE3AbsoluteAndSignPacked(t *testing.T) {
	memory := NewMemory64()
	const codeAddress Address64 = 0x34000
	const dataAddress Address64 = 0x35000
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	memoryData := [32]byte{
		0x00, 0x80, 0xff, 0xff, 0x00, 0x00, 0x01, 0x00,
		0xff, 0x7f, 0x2e, 0xfb, 0x34, 0x12, 0x01, 0x80,
		0xff, 0x00, 0x01, 0xfe, 0x02, 0x00, 0xfd, 0x03,
		0xfc, 0x04, 0xfb, 0x05, 0xfa, 0x06, 0xf9, 0x07,
	}
	if err := memory.Write(dataAddress, memoryData[:]); err != nil {
		t.Fatal(err)
	}

	state := NewMachineState64(memory)
	state.Set(RDI, uint64(dataAddress))
	for i := range state.XMM[0] {
		state.XMM[0][i] = 0xa5
		state.XMM[2][i] = 0x5a
		state.XMM[4][i] = 0x3c
		state.XMM[6][i] = 0xc3
		state.XMM[8][i] = 0x96
		state.XMM[10][i] = 0x69
	}
	for i, value := range []int8{-128, 127, -127, -1, 0, 1, -2, -128, 7, -8, 9, -10, 11, -12, 13, -14} {
		state.XMM[1][i] = byte(value)
	}
	putWords := func(vector *[16]byte, values ...int16) {
		t.Helper()
		if len(values) != 8 {
			t.Fatalf("word count=%d, want 8", len(values))
		}
		for i, value := range values {
			binary.LittleEndian.PutUint16(vector[i*2:], uint16(value))
		}
	}
	putDwords := func(vector *[16]byte, values ...int32) {
		t.Helper()
		if len(values) != 4 {
			t.Fatalf("dword count=%d, want 4", len(values))
		}
		for i, value := range values {
			binary.LittleEndian.PutUint32(vector[i*4:], uint32(value))
		}
	}
	putWords(&state.XMM[3], -32768, -1, 0, 1, 32767, -1234, 1234, -32767)
	putDwords(&state.XMM[5], -2147483648, -1, 0, 1)
	putBytes := func(vector *[16]byte, values ...int8) {
		t.Helper()
		if len(values) != 16 {
			t.Fatalf("byte count=%d, want 16", len(values))
		}
		for i, value := range values {
			vector[i] = byte(value)
		}
	}
	putBytes(&state.XMM[6], -128, 1, -2, 3, -4, 5, -6, 7, -8, 9, -10, 11, -12, 13, -14, 15)
	putBytes(&state.XMM[7], -1, 0, 1, -2, 2, 0, -3, 3, -4, 4, -5, 5, -6, 6, -7, 7)
	putWords(&state.XMM[8], -32768, 1, -2, 3, -4, 5, -6, 7)
	putWords(&state.XMM[9], -1, 0, 1, -2, 2, 0, -3, 3)
	putDwords(&state.XMM[10], 0x69696969, 0x69696969, 0x69696969, 0x69696969)
	putDwords(&state.XMM[11], -1, 0, 1, -2)
	putBytes(&state.XMM[13], -1, 2, -3, 4, -5, 6, -7, 8, -9, 10, -11, 12, -13, 14, -15, 16)

	code := []byte{
		0x66, 0x0f, 0x38, 0x1c, 0xc1, // pabsb xmm0, xmm1
		0x66, 0x0f, 0x38, 0x1d, 0xd3, // pabsw xmm2, xmm3
		0x66, 0x0f, 0x38, 0x1e, 0xe5, // pabsd xmm4, xmm5
		0x66, 0x0f, 0x38, 0x08, 0xf7, // psignb xmm6, xmm7
		0x66, 0x45, 0x0f, 0x38, 0x09, 0xc1, // psignw xmm8, xmm9
		0x66, 0x45, 0x0f, 0x38, 0x0a, 0xd3, // psignd xmm10, xmm11
		0x66, 0x44, 0x0f, 0x38, 0x1d, 0x27, // pabsw xmm12, [rdi]
		0x66, 0x44, 0x0f, 0x38, 0x08, 0x6f, 0x10, // psignb xmm13, [rdi+16]
		0xf4,
	}
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state.RIP = uint64(codeAddress)
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}

	wantBytes := [16]byte{0x80, 0x7f, 0x7f, 0x01, 0, 1, 2, 0x80, 7, 8, 9, 10, 11, 12, 13, 14}
	if state.XMM[0] != wantBytes {
		t.Fatalf("pabsb=%x, want %x", state.XMM[0], wantBytes)
	}
	wantWords := [16]byte{}
	for i, value := range []int16{-32768, 1, 0, 1, 32767, 1234, 1234, 32767} {
		binary.LittleEndian.PutUint16(wantWords[i*2:], uint16(value))
	}
	if state.XMM[2] != wantWords {
		t.Fatalf("pabsw=%x, want %x", state.XMM[2], wantWords)
	}
	wantDwords := [16]byte{}
	for i, value := range []int32{-2147483648, 1, 0, 1} {
		binary.LittleEndian.PutUint32(wantDwords[i*4:], uint32(value))
	}
	if state.XMM[4] != wantDwords {
		t.Fatalf("pabsd=%x, want %x", state.XMM[4], wantDwords)
	}
	if got, want := state.XMM[6], [16]byte{0x80, 0, 0xfe, 0xfd, 0xfc, 0, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}; got != want {
		t.Fatalf("psignb=%x, want %x", got, want)
	}
	wantSignWords := [16]byte{}
	for i, value := range []int16{-32768, 0, -2, -3, -4, 0, 6, 7} {
		binary.LittleEndian.PutUint16(wantSignWords[i*2:], uint16(value))
	}
	if state.XMM[8] != wantSignWords {
		t.Fatalf("psignw=%x, want %x", state.XMM[8], wantSignWords)
	}
	wantSignDwords := [16]byte{}
	for i, value := range []int32{-0x69696969, 0, 0x69696969, -0x69696969} {
		binary.LittleEndian.PutUint32(wantSignDwords[i*4:], uint32(value))
	}
	if state.XMM[10] != wantSignDwords {
		t.Fatalf("psignd=%x, want %x", state.XMM[10], wantSignDwords)
	}
	wantMemoryWords := [16]byte{}
	for i, value := range []int16{-32768, 1, 0, 1, 32767, 1234, 4660, 32767} {
		binary.LittleEndian.PutUint16(wantMemoryWords[i*2:], uint16(value))
	}
	if state.XMM[12] != wantMemoryWords {
		t.Fatalf("pabsw memory=%x, want %x", state.XMM[12], wantMemoryWords)
	}
	if got, want := state.XMM[13], [16]byte{1, 0, 0xfd, 0xfc, 0xfb, 0, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}; got != want {
		t.Fatalf("psignb memory=%x, want %x", got, want)
	}
}

func TestJIT64PackedSSEFloatArithmetic(t *testing.T) {
	setFloat32 := func(state *MachineState64, xmm uint8, values ...float32) {
		t.Helper()
		if len(values) != 4 {
			t.Fatalf("float32 lane count=%d, want 4", len(values))
		}
		for lane, value := range values {
			binary.LittleEndian.PutUint32(state.XMM[xmm][lane*4:], math.Float32bits(value))
		}
	}
	setFloat64 := func(state *MachineState64, xmm uint8, values ...float64) {
		t.Helper()
		if len(values) != 2 {
			t.Fatalf("float64 lane count=%d, want 2", len(values))
		}
		for lane, value := range values {
			binary.LittleEndian.PutUint64(state.XMM[xmm][lane*8:], math.Float64bits(value))
		}
	}
	getFloat32 := func(state *MachineState64, xmm uint8) [4]float32 {
		var values [4]float32
		for lane := range values {
			values[lane] = math.Float32frombits(binary.LittleEndian.Uint32(state.XMM[xmm][lane*4:]))
		}
		return values
	}
	getFloat64 := func(state *MachineState64, xmm uint8) [2]float64 {
		var values [2]float64
		for lane := range values {
			values[lane] = math.Float64frombits(binary.LittleEndian.Uint64(state.XMM[xmm][lane*8:]))
		}
		return values
	}
	run := func(t *testing.T, state *MachineState64, code []byte, data []byte) {
		t.Helper()
		const codeAddress Address64 = 0x3a000
		if len(data) != 0 {
			const dataAddress Address64 = 0x3b000
			if err := state.Memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
				t.Fatal(err)
			}
			if err := state.Memory.Write(dataAddress, data); err != nil {
				t.Fatal(err)
			}
			state.Set(RDI, uint64(dataAddress))
		}
		if err := state.Memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
		if err := state.Memory.Write(codeAddress, code); err != nil {
			t.Fatal(err)
		}
		state.RIP = uint64(codeAddress)
		trap := NewJIT64(state.Memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}

	state := NewMachineState64(NewMemory64())
	setFloat32(state, 0, 1, 2, 3, 4)
	setFloat32(state, 1, 10, 20, 30, 40)
	setFloat32(state, 2, 20, 18, 16, 14)
	setFloat32(state, 3, 1, 2, 3, 4)
	setFloat32(state, 4, 2, 3, 4, 5)
	setFloat32(state, 5, 10, 20, 30, 40)
	setFloat32(state, 6, 20, 60, 120, 200)
	setFloat32(state, 7, 2, 3, 4, 5)
	setFloat32(state, 10, 1, 4, 9, 16)
	setFloat64(state, 11, 1.5, -2.5)
	setFloat64(state, 12, 2.5, 0.5)
	setFloat64(state, 13, 20, 18)
	setFloat64(state, 14, 2, 3)
	setFloat64(state, 15, 2, 3)
	setFloat64(state, 8, 20, 60)
	setFloat64(state, 9, 2, 3)
	run(t, state, []byte{
		0x0f, 0x58, 0xc1, // addps xmm0, xmm1
		0x0f, 0x5c, 0xd3, // subps xmm2, xmm3
		0x0f, 0x59, 0xe5, // mulps xmm4, xmm5
		0x0f, 0x5e, 0xf7, // divps xmm6, xmm7
		0x45, 0x0f, 0x51, 0xd2, // sqrtps xmm10, xmm10
		0x66, 0x45, 0x0f, 0x5e, 0xc1, // divpd xmm8, xmm9
		0x66, 0x44, 0x0f, 0x51, 0x0f, // sqrtpd xmm9, [rdi]
		0xf4,
	}, []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22, 0x40, // 0, 9
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x30, 0x40, // 0, 16
	})
	if got, want := getFloat32(state, 0), [4]float32{11, 22, 33, 44}; got != want {
		t.Fatalf("addps=%v, want %v", got, want)
	}
	if got, want := getFloat32(state, 2), [4]float32{19, 16, 13, 10}; got != want {
		t.Fatalf("subps=%v, want %v", got, want)
	}
	if got, want := getFloat32(state, 4), [4]float32{20, 60, 120, 200}; got != want {
		t.Fatalf("mulps=%v, want %v", got, want)
	}
	if got, want := getFloat32(state, 6), [4]float32{10, 20, 30, 40}; got != want {
		t.Fatalf("divps=%v, want %v", got, want)
	}
	if got, want := getFloat32(state, 10), [4]float32{1, 2, 3, 4}; got != want {
		t.Fatalf("sqrtps=%v, want %v", got, want)
	}
	if got := getFloat64(state, 8); got != [2]float64{10, 20} {
		t.Fatalf("divpd=%v, want [10 20]", got)
	}
	if got := getFloat64(state, 9); got != [2]float64{3, 4} {
		t.Fatalf("sqrtpd memory=%v, want [3 4]", got)
	}

	state = NewMachineState64(NewMemory64())
	setFloat64(state, 2, 1.5, -2.5)
	setFloat64(state, 3, 2.5, 0.5)
	setFloat64(state, 4, 20, 18)
	setFloat64(state, 5, 2, 3)
	setFloat64(state, 6, 2, 3)
	setFloat64(state, 7, 2, 3)
	run(t, state, []byte{
		0x66, 0x0f, 0x58, 0xd3, // addpd xmm2, xmm3
		0x66, 0x0f, 0x5c, 0xe7, // subpd xmm4, xmm7
		0x66, 0x0f, 0x59, 0xee, // mulpd xmm5, xmm6
		0xf4,
	}, nil)
	if got, want := getFloat64(state, 2), [2]float64{4, -2}; got != want {
		t.Fatalf("addpd=%v, want %v", got, want)
	}
	if got, want := getFloat64(state, 4), [2]float64{18, 15}; got != want {
		t.Fatalf("subpd=%v, want %v", got, want)
	}
	if got, want := getFloat64(state, 5), [2]float64{4, 9}; got != want {
		t.Fatalf("mulpd=%v, want %v", got, want)
	}

	state = NewMachineState64(NewMemory64())
	setFloat32(state, 0, float32(math.Inf(1)), float32(math.Inf(-1)), float32(math.NaN()), 1)
	setFloat32(state, 1, 2, 2, 2, 0)
	run(t, state, []byte{0x0f, 0x5e, 0xc1, 0xf4}, nil) // divps xmm0, xmm1
	got := getFloat32(state, 0)
	if !math.IsInf(float64(got[0]), 1) || !math.IsInf(float64(got[1]), -1) || !math.IsNaN(float64(got[2])) || !math.IsInf(float64(got[3]), 1) {
		t.Fatalf("divps special values=%v, want [+Inf -Inf NaN +Inf]", got)
	}

	state = NewMachineState64(NewMemory64())
	setFloat32(state, 0, 10, 20, 30, 40)
	run(t, state, []byte{0x0f, 0x58, 0x07, 0xf4}, []byte{
		0, 0, 128, 63, // 1
		0, 0, 0, 64, // 2
		0, 0, 64, 64, // 3
		0, 0, 128, 64, // 4
	}) // addps xmm0, [rdi]
	if got, want := getFloat32(state, 0), [4]float32{11, 22, 33, 44}; got != want {
		t.Fatalf("addps memory=%v, want %v", got, want)
	}
}

func TestJIT64PackedSSEFloatMinMaxAndDuplicate(t *testing.T) {
	setFloat32Raw := func(state *MachineState64, xmm uint8, values ...uint32) {
		t.Helper()
		if len(values) != 4 {
			t.Fatalf("float32 raw lane count=%d, want 4", len(values))
		}
		for lane, value := range values {
			binary.LittleEndian.PutUint32(state.XMM[xmm][lane*4:], value)
		}
	}
	setFloat64Raw := func(state *MachineState64, xmm uint8, values ...uint64) {
		t.Helper()
		if len(values) != 2 {
			t.Fatalf("float64 raw lane count=%d, want 2", len(values))
		}
		for lane, value := range values {
			binary.LittleEndian.PutUint64(state.XMM[xmm][lane*8:], value)
		}
	}
	read32 := func(state *MachineState64, xmm uint8) [4]uint32 {
		var values [4]uint32
		for lane := range values {
			values[lane] = binary.LittleEndian.Uint32(state.XMM[xmm][lane*4:])
		}
		return values
	}
	read64 := func(state *MachineState64, xmm uint8) [2]uint64 {
		var values [2]uint64
		for lane := range values {
			values[lane] = binary.LittleEndian.Uint64(state.XMM[xmm][lane*8:])
		}
		return values
	}
	run := func(t *testing.T, state *MachineState64, code []byte, data []byte) {
		t.Helper()
		const codeAddress Address64 = 0x3e000
		if len(data) != 0 {
			const dataAddress Address64 = 0x3f000
			if err := state.Memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
				t.Fatal(err)
			}
			if err := state.Memory.Write(dataAddress, data); err != nil {
				t.Fatal(err)
			}
			state.Set(RDI, uint64(dataAddress))
		}
		if err := state.Memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
		if err := state.Memory.Write(codeAddress, code); err != nil {
			t.Fatal(err)
		}
		state.RIP = uint64(codeAddress)
		trap := NewJIT64(state.Memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
	}

	const (
		positiveZero32 = uint32(0x00000000)
		negativeZero32 = uint32(0x80000000)
		positiveZero64 = uint64(0x0000000000000000)
		negativeZero64 = uint64(0x8000000000000000)
		nan32          = uint32(0x7fc01234)
		nan64          = uint64(0x7ff8000000001234)
	)
	state := NewMachineState64(NewMemory64())
	setFloat32Raw(state, 0, math.Float32bits(1), math.Float32bits(-2), positiveZero32, nan32)
	setFloat32Raw(state, 1, math.Float32bits(2), math.Float32bits(-3), negativeZero32, math.Float32bits(4))
	setFloat32Raw(state, 2, math.Float32bits(1), math.Float32bits(-2), positiveZero32, nan32)
	setFloat32Raw(state, 3, math.Float32bits(2), math.Float32bits(-3), negativeZero32, math.Float32bits(4))
	setFloat64Raw(state, 4, nan64, negativeZero64)
	setFloat64Raw(state, 5, math.Float64bits(3), positiveZero64)
	setFloat64Raw(state, 6, nan64, negativeZero64)
	setFloat64Raw(state, 7, math.Float64bits(3), positiveZero64)
	setFloat32Raw(state, 8, math.Float32bits(10), math.Float32bits(20), math.Float32bits(30), math.Float32bits(40))
	setFloat32Raw(state, 9, math.Float32bits(9), math.Float32bits(8), math.Float32bits(7), math.Float32bits(6))
	setFloat32Raw(state, 13, math.Float32bits(10), math.Float32bits(20), math.Float32bits(30), math.Float32bits(40))
	run(t, state, []byte{
		0x0f, 0x5d, 0xc1, // minps xmm0, xmm1
		0x0f, 0x5f, 0xd3, // maxps xmm2, xmm3
		0x66, 0x0f, 0x5d, 0xe5, // minpd xmm4, xmm5
		0x66, 0x0f, 0x5f, 0xf7, // maxpd xmm6, xmm7
		0x44, 0x0f, 0x5d, 0x2f, // minps xmm13, [rdi]
		0xf3, 0x45, 0x0f, 0x12, 0xc1, // movsldup xmm8, xmm9
		0xf3, 0x45, 0x0f, 0x16, 0xd1, // movshdup xmm10, xmm9
		0xf3, 0x44, 0x0f, 0x12, 0x1f, // movsldup xmm11, [rdi]
		0xf3, 0x44, 0x0f, 0x16, 0x27, // movshdup xmm12, [rdi]
		0xf4,
	}, []byte{
		0, 0, 16, 65, // 9
		0, 0, 0, 65, // 8
		0, 0, 224, 64, // 7
		0, 0, 192, 64, // 6
	})
	if got, want := read32(state, 0), [4]uint32{math.Float32bits(1), math.Float32bits(-3), negativeZero32, math.Float32bits(4)}; got != want {
		t.Fatalf("minps=%#v, want %#v", got, want)
	}
	if got, want := read32(state, 2), [4]uint32{math.Float32bits(2), math.Float32bits(-2), negativeZero32, math.Float32bits(4)}; got != want {
		t.Fatalf("maxps=%#v, want %#v", got, want)
	}
	if got, want := read64(state, 4), [2]uint64{math.Float64bits(3), positiveZero64}; got != want {
		t.Fatalf("minpd=%#v, want %#v", got, want)
	}
	if got, want := read64(state, 6), [2]uint64{math.Float64bits(3), positiveZero64}; got != want {
		t.Fatalf("maxpd=%#v, want %#v", got, want)
	}
	if got, want := read32(state, 13), [4]uint32{math.Float32bits(9), math.Float32bits(8), math.Float32bits(7), math.Float32bits(6)}; got != want {
		t.Fatalf("minps memory=%#v, want %#v", got, want)
	}
	if got, want := read32(state, 8), [4]uint32{math.Float32bits(9), math.Float32bits(9), math.Float32bits(7), math.Float32bits(7)}; got != want {
		t.Fatalf("movsldup xmm8=%#v, want %#v", got, want)
	}
	if got, want := read32(state, 10), [4]uint32{math.Float32bits(8), math.Float32bits(8), math.Float32bits(6), math.Float32bits(6)}; got != want {
		t.Fatalf("movshdup xmm10=%#v, want %#v", got, want)
	}
	if got, want := read32(state, 11), [4]uint32{math.Float32bits(9), math.Float32bits(9), math.Float32bits(7), math.Float32bits(7)}; got != want {
		t.Fatalf("movsldup memory=%#v, want %#v", got, want)
	}
	if got, want := read32(state, 12), [4]uint32{math.Float32bits(8), math.Float32bits(8), math.Float32bits(6), math.Float32bits(6)}; got != want {
		t.Fatalf("movshdup memory=%#v, want %#v", got, want)
	}
}

func TestJIT64SIMDNonTemporalAndMaskedStores(t *testing.T) {
	const (
		codeAddress Address64 = 0x42000
		dataAddress Address64 = 0x43000
	)
	memory := NewMemory64()
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	initial := make([]byte, 64)
	for i := range initial {
		initial[i] = 0xcc
	}
	if err := memory.Write(dataAddress, initial); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	code := []byte{
		0x66, 0x0f, 0xe7, 0x47, 0x03, // movntdq [rdi+3], xmm0
		0x0f, 0x2b, 0x47, 0x13, // movntps [rdi+19], xmm0
		0x66, 0x0f, 0xf7, 0xc1, // maskmovdqu xmm0, xmm1
		0x66, 0x45, 0x0f, 0xf7, 0xc8, // maskmovdqu xmm9, xmm8
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress+5))
	for lane := 0; lane < 16; lane++ {
		state.XMM[0][lane] = byte(lane + 1)
		state.XMM[9][lane] = byte(0xa0 + lane)
	}
	for lane, mask := range []byte{
		0x80, 0x00, 0xff, 0x7f, 0x80, 0x01, 0x80, 0x00,
		0x00, 0x80, 0x00, 0x00, 0x80, 0x01, 0x00, 0x80,
	} {
		state.XMM[1][lane] = mask
		state.XMM[8][lane] = mask
	}
	originalRDI := state.Get(RDI)
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.Get(RDI) != originalRDI {
		t.Fatalf("MASKMOVDQU changed RDI from %#x to %#x", originalRDI, state.Get(RDI))
	}
	var got [64]byte
	if err := memory.Read(dataAddress, got[:]); err != nil {
		t.Fatal(err)
	}
	want := initial
	copy(want[8:24], state.XMM[0][:])
	copy(want[24:40], state.XMM[0][:])
	for lane, mask := range state.XMM[1] {
		if mask&0x80 != 0 {
			want[5+lane] = state.XMM[0][lane]
		}
	}
	for lane, mask := range state.XMM[8] {
		if mask&0x80 != 0 {
			want[5+lane] = state.XMM[9][lane]
		}
	}
	if got != [64]byte(want) {
		t.Fatalf("masked/non-temporal stores=%x, want %x", got, want)
	}

	memory = NewMemory64()
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	zeroInitial := make([]byte, 16)
	for i := range zeroInitial {
		zeroInitial[i] = 0xab
	}
	if err := memory.Write(dataAddress, zeroInitial); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(codeAddress, []byte{0x66, 0x0f, 0xf7, 0xc1, 0xf4}); err != nil { // maskmovdqu xmm0, xmm1
		t.Fatal(err)
	}
	state = NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	for lane := range state.XMM[1] {
		state.XMM[0][lane] = byte(lane + 1)
		state.XMM[1][lane] = 0
	}
	trap = NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("zero-mask trap=%#x halted=%v", trap, state.Halted)
	}
	var untouched [16]byte
	if err := memory.Read(dataAddress, untouched[:]); err != nil {
		t.Fatal(err)
	}
	for lane, value := range untouched {
		if value != 0xab {
			t.Fatalf("zero-mask changed byte %d to %#x", lane, value)
		}
	}
}

func TestJIT64PCMPEQQ(t *testing.T) {
	const (
		codeAddress Address64 = 0x45000
		dataAddress Address64 = 0x46000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	var data [16]byte
	binary.LittleEndian.PutUint64(data[0:], 0x1122334455667788)
	binary.LittleEndian.PutUint64(data[8:], 0xdeadbeefcafebabe)
	if err := memory.Write(dataAddress, data[:]); err != nil {
		t.Fatal(err)
	}
	code := []byte{
		0x66, 0x0f, 0x38, 0x29, 0xc1, // pcmpeqq xmm0, xmm1
		0x66, 0x44, 0x0f, 0x38, 0x29, 0xca, // pcmpeqq xmm9, xmm2
		0x66, 0x0f, 0x38, 0x29, 0x1f, // pcmpeqq xmm3, [rdi]
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	binary.LittleEndian.PutUint64(state.XMM[0][0:], 0x1122334455667788)
	binary.LittleEndian.PutUint64(state.XMM[0][8:], 0x0102030405060708)
	binary.LittleEndian.PutUint64(state.XMM[1][0:], 0x1122334455667788)
	binary.LittleEndian.PutUint64(state.XMM[1][8:], 0x1112131415161718)
	binary.LittleEndian.PutUint64(state.XMM[9][0:], 0xaabbccddeeff0011)
	binary.LittleEndian.PutUint64(state.XMM[9][8:], 0x8899aabbccddeeff)
	binary.LittleEndian.PutUint64(state.XMM[2][0:], 0xaabbccddeeff0011)
	binary.LittleEndian.PutUint64(state.XMM[2][8:], 0x8899aabbccddeeee)
	binary.LittleEndian.PutUint64(state.XMM[3][0:], 0x1122334455667788)
	binary.LittleEndian.PutUint64(state.XMM[3][8:], 2)
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	allOnes := uint64(0xffffffffffffffff)
	if got := binary.LittleEndian.Uint64(state.XMM[0][0:]); got != allOnes {
		t.Fatalf("xmm0 lane0=%#x, want all ones", got)
	}
	if got := binary.LittleEndian.Uint64(state.XMM[0][8:]); got != 0 {
		t.Fatalf("xmm0 lane1=%#x, want zero", got)
	}
	if got := binary.LittleEndian.Uint64(state.XMM[9][0:]); got != allOnes {
		t.Fatalf("xmm9 lane0=%#x, want all ones", got)
	}
	if got := binary.LittleEndian.Uint64(state.XMM[9][8:]); got != 0 {
		t.Fatalf("xmm9 lane1=%#x, want zero", got)
	}
	if got := binary.LittleEndian.Uint64(state.XMM[3][0:]); got != allOnes {
		t.Fatalf("xmm3 memory lane0=%#x, want all ones", got)
	}
	if got := binary.LittleEndian.Uint64(state.XMM[3][8:]); got != 0 {
		t.Fatalf("xmm3 memory lane1=%#x, want zero", got)
	}
}

func TestJIT64PackedWideningMoves(t *testing.T) {
	const (
		codeAddress Address64 = 0x48000
		dataAddress Address64 = 0x49000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	memorySource := []byte{0x80, 0x7f, 0x01, 0xff, 0x34, 0x12, 0xfe, 0x7f}
	if err := memory.Write(dataAddress, memorySource); err != nil {
		t.Fatal(err)
	}
	code := []byte{
		0x66, 0x0f, 0x38, 0x20, 0xe0, // pmovsxbw xmm4, xmm0
		0x66, 0x0f, 0x38, 0x21, 0xe9, // pmovsxbd xmm5, xmm1
		0x66, 0x0f, 0x38, 0x22, 0xf2, // pmovsxbq xmm6, xmm2
		0x66, 0x0f, 0x38, 0x23, 0xfb, // pmovsxwd xmm7, xmm3
		0x66, 0x44, 0x0f, 0x38, 0x24, 0xc0, // pmovsxwq xmm8, xmm0
		0x66, 0x44, 0x0f, 0x38, 0x25, 0xc9, // pmovsxdq xmm9, xmm1
		0x66, 0x44, 0x0f, 0x38, 0x30, 0x17, // pmovzxbw xmm10, [rdi]
		0x66, 0x44, 0x0f, 0x38, 0x31, 0x1f, // pmovzxbd xmm11, [rdi]
		0x66, 0x44, 0x0f, 0x38, 0x32, 0x27, // pmovzxbq xmm12, [rdi]
		0x66, 0x44, 0x0f, 0x38, 0x33, 0x2f, // pmovzxwd xmm13, [rdi]
		0x66, 0x44, 0x0f, 0x38, 0x34, 0x37, // pmovzxwq xmm14, [rdi]
		0x66, 0x44, 0x0f, 0x38, 0x35, 0x3f, // pmovzxdq xmm15, [rdi]
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	copy(state.XMM[0][:], []byte{0x80, 0x7f, 0x01, 0xff, 0x34, 0x12, 0xfe, 0x7f})
	copy(state.XMM[1][:], []byte{0x01, 0x00, 0x00, 0x80, 0xff, 0xff, 0xff, 0x7f})
	copy(state.XMM[2][:], []byte{0x80, 0xff})
	copy(state.XMM[3][:], []byte{0x80, 0x80, 0xff, 0x7f, 0x34, 0x12, 0x00, 0x80})
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	putWords := func(values ...uint16) [16]byte {
		var result [16]byte
		for lane, value := range values {
			binary.LittleEndian.PutUint16(result[lane*2:], value)
		}
		return result
	}
	putDwords := func(values ...uint32) [16]byte {
		var result [16]byte
		for lane, value := range values {
			binary.LittleEndian.PutUint32(result[lane*4:], value)
		}
		return result
	}
	putQwords := func(values ...uint64) [16]byte {
		var result [16]byte
		for lane, value := range values {
			binary.LittleEndian.PutUint64(result[lane*8:], value)
		}
		return result
	}
	assertXMM := func(xmm uint8, expected [16]byte) {
		t.Helper()
		if state.XMM[xmm] != expected {
			t.Fatalf("xmm%d=%x, want %x", xmm, state.XMM[xmm], expected)
		}
	}
	assertXMM(4, putWords(0xff80, 0x007f, 0x0001, 0xffff, 0x0034, 0x0012, 0xfffe, 0x007f))
	assertXMM(5, putDwords(0x00000001, 0x00000000, 0x00000000, 0xffffff80))
	assertXMM(6, putQwords(0xffffffffffffff80, 0xffffffffffffffff))
	assertXMM(7, putDwords(0xffff8080, 0x00007fff, 0x00001234, 0xffff8000))
	assertXMM(8, putQwords(0x0000000000007f80, 0xffffffffffffff01))
	assertXMM(9, putQwords(0xffffffff80000001, 0x000000007fffffff))
	assertXMM(10, putWords(0x0080, 0x007f, 0x0001, 0x00ff, 0x0034, 0x0012, 0x00fe, 0x007f))
	assertXMM(11, putDwords(0x00000080, 0x0000007f, 0x00000001, 0x000000ff))
	assertXMM(12, putQwords(0x0000000000000080, 0x000000000000007f))
	assertXMM(13, putDwords(0x00007f80, 0x0000ff01, 0x00001234, 0x00007ffe))
	assertXMM(14, putQwords(0x0000000000007f80, 0x000000000000ff01))
	assertXMM(15, putQwords(0x00000000ff017f80, 0x000000007ffe1234))
}

func TestJIT64PackedRounding(t *testing.T) {
	const (
		codeAddress Address64 = 0x4a000
		dataAddress Address64 = 0x4b000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	memorySource := make([]byte, 16)
	for lane, value := range []float32{-1.5, 2.5, 3.5, -3.5} {
		binary.LittleEndian.PutUint32(memorySource[lane*4:], math.Float32bits(value))
	}
	if err := memory.Write(dataAddress, memorySource); err != nil {
		t.Fatal(err)
	}
	code := []byte{
		0x66, 0x0f, 0x3a, 0x08, 0xc1, 0x00, // roundps xmm0, xmm1, 0 (nearest-even)
		0x66, 0x0f, 0x3a, 0x09, 0xca, 0x01, // roundpd xmm1, xmm2, 1 (floor)
		0x66, 0x0f, 0x3a, 0x0a, 0xdb, 0x02, // roundss xmm3, xmm3, 2 (ceil)
		0x66, 0x0f, 0x3a, 0x0b, 0xe4, 0x03, // roundsd xmm4, xmm4, 3 (truncate)
		0x66, 0x0f, 0x3a, 0x08, 0x2f, 0x00, // roundps xmm5, [rdi], 0
		0x66, 0x44, 0x0f, 0x3a, 0x09, 0xf6, 0x01, // roundpd xmm14, xmm6, 1
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	putFloat32 := func(destination *[16]byte, values ...float32) {
		for lane, value := range values {
			binary.LittleEndian.PutUint32(destination[lane*4:], math.Float32bits(value))
		}
	}
	putFloat64 := func(destination *[16]byte, values ...float64) {
		for lane, value := range values {
			binary.LittleEndian.PutUint64(destination[lane*8:], math.Float64bits(value))
		}
	}
	putFloat32(&state.XMM[1], 1.5, -1.5, 2.5, -2.5)
	putFloat64(&state.XMM[2], 1.5, -1.5)
	putFloat32(&state.XMM[3], 1.5, 101.25, -202.5, 303.75)
	putFloat64(&state.XMM[4], -1.5, 404.5)
	putFloat64(&state.XMM[6], 1.5, -2.5)
	for lane := range state.XMM[14] {
		state.XMM[14][lane] = 0xa5
	}
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	assertFloat32 := func(xmm uint8, values ...float32) {
		t.Helper()
		var expected [16]byte
		putFloat32(&expected, values...)
		if state.XMM[xmm] != expected {
			t.Fatalf("xmm%d=%x, want %x", xmm, state.XMM[xmm], expected)
		}
	}
	assertFloat64 := func(xmm uint8, values ...float64) {
		t.Helper()
		var expected [16]byte
		putFloat64(&expected, values...)
		if state.XMM[xmm] != expected {
			t.Fatalf("xmm%d=%x, want %x", xmm, state.XMM[xmm], expected)
		}
	}
	assertFloat32(0, 2, -2, 2, -2)
	assertFloat64(1, 1, -2)
	if got := math.Float32frombits(binary.LittleEndian.Uint32(state.XMM[3][:4])); got != 2 {
		t.Fatalf("roundss low=%v, want 2", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(state.XMM[3][4:])); got != 101.25 {
		t.Fatalf("roundss upper lane1=%v, want 101.25", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(state.XMM[3][8:])); got != -202.5 {
		t.Fatalf("roundss upper lane2=%v, want -202.5", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(state.XMM[3][12:])); got != 303.75 {
		t.Fatalf("roundss upper lane3=%v, want 303.75", got)
	}
	if got := math.Float64frombits(binary.LittleEndian.Uint64(state.XMM[4][:8])); got != -1 {
		t.Fatalf("roundsd low=%v, want -1", got)
	}
	if got := math.Float64frombits(binary.LittleEndian.Uint64(state.XMM[4][8:])); got != 404.5 {
		t.Fatalf("roundsd upper lane=%v, want 404.5", got)
	}
	assertFloat32(5, -2, 2, 4, -4)
	assertFloat64(14, 1, -3)
}

func TestJIT64PackedBlend(t *testing.T) {
	const (
		codeAddress Address64 = 0x4c000
		dataAddress Address64 = 0x4d000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	memorySource := make([]byte, 16)
	for lane, value := range []float32{100, 200, 300, 400} {
		binary.LittleEndian.PutUint32(memorySource[lane*4:], math.Float32bits(value))
	}
	if err := memory.Write(dataAddress, memorySource); err != nil {
		t.Fatal(err)
	}
	code := []byte{
		0x66, 0x0f, 0x3a, 0x0c, 0xca, 0x0a, // blendps xmm1, xmm2, 0b1010
		0x66, 0x0f, 0x3a, 0x0d, 0xdc, 0x01, // blendpd xmm3, xmm4, 0b01
		0x66, 0x0f, 0x38, 0x14, 0x2f, // blendvps xmm5, [rdi], xmm0
		0x66, 0x44, 0x0f, 0x38, 0x15, 0xf6, // blendvpd xmm14, xmm6, xmm0
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	putFloat32 := func(destination *[16]byte, values ...float32) {
		for lane, value := range values {
			binary.LittleEndian.PutUint32(destination[lane*4:], math.Float32bits(value))
		}
	}
	putFloat64 := func(destination *[16]byte, values ...float64) {
		for lane, value := range values {
			binary.LittleEndian.PutUint64(destination[lane*8:], math.Float64bits(value))
		}
	}
	putFloat32(&state.XMM[0], 0, math.Float32frombits(0x80000000), 0, math.Float32frombits(0x80000000))
	putFloat32(&state.XMM[1], 10, 20, 30, 40)
	putFloat32(&state.XMM[2], 1, 2, 3, 4)
	putFloat64(&state.XMM[3], 10, 20)
	putFloat64(&state.XMM[4], 1, 2)
	putFloat32(&state.XMM[5], 5, 6, 7, 8)
	putFloat64(&state.XMM[6], 60, 70)
	for lane := range state.XMM[14] {
		state.XMM[14][lane] = 0xa5
	}
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	assertFloat32 := func(xmm uint8, values ...float32) {
		t.Helper()
		var expected [16]byte
		putFloat32(&expected, values...)
		if state.XMM[xmm] != expected {
			t.Fatalf("xmm%d=%x, want %x", xmm, state.XMM[xmm], expected)
		}
	}
	assertFloat64 := func(xmm uint8, values ...float64) {
		t.Helper()
		var expected [16]byte
		putFloat64(&expected, values...)
		if state.XMM[xmm] != expected {
			t.Fatalf("xmm%d=%x, want %x", xmm, state.XMM[xmm], expected)
		}
	}
	assertFloat32(1, 10, 2, 30, 4)
	assertFloat64(3, 1, 20)
	assertFloat32(5, 5, 200, 7, 400)
	assertFloat64(14, 60, 70)
}

func TestJIT64DotProducts(t *testing.T) {
	const (
		codeAddress Address64 = 0x4e000
		dataAddress Address64 = 0x4f000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	memorySource := make([]byte, 16)
	for lane, value := range []float32{10, 20, 30, 40} {
		binary.LittleEndian.PutUint32(memorySource[lane*4:], math.Float32bits(value))
	}
	if err := memory.Write(dataAddress, memorySource); err != nil {
		t.Fatal(err)
	}
	code := []byte{
		0x66, 0x0f, 0x3a, 0x40, 0xca, 0xf1, // dpps xmm1, xmm2, 0xf1
		0x66, 0x0f, 0x3a, 0x41, 0xdc, 0x33, // dppd xmm3, xmm4, 0x33
		0x66, 0x0f, 0x3a, 0x40, 0x2f, 0xa5, // dpps xmm5, [rdi], 0xa5
		0x66, 0x44, 0x0f, 0x3a, 0x41, 0xf6, 0x21, // dppd xmm14, xmm6, 0x21
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	putFloat32 := func(destination *[16]byte, values ...float32) {
		for lane, value := range values {
			binary.LittleEndian.PutUint32(destination[lane*4:], math.Float32bits(value))
		}
	}
	putFloat64 := func(destination *[16]byte, values ...float64) {
		for lane, value := range values {
			binary.LittleEndian.PutUint64(destination[lane*8:], math.Float64bits(value))
		}
	}
	putFloat32(&state.XMM[1], 1, 2, 3, 4)
	putFloat32(&state.XMM[2], 10, 20, 30, 40)
	putFloat64(&state.XMM[3], 1.5, -2)
	putFloat64(&state.XMM[4], 2, 3)
	putFloat32(&state.XMM[5], 1, 2, 3, 4)
	putFloat64(&state.XMM[6], 2, 3)
	putFloat64(&state.XMM[14], 5, 7)
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	assertFloat32 := func(xmm uint8, values ...float32) {
		t.Helper()
		var expected [16]byte
		putFloat32(&expected, values...)
		if state.XMM[xmm] != expected {
			t.Fatalf("xmm%d=%x, want %x", xmm, state.XMM[xmm], expected)
		}
	}
	assertFloat64 := func(xmm uint8, values ...float64) {
		t.Helper()
		var expected [16]byte
		putFloat64(&expected, values...)
		if state.XMM[xmm] != expected {
			t.Fatalf("xmm%d=%x, want %x", xmm, state.XMM[xmm], expected)
		}
	}
	assertFloat32(1, 300, 0, 0, 0)
	assertFloat64(3, -3, -3)
	assertFloat32(5, 200, 0, 200, 0)
	assertFloat64(14, 21, 0)
}

func TestJIT64PHMINPOSUW(t *testing.T) {
	const (
		codeAddress Address64 = 0x50000
		dataAddress Address64 = 0x51000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	memorySource := make([]byte, 16)
	memoryValues := []uint16{5, 3, 3, 8, 0xffff, 9, 10, 11}
	for lane, value := range memoryValues {
		binary.LittleEndian.PutUint16(memorySource[lane*2:], value)
	}
	if err := memory.Write(dataAddress, memorySource); err != nil {
		t.Fatal(err)
	}
	code := []byte{
		0x66, 0x0f, 0x38, 0x41, 0xca, // phminposuw xmm1, xmm2
		0x66, 0x0f, 0x38, 0x41, 0x1f, // phminposuw xmm3, [rdi]
		0x66, 0x44, 0x0f, 0x38, 0x41, 0xf6, // phminposuw xmm14, xmm6
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	for lane := range state.XMM[1] {
		state.XMM[1][lane] = 0xa5
		state.XMM[3][lane] = 0x5a
		state.XMM[14][lane] = 0x3c
	}
	putWords := func(destination *[16]byte, values ...uint16) {
		for lane, value := range values {
			binary.LittleEndian.PutUint16(destination[lane*2:], value)
		}
	}
	putWords(&state.XMM[2], 9, 8, 7, 6, 5, 4, 3, 3)
	putWords(&state.XMM[6], 0xffff, 0x8000, 0x1234, 0x0100, 0x0200, 0x0100, 0x0000, 0xffff)
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	var expectedRegister [16]byte
	binary.LittleEndian.PutUint16(expectedRegister[:2], 3)
	binary.LittleEndian.PutUint16(expectedRegister[2:4], 6)
	if state.XMM[1] != expectedRegister {
		t.Fatalf("xmm1=%x, want %x", state.XMM[1], expectedRegister)
	}
	var expectedMemory [16]byte
	binary.LittleEndian.PutUint16(expectedMemory[:2], 3)
	binary.LittleEndian.PutUint16(expectedMemory[2:4], 1)
	if state.XMM[3] != expectedMemory {
		t.Fatalf("xmm3=%x, want %x", state.XMM[3], expectedMemory)
	}
	var expectedHigh [16]byte
	binary.LittleEndian.PutUint16(expectedHigh[:2], 0)
	binary.LittleEndian.PutUint16(expectedHigh[2:4], 6)
	if state.XMM[14] != expectedHigh {
		t.Fatalf("xmm14=%x, want %x", state.XMM[14], expectedHigh)
	}
}

func TestJIT64SSE41IntegerMinMax(t *testing.T) {
	const (
		codeAddress Address64 = 0x52000
		dataAddress Address64 = 0x53000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	memorySource := make([]byte, 16)
	for lane, value := range []uint32{1, 2, 0x7fffffff, 4} {
		binary.LittleEndian.PutUint32(memorySource[lane*4:], value)
	}
	if err := memory.Write(dataAddress, memorySource); err != nil {
		t.Fatal(err)
	}
	code := []byte{
		0x66, 0x0f, 0x38, 0x3a, 0xca, // pminuw xmm1, xmm2
		0x66, 0x0f, 0x38, 0x3b, 0xdc, // pminud xmm3, xmm4
		0x66, 0x0f, 0x38, 0x3e, 0xee, // pmaxuw xmm5, xmm6
		0x66, 0x41, 0x0f, 0x38, 0x3c, 0xf8, // pmaxsb xmm7, xmm8
		0x66, 0x45, 0x0f, 0x38, 0x3d, 0xca, // pmaxsd xmm9, xmm10
		0x66, 0x45, 0x0f, 0x38, 0x39, 0xdc, // pminsd xmm11, xmm12
		0x66, 0x44, 0x0f, 0x38, 0x3f, 0x2f, // pmaxud xmm13, [rdi]
		0x66, 0x45, 0x0f, 0x38, 0x38, 0xf7, // pminsb xmm14, xmm15
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	putWords := func(destination *[16]byte, values ...uint16) {
		for lane, value := range values {
			binary.LittleEndian.PutUint16(destination[lane*2:], value)
		}
	}
	putDwords := func(destination *[16]byte, values ...uint32) {
		for lane, value := range values {
			binary.LittleEndian.PutUint32(destination[lane*4:], value)
		}
	}
	putWords(&state.XMM[1], 0xffff, 0, 0x8000, 1, 2, 3, 4, 5)
	putWords(&state.XMM[2], 0, 1, 0x7fff, 2, 1, 4, 3, 6)
	putDwords(&state.XMM[3], 0xffffffff, 0, 0x80000000, 7)
	putDwords(&state.XMM[4], 1, 2, 0x7fffffff, 8)
	putWords(&state.XMM[5], 0, 0xffff, 0x8000, 1, 2, 3, 4, 5)
	putWords(&state.XMM[6], 1, 0, 0x7fff, 2, 1, 4, 3, 6)
	copy(state.XMM[7][:], []byte{0x80, 0x7f, 0xff, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13})
	copy(state.XMM[8][:], []byte{0x7f, 0x80, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	putDwords(&state.XMM[9], 0x80000000, 0x7fffffff, 0xffffffff, 1)
	putDwords(&state.XMM[10], 0, 0xffffffff, 0xfffffffe, 2)
	putDwords(&state.XMM[11], 0x80000000, 0x7fffffff, 0xffffffff, 1)
	putDwords(&state.XMM[12], 0, 0xffffffff, 0xfffffffe, 2)
	putDwords(&state.XMM[13], 0, 0xffffffff, 0x80000000, 3)
	copy(state.XMM[14][:], []byte{0x80, 0x7f, 0xff, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13})
	copy(state.XMM[15][:], []byte{0x7f, 0x80, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	wantWords := func(reg int, want ...uint16) {
		t.Helper()
		for lane, expected := range want {
			if got := binary.LittleEndian.Uint16(state.XMM[reg][lane*2:]); got != expected {
				t.Fatalf("xmm%d word lane %d = %#x, want %#x", reg, lane, got, expected)
			}
		}
	}
	wantDwords := func(reg int, want ...uint32) {
		t.Helper()
		for lane, expected := range want {
			if got := binary.LittleEndian.Uint32(state.XMM[reg][lane*4:]); got != expected {
				t.Fatalf("xmm%d dword lane %d = %#x, want %#x", reg, lane, got, expected)
			}
		}
	}
	wantBytes := func(reg int, want ...byte) {
		t.Helper()
		if string(state.XMM[reg][:]) != string(want) {
			t.Fatalf("xmm%d bytes = % x, want % x", reg, state.XMM[reg], want)
		}
	}
	wantWords(1, 0, 0, 0x7fff, 1, 1, 3, 3, 5)
	wantDwords(3, 1, 0, 0x7fffffff, 7)
	wantWords(5, 1, 0xffff, 0x8000, 2, 2, 4, 4, 6)
	wantBytes(7, 0x7f, 0x7f, 0, 2, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13)
	wantDwords(9, 0, 0x7fffffff, 0xffffffff, 2)
	wantDwords(11, 0x80000000, 0xffffffff, 0xfffffffe, 1)
	wantDwords(13, 1, 0xffffffff, 0x80000000, 4)
	wantBytes(14, 0x80, 0x80, 0xff, 1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
}

func TestJIT64CRC32(t *testing.T) {
	const (
		codeAddress Address64 = 0x54000
		dataAddress Address64 = 0x55000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(dataAddress, []byte("12345678")); err != nil {
		t.Fatal(err)
	}
	code := []byte{
		0xf2, 0x0f, 0x38, 0xf0, 0xc1, // crc32 eax, cl
		0x66, 0xf2, 0x44, 0x0f, 0x38, 0xf1, 0xc1, // crc32 r8d, cx
		0xf2, 0x44, 0x0f, 0x38, 0xf1, 0xc9, // crc32 r9d, ecx
		0xf2, 0x4c, 0x0f, 0x38, 0xf1, 0xd9, // crc32 r11, rcx
		0xf2, 0x0f, 0x38, 0xf0, 0x07, // crc32 eax, byte ptr [rdi]
		0x66, 0xf2, 0x44, 0x0f, 0x38, 0xf1, 0x07, // crc32 r8d, word ptr [rdi]
		0xf2, 0x44, 0x0f, 0x38, 0xf1, 0x0f, // crc32 r9d, dword ptr [rdi]
		0xf2, 0x4c, 0x0f, 0x38, 0xf1, 0x17, // crc32 r10, qword ptr [rdi]
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RCX, 0x3837363534333231)
	state.Set(RDI, uint64(dataAddress))
	state.Set(RAX, 0x1111111100000000)
	state.Set(R8, 0x2222222200000000)
	state.Set(R9, 0x3333333300000000)
	state.Set(R10, 0x4444444400000000)
	state.Set(R11, 0x5555555500000000)
	state.RFLAGS = Flag64IF | Flag64CF | Flag64ZF | Flag64SF
	flagsBefore := state.RFLAGS
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("RFLAGS changed from %#x to %#x", flagsBefore, state.RFLAGS)
	}
	want := map[Reg64]uint64{
		RAX: 0x60053794,
		R8:  0xf7de23e8,
		R9:  0x86b2bb7f,
		R11: 0x6087809a,
		R10: 0x6087809a,
	}
	for reg, expected := range want {
		if got := state.Get(reg); got != expected {
			t.Fatalf("%s=%#x, want %#x", reg, got, expected)
		}
	}
}

func TestJIT64PCMPISTRIEqualEachAnyMemoryREXR(t *testing.T) {
	const (
		codeAddress Address64 = 0x56000
		dataAddress Address64 = 0x57000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	// PCMPISTRI xmm0,xmm1,0x08: equal-each, least-significant match.
	// PCMPISTRI xmm8,[rdi],0x00: equal-any, with REX.R and a memory source.
	code := []byte{
		0x66, 0x0f, 0x3a, 0x63, 0xc1, 0x08,
		0x66, 0x44, 0x0f, 0x3a, 0x63, 0x07, 0x00,
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(dataAddress, []byte{'x', 'a', 0}); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	for i := 0; i < 16; i++ {
		state.XMM[0][i] = byte(i + 1)
		state.XMM[1][i] = byte(i + 1)
	}
	state.XMM[1][0] = 9 // Equal-each first match is lane 1.
	copy(state.XMM[8][:], []byte{'a', 'b', 0})

	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if got := state.Get(RCX); got != 1 {
		t.Fatalf("PCMPISTRI result=%d, want 1 from equal-any memory operation", got)
	}
}

func TestJIT64PCMPESTRIOrderedAndMask(t *testing.T) {
	const (
		codeAddress Address64 = 0x58000
		dataAddress Address64 = 0x59000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	// First operation returns the ordered match index. The second operation
	// uses the same inputs with bit 6 set and writes an expanded byte mask.
	code := []byte{
		0x66, 0x0f, 0x3a, 0x61, 0xc1, 0x0c, // pcmpestri xmm0,xmm1, equal-ordered
		0x66, 0x0f, 0x3a, 0x60, 0xc1, 0x4c, // pcmpestrm xmm0,xmm1, expanded mask
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RAX, 2) // xmm0: needle "bc"
	state.Set(RDX, 4) // xmm1: haystack "xabc"
	copy(state.XMM[0][:], []byte{'b', 'c'})
	copy(state.XMM[1][:], []byte{'x', 'a', 'b', 'c'})

	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	// The final instruction is PCMPESTRM, so the index produced by the first
	// instruction is checked by running the instruction in isolation below.
	if state.XMM[0][2] != 0xff || state.XMM[0][0] != 0 || state.XMM[0][1] != 0 || state.XMM[0][3] != 0 {
		t.Fatalf("expanded mask=%x, want only haystack lane 2 selected", state.XMM[0])
	}
	if !state.Flag(Flag64CF) || !state.Flag(Flag64ZF) || !state.Flag(Flag64SF) || state.Flag(Flag64OF) {
		t.Fatalf("flags cf=%v zf=%v sf=%v of=%v, want 1 1 1 0", state.Flag(Flag64CF), state.Flag(Flag64ZF), state.Flag(Flag64SF), state.Flag(Flag64OF))
	}

	// Re-run only PCMPESTRI to verify ECX=2 before PCMPESTRM overwrites XMM0.
	memory2 := NewMemory64()
	if err := memory2.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory2.Write(codeAddress, []byte{0x66, 0x0f, 0x3a, 0x61, 0xc1, 0x0c, 0xf4}); err != nil {
		t.Fatal(err)
	}
	state2 := NewMachineState64(memory2)
	state2.RIP = uint64(codeAddress)
	state2.Set(RAX, 2)
	state2.Set(RDX, 4)
	copy(state2.XMM[0][:], []byte{'b', 'c'})
	copy(state2.XMM[1][:], []byte{'x', 'a', 'b', 'c'})
	if trap := NewJIT64(memory2).RunToInterrupt(state2); trap != Trap64Timer || !state2.Halted {
		t.Fatalf("isolated PCMPESTRI trap=%#x halted=%v", trap, state2.Halted)
	}
	if got := state2.Get(RCX); got != 2 {
		t.Fatalf("PCMPESTRI index=%d, want 2", got)
	}
}

func TestJIT64MPSADBWAndPCLMULQDQ(t *testing.T) {
	const (
		codeAddress Address64 = 0x5a000
		dataAddress Address64 = 0x5b000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	// MPSADBW xmm0,xmm1,6; MPSADBW xmm8,[rdi],1;
	// PCLMULQDQ xmm2,xmm3,11h; PCLMULQDQ xmm9,[rdi],0.
	code := []byte{
		0x66, 0x0f, 0x3a, 0x42, 0xc1, 0x06,
		0x66, 0x44, 0x0f, 0x3a, 0x42, 0x07, 0x01,
		0x66, 0x0f, 0x3a, 0x44, 0xd3, 0x11,
		0x66, 0x44, 0x0f, 0x3a, 0x44, 0x0f, 0x00,
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	var memorySource [16]byte
	for i := range memorySource {
		memorySource[i] = byte(30 + i)
	}
	if err := memory.Write(dataAddress, memorySource[:]); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	for i := range state.XMM[0] {
		state.XMM[0][i] = byte(i)
		state.XMM[8][i] = byte(20 + i)
	}
	state.XMM[1] = [16]byte{}
	for i := range state.XMM[1] {
		state.XMM[1][i] = byte(10 + i)
	}
	state.XMM[2] = [16]byte{}
	binary.LittleEndian.PutUint64(state.XMM[2][8:], 0x8000000000000001)
	state.XMM[3] = [16]byte{}
	binary.LittleEndian.PutUint64(state.XMM[3][8:], 0x3)
	state.XMM[9] = [16]byte{}
	binary.LittleEndian.PutUint64(state.XMM[9][0:], 1)

	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	for lane, want := range []uint16{56, 52, 48, 44, 40, 36, 32, 28} {
		if got := binary.LittleEndian.Uint16(state.XMM[0][lane*2:]); got != want {
			t.Fatalf("MPSADBW register lane %d=%d, want %d", lane, got, want)
		}
	}
	for lane, want := range []uint16{56, 52, 48, 44, 40, 36, 32, 28} {
		if got := binary.LittleEndian.Uint16(state.XMM[8][lane*2:]); got != want {
			t.Fatalf("MPSADBW memory/REX.R lane %d=%d, want %d", lane, got, want)
		}
	}
	if got := binary.LittleEndian.Uint64(state.XMM[2][0:]); got != 0x8000000000000003 || binary.LittleEndian.Uint64(state.XMM[2][8:]) != 1 {
		t.Fatalf("PCLMULQDQ high/high=%x, want high=1 low=8000000000000003", state.XMM[2])
	}
	if got := binary.LittleEndian.Uint64(state.XMM[9][0:]); got != 0x2524232221201f1e || binary.LittleEndian.Uint64(state.XMM[9][8:]) != 0 {
		t.Fatalf("PCLMULQDQ memory low/low=%x, want high=0 low=2524232221201f1e", state.XMM[9])
	}
}

func shaTestWords(vector [16]byte) [4]uint32 {
	return [4]uint32{
		binary.LittleEndian.Uint32(vector[0:4]),
		binary.LittleEndian.Uint32(vector[4:8]),
		binary.LittleEndian.Uint32(vector[8:12]),
		binary.LittleEndian.Uint32(vector[12:16]),
	}
}

func setSHATestWords(vector *[16]byte, words [4]uint32) {
	for lane, word := range words {
		binary.LittleEndian.PutUint32(vector[lane*4:lane*4+4], word)
	}
}

func shaTestSigma0Small(value uint32) uint32 {
	return bits.RotateLeft32(value, -7) ^ bits.RotateLeft32(value, -18) ^ (value >> 3)
}

func shaTestSigma1Small(value uint32) uint32 {
	return bits.RotateLeft32(value, -17) ^ bits.RotateLeft32(value, -19) ^ (value >> 10)
}

func shaTestSigma0Big(value uint32) uint32 {
	return bits.RotateLeft32(value, -2) ^ bits.RotateLeft32(value, -13) ^ bits.RotateLeft32(value, -22)
}

func shaTestSigma1Big(value uint32) uint32 {
	return bits.RotateLeft32(value, -6) ^ bits.RotateLeft32(value, -11) ^ bits.RotateLeft32(value, -25)
}

func shaTestChoose(x, y, z uint32) uint32 {
	return (x & y) ^ (^x & z)
}

func shaTestMajority(x, y, z uint32) uint32 {
	return (x & y) ^ (x & z) ^ (y & z)
}

func shaTestMessage1(src1, src2 [4]uint32) [4]uint32 {
	return [4]uint32{src2[2] ^ src1[0], src2[3] ^ src1[1], src1[0] ^ src1[2], src1[1] ^ src1[3]}
}

func shaTestMessage2SHA1(src1, src2 [4]uint32) [4]uint32 {
	w16 := bits.RotateLeft32(src1[3]^src2[2], 1)
	w17 := bits.RotateLeft32(src1[2]^src2[1], 1)
	w18 := bits.RotateLeft32(src1[1]^src2[0], 1)
	w19 := bits.RotateLeft32(src1[0]^w16, 1)
	return [4]uint32{w19, w18, w17, w16}
}

func shaTestMessage1SHA256(src1, src2 [4]uint32) [4]uint32 {
	return [4]uint32{
		src1[0] + shaTestSigma0Small(src1[1]),
		src1[1] + shaTestSigma0Small(src1[2]),
		src1[2] + shaTestSigma0Small(src1[3]),
		src1[3] + shaTestSigma0Small(src2[0]),
	}
}

func shaTestMessage2SHA256(src1, src2 [4]uint32) [4]uint32 {
	w16 := src1[0] + shaTestSigma1Small(src2[2])
	w17 := src1[1] + shaTestSigma1Small(src2[3])
	w18 := src1[2] + shaTestSigma1Small(w16)
	w19 := src1[3] + shaTestSigma1Small(w17)
	return [4]uint32{w16, w17, w18, w19}
}

func shaTestRounds2(src1, src2, implicit [4]uint32) [4]uint32 {
	a, b, c, d := src2[3], src2[2], src1[3], src1[2]
	e, f, g, h := src2[1], src2[0], src1[1], src1[0]
	for _, wk := range implicit[:2] {
		t1 := shaTestChoose(e, f, g) + shaTestSigma1Big(e) + wk + h
		t2 := shaTestMajority(a, b, c) + shaTestSigma0Big(a)
		a, b, c, d, e, f, g, h = t1+t2, a, b, c, t1+d, e, f, g
	}
	return [4]uint32{f, e, b, a}
}

func TestJIT64SHAExtensions(t *testing.T) {
	const (
		codeAddress Address64 = 0x5c000
		dataAddress Address64 = 0x5d000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	// SHA256RNDS2 xmm8,xmm9,xmm0; SHA1MSG1 xmm1,xmm2;
	// SHA1MSG2 xmm3,xmm4; SHA256MSG1 xmm5,xmm6;
	// SHA256MSG2 xmm15,[rdi] (REX.R); then HLT.
	code := []byte{
		0x45, 0x0f, 0x38, 0xcb, 0xc1,
		0x0f, 0x38, 0xc9, 0xca,
		0x0f, 0x38, 0xca, 0xdc,
		0x0f, 0x38, 0xcc, 0xee,
		0x44, 0x0f, 0x38, 0xcd, 0x3f,
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	memoryWords := [4]uint32{0x12345678, 0x9abcdef0, 0x0badcafe, 0xfeedface}
	var memoryVector [16]byte
	setSHATestWords(&memoryVector, memoryWords)
	if err := memory.Write(dataAddress, memoryVector[:]); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	implicit := [4]uint32{0x01020304, 0x11121314, 0x21222324, 0x31323334}
	sha1Message1Src := [4]uint32{0x10203040, 0x50607080, 0x90a0b0c0, 0xd0e0f000}
	sha1Message1Other := [4]uint32{0x11111111, 0x22222222, 0x33333333, 0x44444444}
	sha1Message2Src := [4]uint32{0x01010101, 0x02020202, 0x03030303, 0x04040404}
	sha1Message2Other := [4]uint32{0x55555555, 0x66666666, 0x77777777, 0x88888888}
	sha256Message1Src := [4]uint32{0x0f1e2d3c, 0x4b5a6978, 0x8796a5b4, 0xc3d2e1f0}
	sha256Message1Other := [4]uint32{0x13579bdf, 0x2468ace0, 0xdeadbeef, 0x10293847}
	sha256RoundsSource1 := [4]uint32{0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a}
	sha256RoundsSource2 := [4]uint32{0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19}
	setSHATestWords(&state.XMM[0], implicit)
	setSHATestWords(&state.XMM[1], sha1Message1Src)
	setSHATestWords(&state.XMM[2], sha1Message1Other)
	setSHATestWords(&state.XMM[3], sha1Message2Src)
	setSHATestWords(&state.XMM[4], sha1Message2Other)
	setSHATestWords(&state.XMM[5], sha256Message1Src)
	setSHATestWords(&state.XMM[6], sha256Message1Other)
	setSHATestWords(&state.XMM[8], sha256RoundsSource1)
	setSHATestWords(&state.XMM[9], sha256RoundsSource2)
	state.RFLAGS = Flag64IF | Flag64CF | Flag64ZF | Flag64SF
	flagsBefore := state.RFLAGS

	wantRounds := shaTestRounds2(sha256RoundsSource1, sha256RoundsSource2, implicit)
	wantSHA1Message1 := shaTestMessage1(sha1Message1Src, sha1Message1Other)
	wantSHA1Message2 := shaTestMessage2SHA1(sha1Message2Src, sha1Message2Other)
	wantSHA256Message1 := shaTestMessage1SHA256(sha256Message1Src, sha256Message1Other)
	wantSHA256Message2 := shaTestMessage2SHA256(shaTestWords(state.XMM[7]), memoryWords)

	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("SHA instructions changed RFLAGS from %#x to %#x", flagsBefore, state.RFLAGS)
	}
	for lane, want := range wantRounds {
		if got := binary.LittleEndian.Uint32(state.XMM[8][lane*4:]); got != want {
			t.Fatalf("SHA256RNDS2 lane %d=%#x, want %#x", lane, got, want)
		}
	}
	for lane, want := range wantSHA1Message1 {
		if got := binary.LittleEndian.Uint32(state.XMM[1][lane*4:]); got != want {
			t.Fatalf("SHA1MSG1 lane %d=%#x, want %#x", lane, got, want)
		}
	}
	for lane, want := range wantSHA1Message2 {
		if got := binary.LittleEndian.Uint32(state.XMM[3][lane*4:]); got != want {
			t.Fatalf("SHA1MSG2 lane %d=%#x, want %#x", lane, got, want)
		}
	}
	for lane, want := range wantSHA256Message1 {
		if got := binary.LittleEndian.Uint32(state.XMM[5][lane*4:]); got != want {
			t.Fatalf("SHA256MSG1 lane %d=%#x, want %#x", lane, got, want)
		}
	}
	for lane, want := range wantSHA256Message2 {
		if got := binary.LittleEndian.Uint32(state.XMM[15][lane*4:]); got != want {
			t.Fatalf("SHA256MSG2/REX.R lane %d=%#x, want %#x", lane, got, want)
		}
	}
}

func TestJIT64AESInstructions(t *testing.T) {
	const (
		codeAddress Address64 = 0x5e000
		dataAddress Address64 = 0x5f000
	)
	key0 := [16]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
	key1 := [16]byte{0xd6, 0xaa, 0x74, 0xfd, 0xd2, 0xaf, 0x72, 0xfa, 0xda, 0xa6, 0x78, 0xf1, 0xd6, 0xab, 0x76, 0xfe}
	fipsRoundInput := [16]byte{0x00, 0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80, 0x90, 0xa0, 0xb0, 0xc0, 0xd0, 0xe0, 0xf0}
	fipsRoundOutput := [16]byte{0x89, 0xd8, 0x10, 0xe8, 0x85, 0x5a, 0xce, 0x68, 0x2d, 0x18, 0x43, 0xd8, 0xcb, 0x12, 0x8f, 0xe4}

	run := func(t *testing.T, code []byte, setup func(*MachineState64), data [16]byte) *MachineState64 {
		t.Helper()
		memory := NewMemory64()
		if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
			t.Fatal(err)
		}
		if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
			t.Fatal(err)
		}
		if err := memory.Write(codeAddress, code); err != nil {
			t.Fatal(err)
		}
		if err := memory.Write(dataAddress, data[:]); err != nil {
			t.Fatal(err)
		}
		state := NewMachineState64(memory)
		state.RIP = uint64(codeAddress)
		state.Set(RDI, uint64(dataAddress))
		state.RFLAGS = Flag64IF | Flag64CF | Flag64ZF | Flag64SF
		flagsBefore := state.RFLAGS
		setup(state)
		trap := NewJIT64(memory).RunToInterrupt(state)
		if trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
		if state.RFLAGS != flagsBefore {
			t.Fatalf("AES instruction changed RFLAGS from %#x to %#x", flagsBefore, state.RFLAGS)
		}
		return state
	}

	t.Run("AESENC-FIPS-round-memory", func(t *testing.T) {
		state := run(t, []byte{0x66, 0x0f, 0x38, 0xdc, 0x07, 0xf4}, func(state *MachineState64) {
			state.XMM[0] = fipsRoundInput
		}, key1)
		if state.XMM[0] != fipsRoundOutput {
			t.Fatalf("AESENC round=%x, want %x", state.XMM[0], fipsRoundOutput)
		}
	})

	t.Run("AESDEC-equivalent-inverse-round", func(t *testing.T) {
		stateInput := [16]byte{0xbd, 0x6e, 0x7c, 0x3d, 0xf2, 0xb5, 0x77, 0x9e, 0x0b, 0x61, 0x21, 0x6e, 0x8b, 0x10, 0xb6, 0x89}
		key9 := [16]byte{0x54, 0x99, 0x32, 0xd1, 0xf0, 0x85, 0x57, 0x68, 0x10, 0x93, 0xed, 0x9c, 0xbe, 0x2c, 0x97, 0x4e}
		want := [16]byte{0xf4, 0xdf, 0x52, 0x42, 0x4d, 0x0a, 0xf9, 0x89, 0x8b, 0xaf, 0x73, 0x1a, 0xf4, 0xdc, 0xa2, 0x5e}
		state := run(t, []byte{
			0x66, 0x0f, 0x38, 0xde, 0xc1, // AESDEC xmm0,xmm1
			0xf4,
		}, func(state *MachineState64) {
			state.XMM[0] = stateInput
			state.XMM[1] = aesMixColumns64(key9, true)
		}, key0)
		if state.XMM[0] != want {
			t.Fatalf("AESDEC equivalent-inverse round=%x, want %x", state.XMM[0], want)
		}
	})

	t.Run("AESIMC-register", func(t *testing.T) {
		state := run(t, []byte{0x66, 0x0f, 0x38, 0xdb, 0xc9, 0xf4}, func(state *MachineState64) {
			state.XMM[1] = key1
		}, key0)
		wantInvMix := aesMixColumns64(key1, true)
		if state.XMM[1] != wantInvMix {
			t.Fatalf("AESIMC key=%x, want %x", state.XMM[1], wantInvMix)
		}
	})

	t.Run("AESENCLAST-FIPS-final", func(t *testing.T) {
		round9 := [16]byte{0xbd, 0x6e, 0x7c, 0x3d, 0xf2, 0xb5, 0x77, 0x9e, 0x0b, 0x61, 0x21, 0x6e, 0x8b, 0x10, 0xb6, 0x89}
		key10 := [16]byte{0x13, 0x11, 0x1d, 0x7f, 0xe3, 0x94, 0x4a, 0x17, 0xf3, 0x07, 0xa7, 0x8b, 0x4d, 0x2b, 0x30, 0xc5}
		want := [16]byte{0x69, 0xc4, 0xe0, 0xd8, 0x6a, 0x7b, 0x04, 0x30, 0xd8, 0xcd, 0xb7, 0x80, 0x70, 0xb4, 0xc5, 0x5a}
		state := run(t, []byte{0x66, 0x0f, 0x38, 0xdd, 0xc1, 0xf4}, func(state *MachineState64) {
			state.XMM[0] = round9
			state.XMM[1] = key10
		}, key0)
		if state.XMM[0] != want {
			t.Fatalf("AESENCLAST FIPS final=%x, want %x", state.XMM[0], want)
		}
	})

	t.Run("AESDECLAST-vector", func(t *testing.T) {
		input := [16]byte{0x3a, 0x94, 0x17, 0x5f, 0x21, 0xc0, 0x8e, 0x67, 0x10, 0x4d, 0xb2, 0x7a, 0xf8, 0x33, 0x6c, 0x09}
		want := [16]byte{0x74, 0xcc, 0x4a, 0xf7, 0xa9, 0x48, 0xca, 0x47, 0xa6, 0xb9, 0xff, 0xb1, 0x37, 0xce, 0x90, 0x7a}
		state := run(t, []byte{0x66, 0x0f, 0x38, 0xdf, 0xc1, 0xf4}, func(state *MachineState64) {
			state.XMM[0] = input
			state.XMM[1] = key1
		}, key0)
		if state.XMM[0] != want {
			t.Fatalf("AESDECLAST=%x, want %x", state.XMM[0], want)
		}
	})

	t.Run("AESIMC-REXR-memory", func(t *testing.T) {
		state := run(t, []byte{0x66, 0x44, 0x0f, 0x38, 0xdb, 0x0f, 0xf4}, func(state *MachineState64) {}, key1)
		want := aesMixColumns64(key1, true)
		if state.XMM[9] != want {
			t.Fatalf("AESIMC xmm9,[rdi] = %x, want %x", state.XMM[9], want)
		}
	})

	t.Run("AESKEYGENASSIST-memory", func(t *testing.T) {
		state := run(t, []byte{0x66, 0x0f, 0x3a, 0xdf, 0x1f, 0x01, 0xf4}, func(state *MachineState64) {}, key0)
		want := [16]byte{0xf2, 0x6b, 0x6f, 0xc5, 0x6a, 0x6f, 0xc5, 0xf2, 0xfe, 0xd7, 0xab, 0x76, 0xd6, 0xab, 0x76, 0xfe}
		if state.XMM[3] != want {
			t.Fatalf("AESKEYGENASSIST=%x, want %x", state.XMM[3], want)
		}
	})
}

func TestJIT64RDRAND(t *testing.T) {
	const codeAddress Address64 = 0x50000

	t.Run("success widths and flags", func(t *testing.T) {
		memory := NewMemory64()
		code := []byte{
			0x0f, 0xc7, 0xf0, // rdrand eax
			0x48, 0x0f, 0xc7, 0xf1, // rdrand rcx
			0x66, 0x0f, 0xc7, 0xf2, // rdrand dx
			0x41, 0x0f, 0xc7, 0xf3, // rdrand r11d (REX.B)
			0xf4, // hlt
		}
		mapExecutable64(t, memory, codeAddress, code)
		state := NewMachineState64(memory)
		state.RIP = uint64(codeAddress)
		state.Set(RDX, 0xaabbccdd00000000)
		state.RFLAGS = Flag64IF | Flag64CF | Flag64PF | Flag64AF | Flag64ZF | Flag64SF | Flag64OF
		calls := make([]uint8, 0, 4)
		state.RDRAND = func(width uint8) (uint64, bool) {
			calls = append(calls, width)
			switch width {
			case 4:
				if len(calls) == 1 {
					return 0x11223344, true
				}
				return 0x55667788, true
			case 8:
				return 0x0102030405060708, true
			case 2:
				return 0xbeef, true
			default:
				return 0, false
			}
		}
		if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
		if got := state.Get(RAX); got != 0x11223344 {
			t.Fatalf("rdrand eax=%#x, want %#x", got, uint64(0x11223344))
		}
		if got := state.Get(RCX); got != 0x0102030405060708 {
			t.Fatalf("rdrand rcx=%#x, want %#x", got, uint64(0x0102030405060708))
		}
		if got := state.Get(RDX); got != 0xaabbccdd0000beef {
			t.Fatalf("rdrand dx=%#x, want %#x", got, uint64(0xaabbccdd0000beef))
		}
		if got := state.Get(R11); got != 0x55667788 {
			t.Fatalf("rdrand r11d=%#x, want %#x", got, uint64(0x55667788))
		}
		if len(calls) != 4 || calls[0] != 4 || calls[1] != 8 || calls[2] != 2 || calls[3] != 4 {
			t.Fatalf("provider widths=%v, want [4 8 2 4]", calls)
		}
		if got := state.RFLAGS; got != Flag64IF|Flag64CF {
			t.Fatalf("success flags=%#x, want %#x", got, uint64(Flag64IF|Flag64CF))
		}
	})

	t.Run("failure clears destination and arithmetic flags", func(t *testing.T) {
		memory := NewMemory64()
		mapExecutable64(t, memory, codeAddress, []byte{0x0f, 0xc7, 0xf0, 0xf4}) // rdrand eax; hlt
		state := NewMachineState64(memory)
		state.RIP = uint64(codeAddress)
		state.Set(RAX, 0xffffffffffffffff)
		state.RFLAGS = Flag64IF | Flag64CF | Flag64PF | Flag64AF | Flag64ZF | Flag64SF | Flag64OF
		state.RDRAND = func(width uint8) (uint64, bool) {
			if width != 4 {
				t.Fatalf("failure provider width=%d, want 4", width)
			}
			return 0, false
		}
		if trap := NewJIT64(memory).RunToInterrupt(state); trap != Trap64Timer || !state.Halted {
			t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
		}
		if got := state.Get(RAX); got != 0 {
			t.Fatalf("failed rdrand eax=%#x, want 0", got)
		}
		if got := state.RFLAGS; got != Flag64IF {
			t.Fatalf("failure flags=%#x, want %#x", got, uint64(Flag64IF))
		}
	})
}

func TestJIT64SSE3HorizontalArithmeticAndMasks(t *testing.T) {
	const codeAddress Address64 = 0x2d000
	const dataAddress Address64 = 0x2e000

	putFloat32s := func(memory *Memory64, address Address64, values []float32) {
		var raw [16]byte
		for i, value := range values {
			binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(value))
		}
		if err := memory.Write(address, raw[:]); err != nil {
			t.Fatal(err)
		}
	}
	putFloat64s := func(memory *Memory64, address Address64, values []float64) {
		var raw [16]byte
		for i, value := range values {
			binary.LittleEndian.PutUint64(raw[i*8:], math.Float64bits(value))
		}
		if err := memory.Write(address, raw[:]); err != nil {
			t.Fatal(err)
		}
	}
	readFloat32s := func(t *testing.T, memory *Memory64, address Address64) [4]float32 {
		t.Helper()
		var raw [16]byte
		if err := memory.Read(address, raw[:]); err != nil {
			t.Fatal(err)
		}
		var values [4]float32
		for i := range values {
			values[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		}
		return values
	}
	readFloat64s := func(t *testing.T, memory *Memory64, address Address64) [2]float64 {
		t.Helper()
		var raw [16]byte
		if err := memory.Read(address, raw[:]); err != nil {
			t.Fatal(err)
		}
		var values [2]float64
		for i := range values {
			values[i] = math.Float64frombits(binary.LittleEndian.Uint64(raw[i*8:]))
		}
		return values
	}
	checkFloat32 := func(t *testing.T, got [4]float32, want [4]float32) {
		t.Helper()
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("float32 lane %d = %v, want %v", i, got[i], want[i])
			}
		}
	}
	checkFloat64 := func(t *testing.T, got [2]float64, want [2]float64) {
		t.Helper()
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("float64 lane %d = %v, want %v", i, got[i], want[i])
			}
		}
	}

	psMemory := NewMemory64()
	psCode := []byte{
		0xf3, 0x0f, 0x6f, 0x07, // movdqu xmm0, [rdi]
		0xf3, 0x0f, 0x6f, 0x4f, 0x10, // movdqu xmm1, [rdi+16]
		0xf2, 0x0f, 0xd0, 0xc1, // addsubps xmm0, xmm1
		0xf3, 0x0f, 0x7f, 0x47, 0x20, // movdqu [rdi+32], xmm0
		0xf3, 0x0f, 0x6f, 0x07, // reload xmm0
		0xf3, 0x0f, 0x6f, 0x4f, 0x10, // reload xmm1
		0xf2, 0x0f, 0x7c, 0xc1, // haddps xmm0, xmm1
		0xf3, 0x0f, 0x7f, 0x47, 0x30, // movdqu [rdi+48], xmm0
		0xf3, 0x0f, 0x6f, 0x07, // reload xmm0
		0xf3, 0x0f, 0x6f, 0x4f, 0x10, // reload xmm1
		0xf2, 0x0f, 0x7d, 0xc1, // hsubps xmm0, xmm1
		0xf3, 0x0f, 0x7f, 0x47, 0x40, // movdqu [rdi+64], xmm0
		0xf4,
	}
	mapExecutable64(t, psMemory, codeAddress, psCode)
	if err := psMemory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	putFloat32s(psMemory, dataAddress, []float32{1, 2, 4, 8})
	putFloat32s(psMemory, dataAddress+16, []float32{3, 4, 10, 6})
	psState := NewMachineState64(psMemory)
	psState.RIP = uint64(codeAddress)
	psState.Set(RDI, uint64(dataAddress))
	if trap := NewJIT64(psMemory).RunToInterrupt(psState); trap != Trap64Timer || !psState.Halted {
		t.Fatalf("packed single trap=%#x halted=%v", trap, psState.Halted)
	}
	checkFloat32(t, readFloat32s(t, psMemory, dataAddress+32), [4]float32{-2, 6, -6, 14})
	checkFloat32(t, readFloat32s(t, psMemory, dataAddress+48), [4]float32{3, 12, 7, 16})
	checkFloat32(t, readFloat32s(t, psMemory, dataAddress+64), [4]float32{-1, -4, -1, 4})

	pdMemory := NewMemory64()
	pdCode := []byte{
		0x66, 0x0f, 0x6f, 0x07, // movdqa xmm0, [rdi]
		0x66, 0x0f, 0x6f, 0x4f, 0x10, // movdqa xmm1, [rdi+16]
		0x66, 0x0f, 0xd0, 0xc1, // addsubpd xmm0, xmm1
		0xf3, 0x0f, 0x7f, 0x47, 0x20, // movdqu [rdi+32], xmm0
		0x66, 0x0f, 0x6f, 0x07, // reload xmm0
		0x66, 0x0f, 0x6f, 0x4f, 0x10, // reload xmm1
		0x66, 0x0f, 0x7c, 0xc1, // haddpd xmm0, xmm1
		0xf3, 0x0f, 0x7f, 0x47, 0x30, // movdqu [rdi+48], xmm0
		0x66, 0x0f, 0x6f, 0x07, // reload xmm0
		0x66, 0x0f, 0x6f, 0x4f, 0x10, // reload xmm1
		0x66, 0x0f, 0x7d, 0xc1, // hsubpd xmm0, xmm1
		0xf3, 0x0f, 0x7f, 0x47, 0x40, // movdqu [rdi+64], xmm0
		0xf4,
	}
	mapExecutable64(t, pdMemory, codeAddress, pdCode)
	if err := pdMemory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	putFloat64s(pdMemory, dataAddress, []float64{1.5, 4})
	putFloat64s(pdMemory, dataAddress+16, []float64{0.5, 2})
	pdState := NewMachineState64(pdMemory)
	pdState.RIP = uint64(codeAddress)
	pdState.Set(RDI, uint64(dataAddress))
	if trap := NewJIT64(pdMemory).RunToInterrupt(pdState); trap != Trap64Timer || !pdState.Halted {
		t.Fatalf("packed double trap=%#x halted=%v", trap, pdState.Halted)
	}
	checkFloat64(t, readFloat64s(t, pdMemory, dataAddress+32), [2]float64{1, 6})
	checkFloat64(t, readFloat64s(t, pdMemory, dataAddress+48), [2]float64{5.5, 2.5})
	checkFloat64(t, readFloat64s(t, pdMemory, dataAddress+64), [2]float64{-2.5, -1.5})

	maskMemory := NewMemory64()
	maskCode := []byte{
		0xf3, 0x0f, 0x6f, 0x07, // movdqu xmm0, [rdi]
		0x0f, 0x50, 0xc0, // movmskps eax, xmm0
		0x66, 0x4c, 0x0f, 0x50, 0xc8, // movmskpd r9, xmm0 (66+REX.W+REX.R)
		0x66, 0x0f, 0x38, 0x17, 0x47, 0x10, // ptest xmm0, [rdi+16]
		0xf4,
	}
	mapExecutable64(t, maskMemory, codeAddress, maskCode)
	if err := maskMemory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	var signData [32]byte
	binary.LittleEndian.PutUint32(signData[4:], 0x80000000)
	binary.LittleEndian.PutUint32(signData[12:], 0x80000000)
	for i := 16; i < len(signData); i++ {
		signData[i] = 1
	}
	if err := maskMemory.Write(dataAddress, signData[:]); err != nil {
		t.Fatal(err)
	}
	maskState := NewMachineState64(maskMemory)
	maskState.RIP = uint64(codeAddress)
	maskState.Set(RDI, uint64(dataAddress))
	maskState.RFLAGS = Flag64CF | Flag64PF | Flag64AF | Flag64ZF | Flag64SF | Flag64OF
	if trap := NewJIT64(maskMemory).RunToInterrupt(maskState); trap != Trap64Timer || !maskState.Halted {
		t.Fatalf("mask trap=%#x halted=%v", trap, maskState.Halted)
	}
	if got := maskState.Get(RAX); got != 10 {
		t.Fatalf("movmskps eax=%#x, want 0xa", got)
	}
	if got := maskState.Get(R9); got != 3 {
		t.Fatalf("movmskpd r9=%#x, want 3", got)
	}
	if !maskState.Flag(Flag64ZF) || maskState.Flag(Flag64CF) || maskState.Flag(Flag64PF) || maskState.Flag(Flag64AF) || maskState.Flag(Flag64SF) || maskState.Flag(Flag64OF) {
		t.Fatalf("ptest flags rflags=%#x, want ZF=1 and all other arithmetic flags clear", maskState.RFLAGS)
	}
}

func TestJIT64PMULLDPMULDQ(t *testing.T) {
	memory := NewMemory64()
	const dataAddress Address64 = 0x66000
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	var memoryVector [32]byte
	for lane, value := range []int32{2, -3, 0x7fffffff, -0x80000000} {
		binary.LittleEndian.PutUint32(memoryVector[lane*4:], uint32(value))
		binary.LittleEndian.PutUint32(memoryVector[16+lane*4:], uint32(value+1))
	}
	if err := memory.Write(dataAddress, memoryVector[:]); err != nil {
		t.Fatal(err)
	}

	state := NewMachineState64(memory)
	state.RIP = 0x65000
	state.Set(RDI, uint64(dataAddress))
	for xmm, values := range map[uint8][]int32{
		0:  {2, -3, 0x7fffffff, -0x80000000},
		1:  {-5, 7, 2, -1},
		2:  {2, 123, 0x7fffffff, -456},
		3:  {3, -9, -2, 777},
		8:  {-0x80000000, 11, 0x40000000, -7},
		9:  {2, -13, 3, -2},
		10: {9, 101, -0x80000000, 202},
		11: {-3, 303, -1, 404},
		12: {2, -3, 0x7fffffff, -0x80000000},
		13: {2, 0, 0x7fffffff, 0},
	} {
		for lane, value := range values {
			binary.LittleEndian.PutUint32(state.XMM[xmm][lane*4:], uint32(value))
		}
	}
	code := []byte{
		0x66, 0x0f, 0x38, 0x40, 0xc1, // pmulld xmm0, xmm1
		0x66, 0x0f, 0x38, 0x28, 0xd3, // pmuldq xmm2, xmm3
		0x66, 0x45, 0x0f, 0x38, 0x40, 0xc1, // pmulld xmm8, xmm9
		0x66, 0x45, 0x0f, 0x38, 0x28, 0xd3, // pmuldq xmm10, xmm11
		0x66, 0x44, 0x0f, 0x38, 0x40, 0x27, // pmulld xmm12, [rdi]
		0x66, 0x44, 0x0f, 0x38, 0x28, 0x6f, 0x10, // pmuldq xmm13, [rdi+16]
		0xf4,
	}
	if err := memory.Map(0x65000, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(0x65000, code); err != nil {
		t.Fatal(err)
	}
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("PMULLD/PMULDQ run trap=%#x halted=%v", trap, state.Halted)
	}

	for lane, want := range []uint32{0xfffffff6, 0xffffffeb, 0xfffffffe, 0x80000000} {
		if got := binary.LittleEndian.Uint32(state.XMM[0][lane*4:]); got != want {
			t.Fatalf("PMULLD xmm0 lane %d=%#x, want %#x", lane, got, want)
		}
	}
	if got := binary.LittleEndian.Uint64(state.XMM[2][0:]); got != 6 {
		t.Fatalf("PMULDQ xmm2 low=%#x, want 6", got)
	}
	if got := binary.LittleEndian.Uint64(state.XMM[2][8:]); got != 0xffffffff00000002 {
		t.Fatalf("PMULDQ xmm2 high=%#x, want %#x", got, uint64(0xffffffff00000002))
	}
	if got := binary.LittleEndian.Uint32(state.XMM[8][0:]); got != 0 {
		t.Fatalf("PMULLD xmm8 lane0=%#x, want 0", got)
	}
	if got := binary.LittleEndian.Uint32(state.XMM[8][4:]); got != 0xffffff71 {
		t.Fatalf("PMULLD xmm8 lane1=%#x, want %#x", got, uint32(0xffffff71))
	}
	if got := binary.LittleEndian.Uint32(state.XMM[8][8:]); got != 0xc0000000 {
		t.Fatalf("PMULLD xmm8 lane2=%#x, want %#x", got, uint32(0xc0000000))
	}
	if got := binary.LittleEndian.Uint32(state.XMM[8][12:]); got != 14 {
		t.Fatalf("PMULLD xmm8 lane3=%#x, want 14", got)
	}
	if got := binary.LittleEndian.Uint64(state.XMM[10][0:]); got != 0xffffffffffffffe5 {
		t.Fatalf("PMULDQ xmm10 low=%#x, want %#x", got, uint64(0xffffffffffffffe5))
	}
	if got := binary.LittleEndian.Uint64(state.XMM[10][8:]); got != 0x80000000 {
		t.Fatalf("PMULDQ xmm10 high=%#x, want 0x80000000", got)
	}
	for lane, want := range []uint32{4, 9, 1, 0} {
		if got := binary.LittleEndian.Uint32(state.XMM[12][lane*4:]); got != want {
			t.Fatalf("PMULLD memory lane %d=%#x, want %#x", lane, got, want)
		}
	}
	if got := binary.LittleEndian.Uint64(state.XMM[13][0:]); got != 6 {
		t.Fatalf("PMULDQ memory low=%#x, want 6", got)
	}
	if got := binary.LittleEndian.Uint64(state.XMM[13][8:]); got != 0xc000000080000000 {
		t.Fatalf("PMULDQ memory high=%#x, want %#x", got, uint64(0xc000000080000000))
	}
}

func TestJIT64SSEFloatComparePredicates(t *testing.T) {
	const (
		codeAddress Address64 = 0x6a000
		dataAddress Address64 = 0x6b000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	// CMPPS xmm0,xmm1,EQ_OQ; CMPPD xmm2,xmm3,UNORD_Q;
	// CMPSS xmm4,xmm5,NEQ_UQ; CMPSD_XMM xmm6,[rdi],ORD_Q; hlt.
	code := []byte{
		0x0f, 0xc2, 0xc1, 0x00,
		0x66, 0x0f, 0xc2, 0xd3, 0x03,
		0xf3, 0x0f, 0xc2, 0xe5, 0x04,
		0xf2, 0x0f, 0xc2, 0x37, 0x07,
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	var memoryValue [8]byte
	binary.LittleEndian.PutUint64(memoryValue[:], math.Float64bits(math.NaN()))
	if err := memory.Write(dataAddress, memoryValue[:]); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	state.RFLAGS = Flag64CF | Flag64ZF | Flag64SF
	flagsBefore := state.RFLAGS
	putFloat32 := func(index uint8, values ...float32) {
		var raw [16]byte
		for lane, value := range values {
			binary.LittleEndian.PutUint32(raw[lane*4:], math.Float32bits(value))
		}
		copy(state.XMM[index][:], raw[:])
	}
	putFloat64 := func(index uint8, values ...float64) {
		var raw [16]byte
		for lane, value := range values {
			binary.LittleEndian.PutUint64(raw[lane*8:], math.Float64bits(value))
		}
		copy(state.XMM[index][:], raw[:])
	}
	putFloat32(0, 1, 2, float32(math.NaN()), -1)
	putFloat32(1, 1, 3, 0, -2)
	putFloat64(2, 1, math.NaN())
	putFloat64(3, 2, 0)
	putFloat32(4, 1)
	putFloat32(5, 2)
	putFloat64(6, -1)
	for i := 4; i < 16; i++ {
		state.XMM[4][i] = byte(0xa0 + i)
		state.XMM[6][i] = byte(0xb0 + i)
	}
	upper4 := state.XMM[4][4:]
	upper6 := state.XMM[6][8:]
	jit := NewJIT64(memory)
	trap := jit.RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	var want0 [16]byte
	binary.LittleEndian.PutUint32(want0[0:], math.Float32bits(float32(math.NaN())))
	binary.LittleEndian.PutUint32(want0[4:], 0)
	binary.LittleEndian.PutUint32(want0[8:], 0)
	binary.LittleEndian.PutUint32(want0[12:], 0)
	// The NaN lane is false for EQ_OQ; replace the first lane with true.
	binary.LittleEndian.PutUint32(want0[0:], ^uint32(0))
	if state.XMM[0] != want0 {
		t.Fatalf("CMPPS result=%x want=%x", state.XMM[0], want0)
	}
	var want2 [16]byte
	binary.LittleEndian.PutUint64(want2[0:], 0)
	binary.LittleEndian.PutUint64(want2[8:], ^uint64(0))
	if state.XMM[2] != want2 {
		t.Fatalf("CMPPD result=%x want=%x", state.XMM[2], want2)
	}
	if got := binary.LittleEndian.Uint32(state.XMM[4][:]); got != ^uint32(0) {
		t.Fatalf("CMPSS low=%#x want all ones", got)
	}
	if !bytes.Equal(state.XMM[4][4:], upper4) {
		t.Fatalf("CMPSS upper=%x want=%x", state.XMM[4][4:], upper4)
	}
	if got := binary.LittleEndian.Uint64(state.XMM[6][:]); got != 0 {
		t.Fatalf("CMPSD_XMM low=%#x want zero for unordered ORD_Q", got)
	}
	if !bytes.Equal(state.XMM[6][8:], upper6) {
		t.Fatalf("CMPSD_XMM upper=%x want=%x", state.XMM[6][8:], upper6)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("CMP predicates changed flags: got=%#x want=%#x", state.RFLAGS, flagsBefore)
	}
}

func TestJIT64PCMPGTQAndPACKUSDW(t *testing.T) {
	const (
		codeAddress Address64 = 0x5e000
		dataAddress Address64 = 0x5f000
	)
	memory := NewMemory64()
	if err := memory.Map(codeAddress, Page64Size, PRead|PWrite|PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	// PCMPGTQ xmm0,xmm1; PCMPGTQ xmm8,[rdi];
	// PACKUSDW xmm2,xmm3; PACKUSDW xmm9,[rdi].
	code := []byte{
		0x66, 0x0f, 0x38, 0x37, 0xc1,
		0x66, 0x44, 0x0f, 0x38, 0x37, 0x07,
		0x66, 0x0f, 0x38, 0x2b, 0xd3,
		0x66, 0x44, 0x0f, 0x38, 0x2b, 0x0f,
		0xf4,
	}
	if err := memory.Write(codeAddress, code); err != nil {
		t.Fatal(err)
	}
	var source [16]byte
	binary.LittleEndian.PutUint32(source[0:], 2)
	binary.LittleEndian.PutUint32(source[4:], 0)
	binary.LittleEndian.PutUint32(source[8:], 65536)
	binary.LittleEndian.PutUint32(source[12:], 1)
	if err := memory.Write(dataAddress, source[:]); err != nil {
		t.Fatal(err)
	}

	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	binary.LittleEndian.PutUint64(state.XMM[0][0:], 3)
	binary.LittleEndian.PutUint64(state.XMM[0][8:], ^uint64(1))
	binary.LittleEndian.PutUint64(state.XMM[1][0:], 2)
	binary.LittleEndian.PutUint64(state.XMM[1][8:], ^uint64(0))
	binary.LittleEndian.PutUint64(state.XMM[8][0:], 3)
	binary.LittleEndian.PutUint64(state.XMM[8][8:], ^uint64(1))
	for i, value := range []int32{-1, 0, 65536, 65535} {
		binary.LittleEndian.PutUint32(state.XMM[2][i*4:], uint32(value))
	}
	for i, value := range []int32{1, 2, -2, 100000} {
		binary.LittleEndian.PutUint32(state.XMM[3][i*4:], uint32(value))
	}
	for i, value := range []int32{-3, 0, 65536, 7} {
		binary.LittleEndian.PutUint32(state.XMM[9][i*4:], uint32(value))
	}
	state.RFLAGS = Flag64IF | Flag64CF | Flag64ZF | Flag64SF
	flagsBefore := state.RFLAGS

	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("SIMD instructions changed RFLAGS from %#x to %#x", flagsBefore, state.RFLAGS)
	}
	if got := binary.LittleEndian.Uint64(state.XMM[0][0:]); got != ^uint64(0) || binary.LittleEndian.Uint64(state.XMM[0][8:]) != 0 {
		t.Fatalf("PCMPGTQ register=%x, want low all-ones/high zero", state.XMM[0])
	}
	if got := binary.LittleEndian.Uint64(state.XMM[8][0:]); got != ^uint64(0) || binary.LittleEndian.Uint64(state.XMM[8][8:]) != 0 {
		t.Fatalf("PCMPGTQ memory/REX.R=%x, want low all-ones/high zero", state.XMM[8])
	}
	wantRegister := []uint16{0, 0, 65535, 65535, 1, 2, 0, 65535}
	for lane, want := range wantRegister {
		if got := binary.LittleEndian.Uint16(state.XMM[2][lane*2:]); got != want {
			t.Fatalf("PACKUSDW register lane %d=%d, want %d", lane, got, want)
		}
	}
	wantMemory := []uint16{0, 0, 65535, 7, 2, 0, 65535, 1}
	for lane, want := range wantMemory {
		if got := binary.LittleEndian.Uint16(state.XMM[9][lane*2:]); got != want {
			t.Fatalf("PACKUSDW memory/REX.R lane %d=%d, want %d", lane, got, want)
		}
	}
}

func TestJIT64MOVNTDQAFromMemory(t *testing.T) {
	const (
		codeAddress Address64 = 0x60000
		dataAddress Address64 = 0x61000
	)
	memory := NewMemory64()
	mapExecutable64(t, memory, codeAddress, []byte{
		0x66, 0x44, 0x0f, 0x38, 0x2a, 0x07, // movntdqa xmm8,[rdi]
		0xf4, // hlt
	})
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	input := [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	if err := memory.Write(dataAddress, input[:]); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	for i := range state.XMM[8] {
		state.XMM[8][i] = 0xa5
	}
	state.RFLAGS = Flag64IF | Flag64CF | Flag64ZF | Flag64SF
	flagsBefore := state.RFLAGS
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.XMM[8] != input {
		t.Fatalf("MOVNTDQA xmm8=%x, want %x", state.XMM[8], input)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("MOVNTDQA changed RFLAGS from %#x to %#x", flagsBefore, state.RFLAGS)
	}
	var sourceAfter [16]byte
	if err := memory.Read(dataAddress, sourceAfter[:]); err != nil {
		t.Fatal(err)
	}
	if sourceAfter != input {
		t.Fatalf("MOVNTDQA modified source=%x, want %x", sourceAfter, input)
	}
}

func TestJIT64HalfRegisterMoves(t *testing.T) {
	const (
		codeAddress Address64 = 0x62000
		dataAddress Address64 = 0x63000
	)
	memory := NewMemory64()
	code := []byte{
		0x0f, 0x12, 0xc1, // movhlps xmm0,xmm1
		0x0f, 0x16, 0xc2, // movlhps xmm0,xmm2
		0x0f, 0x16, 0x5f, 0x10, // movhps xmm3,[rdi+0x10]
		0x0f, 0x12, 0x67, 0x18, // movlps xmm4,[rdi+0x18]
		0x66, 0x0f, 0x16, 0x6f, 0x20, // movhpd xmm5,[rdi+0x20]
		0x66, 0x0f, 0x12, 0x77, 0x28, // movlpd xmm6,[rdi+0x28]
		0x0f, 0x17, 0x47, 0x30, // movhps [rdi+0x30],xmm0
		0x0f, 0x13, 0x4f, 0x38, // movlps [rdi+0x38],xmm1
		0x66, 0x0f, 0x17, 0x57, 0x40, // movhpd [rdi+0x40],xmm2
		0x66, 0x0f, 0x13, 0x5f, 0x48, // movlpd [rdi+0x48],xmm3
		0x44, 0x0f, 0x16, 0x07, // movhps xmm8,[rdi] (REX.R)
		0x44, 0x0f, 0x12, 0x4f, 0x08, // movlps xmm9,[rdi+8] (REX.R)
		0x45, 0x0f, 0x12, 0xca, // movhlps xmm9,xmm10 (REX.R+B)
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	var data [128]byte
	for i := range data {
		data[i] = byte(0xa0 + i)
	}
	if err := memory.Write(dataAddress, data[:]); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	for register := 0; register <= 10; register++ {
		for lane := range state.XMM[register] {
			state.XMM[register][lane] = byte(register*0x10 + lane)
		}
	}
	state.RFLAGS = Flag64IF | Flag64CF | Flag64ZF | Flag64SF
	flagsBefore := state.RFLAGS
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("half-register moves changed RFLAGS from %#x to %#x", flagsBefore, state.RFLAGS)
	}

	var want0 [16]byte
	copy(want0[:], []byte{0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27})
	if state.XMM[0] != want0 {
		t.Fatalf("MOVHLPS/MOVLHPS xmm0=%x, want %x", state.XMM[0], want0)
	}
	var want3 [16]byte
	for i := 0; i < 8; i++ {
		want3[i] = byte(0x30 + i)
		want3[8+i] = byte(0xb0 + i)
	}
	if state.XMM[3] != want3 {
		t.Fatalf("MOVHPS xmm3=%x, want %x", state.XMM[3], want3)
	}
	var want4 [16]byte
	for i := 0; i < 8; i++ {
		want4[i] = byte(0xb8 + i)
		want4[8+i] = byte(0x48 + i)
	}
	if state.XMM[4] != want4 {
		t.Fatalf("MOVLPS xmm4=%x, want %x", state.XMM[4], want4)
	}
	var want5 [16]byte
	for i := 0; i < 8; i++ {
		want5[i] = byte(0x50 + i)
		want5[8+i] = byte(0xc0 + i)
	}
	if state.XMM[5] != want5 {
		t.Fatalf("MOVHPD xmm5=%x, want %x", state.XMM[5], want5)
	}
	var want6 [16]byte
	for i := 0; i < 8; i++ {
		want6[i] = byte(0xc8 + i)
		want6[8+i] = byte(0x68 + i)
	}
	if state.XMM[6] != want6 {
		t.Fatalf("MOVLPD xmm6=%x, want %x", state.XMM[6], want6)
	}
	for offset, want := range map[int][]byte{
		0x30: {0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27},
		0x38: {0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17},
		0x40: {0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f},
		0x48: {0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37},
	} {
		got := make([]byte, 8)
		if err := memory.Read(dataAddress+Address64(offset), got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("half move store at %#x=%x, want %x", offset, got, want)
		}
	}
	var want8 [16]byte
	for i := 0; i < 8; i++ {
		want8[i] = byte(0x80 + i)
		want8[8+i] = byte(0xa0 + i)
	}
	if state.XMM[8] != want8 {
		t.Fatalf("REX.R MOVHPS xmm8=%x, want %x", state.XMM[8], want8)
	}
	var want9 [16]byte
	for i := 0; i < 8; i++ {
		want9[i] = byte(0xa8 + i)
		want9[8+i] = byte(0x98 + i)
	}
	if state.XMM[9] != want9 {
		t.Fatalf("REX.R/B MOVLPS/MOVHLPS xmm9=%x, want %x", state.XMM[9], want9)
	}
}

func TestJIT64PBLENDVB(t *testing.T) {
	const (
		codeAddress Address64 = 0x64000
		dataAddress Address64 = 0x65000
	)
	memory := NewMemory64()
	code := []byte{
		0x66, 0x0f, 0x38, 0x10, 0xd3, // pblendvb xmm2,xmm3
		0x66, 0x44, 0x0f, 0x38, 0x10, 0x07, // pblendvb xmm8,[rdi] (REX.R)
		0x66, 0x44, 0x0f, 0x38, 0x10, 0x4f, 0x10, // pblendvb xmm9,[rdi+0x10] (REX.R)
		0x66, 0x0f, 0x38, 0x10, 0xc1, // pblendvb xmm0,xmm1 (implicit mask aliases destination)
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	var data [64]byte
	for i := range data {
		data[i] = byte(0x40 + i)
	}
	if err := memory.Write(dataAddress, data[:]); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Set(RDI, uint64(dataAddress))
	for lane := range state.XMM[1] {
		state.XMM[1][lane] = byte(0x20 + lane)
		state.XMM[2][lane] = byte(0xa0 + lane)
		state.XMM[3][lane] = byte(0xb0 + lane)
		state.XMM[8][lane] = byte(0xc0 + lane)
		state.XMM[9][lane] = byte(0xd0 + lane)
		if lane%2 == 0 {
			state.XMM[0][lane] = 0x80
		}
	}
	state.RFLAGS = Flag64IF | Flag64CF | Flag64ZF | Flag64SF
	flagsBefore := state.RFLAGS
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("PBLENDVB changed RFLAGS from %#x to %#x", flagsBefore, state.RFLAGS)
	}
	var want2 [16]byte
	var want8 [16]byte
	var want9 [16]byte
	var want0 [16]byte
	for lane := 0; lane < 16; lane++ {
		if lane%2 == 0 {
			want2[lane] = byte(0xb0 + lane)
			want8[lane] = byte(0x40 + lane)
			want9[lane] = byte(0x50 + lane)
			want0[lane] = byte(0x20 + lane)
		}
	}
	for lane := 1; lane < 16; lane += 2 {
		want2[lane] = byte(0xa0 + lane)
		want8[lane] = byte(0xc0 + lane)
		want9[lane] = byte(0xd0 + lane)
	}
	if state.XMM[2] != want2 {
		t.Fatalf("PBLENDVB register result=%x, want %x", state.XMM[2], want2)
	}
	if state.XMM[8] != want8 {
		t.Fatalf("PBLENDVB memory REX.R result=%x, want %x", state.XMM[8], want8)
	}
	if state.XMM[9] != want9 {
		t.Fatalf("PBLENDVB memory REX.R+disp result=%x, want %x", state.XMM[9], want9)
	}
	if state.XMM[0] != want0 {
		t.Fatalf("PBLENDVB XMM0 alias result=%x, want %x", state.XMM[0], want0)
	}
}

func TestJIT64MOVDQ2QAndMOVQ2DQ(t *testing.T) {
	const codeAddress Address64 = 0x67000
	memory := NewMemory64()
	code := []byte{
		0xf2, 0x0f, 0xd6, 0xc1, // movdq2q mm0,xmm1
		0xf3, 0x0f, 0xd6, 0xd0, // movq2dq xmm2,mm0
		0xf2, 0x41, 0x0f, 0xd6, 0xc8, // movdq2q mm1,xmm8 (REX.B)
		0xf3, 0x44, 0x0f, 0xd6, 0xc9, // movq2dq xmm9,mm1 (REX.R)
		0xf4, // hlt
	}
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	for lane := uint8(0); lane < 8; lane++ {
		state.XMM[1][lane] = 0x10 + lane
		state.XMM[1][8+lane] = 0xa0 + lane
		state.XMM[8][lane] = 0x20 + lane
		state.XMM[8][8+lane] = 0xb0 + lane
	}
	for lane := uint8(8); lane < 16; lane++ {
		state.XMM[2][lane] = 0xc0 + lane
		state.XMM[9][lane] = 0xd0 + lane
	}
	state.MMX[0] = 0x1111111111111111
	state.MMX[1] = 0x2222222222222222
	state.SetFPUTop(5)
	state.RFLAGS = Flag64IF | Flag64CF | Flag64ZF | Flag64SF
	flagsBefore := state.RFLAGS
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("MMX bridge changed RFLAGS from %#x to %#x", flagsBefore, state.RFLAGS)
	}
	if got, want := state.MMX[0], uint64(0x1716151413121110); got != want {
		t.Fatalf("MMX0=%#x, want %#x", got, want)
	}
	if got, want := state.MMX[1], uint64(0x2726252423222120); got != want {
		t.Fatalf("MMX1=%#x, want %#x", got, want)
	}
	var want2 [16]byte
	var want9 [16]byte
	for lane := uint8(0); lane < 8; lane++ {
		want2[lane] = 0x10 + lane
		want9[lane] = 0x20 + lane
	}
	if state.XMM[2] != want2 {
		t.Fatalf("XMM2=%x, want %x", state.XMM[2], want2)
	}
	if state.XMM[9] != want9 {
		t.Fatalf("XMM9=%x, want %x", state.XMM[9], want9)
	}
	if got := state.FPUTop(); got != 0 {
		t.Fatalf("FPU TOP=%d after MMX transition, want 0", got)
	}
}

func TestJIT64PSHUFW(t *testing.T) {
	const codeAddress Address64 = 0x68000
	const dataAddress Address64 = 0x6a000
	memory := NewMemory64()
	code := []byte{
		0x0f, 0x70, 0xc1, 0x1b, // pshufw mm0,mm1,0x1b: reverse words
		0x0f, 0x70, 0xc9, 0x1b, // pshufw mm1,mm1,0x1b: aliasing source/destination
		0x49, 0xbf, 0, 0, 0, 0, 0, 0, 0, 0, // mov r15,dataAddress
		0x41, 0x0f, 0x70, 0x17, 0xb1, // pshufw mm2,[r15],0xb1
		0xf4, // hlt
	}
	binary.LittleEndian.PutUint64(code[10:], uint64(dataAddress))
	mapExecutable64(t, memory, codeAddress, code)
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	var raw [8]byte
	binary.LittleEndian.PutUint16(raw[0:2], 0x1101)
	binary.LittleEndian.PutUint16(raw[2:4], 0x2202)
	binary.LittleEndian.PutUint16(raw[4:6], 0x3303)
	binary.LittleEndian.PutUint16(raw[6:8], 0x4404)
	if err := memory.Write(dataAddress, raw[:]); err != nil {
		t.Fatal(err)
	}
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.MMX[1] = 0x4404330322021101
	state.MMX[0] = 0xaaaaaaaaaaaaaaaa
	state.MMX[2] = 0xbbbbbbbbbbbbbbbb
	state.SetFPUTop(6)
	state.RFLAGS = Flag64OF | Flag64AF | Flag64PF | Flag64CF
	flagsBefore := state.RFLAGS
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("PSHUFW changed RFLAGS from %#x to %#x", flagsBefore, state.RFLAGS)
	}
	if got, want := state.MMX[0], uint64(0x1101220233034404); got != want {
		t.Fatalf("MMX0=%#x, want %#x", got, want)
	}
	if got, want := state.MMX[1], uint64(0x1101220233034404); got != want {
		t.Fatalf("MMX1 alias result=%#x, want %#x", got, want)
	}
	if got, want := state.MMX[2], uint64(0x3303440411012202); got != want {
		t.Fatalf("MMX2 memory result=%#x, want %#x", got, want)
	}
	if got := state.FPUTop(); got != 0 {
		t.Fatalf("FPU TOP=%d after PSHUFW, want 0", got)
	}
}

func TestJIT64PauseAndPrefetchHints(t *testing.T) {
	const codeAddress Address64 = 0x6c000
	const unmappedAddress Address64 = 0x7f0000000000
	memory := NewMemory64()
	code := []byte{
		0x48, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, // mov rax,unmappedAddress
		0xf3, 0x90, // pause
		0x0f, 0x18, 0x00, // prefetchnta [rax]
		0x0f, 0x18, 0x08, // prefetcht0 [rax]
		0x0f, 0x18, 0x10, // prefetcht1 [rax]
		0x0f, 0x18, 0x18, // prefetcht2 [rax]
		0x0f, 0x0d, 0x08, // prefetchw [rax]
		0xf4, // hlt
	}
	binary.LittleEndian.PutUint64(code[2:], uint64(unmappedAddress))
	mapExecutable64(t, memory, codeAddress, code)
	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.Regs[RDI] = 0x1122334455667788
	state.XMM[3] = [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	state.MMX[3] = 0x8877665544332211
	state.SetFPUTop(4)
	state.RFLAGS = Flag64OF | Flag64SF | Flag64AF | Flag64PF | Flag64CF | Flag64ZF
	flagsBefore := state.RFLAGS
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("hints changed RFLAGS from %#x to %#x", flagsBefore, state.RFLAGS)
	}
	if state.Get(RAX) != uint64(unmappedAddress) {
		t.Fatalf("RAX=%#x, want %#x", state.Get(RAX), unmappedAddress)
	}
	if state.Get(RDI) != 0x1122334455667788 {
		t.Fatalf("RDI changed to %#x", state.Get(RDI))
	}
	if state.XMM[3] != [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} {
		t.Fatalf("XMM3 changed to %x", state.XMM[3])
	}
	if state.MMX[3] != 0x8877665544332211 {
		t.Fatalf("MMX3 changed to %#x", state.MMX[3])
	}
	if state.FPUTop() != 4 {
		t.Fatalf("FPU TOP changed to %d, want 4", state.FPUTop())
	}
}

func TestJIT64NonTemporalStores(t *testing.T) {
	const codeAddress Address64 = 0x74000
	const dataAddress Address64 = 0x76000
	memory := NewMemory64()
	code := []byte{
		0x49, 0xbf, 0, 0, 0, 0, 0, 0, 0, 0, // mov r15,dataAddress
		0xb8, 0xd4, 0xc3, 0xb2, 0xa1, // mov eax,0xa1b2c3d4
		0x41, 0x0f, 0xc3, 0x07, // movnti [r15],eax
		0x48, 0xb8, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, // mov rax,0x1122334455667788
		0x49, 0x0f, 0xc3, 0x47, 0x08, // movnti [r15+8],rax
		0x41, 0x0f, 0xe7, 0x47, 0x10, // movntq [r15+16],mm0
		0x66, 0x41, 0x0f, 0x2b, 0x47, 0x18, // movntpd [r15+24],xmm0
		0xf2, 0x41, 0x0f, 0x2b, 0x4f, 0x28, // movntsd [r15+40],xmm1
		0xf3, 0x41, 0x0f, 0x2b, 0x57, 0x30, // movntss [r15+48],xmm2
		0xf4, // hlt
	}
	binary.LittleEndian.PutUint64(code[2:], uint64(dataAddress))
	if err := memory.Map(dataAddress, Page64Size, PRead|PWrite); err != nil {
		t.Fatal(err)
	}
	initial := make([]byte, 64)
	for i := range initial {
		initial[i] = 0xaa
	}
	if err := memory.Write(dataAddress, initial); err != nil {
		t.Fatal(err)
	}
	mapExecutable64(t, memory, codeAddress, code)

	state := NewMachineState64(memory)
	state.RIP = uint64(codeAddress)
	state.MMX[0] = 0x8877665544332211
	state.XMM[0] = [16]byte{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f}
	state.XMM[1] = [16]byte{0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f}
	state.XMM[2] = [16]byte{0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f}
	state.SetFPUTop(5)
	state.RFLAGS = Flag64OF | Flag64AF | Flag64PF | Flag64ZF | Flag64SF | Flag64CF
	flagsBefore := state.RFLAGS
	mmxBefore := state.MMX[0]
	xmm0Before := state.XMM[0]
	xmm1Before := state.XMM[1]
	xmm2Before := state.XMM[2]
	trap := NewJIT64(memory).RunToInterrupt(state)
	if trap != Trap64Timer || !state.Halted {
		t.Fatalf("trap=%#x halted=%v rip=%#x", trap, state.Halted, state.RIP)
	}
	if state.RFLAGS != flagsBefore {
		t.Fatalf("non-temporal stores changed RFLAGS from %#x to %#x", flagsBefore, state.RFLAGS)
	}
	if state.MMX[0] != mmxBefore || state.XMM[0] != xmm0Before || state.XMM[1] != xmm1Before || state.XMM[2] != xmm2Before {
		t.Fatal("non-temporal stores changed a source register")
	}
	if state.Get(RAX) != 0x1122334455667788 || state.Get(R15) != uint64(dataAddress) {
		t.Fatalf("GPR state changed: RAX=%#x R15=%#x", state.Get(RAX), state.Get(R15))
	}
	if got := state.FPUTop(); got != 0 {
		t.Fatalf("MOVNTQ did not enter MMX mode: TOP=%d", got)
	}

	var got [64]byte
	if err := memory.Read(dataAddress, got[:]); err != nil {
		t.Fatal(err)
	}
	want := [64]byte{}
	for i := range want {
		want[i] = 0xaa
	}
	binary.LittleEndian.PutUint32(want[0:4], 0xa1b2c3d4)
	binary.LittleEndian.PutUint64(want[8:16], 0x1122334455667788)
	binary.LittleEndian.PutUint64(want[16:24], 0x8877665544332211)
	copy(want[24:40], xmm0Before[:])
	copy(want[40:48], xmm1Before[:8])
	copy(want[48:52], xmm2Before[:4])
	if got != want {
		t.Fatalf("non-temporal store memory=%x, want %x", got, want)
	}
}
