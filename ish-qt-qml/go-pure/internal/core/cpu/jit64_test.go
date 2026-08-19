package cpu

import (
	"math"
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
