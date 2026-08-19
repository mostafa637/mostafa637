package syscall

import (
	"context"
	"encoding/binary"
	"path/filepath"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

func newABI64FilesystemTest(t *testing.T) (*corefs.FS, *corecpu.Memory64, *Context64, *Dispatcher64, *corecpu.MachineState64) {
	t.Helper()
	root := t.TempDir()
	db, err := storage.Open(context.Background(), filepath.Join(root, "meta.db"))
	if err != nil {
		t.Fatal(err)
	}
	fake, err := corefs.New(root, db)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = fake.Close()
	})
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0x10000
	if err := memory.Map(area, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.FS = fake
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)
	return fake, memory, ctx, dispatcher, state
}

func set64Syscall(state *corecpu.MachineState64, number Number64, args ...uint64) {
	state.Set(corecpu.RAX, uint64(number))
	registers := []corecpu.Reg64{corecpu.RDI, corecpu.RSI, corecpu.RDX, corecpu.R10, corecpu.R8, corecpu.R9}
	for i, register := range registers {
		var value uint64
		if i < len(args) {
			value = args[i]
		}
		state.Set(register, value)
	}
}

func writeABI64CString(t *testing.T, memory *corecpu.Memory64, address corecpu.Address64, value string) {
	t.Helper()
	if err := memory.Write(address, append([]byte(value), 0)); err != nil {
		t.Fatal(err)
	}
}

func TestStatx64LayoutAndFlags(t *testing.T) {
	fake, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/hello", 0o644, 1000, 1001)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("hello")); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	const pathAddress corecpu.Address64 = 0x10000
	const statxAddress corecpu.Address64 = 0x10400
	writeABI64CString(t, memory, pathAddress, "/hello")
	set64Syscall(state, Sys64Statx, atFDCWD64, uint64(pathAddress), 0, statxBasicStats64, uint64(statxAddress))
	if resume, err := dispatcher.Dispatch(state); err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("statx: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	var statx [statxGuestSize]byte
	if err := memory.Read(statxAddress, statx[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(statx[0:4]); got != statxBasicStats64 {
		t.Fatalf("statx mask = %#x, want %#x", got, statxBasicStats64)
	}
	if got := binary.LittleEndian.Uint32(statx[4:8]); got != 4096 {
		t.Fatalf("statx blksize = %d", got)
	}
	if got := binary.LittleEndian.Uint32(statx[16:20]); got != 1 {
		t.Fatalf("statx nlink = %d", got)
	}
	if got := binary.LittleEndian.Uint32(statx[20:24]); got != 1000 {
		t.Fatalf("statx uid = %d", got)
	}
	if got := binary.LittleEndian.Uint32(statx[24:28]); got != 1001 {
		t.Fatalf("statx gid = %d", got)
	}
	if got := binary.LittleEndian.Uint16(statx[28:30]); got != uint16(corefs.ModeRegular|0o644) {
		t.Fatalf("statx mode = %#o", got)
	}
	if got := binary.LittleEndian.Uint64(statx[40:48]); got != 5 {
		t.Fatalf("statx size = %d", got)
	}
	if got := binary.LittleEndian.Uint64(statx[48:56]); got != 1 {
		t.Fatalf("statx blocks = %d", got)
	}
	if got := binary.LittleEndian.Uint64(statx[64:72]); got == 0 {
		t.Fatal("statx atime is unexpectedly zero")
	}

	if err := fake.Symlink("/hello", "/link", 1000, 1001); err != nil {
		t.Fatal(err)
	}
	writeABI64CString(t, memory, pathAddress, "/link")
	set64Syscall(state, Sys64Statx, atFDCWD64, uint64(pathAddress), atSymlinkNoFollow64, statxBasicStats64, uint64(statxAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("statx nofollow: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(statxAddress, statx[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint16(statx[28:30]); got != uint16(corefs.ModeSymlink|0o777) {
		t.Fatalf("statx nofollow mode = %#o", got)
	}

	writeABI64CString(t, memory, pathAddress, "")
	fd := openABI64TestFile(t, dispatcher, state, memory, "/hello", 0x10800)
	set64Syscall(state, Sys64Statx, uint64(fd), uint64(pathAddress), atEmptyPath64, statxBasicStats64, uint64(statxAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("statx empty path: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Close, uint64(fd))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("close: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64AtFamilyAndPrlimit(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/source", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	const sourceAddress corecpu.Address64 = 0x10000
	const destinationAddress corecpu.Address64 = 0x10100
	const statAddress corecpu.Address64 = 0x10400
	writeABI64CString(t, memory, sourceAddress, "/source")
	writeABI64CString(t, memory, destinationAddress, "/hard")
	set64Syscall(state, Sys64Linkat, atFDCWD64, uint64(sourceAddress), atFDCWD64, uint64(destinationAddress), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("linkat: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if _, err := fake.Stat("/hard"); err != nil {
		t.Fatalf("hard link: %v", err)
	}

	writeABI64CString(t, memory, destinationAddress, "/sym")
	set64Syscall(state, Sys64Symlinkat, uint64(sourceAddress), atFDCWD64, uint64(destinationAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("symlinkat: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	writeABI64CString(t, memory, destinationAddress, "/renamed")
	set64Syscall(state, Sys64Renameat, atFDCWD64, uint64(sourceAddress), atFDCWD64, uint64(destinationAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("renameat: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	writeABI64CString(t, memory, destinationAddress, "/hard")
	set64Syscall(state, Sys64Unlinkat, atFDCWD64, uint64(destinationAddress), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("unlinkat: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	var limit [16]byte
	binary.LittleEndian.PutUint64(limit[0:8], 128)
	binary.LittleEndian.PutUint64(limit[8:16], 256)
	if err := memory.Write(statAddress, limit[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Prlimit64, ctx.PID, uint64(rlimitNOFILE), uint64(statAddress), uint64(statAddress+0x40))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("prlimit64: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var old [16]byte
	if err := memory.Read(statAddress+0x40, old[:]); err != nil {
		t.Fatal(err)
	}
	if cur, max := binary.LittleEndian.Uint64(old[0:8]), binary.LittleEndian.Uint64(old[8:16]); cur != 1024 || max != 4096 {
		t.Fatalf("old nofile limit = %d/%d", cur, max)
	}
}

func openABI64TestFile(t *testing.T, dispatcher *Dispatcher64, state *corecpu.MachineState64, memory *corecpu.Memory64, name string, pathAddress corecpu.Address64) int64 {
	t.Helper()
	writeABI64CString(t, memory, pathAddress, name)
	set64Syscall(state, Sys64Open, uint64(pathAddress), 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	fd := int64(state.Get(corecpu.RAX))
	if fd < 0 {
		t.Fatalf("open %q = %d", name, fd)
	}
	return fd
}

func TestABI64DirFDResolutionAndFlagValidation(t *testing.T) {
	fake, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	if err := fake.Mkdir("/dir", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	file, err := fake.Create("/dir/file", 0o600, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	const dirPathAddress corecpu.Address64 = 0x10000
	const childPathAddress corecpu.Address64 = 0x10100
	const statAddress corecpu.Address64 = 0x10400
	writeABI64CString(t, memory, dirPathAddress, "/dir")
	set64Syscall(state, Sys64Open, uint64(dirPathAddress), 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	dirFD := int64(state.Get(corecpu.RAX))
	if dirFD < 0 {
		t.Fatalf("open dir = %d", dirFD)
	}
	writeABI64CString(t, memory, childPathAddress, "file")
	set64Syscall(state, Sys64Fstatat, uint64(dirFD), uint64(childPathAddress), uint64(statAddress), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("fstatat dirfd: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var stat [stat64GuestSize]byte
	if err := memory.Read(statAddress, stat[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(stat[16:20]); got != corefs.ModeRegular|0o600 {
		t.Fatalf("fstatat dirfd mode = %#o", got)
	}

	writeABI64CString(t, memory, childPathAddress, "")
	set64Syscall(state, Sys64Fstatat, uint64(dirFD), uint64(childPathAddress), uint64(statAddress), atEmptyPath64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("fstatat empty path: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(statAddress, stat[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(stat[16:20]); got != corefs.ModeDir|0o755 {
		t.Fatalf("fstatat empty dir mode = %#o", got)
	}

	writeABI64CString(t, memory, childPathAddress, "/dir/file")
	set64Syscall(state, Sys64Statx, atFDCWD64, uint64(childPathAddress), 0x80000000, statxBasicStats64, uint64(statAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("statx invalid flags: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Close, uint64(dirFD))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("close dir: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64LifecycleAndRobustList(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 321
	ctx.TID = 321
	if !ctx.Children.AddChild(uint32(ctx.PID), 654) {
		t.Fatal("add child failed")
	}
	if !ctx.Children.MarkExited(654, 7) {
		t.Fatal("mark child exited failed")
	}
	const statusAddress corecpu.Address64 = 0x10200
	set64Syscall(state, Sys64Wait4, 654, uint64(statusAddress), 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 654 {
		t.Fatalf("wait4: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var status [4]byte
	if err := memory.Read(statusAddress, status[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(status[:]); got != 7<<8 {
		t.Fatalf("wait4 status = %#x", got)
	}
	set64Syscall(state, Sys64Wait4, ^uint64(0), 0, 1, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ECHILD) {
		t.Fatalf("wait4 reaped: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64SetRobust, 0x12345000, 24)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("set robust: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64GetRobust, 0, uint64(statusAddress), uint64(statusAddress+8))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("get robust: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var robust [16]byte
	if err := memory.Read(statusAddress, robust[:]); err != nil {
		t.Fatal(err)
	}
	if head, length := binary.LittleEndian.Uint64(robust[0:8]), binary.LittleEndian.Uint64(robust[8:16]); head != 0x12345000 || length != 24 {
		t.Fatalf("robust list = %#x/%d", head, length)
	}
	set64Syscall(state, Sys64GetRobust, 999, uint64(statusAddress), uint64(statusAddress+8))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ESRCH) {
		t.Fatalf("get robust wrong pid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	for _, syscall := range []Number64{Sys64Clone, Sys64Fork, Sys64Vfork} {
		set64Syscall(state, syscall)
		if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ENOSYS) {
			t.Fatalf("syscall %d: err=%v rax=%d", syscall, err, int64(state.Get(corecpu.RAX)))
		}
	}
}

func TestABI64GetPPIDUsesParentContext(t *testing.T) {
	_, _, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 654
	ctx.ParentPID = 321
	set64Syscall(state, Sys64GetPPID)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 321 {
		t.Fatalf("getppid: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
