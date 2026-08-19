package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64PollAndSelect(t *testing.T) {
	fake, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/ready", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	const (
		pathAddress  corecpu.Address64 = 0x10900
		pollAddress  corecpu.Address64 = 0x10a00
		readAddress  corecpu.Address64 = 0x10b00
		writeAddress corecpu.Address64 = 0x10c00
		timeAddress  corecpu.Address64 = 0x10d00
	)
	writeABI64CString(t, memory, pathAddress, "/ready")
	set64Syscall(state, Sys64Open, uint64(pathAddress), 2, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 0 {
		t.Fatalf("open ready: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	fd := uint32(state.Get(corecpu.RAX))
	var pollEntry [8]byte
	binary.LittleEndian.PutUint32(pollEntry[0:4], fd)
	binary.LittleEndian.PutUint16(pollEntry[4:6], pollIn64|pollOut64)
	if err := memory.Write(pollAddress, pollEntry[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Poll, uint64(pollAddress), 1, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("poll: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(pollAddress+6, pollEntry[6:8]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint16(pollEntry[6:8]); got != pollIn64|pollOut64 {
		t.Fatalf("poll revents=%#x, want %#x", got, pollIn64|pollOut64)
	}
	set64Syscall(state, Sys64Ppoll, uint64(pollAddress), 1, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("ppoll: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Write(readAddress, []byte{1 << uint(fd)}); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(writeAddress, []byte{0}); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Select, uint64(fd+1), uint64(readAddress), uint64(writeAddress), 0, uint64(timeAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("select: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var selected [1]byte
	if err := memory.Read(readAddress, selected[:]); err != nil {
		t.Fatal(err)
	}
	if selected[0] != 1<<uint(fd) {
		t.Fatalf("select read set=%#x, want %#x", selected[0], 1<<uint(fd))
	}
	if err := memory.Write(readAddress, []byte{1 << uint(fd)}); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Pselect6, uint64(fd+1), uint64(readAddress), 0, 0, uint64(timeAddress), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("pselect6: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64PollInvalidDescriptor(t *testing.T) {
	_, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	const address corecpu.Address64 = 0x11000
	var entry [8]byte
	binary.LittleEndian.PutUint32(entry[0:4], 999)
	binary.LittleEndian.PutUint16(entry[4:6], pollIn64)
	if err := memory.Write(address, entry[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Poll, uint64(address), 1, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("invalid poll: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(address+6, entry[6:8]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint16(entry[6:8]); got != pollNval64 {
		t.Fatalf("invalid poll revents=%#x, want %#x", got, pollNval64)
	}
}

func TestABI64PpollAndPselectTemporarySignalMask(t *testing.T) {
	fake, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	file, err := fake.Create("/ready", 0o644, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	const (
		pathAddress    corecpu.Address64 = 0x11100
		pollAddress    corecpu.Address64 = 0x11200
		readAddress    corecpu.Address64 = 0x11300
		timeAddress    corecpu.Address64 = 0x11400
		maskAddress    corecpu.Address64 = 0x11500
		sigmaskAddress corecpu.Address64 = 0x11600
	)
	writeABI64CString(t, memory, pathAddress, "/ready")
	set64Syscall(state, Sys64Open, uint64(pathAddress), 2, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 0 {
		t.Fatalf("open ready: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	fd := uint32(state.Get(corecpu.RAX))
	var pollEntry [8]byte
	binary.LittleEndian.PutUint32(pollEntry[0:4], fd)
	binary.LittleEndian.PutUint16(pollEntry[4:6], pollIn64)
	if err := memory.Write(pollAddress, pollEntry[:]); err != nil {
		t.Fatal(err)
	}
	var blockedMask [8]byte
	binary.LittleEndian.PutUint64(blockedMask[:], signalBit64(10))
	if err := memory.Write(maskAddress, blockedMask[:]); err != nil {
		t.Fatal(err)
	}
	var sigmask [16]byte
	binary.LittleEndian.PutUint64(sigmask[0:8], uint64(maskAddress))
	binary.LittleEndian.PutUint64(sigmask[8:16], 8)
	if err := memory.Write(sigmaskAddress, sigmask[:]); err != nil {
		t.Fatal(err)
	}
	if err := memory.Write(timeAddress, make([]byte, 16)); err != nil {
		t.Fatal(err)
	}
	ctx.SignalMu.Lock()
	ctx.PendingSignals = signalBit64(10)
	ctx.SignalMask = signalBit64(12)
	originalMask := ctx.SignalMask
	ctx.SignalMu.Unlock()
	set64Syscall(state, Sys64Ppoll, uint64(pollAddress), 1, uint64(timeAddress), uint64(maskAddress), 8)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("ppoll masked signal: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	ctx.SignalMu.Lock()
	if ctx.SignalMask != originalMask {
		t.Fatalf("ppoll changed signal mask: got %#x want %#x", ctx.SignalMask, originalMask)
	}
	ctx.SignalMu.Unlock()

	if err := memory.Write(readAddress, []byte{1 << uint(fd)}); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Pselect6, uint64(fd+1), uint64(readAddress), 0, 0, uint64(timeAddress), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINTR) {
		t.Fatalf("pselect unmasked signal: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64PpollInvalidSignalMaskSize(t *testing.T) {
	_, memory, _, dispatcher, state := newABI64FilesystemTest(t)
	const (
		pollAddress corecpu.Address64 = 0x11700
		maskAddress corecpu.Address64 = 0x11800
	)
	if err := memory.Write(maskAddress, make([]byte, 8)); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Ppoll, uint64(pollAddress), 0, 0, uint64(maskAddress), 4)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("ppoll invalid signal mask size: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
