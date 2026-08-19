package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcher64IntegratedWithJIT64(t *testing.T) {
	memory := corecpu.NewMemory64()
	const start corecpu.Address64 = 0x7000
	// mov rax,39; syscall; mov rbx,rax; mov rax,60; mov rdi,9; syscall
	code := []byte{
		0x48, 0xc7, 0xc0, 39, 0, 0, 0,
		0x0f, 0x05,
		0x48, 0x89, 0xc3,
		0x48, 0xc7, 0xc0, 60, 0, 0, 0,
		0x48, 0xc7, 0xc7, 9, 0, 0, 0,
		0x0f, 0x05,
	}
	if err := memory.Map(start, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite|corecpu.PExec); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(start, code); err != nil {
		t.Fatal(err)
	}
	context := NewContext64(memory)
	context.PID = 4321
	dispatcher := NewDispatcher64(context)
	jit := corecpu.NewJIT64(memory)
	jit.OnSyscall64 = func(state *corecpu.MachineState64) (bool, error) {
		return dispatcher.Dispatch(state)
	}
	state := corecpu.NewMachineState64(memory)
	state.RIP = uint64(start)

	trap := jit.RunToInterrupt(state)
	if trap != corecpu.Trap64Exit || !state.Halted {
		t.Fatalf("trap=%#x halted=%v", trap, state.Halted)
	}
	if state.Get(corecpu.RBX) != 4321 || state.Get(corecpu.RAX) != 9 {
		t.Fatalf("registers after syscalls: rbx=%d rax=%d", state.Get(corecpu.RBX), state.Get(corecpu.RAX))
	}
}
