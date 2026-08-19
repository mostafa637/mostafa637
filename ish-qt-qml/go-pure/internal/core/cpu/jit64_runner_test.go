package cpu

import "testing"

func TestJIT64RunToInterruptUsesFallback(t *testing.T) {
	memory := NewMemory64()
	const start Address64 = 0x3000
	// xchg rax,rbx; pause; rdtsc; syscall
	code := []byte{0x48, 0x93, 0xf3, 0x90, 0x0f, 0x31, 0x0f, 0x05}
	mapExecutable64(t, memory, start, code)
	jit := NewJIT64(memory)
	state := NewMachineState64(memory)
	state.RIP = uint64(start)
	state.Set(RAX, 11)
	state.Set(RBX, 22)
	state.Cycle = 0x1122334455667788

	trap := jit.RunToInterrupt(state)
	if trap != Trap64Syscall {
		t.Fatalf("trap=%#x, want syscall", trap)
	}
	if state.Get(RAX) != 0x55667788 || state.Get(RBX) != 11 || state.Get(RDX) != 0x11223344 {
		t.Fatalf("fallback state rax=%#x rbx=%#x rdx=%#x", state.Get(RAX), state.Get(RBX), state.Get(RDX))
	}
	if state.RIP != uint64(start)+8 {
		t.Fatalf("rip=%#x, want %#x", state.RIP, uint64(start)+8)
	}
}

func TestJIT64PokeAndPageFault(t *testing.T) {
	memory := NewMemory64()
	const start Address64 = 0x4000
	mapExecutable64(t, memory, start, []byte{0x90, 0x0f, 0x05})
	jit := NewJIT64(memory)
	jit.Poke()
	state := NewMachineState64(memory)
	state.RIP = uint64(start)
	if trap := jit.RunToInterrupt(state); trap != Trap64Timer {
		t.Fatalf("poke trap=%#x, want timer", trap)
	}

	const faultStart Address64 = 0x5000
	// mov rax,[rip+0x1000]; syscall. The referenced page is deliberately absent.
	mapExecutable64(t, memory, faultStart, []byte{0x48, 0x8b, 0x05, 0x00, 0x10, 0x00, 0x00, 0x0f, 0x05})
	state = NewMachineState64(memory)
	state.RIP = uint64(faultStart)
	if trap := jit.RunToInterrupt(state); trap != Trap64PageFault {
		t.Fatalf("fault trap=%#x, want page fault", trap)
	}
}
