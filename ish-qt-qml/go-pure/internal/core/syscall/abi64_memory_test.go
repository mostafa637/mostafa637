package syscall

import (
	"bytes"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func TestDispatcher64MemorySyscalls(t *testing.T) {
	memory := corecpu.NewMemory64()
	context64 := NewContext64(memory)
	dispatcher := NewDispatcher64(context64)
	state := corecpu.NewMachineState64(memory)

	state.Set(corecpu.RAX, uint64(Sys64Mmap))
	state.Set(corecpu.RDI, 0)
	state.Set(corecpu.RSI, corecpu.Page64Size)
	state.Set(corecpu.RDX, uint64(ProtRead|ProtWrite))
	state.Set(corecpu.R10, uint64(MapPrivate|MapAnonymous))
	state.Set(corecpu.R8, ^uint64(0))
	state.Set(corecpu.R9, 0)
	resume, err := dispatcher.Dispatch(state)
	base := corecpu.Address64(state.Get(corecpu.RAX))
	if err != nil || !resume || base == 0 || uint64(base)&(corecpu.Page64Size-1) != 0 {
		t.Fatalf("mmap: resume=%v err=%v base=%#x", resume, err, base)
	}
	if err := memory.Write(base, []byte("mapped")); err != nil {
		t.Fatalf("mmap write: %v", err)
	}

	state.Set(corecpu.RAX, uint64(Sys64Mprotect))
	state.Set(corecpu.RDI, uint64(base))
	state.Set(corecpu.RSI, corecpu.Page64Size)
	state.Set(corecpu.RDX, uint64(ProtRead))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("mprotect: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Write(base, []byte("blocked")); err != corecpu.ErrProtection {
		t.Fatalf("mprotect write error = %v, want %v", err, corecpu.ErrProtection)
	}

	state.Set(corecpu.RAX, uint64(Sys64Mprotect))
	state.Set(corecpu.RDI, uint64(base+1))
	state.Set(corecpu.RSI, corecpu.Page64Size)
	state.Set(corecpu.RDX, uint64(ProtRead))
	_, _ = dispatcher.Dispatch(state)
	if got := int64(state.Get(corecpu.RAX)); got != int64(EINVAL) {
		t.Fatalf("unaligned mprotect = %d, want %d", got, EINVAL)
	}

	state.Set(corecpu.RAX, uint64(Sys64Munmap))
	state.Set(corecpu.RDI, uint64(base))
	state.Set(corecpu.RSI, corecpu.Page64Size)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("munmap: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if _, ok := memory.MappingFlags(corecpu.Page64(uint64(base) >> corecpu.Page64Bits)); ok {
		t.Fatal("munmap left the page mapped")
	}
}

func TestDispatcher64FixedMappingAndLseek(t *testing.T) {
	memory := corecpu.NewMemory64()
	context64 := NewContext64(memory)
	dispatcher := NewDispatcher64(context64)
	state := corecpu.NewMachineState64(memory)
	const fixed corecpu.Address64 = 0x4000000000

	state.Set(corecpu.RAX, uint64(Sys64Mmap))
	state.Set(corecpu.RDI, uint64(fixed))
	state.Set(corecpu.RSI, corecpu.Page64Size)
	state.Set(corecpu.RDX, uint64(ProtRead|ProtWrite))
	state.Set(corecpu.R10, uint64(MapPrivate|MapAnonymous|MapFixed))
	state.Set(corecpu.R8, ^uint64(0))
	state.Set(corecpu.R9, 0)
	_, _ = dispatcher.Dispatch(state)
	if got := corecpu.Address64(state.Get(corecpu.RAX)); got != fixed {
		t.Fatalf("fixed mmap = %#x, want %#x", got, fixed)
	}

	state.Set(corecpu.RAX, uint64(Sys64Mmap))
	state.Set(corecpu.R10, uint64(MapPrivate|MapAnonymous|MapFixed)|mapFixedNoReplace64)
	_, _ = dispatcher.Dispatch(state)
	if got := int64(state.Get(corecpu.RAX)); got != int64(EEXIST) {
		t.Fatalf("MAP_FIXED_NOREPLACE collision = %d, want %d", got, EEXIST)
	}

	reader := bytes.NewReader([]byte("abcdef"))
	if err := context64.InstallFile(3, &corefd.File{Reader: reader, Seeker: reader}); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Lseek))
	state.Set(corecpu.RDI, 3)
	state.Set(corecpu.RSI, 2)
	state.Set(corecpu.RDX, 0)
	_, _ = dispatcher.Dispatch(state)
	if got := int64(state.Get(corecpu.RAX)); got != 2 {
		t.Fatalf("lseek = %d, want 2", got)
	}

	if err := context64.InstallFile(4, &corefd.File{Writer: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Lseek))
	state.Set(corecpu.RDI, 4)
	_, _ = dispatcher.Dispatch(state)
	if got := int64(state.Get(corecpu.RAX)); got != -29 {
		t.Fatalf("non-seekable lseek = %d, want -29", got)
	}
}
