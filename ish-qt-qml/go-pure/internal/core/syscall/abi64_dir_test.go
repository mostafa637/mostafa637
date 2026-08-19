package syscall

import (
	"bytes"
	"context"
	"encoding/binary"
	"path/filepath"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

func TestDispatcher64Getdents64(t *testing.T) {
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
	if err := fake.Mkdir("/dir", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile("/dir/a", []byte("a"), 0o644, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.Symlink("a", "/dir/link", 0, 0); err != nil {
		t.Fatal(err)
	}

	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xa000
	if err := memory.Map(area, 3*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.FS = fake
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)
	writeCString := func(address corecpu.Address64, value string) {
		t.Helper()
		if err := memory.Write(address, append([]byte(value), 0)); err != nil {
			t.Fatal(err)
		}
	}
	writeCString(area, "/dir")
	state.Set(corecpu.RAX, uint64(Sys64Open))
	state.Set(corecpu.RDI, uint64(area))
	state.Set(corecpu.RSI, 0)
	state.Set(corecpu.RDX, 0)
	if resume, err := dispatcher.Dispatch(state); err != nil || !resume || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("open directory: resume=%v err=%v fd=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	fd := state.Get(corecpu.RAX)
	buffer := area + corecpu.Address64(corecpu.Page64Size)

	readRecords := func(n int64) (names []string, types []byte) {
		t.Helper()
		if n < 0 {
			t.Fatalf("negative getdents result: %d", n)
		}
		data := make([]byte, n)
		if err := memory.Read(buffer, data); err != nil {
			t.Fatal(err)
		}
		for offset := 0; offset < len(data); {
			if len(data)-offset < 19 {
				t.Fatalf("short linux_dirent64 prefix at %d: %d bytes remain", offset, len(data)-offset)
			}
			recordLength := int(binary.LittleEndian.Uint16(data[offset+16 : offset+18]))
			if recordLength < 24 || offset+recordLength > len(data) || recordLength%8 != 0 {
				t.Fatalf("invalid linux_dirent64 reclen=%d offset=%d total=%d", recordLength, offset, len(data))
			}
			nameBytes := data[offset+19 : offset+recordLength]
			if end := bytes.IndexByte(nameBytes, 0); end >= 0 {
				nameBytes = nameBytes[:end]
			} else {
				t.Fatalf("directory name is not NUL-terminated at %d", offset)
			}
			names = append(names, string(nameBytes))
			types = append(types, data[offset+18])
			offset += recordLength
		}
		return names, types
	}

	dispatchGetdents := func(count uint64, destination corecpu.Address64) int64 {
		t.Helper()
		state.Set(corecpu.RAX, uint64(Sys64Getdents64))
		state.Set(corecpu.RDI, fd)
		state.Set(corecpu.RSI, uint64(destination))
		state.Set(corecpu.RDX, count)
		if resume, err := dispatcher.Dispatch(state); err != nil || !resume {
			t.Fatalf("getdents dispatch: resume=%v err=%v", resume, err)
		}
		return int64(state.Get(corecpu.RAX))
	}

	first := dispatchGetdents(64, buffer)
	if first <= 0 {
		t.Fatalf("first getdents = %d", first)
	}
	firstNames, firstTypes := readRecords(first)
	if len(firstNames) != 2 || firstNames[0] != "." || firstNames[1] != ".." {
		t.Fatalf("first names = %v", firstNames)
	}
	if firstTypes[0] != dTypeDir || firstTypes[1] != dTypeDir {
		t.Fatalf("dot types = %v", firstTypes)
	}

	second := dispatchGetdents(64, buffer)
	secondNames, secondTypes := readRecords(second)
	if len(secondNames) != 2 || secondNames[0] != "a" || secondNames[1] != "link" {
		t.Fatalf("second names = %v", secondNames)
	}
	if secondTypes[0] != dTypeRegular || secondTypes[1] != dTypeSymlink {
		t.Fatalf("entry types = %v", secondTypes)
	}
	if got := dispatchGetdents(64, buffer); got != 0 {
		t.Fatalf("EOF getdents = %d, want 0", got)
	}

	file, err := ctx.GetFile(fd)
	if err != nil {
		t.Fatal(err)
	}
	file.DirPos = 0
	if got := dispatchGetdents(8, buffer); got != int64(EINVAL) {
		t.Fatalf("small buffer getdents = %d, want %d", got, EINVAL)
	}
	file.DirPos = 0
	if got := dispatchGetdents(64, area+corecpu.Address64(3*corecpu.Page64Size)); got != int64(EFAULT) {
		t.Fatalf("bad buffer getdents = %d, want %d", got, EFAULT)
	}
}
