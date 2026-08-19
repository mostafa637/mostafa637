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
