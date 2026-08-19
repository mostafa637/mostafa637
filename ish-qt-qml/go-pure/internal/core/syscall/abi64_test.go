package syscall

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

func TestDispatcher64RegisterABI(t *testing.T) {
	memory := corecpu.NewMemory64()
	const buffer corecpu.Address64 = 0x6000
	if err := memory.Map(buffer, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context := NewContext64(memory)
	context.PID = 1234
	context.TID = 1235
	dispatcher := NewDispatcher64(context)
	state := corecpu.NewMachineState64(memory)

	state.Set(corecpu.RAX, uint64(Sys64GetPID))
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != 1234 {
		t.Fatalf("getpid: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}

	state.Set(corecpu.RAX, uint64(Sys64Getrandom))
	state.Set(corecpu.RDI, uint64(buffer))
	state.Set(corecpu.RSI, 32)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != 32 {
		t.Fatalf("getrandom: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	var randomBytes [32]byte
	if err := memory.Read(buffer, randomBytes[:]); err != nil {
		t.Fatal(err)
	}

	state.Set(corecpu.RAX, uint64(Sys64Exit))
	state.Set(corecpu.RDI, 7)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || resume || !state.Halted || state.Get(corecpu.RAX) != 7 {
		t.Fatalf("exit: resume=%v err=%v halted=%v rax=%d", resume, err, state.Halted, state.Get(corecpu.RAX))
	}
}

func TestDispatcher64FDTableIO(t *testing.T) {
	memory := corecpu.NewMemory64()
	const buffer corecpu.Address64 = 0x7000
	if err := memory.Map(buffer, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	input := bytes.NewBufferString("ping")
	var output bytes.Buffer
	context := NewContext64(memory)
	if err := context.InstallFile(0, &File{Reader: input}); err != nil {
		t.Fatal(err)
	}
	if err := context.InstallFile(1, &File{Writer: &output}); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDispatcher64(context)
	state := corecpu.NewMachineState64(memory)

	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, 0)
	state.Set(corecpu.RSI, uint64(buffer))
	state.Set(corecpu.RDX, 4)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 4 {
		t.Fatalf("read: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	var got [4]byte
	if err := memory.Read(buffer, got[:]); err != nil {
		t.Fatal(err)
	}
	if string(got[:]) != "ping" {
		t.Fatalf("read data = %q", got[:])
	}

	state.Set(corecpu.RAX, uint64(Sys64Write))
	state.Set(corecpu.RDI, 1)
	state.Set(corecpu.RSI, uint64(buffer))
	state.Set(corecpu.RDX, 4)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 4 || output.String() != "ping" {
		t.Fatalf("write: resume=%v err=%v rax=%d output=%q", resume, err, state.Get(corecpu.RAX), output.String())
	}

	state.Set(corecpu.RAX, uint64(Sys64Dup))
	state.Set(corecpu.RDI, 1)
	resume, err = dispatcher.Dispatch(state)
	dupFD := int64(state.Get(corecpu.RAX))
	if err != nil || !resume || dupFD < 3 {
		t.Fatalf("dup: resume=%v err=%v fd=%d", resume, err, dupFD)
	}
	state.Set(corecpu.RAX, uint64(Sys64Close))
	state.Set(corecpu.RDI, uint64(dupFD))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("close: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, uint64(dupFD))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != int64(EBADF) {
		t.Fatalf("closed fd: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
}

func TestDispatcher64OpenatFakeFS(t *testing.T) {
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
	const area corecpu.Address64 = 0x8000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(area, append([]byte("/hello"), 0)); err != nil {
		t.Fatal(err)
	}
	context64 := NewContext64(memory)
	context64.FS = fake
	dispatcher := NewDispatcher64(context64)
	state := corecpu.NewMachineState64(memory)
	state.Set(corecpu.RAX, uint64(Sys64Openat))
	state.Set(corecpu.RDI, atFDCWD64)
	state.Set(corecpu.RSI, uint64(area))
	state.Set(corecpu.RDX, 0)
	state.Set(corecpu.R10, 0)
	resume, err := dispatcher.Dispatch(state)
	fd := int64(state.Get(corecpu.RAX))
	if err != nil || !resume || fd < 3 {
		t.Fatalf("openat: resume=%v err=%v fd=%d", resume, err, fd)
	}

	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, uint64(fd))
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, 5)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 5 {
		t.Fatalf("read opened file: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	var got [5]byte
	if err := memory.Read(area+0x100, got[:]); err != nil {
		t.Fatal(err)
	}
	if string(got[:]) != "hello" {
		t.Fatalf("opened file data = %q", got[:])
	}

	state.Set(corecpu.RAX, uint64(Sys64Close))
	state.Set(corecpu.RDI, uint64(fd))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("close opened file: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
}
