package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func TestABI64ReadlinkatAndMkdiratWithDirFD(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	if err := fake.Mkdir("/base", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.Symlink("target", "/base/link", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(5, &corefd.File{Path: "/base"}); err != nil {
		t.Fatal(err)
	}

	const (
		absoluteName corecpu.Address64 = 0x10200
		relativeName corecpu.Address64 = 0x10300
		buffer       corecpu.Address64 = 0x10400
		newName      corecpu.Address64 = 0x10500
	)
	writeABI64CString(t, memory, absoluteName, "/base/link")
	writeABI64CString(t, memory, relativeName, "link")
	writeABI64CString(t, memory, newName, "new")

	set64Syscall(state, Sys64Readlinkat, atFDCWD64, uint64(absoluteName), uint64(buffer), 3)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 3 {
		t.Fatalf("readlinkat absolute: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	var prefix [3]byte
	if err := memory.Read(buffer, prefix[:]); err != nil {
		t.Fatal(err)
	}
	if string(prefix[:]) != "tar" {
		t.Fatalf("readlinkat truncation=%q, want tar", prefix[:])
	}

	set64Syscall(state, Sys64Readlinkat, 5, uint64(relativeName), uint64(buffer), 16)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 6 {
		t.Fatalf("readlinkat dirfd: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	full := make([]byte, 6)
	if err := memory.Read(buffer, full); err != nil {
		t.Fatal(err)
	}
	if string(full) != "target" {
		t.Fatalf("readlinkat dirfd target=%q, want target", full)
	}

	set64Syscall(state, Sys64Mkdirat, 5, uint64(newName), 0o755)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("mkdirat dirfd: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	info, err := fake.Stat("/base/new")
	if err != nil || !info.IsDir() {
		t.Fatalf("mkdirat created entry: info=%#v err=%v", info, err)
	}
}

func TestABI64ReadlinkatAndMkdiratValidation(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	if err := fake.Mkdir("/base", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.Symlink("target", "/base/link", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(5, &corefd.File{Path: "/base"}); err != nil {
		t.Fatal(err)
	}
	const name corecpu.Address64 = 0x10600
	writeABI64CString(t, memory, name, "link")

	set64Syscall(state, Sys64Readlinkat, 99, uint64(name), uint64(name+0x100), 8)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EBADF) {
		t.Fatalf("readlinkat invalid dirfd: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Readlinkat, 5, uint64(name), 0x200000, 8)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("readlinkat invalid buffer: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Mkdirat, 5, uint64(name), uint64(1)<<32)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("mkdirat invalid mode: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Mkdirat, 5, uint64(name), 0o755)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EEXIST) {
		t.Fatalf("mkdirat existing symlink: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Readlinkat, 5, uint64(name), uint64(name+0x100), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("readlinkat zero buffer: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
}
