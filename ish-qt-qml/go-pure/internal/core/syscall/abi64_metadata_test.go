package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

func TestABI64MetadataAndUmask(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	if err := fake.Mkdir("/tmp", 0o777, 0, 0); err != nil {
		t.Fatal(err)
	}
	const (
		pathAddress  corecpu.Address64 = 0x10900
		emptyAddress corecpu.Address64 = 0x10b00
	)
	if err := memory.Write(pathAddress, append([]byte("/tmp/abi64-metadata"), 0)); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Umask, 0o027)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0o022 {
		t.Fatalf("umask set: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Open, uint64(pathAddress), uint64(guestOpenCreat|2), 0o666)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 0 {
		t.Fatalf("open create: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	fd := state.Get(corecpu.RAX)
	info, err := fake.Stat("/tmp/abi64-metadata")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode.Mode != corefs.ModeRegular|0o640 {
		t.Fatalf("created mode=%#o, want %#o", info.Mode.Mode, corefs.ModeRegular|0o640)
	}
	set64Syscall(state, Sys64Fchmod, fd, 0o600)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("fchmod: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Fchown, fd, 1001, 1002)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("fchown: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	info, err = fake.Stat("/tmp/abi64-metadata")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode.Mode != corefs.ModeRegular|0o600 || info.Mode.UID != 1001 || info.Mode.GID != 1002 {
		t.Fatalf("updated metadata=%#v", info.Mode)
	}
	set64Syscall(state, Sys64Fchmodat, atFDCWD64, uint64(pathAddress), 0o644, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("fchmodat: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Write(emptyAddress, []byte{0}); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Fchownat, fd, uint64(emptyAddress), 1003, 1004, atEmptyPath64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("fchownat empty path: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	info, err = fake.Stat("/tmp/abi64-metadata")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode.UID != 1003 || info.Mode.GID != 1004 || info.Mode.Mode != corefs.ModeRegular|0o644 {
		t.Fatalf("at metadata=%#v", info.Mode)
	}
	ctx.Umask = 0o777
	if err := memory.Write(pathAddress, append([]byte("/tmp/abi64-umask-777"), 0)); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Open, uint64(pathAddress), uint64(guestOpenCreat|2), 0o666)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 0 {
		t.Fatalf("second open create: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	second, err := fake.Stat("/tmp/abi64-umask-777")
	if err != nil {
		t.Fatal(err)
	}
	if second.Mode.Mode != corefs.ModeRegular {
		t.Fatalf("umask=0777 created mode=%#o", second.Mode.Mode)
	}
}
