package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcher64FileHandleRoundTripAndSizeNegotiation(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	if err := fake.WriteFile("/data", []byte("handle-data"), 0o644, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.Mkdir("/mnt", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	const (
		pathAddress corecpu.Address64 = 0x10000
		handleAddr  corecpu.Address64 = 0x10200
		mountIDAddr corecpu.Address64 = 0x10220
	)
	writeABI64CString(t, memory, pathAddress, "/data")

	set64Syscall(state, Sys64NameToHandleAt, atFDCWD64, uint64(pathAddress), uint64(handleAddr), uint64(mountIDAddr), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EOVERFLOW) {
		t.Fatalf("size negotiation: err=%v rax=%d, want EOVERFLOW", err, int64(state.Get(corecpu.RAX)))
	}
	var header [8]byte
	if err := memory.Read(handleAddr, header[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(header[0:4]); got != fileHandleBytes64 {
		t.Fatalf("required handle bytes = %d, want %d", got, fileHandleBytes64)
	}
	if got := binary.LittleEndian.Uint32(header[4:8]); got != fileHandleType64 {
		t.Fatalf("negotiated handle type = %d, want %d", got, fileHandleType64)
	}

	binary.LittleEndian.PutUint32(header[0:4], fileHandleBytes64)
	if err := memory.Write(handleAddr, header[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64NameToHandleAt, atFDCWD64, uint64(pathAddress), uint64(handleAddr), uint64(mountIDAddr), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("name_to_handle_at: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	var mountID [4]byte
	if err := memory.Read(mountIDAddr, mountID[:]); err != nil {
		t.Fatal(err)
	}
	if got := int32(binary.LittleEndian.Uint32(mountID[:])); got != fileHandleMountID64 {
		t.Fatalf("mount id = %d, want %d", got, fileHandleMountID64)
	}

	writeABI64CString(t, memory, pathAddress, "/mnt")
	set64Syscall(state, Sys64Open, uint64(pathAddress), guestOpenDirectory, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	mountFD := state.Get(corecpu.RAX)
	set64Syscall(state, Sys64OpenByHandleAt, mountFD, uint64(handleAddr), 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatalf("open_by_handle_at: err=%v", err)
	}
	openedFD := state.Get(corecpu.RAX)
	opened, err := ctx.GetFile(openedFD)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Path != "/data" {
		t.Fatalf("opened path = %q, want /data", opened.Path)
	}

	if err := fake.Rename("/data", "/renamed"); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64OpenByHandleAt, mountFD, uint64(handleAddr), 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatalf("open renamed handle: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	renamedFD := state.Get(corecpu.RAX)
	renamed, err := ctx.GetFile(renamedFD)
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Path != "/renamed" {
		t.Fatalf("renamed handle path = %q, want /renamed", renamed.Path)
	}
}

func TestDispatcher64FileHandleValidationAndDeletedInode(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	if err := fake.WriteFile("/data", []byte("handle-data"), 0o644, 0, 0); err != nil {
		t.Fatal(err)
	}
	const (
		pathAddress corecpu.Address64 = 0x10000
		handleAddr  corecpu.Address64 = 0x10200
		mountIDAddr corecpu.Address64 = 0x10220
	)
	writeABI64CString(t, memory, pathAddress, "/data")
	binaryHeader := make([]byte, fileHandleStructSize64)
	binary.LittleEndian.PutUint32(binaryHeader[0:4], fileHandleBytes64)
	if err := memory.Write(handleAddr, binaryHeader); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64NameToHandleAt, atFDCWD64, uint64(pathAddress), uint64(handleAddr), uint64(mountIDAddr), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("name_to_handle_at: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}

	set64Syscall(state, Sys64OpenByHandleAt, 99, uint64(handleAddr), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EBADF) {
		t.Fatalf("bad mount fd: err=%v rax=%d, want EBADF", err, int64(state.Get(corecpu.RAX)))
	}

	var invalidType [4]byte
	binary.LittleEndian.PutUint32(invalidType[:], 99)
	if err := memory.Write(handleAddr+4, invalidType[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64OpenByHandleAt, atFDCWD64, uint64(handleAddr), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EBADF) {
		t.Fatalf("AT_FDCWD mount fd: err=%v rax=%d, want EBADF", err, int64(state.Get(corecpu.RAX)))
	}

	if err := fake.Mkdir("/mnt", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	writeABI64CString(t, memory, pathAddress, "/mnt")
	set64Syscall(state, Sys64Open, uint64(pathAddress), guestOpenDirectory, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	mountFD := state.Get(corecpu.RAX)
	set64Syscall(state, Sys64OpenByHandleAt, mountFD, uint64(handleAddr), 0x04)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid open flags: err=%v rax=%d, want EINVAL", err, int64(state.Get(corecpu.RAX)))
	}

	writeABI64CString(t, memory, pathAddress, "/data")
	if err := fake.Unlink("/data"); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint32(invalidType[:], fileHandleType64)
	if err := memory.Write(handleAddr+4, invalidType[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64OpenByHandleAt, mountFD, uint64(handleAddr), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ENOENT) {
		t.Fatalf("deleted handle: err=%v rax=%d, want ENOENT", err, int64(state.Get(corecpu.RAX)))
	}
	_ = ctx
}
