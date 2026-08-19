package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func dispatchDup64(t *testing.T, dispatcher *Dispatcher64, state *corecpu.MachineState64, number Number64, args ...uint64) int64 {
	t.Helper()
	set64Syscall(state, number, args...)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatalf("dispatch syscall %d: %v", number, err)
	}
	return int64(state.Get(corecpu.RAX))
}

func TestDup3ABI64CloexecIsPerDescriptor(t *testing.T) {
	memory := corecpu.NewMemory64()
	ctx := NewContext64(memory)
	source, err := ctx.FDs.Open(&corefd.File{})
	if err != nil {
		t.Fatal(err)
	}
	target, err := ctx.FDs.Open(&corefd.File{})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	if got := dispatchDup64(t, dispatcher, state, Sys64Dup3, uint64(source), 5, guestOpenCloexec); got != 5 {
		t.Fatalf("dup3 = %d, want 5", got)
	}
	if cloexec, err := ctx.FDs.Cloexec(source); err != nil || cloexec {
		t.Fatalf("source cloexec=%v err=%v, want false", cloexec, err)
	}
	if cloexec, err := ctx.FDs.Cloexec(5); err != nil || !cloexec {
		t.Fatalf("target cloexec=%v err=%v, want true", cloexec, err)
	}
	if got := dispatchDup64(t, dispatcher, state, Sys64Fcntl, 5, fcntlGetFD, 0); got != int64(fdCloexec) {
		t.Fatalf("F_GETFD dup3 target=%d, want %d", got, fdCloexec)
	}
	if got := dispatchDup64(t, dispatcher, state, Sys64Fcntl, uint64(source), fcntlGetFD, 0); got != 0 {
		t.Fatalf("F_GETFD dup3 source=%d, want 0", got)
	}

	removed := ctx.FDs.CloseOnExec()
	if len(removed) != 1 || removed[0] != 5 {
		t.Fatalf("close-on-exec removed=%v, want [5]", removed)
	}
	if _, err := ctx.FDs.Get(source); err != nil {
		t.Fatalf("source after close-on-exec: %v", err)
	}
	if _, err := ctx.FDs.Get(target); err != nil {
		t.Fatalf("unrelated target after close-on-exec: %v", err)
	}
	if _, err := ctx.FDs.Get(5); err != corefd.ErrBadFD {
		t.Fatalf("dup3 target lookup after close-on-exec=%v", err)
	}
}

func TestDup3ABI64ValidationAndDup2Cloexec(t *testing.T) {
	memory := corecpu.NewMemory64()
	ctx := NewContext64(memory)
	source, err := ctx.FDs.Open(&corefd.File{Cloexec: true})
	if err != nil {
		t.Fatal(err)
	}
	target, err := ctx.FDs.Open(&corefd.File{Cloexec: true})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	if got := dispatchDup64(t, dispatcher, state, Sys64Dup3, uint64(source), uint64(source), 0); got != int64(EINVAL) {
		t.Fatalf("dup3 same fd=%d, want %d", got, EINVAL)
	}
	if got := dispatchDup64(t, dispatcher, state, Sys64Dup3, uint64(source), 6, guestOpenCloexec|0x400); got != int64(EINVAL) {
		t.Fatalf("dup3 invalid flags=%d, want %d", got, EINVAL)
	}
	if got := dispatchDup64(t, dispatcher, state, Sys64Dup3, 999, 6, 0); got != int64(EBADF) {
		t.Fatalf("dup3 invalid source=%d, want %d", got, EBADF)
	}
	if got := dispatchDup64(t, dispatcher, state, Sys64Dup2, uint64(source), uint64(target)); got != int64(target) {
		t.Fatalf("dup2=%d, want %d", got, target)
	}
	if cloexec, err := ctx.FDs.Cloexec(source); err != nil || !cloexec {
		t.Fatalf("dup2 source cloexec=%v err=%v, want true", cloexec, err)
	}
	if cloexec, err := ctx.FDs.Cloexec(target); err != nil || cloexec {
		t.Fatalf("dup2 target cloexec=%v err=%v, want false", cloexec, err)
	}
}
