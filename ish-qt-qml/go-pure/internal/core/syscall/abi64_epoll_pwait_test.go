package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcher64EpollPwaitVariants(t *testing.T) {
	ctx, dispatcher, state, area := newEventsTestContext(t)

	set64Syscall(state, Sys64Eventfd2, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 0 {
		t.Fatalf("eventfd2: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	eventFD := state.Get(corecpu.RAX)

	set64Syscall(state, Sys64EpollCreate1, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 0 {
		t.Fatalf("epoll_create1: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	epollFD := state.Get(corecpu.RAX)

	var registration [16]byte
	binary.LittleEndian.PutUint32(registration[:4], epollIn64)
	binary.LittleEndian.PutUint64(registration[8:], 0xcafe)
	if err := ctx.Memory.Write(area, registration[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64EpollCtl, epollFD, epollCtlAdd64, eventFD, uint64(area))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("epoll_ctl: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64EpollPwait, epollFD, uint64(area+0x100), 1, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("epoll_pwait zero timeout: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	var value [8]byte
	binary.LittleEndian.PutUint64(value[:], 1)
	if err := ctx.Memory.Write(area+0x200, value[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Write, eventFD, uint64(area+0x200), 8)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 8 {
		t.Fatalf("eventfd write: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	set64Syscall(state, Sys64EpollPwait, epollFD, uint64(area+0x100), 1, 100, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("epoll_pwait ready: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var event [16]byte
	if err := ctx.Memory.Read(area+0x100, event[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(event[:4]); got&epollIn64 == 0 || binary.LittleEndian.Uint64(event[8:]) != 0xcafe {
		t.Fatalf("epoll_pwait event: events=%#x data=%#x", binary.LittleEndian.Uint32(event[:4]), binary.LittleEndian.Uint64(event[8:]))
	}
	set64Syscall(state, Sys64Read, eventFD, uint64(area+0x250), 8)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 8 {
		t.Fatalf("eventfd read: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	var timespec [16]byte
	binary.LittleEndian.PutUint64(timespec[0:8], 0)
	binary.LittleEndian.PutUint64(timespec[8:16], 0)
	if err := ctx.Memory.Write(area+0x300, timespec[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64EpollPwait2, epollFD, uint64(area+0x100), 1, uint64(area+0x300), 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("epoll_pwait2 zero timeout: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	ctx.SignalMu.Lock()
	ctx.PendingSignals = signalBit64(10)
	ctx.SignalMask = 0
	ctx.SignalMu.Unlock()
	set64Syscall(state, Sys64EpollPwait, epollFD, uint64(area+0x100), 1, 0, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINTR) {
		t.Fatalf("epoll_pwait pending signal: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	var blockedMask [8]byte
	binary.LittleEndian.PutUint64(blockedMask[:], signalBit64(10))
	if err := ctx.Memory.Write(area+0x400, blockedMask[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64EpollPwait, epollFD, uint64(area+0x100), 1, 0, uint64(area+0x400), sigSetSize64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("epoll_pwait blocked signal: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
