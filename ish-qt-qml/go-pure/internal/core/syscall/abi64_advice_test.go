package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64FadviseValidation(t *testing.T) {
	fake, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/advice", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	const pathAddress corecpu.Address64 = 0x11900
	writeABI64CString(t, memory, pathAddress, "/advice")
	set64Syscall(state, Sys64Open, uint64(pathAddress), 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 0 {
		t.Fatalf("open advice: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	fd := state.Get(corecpu.RAX)

	for _, advice := range []uint64{fadviseNormal64, fadviseSequential64, fadviseNoReuse64} {
		set64Syscall(state, Sys64Fadvise64, fd, 0, 0, advice)
		if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
			t.Fatalf("fadvise advice=%d: err=%v rax=%d", advice, err, int64(state.Get(corecpu.RAX)))
		}
	}

	set64Syscall(state, Sys64Fadvise64, 999, 0, 0, fadviseNormal64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EBADF) {
		t.Fatalf("invalid fd: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Fadvise64, fd, 0, 0, fadviseNoReuse64+1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid advice: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Fadvise64, fd, ^uint64(0), 0, fadviseNormal64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid offset: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Fadvise64, fd, 0, ^uint64(0), fadviseNormal64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid length: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Close, fd)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("close advice: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
