package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64MembarrierQueryAndRegistration(t *testing.T) {
	_, _, ctx, dispatcher, state := newABI64FilesystemTest(t)

	set64Syscall(state, Sys64Membarrier, membarrierCmdQuery64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(membarrierSupportedCommands64) {
		t.Fatalf("query: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Membarrier, membarrierCmdGlobalExpedited64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("unregistered global expedited: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Membarrier, membarrierCmdRegisterGlobalExpedited64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("register global: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Membarrier, membarrierCmdGlobalExpedited64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("global expedited: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Membarrier, membarrierCmdRegisterPrivateExpedited64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("register private: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	before := ctx.MembarrierEpoch
	set64Syscall(state, Sys64Membarrier, membarrierCmdPrivateExpedited64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("private expedited: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if ctx.MembarrierEpoch <= before {
		t.Fatalf("private expedited did not execute barrier: before=%d after=%d", before, ctx.MembarrierEpoch)
	}

	set64Syscall(state, Sys64Membarrier, membarrierCmdGetRegistrations64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(membarrierCmdRegisterGlobalExpedited64|membarrierCmdRegisterPrivateExpedited64) {
		t.Fatalf("registrations: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64MembarrierValidation(t *testing.T) {
	_, _, _, dispatcher, state := newABI64FilesystemTest(t)
	set64Syscall(state, Sys64Membarrier, membarrierCmdQuery64, 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("non-zero flags: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Membarrier, 1<<20)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("unknown command: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Membarrier, membarrierCmdPrivateExpedited64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("unregistered private expedited: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
