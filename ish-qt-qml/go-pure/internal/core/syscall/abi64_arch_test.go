package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64ArchPrctlTLS(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	const address corecpu.Address64 = 0x10600
	state.FSBase = 0x11110000
	state.GSBase = 0x22220000
	set64Syscall(state, Sys64ArchPrctl, archGetFS64, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("get fs: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var raw [8]byte
	if err := memory.Read(address, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(raw[:]); got != 0x11110000 {
		t.Fatalf("fs base=%#x", got)
	}
	set64Syscall(state, Sys64ArchPrctl, archSetFS64, 0x33330000)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 || state.FSBase != 0x33330000 || ctx.FSBase != 0x33330000 {
		t.Fatalf("set fs: err=%v rax=%d state=%#x ctx=%#x", err, int64(state.Get(corecpu.RAX)), state.FSBase, ctx.FSBase)
	}
	set64Syscall(state, Sys64ArchPrctl, archGetGS64, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("get gs: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(address, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(raw[:]); got != 0x22220000 {
		t.Fatalf("gs base=%#x", got)
	}
	set64Syscall(state, Sys64ArchPrctl, archGetCPUID64, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("get cpuid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(address, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(raw[:]); got != 1 {
		t.Fatalf("cpuid state=%d", got)
	}
	set64Syscall(state, Sys64ArchPrctl, archSetCPUID64, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 || ctx.CPUIDEnabled {
		t.Fatalf("set cpuid: err=%v rax=%d enabled=%v", err, int64(state.Get(corecpu.RAX)), ctx.CPUIDEnabled)
	}
	set64Syscall(state, Sys64ArchPrctl, archSetCPUID64, 2)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid cpuid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
