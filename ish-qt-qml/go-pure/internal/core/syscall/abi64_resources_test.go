package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64ResourceLimitsAndGroups(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 321
	ctx.TID = 321
	const limitAddress corecpu.Address64 = 0x10200
	set64Syscall(state, Sys64GetRlimit, uint64(rlimitNOFILE), uint64(limitAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("getrlimit: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var limit [16]byte
	if err := memory.Read(limitAddress, limit[:]); err != nil {
		t.Fatal(err)
	}
	if cur, max := binary.LittleEndian.Uint64(limit[0:8]), binary.LittleEndian.Uint64(limit[8:16]); cur != 1024 || max != 4096 {
		t.Fatalf("initial nofile limit = %d/%d", cur, max)
	}

	binary.LittleEndian.PutUint64(limit[0:8], 128)
	binary.LittleEndian.PutUint64(limit[8:16], 256)
	if err := memory.Write(limitAddress, limit[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64SetRlimit, uint64(rlimitNOFILE), uint64(limitAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("setrlimit: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64GetRlimit, uint64(rlimitNOFILE), uint64(limitAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("getrlimit after set: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(limitAddress, limit[:]); err != nil {
		t.Fatal(err)
	}
	if cur, max := binary.LittleEndian.Uint64(limit[0:8]), binary.LittleEndian.Uint64(limit[8:16]); cur != 128 || max != 256 {
		t.Fatalf("updated nofile limit = %d/%d", cur, max)
	}
	binary.LittleEndian.PutUint64(limit[0:8], 257)
	binary.LittleEndian.PutUint64(limit[8:16], 256)
	if err := memory.Write(limitAddress, limit[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64SetRlimit, uint64(rlimitNOFILE), uint64(limitAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid setrlimit: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64GetGroups, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("getgroups count: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	const groupsAddress corecpu.Address64 = 0x10300
	var groups [8]byte
	binary.LittleEndian.PutUint32(groups[0:4], 1000)
	binary.LittleEndian.PutUint32(groups[4:8], 1001)
	if err := memory.Write(groupsAddress, groups[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64SetGroups, 2, uint64(groupsAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("setgroups: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64GetGroups, 2, uint64(groupsAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 2 {
		t.Fatalf("getgroups values: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if got := binary.LittleEndian.Uint32(groups[0:4]); got != 1000 {
		// Re-read after the syscall; the old local array is only the input buffer.
		if err := memory.Read(groupsAddress, groups[:]); err != nil {
			t.Fatal(err)
		}
	}
	if got := binary.LittleEndian.Uint32(groups[0:4]); got != 1000 || binary.LittleEndian.Uint32(groups[4:8]) != 1001 {
		t.Fatalf("groups = %d/%d", got, binary.LittleEndian.Uint32(groups[4:8]))
	}
}

func TestABI64ThreadSignals(t *testing.T) {
	_, _, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 321
	ctx.TID = 654
	set64Syscall(state, Sys64Tkill, 654, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("tkill self probe: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Tgkill, 321, 654, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("tgkill self probe: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Tkill, 999, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ESRCH) {
		t.Fatalf("tkill foreign tid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64RtSigaction, 15, 0, 0, 8)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("rt_sigaction: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
