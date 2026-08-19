package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func writeOpenHow64(t *testing.T, memory *corecpu.Memory64, address corecpu.Address64, flags, mode, resolve uint64, size int) {
	t.Helper()
	buffer := make([]byte, size)
	binary.LittleEndian.PutUint64(buffer[0:8], flags)
	binary.LittleEndian.PutUint64(buffer[8:16], mode)
	binary.LittleEndian.PutUint64(buffer[16:24], resolve)
	if err := memory.Write(address, buffer); err != nil {
		t.Fatal(err)
	}
}

func TestABI64Openat2DirFDAndResolve(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	if err := fake.Mkdir("/base", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	created, err := fake.Create("/base/file", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = created.Close()
	if err := fake.Symlink("file", "/base/link", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(5, &corefd.File{Path: "/base"}); err != nil {
		t.Fatal(err)
	}
	const (
		pathAddress corecpu.Address64 = 0x10200
		howAddress  corecpu.Address64 = 0x10300
	)
	writeABI64CString(t, memory, pathAddress, "file")
	writeOpenHow64(t, memory, howAddress, 0, 0, 0, openHowSize64)

	set64Syscall(state, Sys64Openat2, 5, uint64(pathAddress), uint64(howAddress), openHowSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("openat2 dirfd: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	fd := int32(state.Get(corecpu.RAX))
	file, err := ctx.GetFile(uint64(fd))
	if err != nil || file.Path != "/base/file" {
		t.Fatalf("openat2 path=%q err=%v", file.Path, err)
	}
	if err := ctx.FDs.Close(fd); err != nil {
		t.Fatal(err)
	}

	writeABI64CString(t, memory, pathAddress, "created")
	writeOpenHow64(t, memory, howAddress, guestOpenCreat|guestOpenCloexec, 0o640, 0, openHowSize64)
	set64Syscall(state, Sys64Openat2, 5, uint64(pathAddress), uint64(howAddress), openHowSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("openat2 O_CREAT: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	createdFD := int32(state.Get(corecpu.RAX))
	if cloexec, err := ctx.FDs.Cloexec(createdFD); err != nil || !cloexec {
		t.Fatalf("openat2 O_CLOEXEC=%v err=%v", cloexec, err)
	}
	if _, err := fake.Stat("/base/created"); err != nil {
		t.Fatalf("openat2 created file: %v", err)
	}
	_ = ctx.FDs.Close(createdFD)

	writeABI64CString(t, memory, pathAddress, "../file")
	writeOpenHow64(t, memory, howAddress, 0, 0, resolveBeneath64, openHowSize64)
	set64Syscall(state, Sys64Openat2, 5, uint64(pathAddress), uint64(howAddress), openHowSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EXDEV) {
		t.Fatalf("openat2 BENEATH escape: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	writeABI64CString(t, memory, pathAddress, "../file")
	writeOpenHow64(t, memory, howAddress, 0, 0, resolveInRoot64, openHowSize64)
	set64Syscall(state, Sys64Openat2, 5, uint64(pathAddress), uint64(howAddress), openHowSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("openat2 IN_ROOT clamp: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	_ = ctx.FDs.Close(int32(state.Get(corecpu.RAX)))

	writeABI64CString(t, memory, pathAddress, "link")
	writeOpenHow64(t, memory, howAddress, 0, 0, resolveNoSymlinks64, openHowSize64)
	set64Syscall(state, Sys64Openat2, 5, uint64(pathAddress), uint64(howAddress), openHowSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ELOOP) {
		t.Fatalf("openat2 NO_SYMLINKS: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64Openat2Validation(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	const (
		pathAddress corecpu.Address64 = 0x10400
		howAddress  corecpu.Address64 = 0x10500
	)
	writeABI64CString(t, memory, pathAddress, "/missing")
	writeOpenHow64(t, memory, howAddress, 0, 0, 0, openHowSize64)

	set64Syscall(state, Sys64Openat2, atFDCWD64, uint64(pathAddress), uint64(howAddress), 16)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("openat2 short how: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	writeOpenHow64(t, memory, howAddress, 0, 0, 0, 32)
	set64Syscall(state, Sys64Openat2, atFDCWD64, uint64(pathAddress), uint64(howAddress), 32)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ENOENT) {
		t.Fatalf("openat2 zero extension: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Write(howAddress+24, []byte{1}); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Openat2, atFDCWD64, uint64(pathAddress), uint64(howAddress), 32)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(E2BIG) {
		t.Fatalf("openat2 nonzero extension: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	writeOpenHow64(t, memory, howAddress, 0x200000, 0, 0, openHowSize64)
	set64Syscall(state, Sys64Openat2, atFDCWD64, uint64(pathAddress), uint64(howAddress), openHowSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("openat2 invalid flags: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Openat2, 99, uint64(pathAddress), uint64(howAddress), openHowSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EBADF) {
		t.Fatalf("openat2 invalid dirfd: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Openat2, atFDCWD64, uint64(pathAddress), 0x200000, openHowSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("openat2 invalid how pointer: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	_ = ctx
}
