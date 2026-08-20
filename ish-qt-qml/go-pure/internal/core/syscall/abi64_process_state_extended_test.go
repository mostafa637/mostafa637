package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64PrctlExtendedProcessState(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	const address corecpu.Address64 = 0x10800

	set64Syscall(state, Sys64Prctl, prSetKeepCaps64, 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 || !ctx.KeepCaps {
		t.Fatalf("set keepcaps: err=%v rax=%d state=%v", err, int64(state.Get(corecpu.RAX)), ctx.KeepCaps)
	}
	set64Syscall(state, Sys64Prctl, prGetKeepCaps64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("get keepcaps: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Prctl, prSetKeepCaps64, 2)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid keepcaps: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Prctl, prSetSecureBits64, 0x1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 || ctx.SecureBits != 0x1 {
		t.Fatalf("set securebits: err=%v rax=%d state=%#x", err, int64(state.Get(corecpu.RAX)), ctx.SecureBits)
	}
	set64Syscall(state, Sys64Prctl, prGetSecureBits64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0x1 {
		t.Fatalf("get securebits: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Prctl, prSetSecureBits64, 0x40)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid securebits: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Prctl, prSetSecureBits64, 0x2)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("lock securebits: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Prctl, prSetSecureBits64, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EPERM) {
		t.Fatalf("clear locked securebits: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Prctl, prSetChildSubreaper64, 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 || !ctx.ChildSubreaper {
		t.Fatalf("set child subreaper: err=%v rax=%d state=%v", err, int64(state.Get(corecpu.RAX)), ctx.ChildSubreaper)
	}
	set64Syscall(state, Sys64Prctl, prGetChildSubreaper64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("get child subreaper: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Prctl, prSetChildSubreaper64, 2)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid child subreaper: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Prctl, prSetTimerSlack64, 1234)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 || ctx.TimerSlack != 1234 {
		t.Fatalf("set timer slack: err=%v rax=%d state=%d", err, int64(state.Get(corecpu.RAX)), ctx.TimerSlack)
	}
	set64Syscall(state, Sys64Prctl, prGetTimerSlack64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1234 {
		t.Fatalf("get timer slack: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Prctl, prSetTimerSlack64, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || ctx.TimerSlack != defaultTimerSlack64 {
		t.Fatalf("reset timer slack: err=%v state=%d", err, ctx.TimerSlack)
	}

	set64Syscall(state, Sys64Prctl, prSetNoNewPrivs64, 1, 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid no-new-privs arguments: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Prctl, 0x7fff)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("unknown prctl: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Prctl, prGetName64, 0x200000)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("invalid get name pointer: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	_ = memory
}
