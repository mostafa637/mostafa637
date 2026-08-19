package syscall

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

const (
	efdSemaphore64 uint64 = 1
	efdNonblock64  uint64 = 0x800
	efdCloexec64   uint64 = 0x80000

	tfdNonblock64 uint64 = 0x800
	tfdCloexec64  uint64 = 0x80000

	epollIn64         uint32 = 0x001
	epollOut64        uint32 = 0x004
	epollErr64        uint32 = 0x008
	epollHup64        uint32 = 0x010
	epollCtlAdd64            = 1
	epollCtlDel64            = 2
	epollCtlMod64            = 3
	clockMonotonic64         = 1
	inotifyNonblock64        = 0x800
	inotifyCloexec64         = 0x80000
)

var (
	errWouldBlock64     = errors.New("x86-64 guest operation would block")
	errInvalidEventFD64 = errors.New("x86-64 guest invalid eventfd value")
)

type eventFD64 struct {
	mu        sync.Mutex
	cond      *sync.Cond
	counter   uint64
	semaphore bool
	nonblock  bool
	closed    bool
}

func newEventFD64(initial uint64, flags uint64) *eventFD64 {
	e := &eventFD64{counter: initial, semaphore: flags&efdSemaphore64 != 0, nonblock: flags&efdNonblock64 != 0}
	e.cond = sync.NewCond(&e.mu)
	return e
}

func (e *eventFD64) Read(dst []byte) (int, error) {
	if len(dst) < 8 {
		return 0, io.ErrShortBuffer
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for e.counter == 0 && !e.closed {
		if e.nonblock {
			return 0, errWouldBlock64
		}
		e.cond.Wait()
	}
	if e.closed && e.counter == 0 {
		return 0, io.EOF
	}
	value := e.counter
	if e.semaphore {
		value = 1
		e.counter--
	} else {
		e.counter = 0
	}
	binary.LittleEndian.PutUint64(dst[:8], value)
	e.cond.Broadcast()
	return 8, nil
}

func (e *eventFD64) Write(src []byte) (int, error) {
	if len(src) < 8 {
		return 0, io.ErrShortBuffer
	}
	value := binary.LittleEndian.Uint64(src[:8])
	if value == ^uint64(0) {
		return 0, errInvalidEventFD64
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return 0, io.ErrClosedPipe
	}
	if ^uint64(0)-1-e.counter < value {
		if e.nonblock {
			return 0, errWouldBlock64
		}
		for ^uint64(0)-1-e.counter < value && !e.closed {
			e.cond.Wait()
		}
		if e.closed {
			return 0, io.ErrClosedPipe
		}
	}
	e.counter += value
	e.cond.Broadcast()
	return 8, nil
}

func (e *eventFD64) Close() error {
	e.mu.Lock()
	e.closed = true
	e.cond.Broadcast()
	e.mu.Unlock()
	return nil
}

func (e *eventFD64) Poll(events uint16) uint16 {
	e.mu.Lock()
	defer e.mu.Unlock()
	var ready uint16
	if e.counter > 0 {
		ready |= uint16(epollIn64)
	}
	if e.counter < ^uint64(0)-1 && !e.closed {
		ready |= uint16(epollOut64)
	}
	return ready & events
}

type timerFD64 struct {
	mu       sync.Mutex
	cond     *sync.Cond
	interval time.Duration
	next     time.Time
	nonblock bool
	closed   bool
}

func newTimerFD64(flags uint64) *timerFD64 {
	t := &timerFD64{nonblock: flags&tfdNonblock64 != 0}
	t.cond = sync.NewCond(&t.mu)
	return t
}

func (t *timerFD64) Read(dst []byte) (int, error) {
	if len(dst) < 8 {
		return 0, io.ErrShortBuffer
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for {
		if t.closed {
			return 0, io.EOF
		}
		if t.next.IsZero() {
			if t.nonblock {
				return 0, errWouldBlock64
			}
			t.cond.Wait()
			continue
		}
		now := time.Now()
		if now.Before(t.next) {
			if t.nonblock {
				return 0, errWouldBlock64
			}
			delay := time.Until(t.next)
			t.mu.Unlock()
			timer := time.NewTimer(delay)
			<-timer.C
			t.mu.Lock()
			continue
		}
		count := uint64(1)
		if t.interval > 0 {
			count += uint64(now.Sub(t.next) / t.interval)
			t.next = t.next.Add(time.Duration(count) * t.interval)
		} else {
			t.next = time.Time{}
		}
		binary.LittleEndian.PutUint64(dst[:8], count)
		return 8, nil
	}
}

func (t *timerFD64) Write([]byte) (int, error) { return 0, errInvalidEventFD64 }

func (t *timerFD64) Close() error {
	t.mu.Lock()
	t.closed = true
	t.cond.Broadcast()
	t.mu.Unlock()
	return nil
}

func (t *timerFD64) Poll(events uint16) uint16 {
	t.mu.Lock()
	defer t.mu.Unlock()
	var ready uint16
	if !t.next.IsZero() && !time.Now().Before(t.next) {
		ready |= uint16(epollIn64)
	}
	if !t.closed {
		ready |= uint16(epollOut64)
	}
	return ready & events
}

type epollWatch64 struct {
	fd     uint64
	events uint32
	data   uint64
}

type epollFD64 struct {
	mu      sync.Mutex
	watches map[uint64]epollWatch64
	closed  bool
}

func newEpollFD64() *epollFD64 { return &epollFD64{watches: make(map[uint64]epollWatch64)} }
func (e *epollFD64) Close() error {
	e.mu.Lock()
	e.closed = true
	e.mu.Unlock()
	return nil
}

func eventfd2_64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil || args[0] == ^uint64(0) {
		return int64(EINVAL)
	}
	if args[1]&^(efdSemaphore64|efdNonblock64|efdCloexec64) != 0 {
		return int64(EINVAL)
	}
	file := &corefd.File{Reader: newEventFD64(args[0], args[1]), Cloexec: args[1]&efdCloexec64 != 0}
	handle := file.Reader.(*eventFD64)
	file.Writer = handle
	file.Closer = handle
	file.Poll = handle.Poll
	fd, err := ctx.FDs.Open(file)
	if err != nil {
		return int64(ENOMEM)
	}
	return int64(fd)
}

func timerfdCreate64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil || args[0] != 0 && args[0] != clockMonotonic64 {
		return int64(EINVAL)
	}
	if args[1]&^(tfdNonblock64|tfdCloexec64) != 0 {
		return int64(EINVAL)
	}
	handle := newTimerFD64(args[1])
	file := &corefd.File{Reader: handle, Writer: handle, Closer: handle, Poll: handle.Poll, Cloexec: args[1]&tfdCloexec64 != 0}
	fd, err := ctx.FDs.Open(file)
	if err != nil {
		return int64(ENOMEM)
	}
	return int64(fd)
}

func readTimespec64(ctx *Context64, address corecpu.Address64) (time.Duration, bool) {
	var raw [16]byte
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Read(address, raw[:]) != nil {
		return 0, false
	}
	seconds := int64(binary.LittleEndian.Uint64(raw[:8]))
	nanos := int64(binary.LittleEndian.Uint64(raw[8:]))
	if seconds < 0 || nanos < 0 || nanos >= int64(time.Second) {
		return 0, false
	}
	return time.Duration(seconds)*time.Second + time.Duration(nanos), true
}

func writeTimespec64(ctx *Context64, address corecpu.Address64, duration time.Duration) bool {
	if duration < 0 {
		duration = 0
	}
	var raw [16]byte
	binary.LittleEndian.PutUint64(raw[:8], uint64(duration/time.Second))
	binary.LittleEndian.PutUint64(raw[8:], uint64(duration%time.Second))
	return ctx != nil && ctx.Memory != nil && ctx.Memory.Write(address, raw[:]) == nil
}

func timerfdSettime64(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := file.Reader.(*timerFD64)
	if !ok {
		return int64(EINVAL)
	}
	if args[1]&^uint64(1) != 0 {
		return int64(EINVAL)
	}
	interval, ok := readTimespec64(ctx, corecpu.Address64(args[2]))
	if !ok {
		return int64(EFAULT)
	}
	var value [16]byte
	if ctx.Memory.Read(corecpu.Address64(args[2])+16, value[:]) != nil {
		return int64(EFAULT)
	}
	seconds := int64(binary.LittleEndian.Uint64(value[:8]))
	nanos := int64(binary.LittleEndian.Uint64(value[8:]))
	if seconds < 0 || nanos < 0 || nanos >= int64(time.Second) {
		return int64(EINVAL)
	}
	initial := time.Duration(seconds)*time.Second + time.Duration(nanos)
	handle.mu.Lock()
	if args[3] != 0 {
		remaining := time.Duration(0)
		if !handle.next.IsZero() {
			remaining = time.Until(handle.next)
		}
		if !writeTimespec64(ctx, corecpu.Address64(args[3]), remaining) || !writeTimespec64(ctx, corecpu.Address64(args[3])+16, handle.interval) {
			handle.mu.Unlock()
			return int64(EFAULT)
		}
	}
	handle.interval = interval
	if initial == 0 {
		handle.next = time.Time{}
	} else {
		handle.next = time.Now().Add(initial)
	}
	handle.cond.Broadcast()
	handle.mu.Unlock()
	return 0
}

func timerfdGettime64(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := file.Reader.(*timerFD64)
	if !ok {
		return int64(EINVAL)
	}
	handle.mu.Lock()
	remaining := time.Duration(0)
	if !handle.next.IsZero() {
		remaining = time.Until(handle.next)
	}
	interval := handle.interval
	handle.mu.Unlock()
	if !writeTimespec64(ctx, corecpu.Address64(args[1]), remaining) || !writeTimespec64(ctx, corecpu.Address64(args[1])+16, interval) {
		return int64(EFAULT)
	}
	return 0
}

func epollCreate164(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil || args[0]&^uint64(0x80000) != 0 {
		return int64(EINVAL)
	}
	handle := newEpollFD64()
	file := &corefd.File{Opaque: handle, Closer: handle}
	fd, err := ctx.FDs.Open(file)
	if err != nil {
		return int64(ENOMEM)
	}
	file.Cloexec = args[0]&0x80000 != 0
	return int64(fd)
}

func epollCtl64(ctx *Context64, args [6]uint64) int64 {
	epollFile, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := epollFile.Opaque.(*epollFD64)
	if !ok {
		return int64(EINVAL)
	}
	if _, err := ctx.GetFile(args[2]); err != nil {
		return int64(EBADF)
	}
	var event [16]byte
	if args[3] != 0 && ctx.Memory.Read(corecpu.Address64(args[3]), event[:]) != nil {
		return int64(EFAULT)
	}
	watch := epollWatch64{fd: args[2], events: binary.LittleEndian.Uint32(event[:4]), data: binary.LittleEndian.Uint64(event[8:])}
	handle.mu.Lock()
	defer handle.mu.Unlock()
	switch args[1] {
	case epollCtlAdd64:
		if _, exists := handle.watches[args[2]]; exists {
			return int64(EEXIST)
		}
		handle.watches[args[2]] = watch
	case epollCtlMod64:
		if _, exists := handle.watches[args[2]]; !exists {
			return int64(ENOENT)
		}
		handle.watches[args[2]] = watch
	case epollCtlDel64:
		if _, exists := handle.watches[args[2]]; !exists {
			return int64(ENOENT)
		}
		delete(handle.watches, args[2])
	default:
		return int64(EINVAL)
	}
	return 0
}

func epollWait64(ctx *Context64, args [6]uint64) int64 {
	epollFile, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := epollFile.Opaque.(*epollFD64)
	if !ok || args[2] == 0 || args[2] > 4096 {
		return int64(EINVAL)
	}
	deadline := time.Time{}
	if int64(args[3]) >= 0 {
		deadline = time.Now().Add(time.Duration(args[3]) * time.Millisecond)
	}
	for {
		ready := make([]epollWatch64, 0)
		handle.mu.Lock()
		watches := make([]epollWatch64, 0, len(handle.watches))
		for _, watch := range handle.watches {
			watches = append(watches, watch)
		}
		handle.mu.Unlock()
		for _, watch := range watches {
			file, getErr := ctx.GetFile(watch.fd)
			if getErr != nil {
				continue
			}
			mask := uint16(watch.events)
			if file.Poll != nil {
				mask = file.Poll(mask)
			}
			if mask != 0 {
				watch.events = uint32(mask)
				ready = append(ready, watch)
				if uint64(len(ready)) >= args[2] {
					break
				}
			}
		}
		if len(ready) > 0 {
			for i, watch := range ready {
				var event [16]byte
				binary.LittleEndian.PutUint32(event[:4], watch.events)
				binary.LittleEndian.PutUint64(event[8:], watch.data)
				if ctx.Memory.Write(corecpu.Address64(args[1]+uint64(i*16)), event[:]) != nil {
					return int64(EFAULT)
				}
			}
			return int64(len(ready))
		}
		if int64(args[3]) == 0 || (!deadline.IsZero() && !time.Now().Before(deadline)) {
			return 0
		}
		time.Sleep(time.Millisecond)
	}
}

type inotifyWatch64 struct {
	path string
	mask uint32
}

type inotifyFD64 struct {
	mu      sync.Mutex
	nextWD  int32
	watches map[int32]inotifyWatch64
	closed  bool
}

func (i *inotifyFD64) Read([]byte) (int, error) { return 0, errWouldBlock64 }
func (i *inotifyFD64) Close() error {
	i.mu.Lock()
	i.closed = true
	i.watches = nil
	i.mu.Unlock()
	return nil
}
func (i *inotifyFD64) Poll(uint16) uint16 { return 0 }

func inotifyInit164(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil || args[0]&^(inotifyNonblock64|inotifyCloexec64) != 0 {
		return int64(EINVAL)
	}
	handle := &inotifyFD64{nextWD: 1, watches: make(map[int32]inotifyWatch64)}
	file := &corefd.File{Reader: handle, Closer: handle, Poll: handle.Poll, Cloexec: args[0]&inotifyCloexec64 != 0}
	fd, err := ctx.FDs.Open(file)
	if err != nil {
		return int64(ENOMEM)
	}
	return int64(fd)
}

func inotifyAddWatch64(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := file.Reader.(*inotifyFD64)
	if !ok {
		return int64(EINVAL)
	}
	name, valid := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	if !valid {
		return int64(EFAULT)
	}
	if ctx.FS != nil {
		if _, statErr := ctx.FS.Stat(name); statErr != nil {
			return int64(ENOENT)
		}
	}
	handle.mu.Lock()
	wd := handle.nextWD
	handle.nextWD++
	handle.watches[wd] = inotifyWatch64{path: name, mask: uint32(args[2])}
	handle.mu.Unlock()
	return int64(wd)
}

func inotifyRmWatch64(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	handle, ok := file.Reader.(*inotifyFD64)
	if !ok {
		return int64(EINVAL)
	}
	handle.mu.Lock()
	defer handle.mu.Unlock()
	if _, exists := handle.watches[int32(args[1])]; !exists {
		return int64(EINVAL)
	}
	delete(handle.watches, int32(args[1]))
	return 0
}

const (
	sfdNonblock64 = 0x800
	sfdCloexec64  = 0x80000
	sigsetSize64  = 8
	sigMax64      = 64
)

// signalFD64 is the guest-visible portion of signalfd. Signals are queued in
// the guest descriptor rather than delivered through the host process, which
// keeps the implementation deterministic and avoids CGo/runtime signal state.
type signalFD64 struct {
	mu       sync.Mutex
	cond     *sync.Cond
	mask     uint64
	nonblock bool
	closed   bool
	queue    [][128]byte
}

func newSignalFD64(mask uint64, flags uint64) *signalFD64 {
	s := &signalFD64{mask: mask, nonblock: flags&sfdNonblock64 != 0}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *signalFD64) accepts(signo uint32) bool {
	return signo >= 1 && signo <= sigMax64 && s.mask&(uint64(1)<<(signo-1)) != 0
}

func (s *signalFD64) enqueue(signo uint32, pid, uid uint32) {
	if s == nil || !s.accepts(signo) {
		return
	}
	var info [128]byte
	binary.LittleEndian.PutUint32(info[0:4], signo)
	binary.LittleEndian.PutUint32(info[12:16], pid)
	binary.LittleEndian.PutUint32(info[16:20], uid)
	s.mu.Lock()
	if !s.closed {
		s.queue = append(s.queue, info)
		s.cond.Broadcast()
	}
	s.mu.Unlock()
}

func (s *signalFD64) Read(dst []byte) (int, error) {
	if len(dst) < 128 {
		return 0, io.ErrShortBuffer
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for len(s.queue) == 0 && !s.closed {
		if s.nonblock {
			return 0, errWouldBlock64
		}
		s.cond.Wait()
	}
	if len(s.queue) == 0 {
		return 0, io.EOF
	}
	copy(dst[:128], s.queue[0][:])
	s.queue = s.queue[1:]
	return 128, nil
}

func (s *signalFD64) Poll(events uint16) uint16 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ready uint16
	if len(s.queue) > 0 {
		ready |= uint16(epollIn64)
	}
	if s.closed {
		ready |= uint16(epollHup64)
	}
	return ready & events
}

func (s *signalFD64) Close() error {
	s.mu.Lock()
	s.closed = true
	s.cond.Broadcast()
	s.mu.Unlock()
	return nil
}

func signalfd4_64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FDs == nil || args[2] != sigsetSize64 {
		return int64(EINVAL)
	}
	if args[3]&^uint64(sfdNonblock64|sfdCloexec64) != 0 {
		return int64(EINVAL)
	}
	var maskBytes [sigsetSize64]byte
	if args[1] == 0 || ctx.Memory.Read(corecpu.Address64(args[1]), maskBytes[:]) != nil {
		return int64(EFAULT)
	}
	mask := binary.LittleEndian.Uint64(maskBytes[:])
	if args[0] != ^uint64(0) {
		if args[0] > maxFD64 {
			return int64(EBADF)
		}
		file, err := ctx.GetFile(args[0])
		if err != nil {
			return int64(EBADF)
		}
		handle, ok := file.Opaque.(*signalFD64)
		if !ok {
			return int64(EINVAL)
		}
		handle.mu.Lock()
		handle.mask = mask
		handle.nonblock = args[3]&sfdNonblock64 != 0
		handle.mu.Unlock()
		return int64(args[0])
	}
	handle := newSignalFD64(mask, args[3])
	file := &corefd.File{Reader: handle, Opaque: handle, Closer: handle, Poll: handle.Poll, Cloexec: args[3]&sfdCloexec64 != 0}
	fd, err := ctx.FDs.Open(file)
	if err != nil {
		return int64(ENOMEM)
	}
	ctx.signalFDs[handle] = struct{}{}
	return int64(fd)
}

func kill64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || args[1] > sigMax64 {
		return int64(EINVAL)
	}
	if args[0] != 0 && args[0] != ctx.PID {
		return int64(ESRCH)
	}
	if args[1] == 0 {
		return 0
	}
	queueSignal64(ctx, args[1])
	for handle := range ctx.signalFDs {
		handle.enqueue(uint32(args[1]), uint32(ctx.PID), 0)
	}
	return 0
}
