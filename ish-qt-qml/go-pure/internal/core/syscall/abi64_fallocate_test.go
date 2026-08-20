package syscall

import (
	"bytes"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func TestDispatcher64FallocateExtendsAndKeepsSize(t *testing.T) {
	fake, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/data", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("abcdefghij")); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	const pathAddress corecpu.Address64 = 0x10000
	writeABI64CString(t, memory, pathAddress, "/data")
	set64Syscall(state, Sys64Open, uint64(pathAddress), 2, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	fd := state.Get(corecpu.RAX)
	if int64(fd) < 3 {
		t.Fatalf("open fd = %d", int64(fd))
	}

	set64Syscall(state, Sys64Fallocate, fd, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("zero-length fallocate: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	if info, err := fake.Stat("/data"); err != nil || info.Size != 10 {
		t.Fatalf("zero-length size = %d, err=%v, want 10", info.Size, err)
	}

	set64Syscall(state, Sys64Fallocate, fd, 0, 10, 20)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("fallocate extend: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	if info, err := fake.Stat("/data"); err != nil || info.Size != 30 {
		t.Fatalf("extended size = %d, err=%v, want 30", info.Size, err)
	}
	content := make([]byte, 20)
	if n, err := fake.ReadAt("/data", content, 10); err != nil || n != len(content) || !bytes.Equal(content, make([]byte, len(content))) {
		t.Fatalf("extended range n=%d err=%v data=%q", n, err, content)
	}

	set64Syscall(state, Sys64Fallocate, fd, fallocateKeepSize64, 100, 50)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("fallocate keep-size: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	if info, err := fake.Stat("/data"); err != nil || info.Size != 30 {
		t.Fatalf("KEEP_SIZE changed size = %d, err=%v, want 30", info.Size, err)
	}
}

func TestDispatcher64FallocatePunchHolePreservesSize(t *testing.T) {
	fake, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/data", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("abcdefghij")); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	const pathAddress corecpu.Address64 = 0x10000
	writeABI64CString(t, memory, pathAddress, "/data")
	set64Syscall(state, Sys64Open, uint64(pathAddress), 2, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	fd := state.Get(corecpu.RAX)

	set64Syscall(state, Sys64Fallocate, fd, fallocatePunchHole64|fallocateKeepSize64, 2, 4)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("punch hole: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	content := make([]byte, 10)
	if n, err := fake.ReadAt("/data", content, 0); err != nil || n != len(content) {
		t.Fatalf("read punched file n=%d err=%v", n, err)
	}
	want := []byte{'a', 'b', 0, 0, 0, 0, 'g', 'h', 'i', 'j'}
	if !bytes.Equal(content, want) {
		t.Fatalf("punched content = %q, want %q", content, want)
	}
	if info, err := fake.Stat("/data"); err != nil || info.Size != 10 {
		t.Fatalf("punched size = %d, err=%v, want 10", info.Size, err)
	}
}

func TestDispatcher64FallocateValidation(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/data", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	const pathAddress corecpu.Address64 = 0x10000
	writeABI64CString(t, memory, pathAddress, "/data")
	set64Syscall(state, Sys64Open, uint64(pathAddress), 2, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	fd := state.Get(corecpu.RAX)

	tests := []struct {
		name string
		args []uint64
		want int64
	}{
		{name: "bad fd", args: []uint64{99, 0, 0, 1}, want: int64(EBADF)},
		{name: "negative offset", args: []uint64{fd, 0, ^uint64(0), 1}, want: int64(EINVAL)},
		{name: "negative length", args: []uint64{fd, 0, 0, ^uint64(0)}, want: int64(EINVAL)},
		{name: "unsupported mode", args: []uint64{fd, 0x04, 0, 1}, want: int64(EOPNOTSUPP)},
		{name: "punch without keep", args: []uint64{fd, fallocatePunchHole64, 0, 1}, want: int64(EOPNOTSUPP)},
		{name: "end overflow", args: []uint64{fd, 0, uint64(^uint64(0) >> 1), 1}, want: int64(EINVAL)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set64Syscall(state, Sys64Fallocate, test.args...)
			if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != test.want {
				t.Fatalf("fallocate: err=%v rax=%d, want %d", err, int64(state.Get(corecpu.RAX)), test.want)
			}
		})
	}

	if err := fake.Mkdir("/dir", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	writeABI64CString(t, memory, pathAddress, "/dir")
	set64Syscall(state, Sys64Open, uint64(pathAddress), guestOpenDirectory, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	dirFD := state.Get(corecpu.RAX)
	set64Syscall(state, Sys64Fallocate, dirFD, 0, 0, 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EOPNOTSUPP) {
		t.Fatalf("directory fallocate: err=%v rax=%d, want EOPNOTSUPP", err, int64(state.Get(corecpu.RAX)))
	}

	var regular bytes.Buffer
	const noPathFD uint64 = 90
	if err := ctx.FDs.InstallAt(int32(noPathFD), &corefd.File{Writer: &regular}, false); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Fallocate, noPathFD, 0, 0, 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("pathless fallocate: err=%v rax=%d, want EINVAL", err, int64(state.Get(corecpu.RAX)))
	}
}
