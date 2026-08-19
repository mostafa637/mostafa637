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

func TestDispatcher64FilesystemCapacityAndDurability(t *testing.T) {
	root := t.TempDir()
	db, err := storage.Open(context.Background(), filepath.Join(root, "meta.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake, err := corefs.New(root, db)
	if err != nil {
		t.Fatal(err)
	}
	defer fake.Close()
	created, err := fake.Create("/hello", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := created.Write([]byte("hello")); err != nil {
		_ = created.Close()
		t.Fatal(err)
	}
	if err := created.Close(); err != nil {
		t.Fatal(err)
	}

	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xc000
	if err := memory.Map(area, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	writeCString := func(address corecpu.Address64, value string) {
		t.Helper()
		if err := memory.Write(address, append([]byte(value), 0)); err != nil {
			t.Fatal(err)
		}
	}
	ctx := NewContext64(memory)
	ctx.FS = fake
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)
	pathAddress := area
	statfsAddress := area + 0x200
	writeCString(pathAddress, "/hello")

	set64Syscall(state, Sys64Statfs, uint64(pathAddress), statfs64GuestSize, uint64(statfsAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("statfs: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var statfs [statfs64GuestSize]byte
	if err := memory.Read(statfsAddress, statfs[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(statfs[0:8]); got != statfs64Magic {
		t.Fatalf("statfs magic = %#x, want %#x", got, statfs64Magic)
	}
	if got := binary.LittleEndian.Uint64(statfs[8:16]); got != uint64(corecpu.Page64Size) {
		t.Fatalf("statfs block size = %d", got)
	}
	if got := binary.LittleEndian.Uint64(statfs[64:72]); got != 255 {
		t.Fatalf("statfs name length = %d", got)
	}

	set64Syscall(state, Sys64Open, uint64(pathAddress), 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	fd := state.Get(corecpu.RAX)
	if int64(fd) < 3 {
		t.Fatalf("open fd = %d", int64(fd))
	}

	set64Syscall(state, Sys64Fstatfs, fd, statfs64GuestSize, uint64(statfsAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("fstatfs: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64Truncate, uint64(pathAddress), 2)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("truncate: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if info, err := fake.Stat("/hello"); err != nil || info.Size != 2 {
		t.Fatalf("size after truncate = %d, err=%v", info.Size, err)
	}

	set64Syscall(state, Sys64Ftruncate, fd, 7)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("ftruncate: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if info, err := fake.Stat("/hello"); err != nil || info.Size != 7 {
		t.Fatalf("size after ftruncate = %d, err=%v", info.Size, err)
	}

	for _, number := range []Number64{Sys64Fsync, Sys64Fdatasync} {
		set64Syscall(state, number, fd)
		if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
			t.Fatalf("sync syscall %d: err=%v rax=%d", number, err, int64(state.Get(corecpu.RAX)))
		}
	}

	set64Syscall(state, Sys64Close, fd)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("close: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
