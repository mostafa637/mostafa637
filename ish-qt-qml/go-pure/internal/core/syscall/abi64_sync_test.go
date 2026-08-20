package syscall

import (
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func TestABI64SyncAndSyncfs(t *testing.T) {
	fake, _, ctx, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/sync-me", 0o644, 1000, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("metadata and bytes")); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InstallFile(3, &corefd.File{Path: "/sync-me"}); err != nil {
		t.Fatal(err)
	}

	set64Syscall(state, Sys64Sync)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("sync: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Syncfs, 3)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("syncfs: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64SyncfsValidation(t *testing.T) {
	_, _, ctx, dispatcher, state := newABI64FilesystemTest(t)
	set64Syscall(state, Sys64Syncfs, 99)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EBADF) {
		t.Fatalf("invalid fd: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := ctx.InstallFile(4, &corefd.File{}); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Syncfs, 4)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("anonymous fd: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
