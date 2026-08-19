package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64TimeHandlers(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	const address corecpu.Address64 = 0x10400
	set64Syscall(state, Sys64Gettimeofday, uint64(address), uint64(address+16))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("gettimeofday: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var timeval [16]byte
	if err := memory.Read(address, timeval[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint64(timeval[0:8]) == 0 {
		t.Fatal("gettimeofday returned zero seconds")
	}

	set64Syscall(state, Sys64ClockGettime, 1, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("clock_gettime: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64ClockGetres, 1, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("clock_getres: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var resolution [16]byte
	if err := memory.Read(address, resolution[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint64(resolution[8:16]) != 1 {
		t.Fatalf("clock resolution nanos=%d", binary.LittleEndian.Uint64(resolution[8:16]))
	}

	set64Syscall(state, Sys64GetRUsage, 0, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("getrusage: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Times, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 0 {
		t.Fatalf("times: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if ctx.StartTime.IsZero() {
		t.Fatal("context start time was not initialized")
	}
}

func TestABI64SignalMaskAndAction(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	const (
		setAddress corecpu.Address64 = 0x10500
		oldAddress corecpu.Address64 = 0x10508
		actionAddr corecpu.Address64 = 0x10520
		oldActAddr corecpu.Address64 = 0x10540
	)
	var set [8]byte
	// SIGUSR1 (10) plus the unmaskable SIGKILL/SIGSTOP bits.
	binary.LittleEndian.PutUint64(set[:], (uint64(1)<<9)|(uint64(1)<<8)|(uint64(1)<<18))
	if err := memory.Write(setAddress, set[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64RtSigprocmask, sigBlock64, uint64(setAddress), uint64(oldAddress), sigSetSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("sigprocmask block: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var old [8]byte
	if err := memory.Read(oldAddress, old[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint64(old[:]) != 0 || ctx.SignalMask != uint64(1)<<9 {
		t.Fatalf("mask after block old=%#x current=%#x", binary.LittleEndian.Uint64(old[:]), ctx.SignalMask)
	}
	set64Syscall(state, Sys64RtSigprocmask, sigUnblock64, uint64(setAddress), uint64(oldAddress), sigSetSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("sigprocmask unblock: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(oldAddress, old[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint64(old[:]) != uint64(1)<<9 || ctx.SignalMask != 0 {
		t.Fatalf("mask after unblock old=%#x current=%#x", binary.LittleEndian.Uint64(old[:]), ctx.SignalMask)
	}

	action := make([]byte, 32)
	for index := range action {
		action[index] = byte(index + 1)
	}
	if err := memory.Write(actionAddr, action); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64RtSigaction, 10, uint64(actionAddr), 0, sigSetSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("sigaction set: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64RtSigaction, 10, 0, uint64(oldActAddr), sigSetSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("sigaction get: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var oldAction [32]byte
	if err := memory.Read(oldActAddr, oldAction[:]); err != nil {
		t.Fatal(err)
	}
	for index, value := range action {
		if oldAction[index] != value {
			t.Fatalf("action byte %d=%d, want %d", index, oldAction[index], value)
		}
	}
	set64Syscall(state, Sys64RtSigprocmask, sigSetMask64, uint64(setAddress), 0, 4)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("sigprocmask bad size: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
