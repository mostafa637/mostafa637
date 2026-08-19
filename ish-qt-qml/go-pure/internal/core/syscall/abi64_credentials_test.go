package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcher64CredentialsAndResIDs(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xc000
	if err := memory.Map(area, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.PID = 42
	ctx.TID = 42
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	set64Syscall(state, Sys64GetUID)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("initial getuid: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}

	set64Syscall(state, Sys64SetResUID, 1000, 1000, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("root setresuid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if ctx.RealUID != 1000 || ctx.EffectiveUID != 1000 || ctx.SavedUID != 0 {
		t.Fatalf("setresuid state: real=%d effective=%d saved=%d", ctx.RealUID, ctx.EffectiveUID, ctx.SavedUID)
	}

	set64Syscall(state, Sys64GetResUID, uint64(area))
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("getresuid: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	var ids [3]uint32
	buffer := make([]byte, len(ids)*4)
	if err := memory.Read(area, buffer); err != nil {
		t.Fatal(err)
	}
	for index := range ids {
		ids[index] = binary.LittleEndian.Uint32(buffer[index*4:])
	}
	if ids != [3]uint32{1000, 1000, 0} {
		t.Fatalf("getresuid values=%v", ids)
	}

	set64Syscall(state, Sys64SetUID, 2000)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EPERM) {
		t.Fatalf("unprivileged setuid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64SetUID, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 || ctx.EffectiveUID != 0 {
		t.Fatalf("restore effective uid: err=%v rax=%d effective=%d", err, int64(state.Get(corecpu.RAX)), ctx.EffectiveUID)
	}

	set64Syscall(state, Sys64SetResGID, 100, 100, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("root setresgid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64GetResGID, uint64(area+0x40))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("getresgid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(area+0x40, buffer); err != nil {
		t.Fatal(err)
	}
	for index := range ids {
		ids[index] = binary.LittleEndian.Uint32(buffer[index*4:])
	}
	if ids != [3]uint32{100, 100, 0} {
		t.Fatalf("getresgid values=%v", ids)
	}
	set64Syscall(state, Sys64SetUID, uint64(1)<<32)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("out-of-range setuid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64SetResGID, uint64(1)<<32, ^uint64(0), ^uint64(0))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("out-of-range setresgid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64GetGID)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 100 {
		t.Fatalf("getgid: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
}

func TestDispatcher64SessionAndProcessGroups(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xd000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.PID = 42
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	set64Syscall(state, Sys64GetPGID, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 42 {
		t.Fatalf("initial getpgid: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}

	set64Syscall(state, Sys64SetPGID, 0, 99)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("setpgid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64GetPGID, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 99 {
		t.Fatalf("getpgid after setpgid: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}

	set64Syscall(state, Sys64GetSID, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 42 {
		t.Fatalf("initial getsid: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	set64Syscall(state, Sys64SetSID)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 42 {
		t.Fatalf("setsid: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	if ctx.PGID != 42 || ctx.SID != 42 {
		t.Fatalf("session state: pgid=%d sid=%d", ctx.PGID, ctx.SID)
	}
	set64Syscall(state, Sys64SetSID)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EPERM) {
		t.Fatalf("second setsid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64GetPGID, 99)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ESRCH) {
		t.Fatalf("unknown pgid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	_ = area
}
