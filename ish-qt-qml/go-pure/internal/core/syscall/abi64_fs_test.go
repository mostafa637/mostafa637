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

func TestDispatcher64FilesystemLifecycle(t *testing.T) {
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
	const area corecpu.Address64 = 0x9000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	writeCString := func(address corecpu.Address64, value string) {
		t.Helper()
		if err := memory.Write(address, append([]byte(value), 0)); err != nil {
			t.Fatal(err)
		}
	}
	context64 := NewContext64(memory)
	context64.FS = fake
	dispatcher := NewDispatcher64(context64)
	state := corecpu.NewMachineState64(memory)
	statAddress := area + 0x300

	writeCString(area, "/hello")
	state.Set(corecpu.RAX, uint64(Sys64Stat))
	state.Set(corecpu.RDI, uint64(area))
	state.Set(corecpu.RSI, uint64(statAddress))
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("stat: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	var stat [stat64GuestSize]byte
	if err := memory.Read(statAddress, stat[:]); err != nil {
		t.Fatal(err)
	}
	if mode := binary.LittleEndian.Uint32(stat[16:20]); mode != corefs.ModeRegular|0o644 {
		t.Fatalf("stat mode = %#o", mode)
	}

	state.Set(corecpu.RAX, uint64(Sys64Open))
	state.Set(corecpu.RDI, uint64(area))
	state.Set(corecpu.RSI, 0)
	state.Set(corecpu.RDX, 0)
	resume, err = dispatcher.Dispatch(state)
	fd := int64(state.Get(corecpu.RAX))
	if err != nil || !resume || fd < 3 {
		t.Fatalf("open: resume=%v err=%v fd=%d", resume, err, fd)
	}
	state.Set(corecpu.RAX, uint64(Sys64Fstat))
	state.Set(corecpu.RDI, uint64(fd))
	state.Set(corecpu.RSI, uint64(statAddress))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("fstat: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}

	writeCString(area+0x100, "/dir")
	state.Set(corecpu.RAX, uint64(Sys64Mkdir))
	state.Set(corecpu.RDI, uint64(area+0x100))
	state.Set(corecpu.RSI, 0o755)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("mkdir: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if info, err := fake.Stat("/dir"); err != nil || !info.IsDir() {
		t.Fatalf("mkdir stat: info=%v err=%v", info, err)
	}

	writeCString(area+0x200, "/renamed")
	state.Set(corecpu.RAX, uint64(Sys64Rename))
	state.Set(corecpu.RDI, uint64(area))
	state.Set(corecpu.RSI, uint64(area+0x200))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("rename: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	writeCString(area+0x400, "/renamed")
	state.Set(corecpu.RAX, uint64(Sys64Unlink))
	state.Set(corecpu.RDI, uint64(area+0x400))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("unlink: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	writeCString(area+0x500, "/dir")
	state.Set(corecpu.RAX, uint64(Sys64Rmdir))
	state.Set(corecpu.RDI, uint64(area+0x500))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("rmdir: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
}
