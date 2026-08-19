package syscall

import (
	"bytes"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64MremapAndMadvise(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	const source corecpu.Address64 = 0x30000
	if err := memory.Map(source, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	payload := []byte("mremap-payload")
	if err := memory.Write(source, payload); err != nil {
		t.Fatal(err)
	}

	set64Syscall(state, Sys64Mremap, uint64(source), corecpu.Page64Size, 2*corecpu.Page64Size, mremapMayMove64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(source) {
		t.Fatalf("mremap grow: err=%v rax=%#x", err, state.Get(corecpu.RAX))
	}
	var got [len("mremap-payload")]byte
	if err := memory.Read(source, got[:]); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got[:], payload) {
		t.Fatalf("mremap payload = %q, want %q", got[:], payload)
	}

	set64Syscall(state, Sys64Madvise, uint64(source), 2*corecpu.Page64Size, madviseDontNeed64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("madvise: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Mremap, uint64(source), 2*corecpu.Page64Size, corecpu.Page64Size, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(source) {
		t.Fatalf("mremap shrink: err=%v rax=%#x", err, state.Get(corecpu.RAX))
	}
	if _, ok := memory.MappingFlags(corecpu.Page64(uint64(source)>>corecpu.Page64Bits) + 1); ok {
		t.Fatal("mremap shrink left the second page mapped")
	}
	if len(ctx.Mappings) != 0 {
		t.Fatalf("anonymous remap metadata length = %d, want 0", len(ctx.Mappings))
	}
}

func TestABI64Faccessat2(t *testing.T) {
	fake, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/hello", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	const pathAddress corecpu.Address64 = 0x10c00
	writeABI64CString(t, memory, pathAddress, "/hello")
	set64Syscall(state, Sys64Faccessat2, atFDCWD64, uint64(pathAddress), accessRead, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("faccessat2 existing: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	writeABI64CString(t, memory, pathAddress, "/missing")
	set64Syscall(state, Sys64Faccessat2, atFDCWD64, uint64(pathAddress), 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ENOENT) {
		t.Fatalf("faccessat2 missing: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	writeABI64CString(t, memory, pathAddress, "/hello")
	set64Syscall(state, Sys64Faccessat2, atFDCWD64, uint64(pathAddress), 0x8, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("faccessat2 invalid mode: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	fd := openABI64TestFile(t, dispatcher, state, memory, "/hello", pathAddress)
	writeABI64CString(t, memory, pathAddress, "")
	set64Syscall(state, Sys64Faccessat2, uint64(fd), uint64(pathAddress), 0, faccessAtEmptyPath64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("faccessat2 empty path: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
