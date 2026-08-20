package syscall

import (
	"encoding/binary"
	"math"
	"sync"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	futexPrivateFlag64  uint32 = 128
	futexRealtimeFlag64 uint32 = 256
)

type FutexRegistry64 struct {
	mu      sync.Mutex
	waiters map[uint64][]*futexWaiter
}

func NewFutexRegistry64() *FutexRegistry64 {
	return &FutexRegistry64{waiters: make(map[uint64][]*futexWaiter)}
}

func gettid64(ctx *Context64, _ [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	tid := ctx.TID
	if tid == 0 {
		tid = ctx.PID
	}
	return int64(tid)
}

func setTIDAddress64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	ctx.TIDAddress = args[0]
	return gettid64(ctx, args)
}

func nanosleep64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[0] == 0 {
		return int64(EFAULT)
	}
	var raw [16]byte
	if err := ctx.Memory.Read(corecpu.Address64(args[0]), raw[:]); err != nil {
		return int64(EFAULT)
	}
	seconds := int64(binary.LittleEndian.Uint64(raw[:8]))
	nanos := int64(binary.LittleEndian.Uint64(raw[8:]))
	if seconds < 0 || nanos < 0 || nanos >= int64(time.Second) {
		return int64(EINVAL)
	}
	maxSeconds := int64(math.MaxInt64 / int64(time.Second))
	if seconds > maxSeconds {
		return int64(EINVAL)
	}
	duration := time.Duration(seconds)*time.Second + time.Duration(nanos)
	time.Sleep(duration)
	if args[1] != 0 && !writeTimespec64(ctx, corecpu.Address64(args[1]), 0) {
		return int64(EFAULT)
	}
	return 0
}

func futex64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if ctx.Futexes == nil {
		ctx.Futexes = NewFutexRegistry64()
	}
	op := uint32(args[1]) &^ (futexPrivateFlag64 | futexRealtimeFlag64)
	switch op {
	case futexWait, futexWaitBitset:
		if op == futexWaitBitset && args[5] == 0 {
			return int64(EINVAL)
		}
		var current [4]byte
		if err := ctx.Memory.Read(corecpu.Address64(args[0]), current[:]); err != nil {
			return int64(EFAULT)
		}
		if binary.LittleEndian.Uint32(current[:]) != uint32(args[2]) {
			return int64(EAGAIN)
		}
		timeout, result := futexTimeout64(ctx, corecpu.Address64(args[3]))
		if result != 0 {
			return result
		}
		return int64(ctx.Futexes.wait(ctx.Memory, corecpu.Address64(args[0]), uint32(args[2]), timeout))
	case futexWake, futexWakeBitset:
		if op == futexWakeBitset && args[5] == 0 {
			return int64(EINVAL)
		}
		return int64(ctx.Futexes.wake(args[0], args[2]))
	default:
		return int64(EINVAL)
	}
}

func futexTimeout64(ctx *Context64, address corecpu.Address64) (*time.Duration, int64) {
	if address == 0 {
		return nil, 0
	}
	if ctx == nil || ctx.Memory == nil {
		return nil, int64(EFAULT)
	}
	duration, ok := readTimespec64(ctx, address)
	if !ok {
		return nil, int64(EINVAL)
	}
	return &duration, 0
}

func (r *FutexRegistry64) wait(memory *corecpu.Memory64, address corecpu.Address64, expected uint32, timeout *time.Duration) int32 {
	if r == nil || memory == nil {
		return EFAULT
	}
	r.mu.Lock()
	var current [4]byte
	if err := memory.Read(address, current[:]); err != nil {
		r.mu.Unlock()
		return EFAULT
	}
	if binary.LittleEndian.Uint32(current[:]) != expected {
		r.mu.Unlock()
		return EAGAIN
	}
	waiter := &futexWaiter{ready: make(chan struct{})}
	r.waiters[uint64(address)] = append(r.waiters[uint64(address)], waiter)
	r.mu.Unlock()

	if timeout == nil {
		<-waiter.ready
		return 0
	}
	timer := time.NewTimer(*timeout)
	defer timer.Stop()
	select {
	case <-waiter.ready:
		return 0
	case <-timer.C:
		r.mu.Lock()
		waiters := r.waiters[uint64(address)]
		for index, candidate := range waiters {
			if candidate == waiter {
				r.waiters[uint64(address)] = append(waiters[:index], waiters[index+1:]...)
				if len(r.waiters[uint64(address)]) == 0 {
					delete(r.waiters, uint64(address))
				}
				break
			}
		}
		r.mu.Unlock()
		return ETIMEDOUT
	}
}

// Wake releases up to count waiters registered on address. It is used by
// lifecycle code for CLONE_CHILD_CLEARTID notification.
func (r *FutexRegistry64) Wake(address uint64, count uint64) int32 {
	return r.wake(address, count)
}

func (r *FutexRegistry64) wake(address uint64, count uint64) int32 {
	if r == nil || count == 0 {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	waiters := r.waiters[address]
	if len(waiters) == 0 {
		return 0
	}
	if count > uint64(len(waiters)) {
		count = uint64(len(waiters))
	}
	for _, waiter := range waiters[:int(count)] {
		close(waiter.ready)
	}
	if int(count) == len(waiters) {
		delete(r.waiters, address)
	} else {
		r.waiters[address] = append([]*futexWaiter(nil), waiters[int(count):]...)
	}
	return int32(count)
}

func rseq64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	flags := uint32(args[2])
	if flags&^rseqUnregister != 0 {
		return int64(EINVAL)
	}
	if flags&rseqUnregister != 0 {
		if args[0] != 0 || args[1] != 0 || args[3] != 0 {
			return int64(EINVAL)
		}
		ctx.RseqAddress, ctx.RseqLength, ctx.RseqSignature = 0, 0, 0
		return 0
	}
	if args[0] == 0 || args[1] < uint64(rseqSize) {
		return int64(EINVAL)
	}
	if ctx.RseqAddress != 0 {
		return int64(EBUSY)
	}
	var initial [rseqSize]byte
	if err := ctx.Memory.Write(corecpu.Address64(args[0]), initial[:]); err != nil {
		return int64(EFAULT)
	}
	ctx.RseqAddress = args[0]
	ctx.RseqLength = args[1]
	ctx.RseqSignature = args[3]
	return 0
}
