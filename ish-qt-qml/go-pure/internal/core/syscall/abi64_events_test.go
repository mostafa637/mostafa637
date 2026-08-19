package syscall

import (
	"encoding/binary"
	"testing"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func newEventsTestContext(t *testing.T) (*Context64, *Dispatcher64, *corecpu.MachineState64, corecpu.Address64) {
	t.Helper()
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xb000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context64 := NewContext64(memory)
	return context64, NewDispatcher64(context64), corecpu.NewMachineState64(memory), area
}

func TestDispatcher64EventFD(t *testing.T) {
	context64, dispatcher, state, area := newEventsTestContext(t)
	state.Set(corecpu.RAX, uint64(Sys64Eventfd2))
	state.Set(corecpu.RDI, 0)
	state.Set(corecpu.RSI, 0)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("eventfd2: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	fd := state.Get(corecpu.RAX)
	var value [8]byte
	binary.LittleEndian.PutUint64(value[:], 7)
	if err := context64.Memory.Write(area, value[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Write))
	state.Set(corecpu.RDI, fd)
	state.Set(corecpu.RSI, uint64(area))
	state.Set(corecpu.RDX, 8)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 8 {
		t.Fatalf("eventfd write: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, fd)
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, 8)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 8 {
		t.Fatalf("eventfd read: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if err := context64.Memory.Read(area+0x100, value[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(value[:]); got != 7 {
		t.Fatalf("eventfd value = %d", got)
	}
}

func TestDispatcher64TimerFD(t *testing.T) {
	context64, dispatcher, state, area := newEventsTestContext(t)
	state.Set(corecpu.RAX, uint64(Sys64TimerfdCreate))
	state.Set(corecpu.RDI, clockMonotonic64)
	state.Set(corecpu.RSI, 0)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("timerfd_create: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	fd := state.Get(corecpu.RAX)
	var spec [32]byte
	binary.LittleEndian.PutUint64(spec[16:24], 0)
	binary.LittleEndian.PutUint64(spec[24:32], uint64(2_000_000))
	if err := context64.Memory.Write(area, spec[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64TimerfdSettime))
	state.Set(corecpu.RDI, fd)
	state.Set(corecpu.RSI, 0)
	state.Set(corecpu.RDX, uint64(area))
	state.Set(corecpu.R10, 0)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("timerfd_settime: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	start := time.Now()
	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, fd)
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, 8)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 8 {
		t.Fatalf("timerfd read: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if time.Since(start) < time.Millisecond {
		t.Fatal("timerfd read returned before timer expiration")
	}

	absolute := time.Since(context64.StartTime) + 2*time.Millisecond
	var absoluteSpec [32]byte
	binary.LittleEndian.PutUint64(absoluteSpec[16:24], uint64(absolute/time.Second))
	binary.LittleEndian.PutUint64(absoluteSpec[24:32], uint64(absolute%time.Second))
	if err := context64.Memory.Write(area+0x200, absoluteSpec[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64TimerfdSettime))
	state.Set(corecpu.RDI, fd)
	state.Set(corecpu.RSI, timerfdTimerAbs64)
	state.Set(corecpu.RDX, uint64(area+0x200))
	state.Set(corecpu.R10, 0)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("timerfd_settime absolute: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	start = time.Now()
	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, fd)
	state.Set(corecpu.RSI, uint64(area+0x300))
	state.Set(corecpu.RDX, 8)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 8 {
		t.Fatalf("timerfd absolute read: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	if time.Since(start) < time.Millisecond {
		t.Fatal("absolute timerfd read returned before timer expiration")
	}
}

func TestDispatcher64EpollAndInotify(t *testing.T) {
	context64, dispatcher, state, area := newEventsTestContext(t)
	state.Set(corecpu.RAX, uint64(Sys64Eventfd2))
	state.Set(corecpu.RDI, 1)
	state.Set(corecpu.RSI, 0)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume {
		t.Fatalf("eventfd for epoll: resume=%v err=%v", resume, err)
	}
	eventFD := state.Get(corecpu.RAX)
	state.Set(corecpu.RAX, uint64(Sys64EpollCreate1))
	state.Set(corecpu.RDI, 0)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("epoll_create1: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	epollFD := state.Get(corecpu.RAX)
	var event [16]byte
	binary.LittleEndian.PutUint32(event[:4], epollIn64)
	binary.LittleEndian.PutUint64(event[8:], 0xfeed)
	if err := context64.Memory.Write(area, event[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64EpollCtl))
	state.Set(corecpu.RDI, epollFD)
	state.Set(corecpu.RSI, epollCtlAdd64)
	state.Set(corecpu.RDX, eventFD)
	state.Set(corecpu.R10, uint64(area))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("epoll_ctl add: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	var value [8]byte
	binary.LittleEndian.PutUint64(value[:], 1)
	if err := context64.Memory.Write(area+0x100, value[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Write))
	state.Set(corecpu.RDI, eventFD)
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, 8)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume {
		t.Fatalf("epoll eventfd write: resume=%v err=%v", resume, err)
	}
	state.Set(corecpu.RAX, uint64(Sys64EpollWait))
	state.Set(corecpu.RDI, epollFD)
	state.Set(corecpu.RSI, uint64(area+0x200))
	state.Set(corecpu.RDX, 1)
	state.Set(corecpu.R10, 0)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("epoll_wait: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}

	var pathname = []byte("/watch\x00")
	if err := context64.Memory.Write(area+0x300, pathname); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64InotifyInit1))
	state.Set(corecpu.RDI, inotifyNonblock64)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("inotify_init1: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	inotifyFD := state.Get(corecpu.RAX)
	state.Set(corecpu.RAX, uint64(Sys64InotifyAdd))
	state.Set(corecpu.RDI, inotifyFD)
	state.Set(corecpu.RSI, uint64(area+0x300))
	state.Set(corecpu.RDX, 0xff)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) <= 0 {
		t.Fatalf("inotify_add_watch: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	wd := state.Get(corecpu.RAX)
	state.Set(corecpu.RAX, uint64(Sys64InotifyRm))
	state.Set(corecpu.RDI, inotifyFD)
	state.Set(corecpu.RSI, wd)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("inotify_rm_watch: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
}

func TestDispatcher64SignalFD(t *testing.T) {
	context64, dispatcher, state, area := newEventsTestContext(t)
	context64.PID = 42
	var mask [8]byte
	binary.LittleEndian.PutUint64(mask[:], uint64(1)<<(15-1))
	if err := context64.Memory.Write(area, mask[:]); err != nil {
		t.Fatal(err)
	}
	state.Set(corecpu.RAX, uint64(Sys64Signalfd4))
	state.Set(corecpu.RDI, ^uint64(0))
	state.Set(corecpu.RSI, uint64(area))
	state.Set(corecpu.RDX, 8)
	state.Set(corecpu.R10, sfdNonblock64)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) < 3 {
		t.Fatalf("signalfd4: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	fd := state.Get(corecpu.RAX)

	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, fd)
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, 128)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != int64(EAGAIN) {
		t.Fatalf("empty nonblock signalfd read: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}

	state.Set(corecpu.RAX, uint64(Sys64Kill))
	state.Set(corecpu.RDI, context64.PID)
	state.Set(corecpu.RSI, 15)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("kill for signalfd: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}

	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, fd)
	state.Set(corecpu.RSI, uint64(area+0x100))
	state.Set(corecpu.RDX, 128)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 128 {
		t.Fatalf("queued signalfd read: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
	var info [128]byte
	if err := context64.Memory.Read(area+0x100, info[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(info[0:4]); got != 15 {
		t.Fatalf("signalfd signo=%d, want 15", got)
	}
	if got := binary.LittleEndian.Uint32(info[12:16]); got != uint32(context64.PID) {
		t.Fatalf("signalfd pid=%d, want %d", got, context64.PID)
	}
}
