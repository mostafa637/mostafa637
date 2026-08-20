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

func TestABI64CloneStartsChildAfterRegistryAndCopiesState(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 77
	ctx.TID = 77
	ctx.Machine = state
	childMemory := memory.Clone()
	var request CloneRequest64
	started := false
	ctx.ProcessFactory = func(parent *Context64, got CloneRequest64) int64 {
		request = got
		child := parent.CloneForChild64(childMemory, 901)
		if child == nil {
			return int64(ENOMEM)
		}
		return 901
	}
	ctx.ChildStarter = func(parent *Context64, pid int64, got CloneRequest64) {
		started = true
		if pid != 901 || got.Flags != request.Flags {
			t.Fatalf("unexpected child starter args: pid=%d request=%+v", pid, got)
		}
		if childPID, _, err := parent.Children.Wait(uint32(parent.PID), int32(pid), WaitNoHang); err != 0 || childPID != 0 {
			t.Fatalf("child was not running in registry: pid=%d err=%d", childPID, err)
		}
		if !parent.Children.MarkExited(uint32(pid), 7) {
			t.Fatal("child exit was not published")
		}
	}
	flags := cloneVM64 | cloneSighand64 | cloneParentTID64
	set64Syscall(state, Sys64Clone, flags, 0x18000, 0x10c00, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 901 {
		t.Fatalf("clone: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if !started {
		t.Fatal("ChildStarter was not called")
	}
	childPID, status, err := ctx.Children.Wait(uint32(ctx.PID), 901, 0)
	if err != 0 || childPID != 901 || status != 7<<8 {
		t.Fatalf("wait child: pid=%d status=%d err=%d", childPID, status, err)
	}
	if request.ChildStack != 0x18000 || request.ParentTID != 0x10c00 {
		t.Fatalf("clone request = %+v", request)
	}
}

func TestContext64CloneForChildCopiesProcessState(t *testing.T) {
	memory := corecpu.NewMemory64()
	if err := memory.MapBytes(0x4000, []byte("parent"), corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	parent := NewContext64(memory)
	parent.PID = 77
	parent.CWD = "/work"
	parent.SignalMask = 0x55
	parent.Groups = []uint32{1, 2}
	child := parent.CloneForChild64(memory.Clone(), 901)
	if child == nil || child.PID != 901 || child.ParentPID != 77 || child.CWD != "/work" || child.SignalMask != 0x55 {
		t.Fatalf("unexpected child context: %+v", child)
	}
	child.Groups[0] = 9
	if parent.Groups[0] != 1 {
		t.Fatal("child groups share parent backing array")
	}
}
