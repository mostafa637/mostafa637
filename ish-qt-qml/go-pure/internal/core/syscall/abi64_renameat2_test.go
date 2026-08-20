package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64Renameat2NoReplaceAndExchange(t *testing.T) {
	fake, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	if err := fake.WriteFile("/source", []byte("source"), 0o644, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile("/existing", []byte("existing"), 0o644, 0, 0); err != nil {
		t.Fatal(err)
	}
	const oldAddress corecpu.Address64 = 0x10000
	const newAddress corecpu.Address64 = 0x10100
	writeABI64CString(t, memory, oldAddress, "/source")
	writeABI64CString(t, memory, newAddress, "/existing")
	set64Syscall(state, Sys64Renameat2, atFDCWD64, uint64(oldAddress), atFDCWD64, uint64(newAddress), renameNoReplace64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EEXIST) {
		t.Fatalf("renameat2 noreplace: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if got, err := fake.ReadFile("/source"); err != nil || string(got) != "source" {
		t.Fatalf("source after failed noreplace: data=%q err=%v", got, err)
	}

	leftBefore, err := fake.Stat("/source")
	if err != nil {
		t.Fatal(err)
	}
	rightBefore, err := fake.Stat("/existing")
	if err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Renameat2, atFDCWD64, uint64(oldAddress), atFDCWD64, uint64(newAddress), renameExchange64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("renameat2 exchange: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if got, err := fake.ReadFile("/source"); err != nil || string(got) != "existing" {
		t.Fatalf("source after exchange: data=%q err=%v", got, err)
	}
	if got, err := fake.ReadFile("/existing"); err != nil || string(got) != "source" {
		t.Fatalf("existing after exchange: data=%q err=%v", got, err)
	}
	leftAfter, err := fake.Stat("/source")
	if err != nil {
		t.Fatal(err)
	}
	rightAfter, err := fake.Stat("/existing")
	if err != nil {
		t.Fatal(err)
	}
	if leftAfter.Inode != rightBefore.Inode || rightAfter.Inode != leftBefore.Inode {
		t.Fatalf("exchange inode mapping = source:%d existing:%d, want source:%d existing:%d", leftAfter.Inode, rightAfter.Inode, rightBefore.Inode, leftBefore.Inode)
	}
}

func TestABI64Renameat2DirFDAndValidation(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	if err := fake.Mkdir("/base", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile("/base/source", []byte("dirfd"), 0o644, 0, 0); err != nil {
		t.Fatal(err)
	}
	const dirAddress corecpu.Address64 = 0x10200
	dirfd := openABI64TestFile(t, dispatcher, state, memory, "/base", dirAddress)
	if dirfd < 0 {
		t.Fatalf("open dirfd = %d", dirfd)
	}
	defer func() {
		set64Syscall(state, Sys64Close, uint64(dirfd))
		_, _ = dispatcher.Dispatch(state)
	}()
	const oldAddress corecpu.Address64 = 0x10300
	const newAddress corecpu.Address64 = 0x10400
	writeABI64CString(t, memory, oldAddress, "source")
	writeABI64CString(t, memory, newAddress, "dest")
	set64Syscall(state, Sys64Renameat2, uint64(dirfd), uint64(oldAddress), uint64(dirfd), uint64(newAddress), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("renameat2 dirfd: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if got, err := fake.ReadFile("/base/dest"); err != nil || string(got) != "dirfd" {
		t.Fatalf("dirfd destination: data=%q err=%v", got, err)
	}

	set64Syscall(state, Sys64Renameat2, atFDCWD64, uint64(oldAddress), atFDCWD64, uint64(newAddress), renameNoReplace64|renameExchange64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid combined flags: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Renameat2, atFDCWD64, uint64(oldAddress), atFDCWD64, uint64(newAddress), renameWhiteout64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EOPNOTSUPP) {
		t.Fatalf("whiteout: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Renameat2, atFDCWD64, uint64(0x13000), atFDCWD64, uint64(newAddress), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("invalid old pointer: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	_ = ctx
}
