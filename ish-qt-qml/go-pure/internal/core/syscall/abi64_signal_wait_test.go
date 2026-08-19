package syscall

import (
	"encoding/binary"
	"testing"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64SignalPendingAndTimedWait(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 42
	const (
		setAddress     corecpu.Address64 = 0x10000
		pendingAddress corecpu.Address64 = 0x10008
		infoAddress    corecpu.Address64 = 0x10010
		timeoutAddress corecpu.Address64 = 0x10090
	)
	var set [8]byte
	binary.LittleEndian.PutUint64(set[:], uint64(1)<<(10-1))
	if err := memory.Write(setAddress, set[:]); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(timeoutAddress, make([]byte, 16)); err != nil {
		t.Fatal(err)
	}

	set64Syscall(state, Sys64Kill, ctx.PID, 10)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("queue signal: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64RtSigpending, uint64(pendingAddress), sigSetSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("rt_sigpending: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var pending [8]byte
	if err := memory.Read(pendingAddress, pending[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(pending[:]); got != uint64(1)<<(10-1) {
		t.Fatalf("pending mask=%#x", got)
	}

	set64Syscall(state, Sys64RtSigtimedwait, uint64(setAddress), uint64(infoAddress), uint64(timeoutAddress), sigSetSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 10 {
		t.Fatalf("rt_sigtimedwait: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var info [128]byte
	if err := memory.Read(infoAddress, info[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(info[:4]); got != 10 {
		t.Fatalf("siginfo signo=%d", got)
	}
	set64Syscall(state, Sys64RtSigpending, uint64(pendingAddress), sigSetSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("rt_sigpending after consume: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(pendingAddress, pending[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(pending[:]); got != 0 {
		t.Fatalf("pending after consume=%#x", got)
	}
}

func TestABI64SignalSuspendRestoresMask(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 43
	const maskAddress corecpu.Address64 = 0x10000
	var mask [8]byte
	if err := memory.Write(maskAddress, mask[:]); err != nil {
		t.Fatal(err)
	}
	ctx.SignalMask = uint64(1) << (12 - 1)
	ctx.SignalMu.Lock()
	ctx.PendingSignals = 0
	ctx.SignalMu.Unlock()

	result := make(chan struct {
		resume bool
		err    error
		value  int64
	}, 1)
	go func() {
		set64Syscall(state, Sys64RtSigsuspend, uint64(maskAddress), sigSetSize64)
		resume, err := dispatcher.Dispatch(state)
		result <- struct {
			resume bool
			err    error
			value  int64
		}{resume, err, int64(state.Get(corecpu.RAX))}
	}()
	select {
	case <-result:
		t.Fatal("rt_sigsuspend returned before a signal")
	case <-time.After(15 * time.Millisecond):
	}
	queueSignal64(ctx, 12)
	select {
	case got := <-result:
		if got.err != nil || !got.resume || got.value != int64(EINTR) {
			t.Fatalf("rt_sigsuspend: resume=%v err=%v rax=%d", got.resume, got.err, got.value)
		}
	case <-time.After(time.Second):
		t.Fatal("rt_sigsuspend did not wake")
	}
	if ctx.SignalMask != uint64(1)<<(12-1) {
		t.Fatalf("signal mask after suspend=%#x", ctx.SignalMask)
	}
}

func TestABI64PauseRequiresSignal(t *testing.T) {
	_, _, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 44
	ctx.SignalMask = 0
	result := make(chan int64, 1)
	go func() {
		set64Syscall(state, Sys64Pause)
		_, _ = dispatcher.Dispatch(state)
		result <- int64(state.Get(corecpu.RAX))
	}()
	select {
	case <-result:
		t.Fatal("pause returned while all signals were blocked")
	case <-time.After(15 * time.Millisecond):
	}
	queueSignal64(ctx, 2)
	select {
	case got := <-result:
		if got != int64(EINTR) {
			t.Fatalf("pause rax=%d, want %d", got, EINTR)
		}
	case <-time.After(time.Second):
		t.Fatal("pause did not wake")
	}
}
