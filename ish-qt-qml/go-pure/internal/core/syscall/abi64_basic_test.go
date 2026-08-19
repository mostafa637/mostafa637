package syscall

import (
	"context"
	"path/filepath"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

func TestDispatcher64BasicIdentityAndPaths(t *testing.T) {
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
	if err := fake.Mkdir("/etc", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile("/etc/config", []byte("pure-go"), 0o644, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.Symlink("/etc/config", "/etc/link", 0, 0); err != nil {
		t.Fatal(err)
	}

	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0x9000
	if err := memory.Map(area, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	writeCString := func(address corecpu.Address64, value string) {
		t.Helper()
		if err := memory.Write(address, append([]byte(value), 0)); err != nil {
			t.Fatal(err)
		}
	}
	readBytes := func(address corecpu.Address64, length int) []byte {
		t.Helper()
		value := make([]byte, length)
		if err := memory.Read(address, value); err != nil {
			t.Fatal(err)
		}
		return value
	}

	ctx := NewContext64(memory)
	ctx.FS = fake
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	writeCString(area, "/etc")
	state.Set(corecpu.RAX, uint64(Sys64Chdir))
	state.Set(corecpu.RDI, uint64(area))
	if resume, err := dispatcher.Dispatch(state); err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("chdir: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	state.Set(corecpu.RAX, uint64(Sys64GetCWD))
	state.Set(corecpu.RDI, uint64(area+0x100))
	state.Set(corecpu.RSI, 64)
	if resume, err := dispatcher.Dispatch(state); err != nil || !resume || int64(state.Get(corecpu.RAX)) != int64(len("/etc")+1) {
		t.Fatalf("getcwd: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if got := string(readBytes(area+0x100, len("/etc")+1)); got != "/etc\x00" {
		t.Fatalf("getcwd = %q", got)
	}

	writeCString(area+0x200, "link")
	state.Set(corecpu.RAX, uint64(Sys64Readlink))
	state.Set(corecpu.RDI, uint64(area+0x200))
	state.Set(corecpu.RSI, uint64(area+0x300))
	state.Set(corecpu.RDX, 64)
	if resume, err := dispatcher.Dispatch(state); err != nil || !resume || int64(state.Get(corecpu.RAX)) != int64(len("/etc/config")) {
		t.Fatalf("readlink relative: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if got := string(readBytes(area+0x300, len("/etc/config"))); got != "/etc/config" {
		t.Fatalf("readlink = %q", got)
	}

	state.Set(corecpu.RAX, uint64(Sys64Uname))
	state.Set(corecpu.RDI, uint64(area+0x400))
	if resume, err := dispatcher.Dispatch(state); err != nil || !resume || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("uname: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	uname := readBytes(area+0x400, uname64FieldSize*6)
	if got := string(uname[:5]); got != "Linux" {
		t.Fatalf("uname sysname = %q", got)
	}
	if got := string(uname[4*uname64FieldSize : 4*uname64FieldSize+6]); got != "x86_64" {
		t.Fatalf("uname machine = %q", got)
	}
}
