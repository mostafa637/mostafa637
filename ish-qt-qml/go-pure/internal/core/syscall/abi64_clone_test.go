package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64CloneFactoryAndParentTID(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 77
	const parentTID corecpu.Address64 = 0x10c00
	var got CloneRequest64
	called := false
	ctx.ProcessFactory = func(parent *Context64, request CloneRequest64) int64 {
		called = true
		got = request
		return 901
	}
	flags := cloneVM64 | cloneSighand64 | cloneParentTID64 | cloneSetTLS64
	set64Syscall(state, Sys64Clone, flags, 0x18000, uint64(parentTID), 0, 0xfeed0000, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 901 {
		t.Fatalf("clone: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if !called {
		t.Fatal("clone did not call process factory")
	}
	if got.Flags != flags || got.ChildStack != 0x18000 || got.ParentTID != uint64(parentTID) || got.TLS != 0xfeed0000 || got.Fork || got.VFork {
		t.Fatalf("unexpected clone request: %+v", got)
	}
	var raw [8]byte
	if err := memory.Read(parentTID, raw[:]); err != nil {
		t.Fatal(err)
	}
	if gotPID := binary.LittleEndian.Uint64(raw[:]); gotPID != 901 {
		t.Fatalf("parent tid=%d", gotPID)
	}
	if !ctx.Children.MarkExited(901, 0) {
		t.Fatal("child was not registered")
	}
	pid, _, _, errCode := ctx.Children.WaitID(uint32(ctx.PID), waitIDTypePID64, 901, WaitExited)
	if errCode != 0 || pid != 901 {
		t.Fatalf("registered child could not be waited: pid=%d err=%d", pid, errCode)
	}
}

func TestABI64CloneValidationAndFallback(t *testing.T) {
	_, _, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 77
	called := false
	ctx.ProcessFactory = func(parent *Context64, request CloneRequest64) int64 {
		called = true
		return 902
	}
	set64Syscall(state, Sys64Clone, uint64(1)<<63, 0, 0, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("unknown clone flag: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if called {
		t.Fatal("factory called for invalid flags")
	}
	set64Syscall(state, Sys64Clone, cloneSighand64, 0, 0, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("sighand without vm: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Clone, cloneThread64|cloneVM64, 0, 0, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("thread without sighand: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Clone, cloneParentTID64, 0, 0, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("null parent tid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	ctx.ProcessFactory = nil
	set64Syscall(state, Sys64Clone, 0, 0, 0, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ENOSYS) {
		t.Fatalf("missing factory: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64ForkAndVforkFactoryRequests(t *testing.T) {
	_, _, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 77
	requests := make([]CloneRequest64, 0, 2)
	ctx.ProcessFactory = func(parent *Context64, request CloneRequest64) int64 {
		requests = append(requests, request)
		return int64(910 + len(requests))
	}
	set64Syscall(state, Sys64Fork, 0, 0, 0, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 911 {
		t.Fatalf("fork: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Vfork, 0, 0, 0, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 912 {
		t.Fatalf("vfork: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if len(requests) != 2 || !requests[0].Fork || requests[0].VFork || requests[1].Fork || !requests[1].VFork {
		t.Fatalf("unexpected lifecycle requests: %+v", requests)
	}
}
