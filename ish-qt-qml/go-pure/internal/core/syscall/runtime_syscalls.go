package syscall

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"math"
	"sync"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	futexWait          uint32 = 0
	futexWake          uint32 = 1
	futexWaitBitset    uint32 = 9
	futexWakeBitset    uint32 = 10
	futexPrivateFlag   uint32 = 128
	futexRealtimeFlag  uint32 = 256
	futexWakeBitsetAll uint32 = ^uint32(0)

	getrandomNonblock uint32 = 1
	getrandomRandom   uint32 = 2

	rseqUnregister uint32 = 1
	rseqSize       uint32 = 32
)

type futexWaiter struct {
	ready chan struct{}
}

type FutexRegistry struct {
	mu      sync.Mutex
	waiters map[uint32][]*futexWaiter
}

func NewFutexRegistry() *FutexRegistry {
	return &FutexRegistry{waiters: make(map[uint32][]*futexWaiter)}
}

func futex(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil {
		return EFAULT
	}
	if context.Futexes == nil {
		context.Futexes = NewFutexRegistry()
	}
	op := args[1] &^ (futexPrivateFlag | futexRealtimeFlag)
	switch op {
	case futexWait, futexWaitBitset:
		if op == futexWaitBitset && args[5] == 0 {
			return EINVAL
		}
		var current [4]byte
		if err := context.Memory.Read(corecpu.Address(args[0]), current[:]); err != nil {
			return EFAULT
		}
		if binary.LittleEndian.Uint32(current[:]) != args[2] {
			return EAGAIN
		}
		timeout, result := futexTimeout(context, args[3], args[1]&futexRealtimeFlag != 0)
		if result != 0 {
			return result
		}
		return context.Futexes.wait(context.Memory, corecpu.Address(args[0]), args[2], timeout)
	case futexWake, futexWakeBitset:
		if op == futexWakeBitset && args[5] == 0 {
			return EINVAL
		}
		return context.Futexes.wake(args[0], int(args[2]))
	default:
		return EINVAL
	}
}

func (r *FutexRegistry) wait(memory *corecpu.Memory, address corecpu.Address, expected uint32, timeout *time.Duration) int32 {
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
	r.waiters[uint32(address)] = append(r.waiters[uint32(address)], waiter)
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
		waiters := r.waiters[uint32(address)]
		for index, candidate := range waiters {
			if candidate == waiter {
				r.waiters[uint32(address)] = append(waiters[:index], waiters[index+1:]...)
				if len(r.waiters[uint32(address)]) == 0 {
					delete(r.waiters, uint32(address))
				}
				break
			}
		}
		r.mu.Unlock()
		return ETIMEDOUT
	}
}

func (r *FutexRegistry) wake(address uint32, count int) int32 {
	if r == nil || count <= 0 {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	waiters := r.waiters[address]
	if len(waiters) == 0 {
		return 0
	}
	if count > len(waiters) {
		count = len(waiters)
	}
	for _, waiter := range waiters[:count] {
		close(waiter.ready)
	}
	if count == len(waiters) {
		delete(r.waiters, address)
	} else {
		r.waiters[address] = append([]*futexWaiter(nil), waiters[count:]...)
	}
	return int32(count)
}

func futexTimeout(context *Context, address uint32, realtime bool) (*time.Duration, int32) {
	if address == 0 {
		return nil, 0
	}
	if context == nil || context.Memory == nil {
		return nil, EFAULT
	}
	var value [8]byte
	if err := context.Memory.Read(corecpu.Address(address), value[:]); err != nil {
		return nil, EFAULT
	}
	seconds := int64(int32(binary.LittleEndian.Uint32(value[0:4])))
	nanoseconds := binary.LittleEndian.Uint32(value[4:8])
	if seconds < 0 || nanoseconds >= 1_000_000_000 {
		return nil, EINVAL
	}
	if seconds > math.MaxInt64/int64(time.Second) {
		return nil, EINVAL
	}
	duration := time.Duration(seconds)*time.Second + time.Duration(nanoseconds)
	if realtime {
		// The host monotonic timer is used for deterministic guest waits. The
		// flag is accepted while the guest timeout remains a bounded duration.
	}
	return &duration, 0
}

func getrandom(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil {
		return EFAULT
	}
	if args[2]&^(getrandomNonblock|getrandomRandom) != 0 {
		return EINVAL
	}
	length, ok := safeLength(args[1])
	if !ok || uint64(args[1]) > uint64(maxGuestIOBytes) {
		return EINVAL
	}
	if length == 0 {
		return 0
	}
	buffer := make([]byte, length)
	if _, err := cryptorand.Read(buffer); err != nil {
		return EIO
	}
	if state == nil || context.Memory.Write(corecpu.Address(args[0]), buffer) != nil {
		return EFAULT
	}
	return int32(length)
}

func prlimit64(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil {
		return EFAULT
	}
	if args[0] != 0 && args[0] != context.PID {
		return ESRCH
	}
	if context.RLimits == nil {
		context.RLimits = defaultResourceLimits()
	}
	old, ok := context.RLimits[args[1]]
	if !ok {
		return EINVAL
	}
	if args[2] != 0 {
		var value [16]byte
		if context.Memory.Read(corecpu.Address(args[2]), value[:]) != nil {
			return EFAULT
		}
		limit := ResourceLimit{Cur: binary.LittleEndian.Uint64(value[0:8]), Max: binary.LittleEndian.Uint64(value[8:16])}
		if limit.Cur > limit.Max {
			return EINVAL
		}
		context.RLimits[args[1]] = limit
	}
	if args[3] != 0 {
		var value [16]byte
		binary.LittleEndian.PutUint64(value[0:8], old.Cur)
		binary.LittleEndian.PutUint64(value[8:16], old.Max)
		if state == nil || context.Memory.Write(corecpu.Address(args[3]), value[:]) != nil {
			return EFAULT
		}
	}
	return 0
}

func rseq(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil {
		return EFAULT
	}
	if args[2]&^rseqUnregister != 0 {
		return EINVAL
	}
	if args[2]&rseqUnregister != 0 {
		if args[0] != 0 || args[1] != 0 || args[3] != 0 {
			return EINVAL
		}
		context.RseqAddress, context.RseqLength, context.RseqSignature = 0, 0, 0
		return 0
	}
	if args[0] == 0 || args[1] < rseqSize {
		return EINVAL
	}
	if context.RseqAddress != 0 {
		return EBUSY
	}
	if state == nil {
		return EFAULT
	}
	var initial [rseqSize]byte
	if context.Memory.Write(corecpu.Address(args[0]), initial[:]) != nil {
		return EFAULT
	}
	context.RseqAddress = args[0]
	context.RseqLength = args[1]
	context.RseqSignature = args[3]
	return 0
}
