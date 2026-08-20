package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64WaitIDExitedChild(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 77
	if !ctx.Children.AddChild(uint32(ctx.PID), 654) || !ctx.Children.MarkExited(654, 7) {
		t.Fatal("failed to create exited child")
	}
	const infoAddress corecpu.Address64 = 0x10900
	set64Syscall(state, Sys64Waitid, waitIDPID64, 654, uint64(infoAddress), uint64(waitIDExited64), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("waitid exited child: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var info [waitIDSiginfoSize64]byte
	if err := memory.Read(infoAddress, info[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(info[waitIDSiSignoOffset64:]); got != waitIDSigCHLD64 {
		t.Fatalf("siginfo signo=%d", got)
	}
	if got := binary.LittleEndian.Uint32(info[waitIDSiCodeOffset64:]); got != waitIDCLDExited64 {
		t.Fatalf("siginfo code=%d", got)
	}
	if got := binary.LittleEndian.Uint32(info[waitIDSiPIDOffset64:]); got != 654 {
		t.Fatalf("siginfo pid=%d", got)
	}
	if got := binary.LittleEndian.Uint32(info[waitIDSiStatusOffset64:]); got != 7 {
		t.Fatalf("siginfo status=%d", got)
	}
	set64Syscall(state, Sys64Waitid, waitIDPID64, 654, uint64(infoAddress), uint64(waitIDExited64|waitIDNoHang64), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ECHILD) {
		t.Fatalf("waitid reaped child: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64WaitIDNoHangAndNoWait(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 77
	if !ctx.Children.AddChild(uint32(ctx.PID), 655) || !ctx.Children.AddChild(uint32(ctx.PID), 656) {
		t.Fatal("failed to create children")
	}
	const infoAddress corecpu.Address64 = 0x10a00
	set64Syscall(state, Sys64Waitid, waitIDPID64, 656, uint64(infoAddress), uint64(waitIDExited64|waitIDNoHang64), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("waitid WNOHANG: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var info [waitIDSiginfoSize64]byte
	if err := memory.Read(infoAddress, info[:]); err != nil {
		t.Fatal(err)
	}
	if info != [waitIDSiginfoSize64]byte{} {
		t.Fatal("WNOHANG wrote non-zero siginfo for running child")
	}
	if !ctx.Children.MarkExited(656, 9) {
		t.Fatal("failed to exit child")
	}
	set64Syscall(state, Sys64Waitid, waitIDPID64, 656, uint64(infoAddress), uint64(waitIDExited64|waitIDNoWait64), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("waitid WNOWAIT: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Waitid, waitIDPID64, 656, uint64(infoAddress), uint64(waitIDExited64), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("waitid after WNOWAIT: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	if !ctx.Children.MarkExited(655, 3) {
		t.Fatal("failed to exit second child")
	}
	set64Syscall(state, Sys64Waitid, waitIDAll64, 0, uint64(infoAddress), uint64(waitIDExited64), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("waitid P_ALL: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64WaitIDValidation(t *testing.T) {
	_, _, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 77
	if !ctx.Children.AddChild(uint32(ctx.PID), 657) {
		t.Fatal("failed to create child")
	}
	set64Syscall(state, Sys64Waitid, waitIDPID64, 657, 0, uint64(waitIDExited64), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("null siginfo: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Waitid, 99, 657, 0x10b00, uint64(waitIDExited64), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid idtype: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Waitid, waitIDPID64, 657, 0x10b00, uint64(waitIDNoHang64), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("missing WEXITED: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Waitid, waitIDPID64, 657, 0x10b00, uint64(waitIDExited64), 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("non-zero rusage: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
