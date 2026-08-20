package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64ExitPublishesTerminationState(t *testing.T) {
	memory := corecpu.NewMemory64()
	ctx := NewContext64(memory)
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)
	set64Syscall(state, Sys64Exit, 7, 0, 0, 0, 0, 0)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || resume || !ctx.Exited || ctx.ExitCode != 7 || !state.Halted {
		t.Fatalf("exit: resume=%v err=%v exited=%v code=%d halted=%v", resume, err, ctx.Exited, ctx.ExitCode, state.Halted)
	}
	if int64(state.Get(corecpu.RAX)) != 7 {
		t.Fatalf("exit return register=%d", int64(state.Get(corecpu.RAX)))
	}

	ctx.Exited = false
	ctx.ExitCode = 0
	state.Halted = false
	set64Syscall(state, Sys64ExitGroup, 9, 0, 0, 0, 0, 0)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || resume || !ctx.Exited || ctx.ExitCode != 9 || !state.Halted {
		t.Fatalf("exit_group: resume=%v err=%v exited=%v code=%d halted=%v", resume, err, ctx.Exited, ctx.ExitCode, state.Halted)
	}
}
