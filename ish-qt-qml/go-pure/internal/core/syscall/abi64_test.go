package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcher64RegisterABI(t *testing.T) {
	memory := corecpu.NewMemory64()
	const buffer corecpu.Address64 = 0x6000
	if err := memory.Map(buffer, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context := NewContext64(memory)
	context.PID = 1234
	context.TID = 1235
	dispatcher := NewDispatcher64(context)
	state := corecpu.NewMachineState64(memory)

	state.Set(corecpu.RAX, uint64(Sys64GetPID))
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != 1234 {
		t.Fatalf("getpid: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}

	state.Set(corecpu.RAX, uint64(Sys64Getrandom))
	state.Set(corecpu.RDI, uint64(buffer))
	state.Set(corecpu.RSI, 32)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != 32 {
		t.Fatalf("getrandom: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	var randomBytes [32]byte
	if err := memory.Read(buffer, randomBytes[:]); err != nil {
		t.Fatal(err)
	}

	state.Set(corecpu.RAX, uint64(Sys64Exit))
	state.Set(corecpu.RDI, 7)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || resume || !state.Halted || state.Get(corecpu.RAX) != 7 {
		t.Fatalf("exit: resume=%v err=%v halted=%v rax=%d", resume, err, state.Halted, state.Get(corecpu.RAX))
	}
}
