package syscall

import (
	"encoding/binary"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	sigSetSize64 = 8
	sigKill64    = 9
	sigStop64    = 19

	sigBlock64   = 0
	sigUnblock64 = 1
	sigSetMask64 = 2

	clockRealtime64 = 0
	clockProcess64  = 2
	clockThread64   = 3
)

func writeTimespecGuest64(memory *corecpu.Memory64, address uint64, seconds int64, nanos int64) bool {
	if memory == nil || address == 0 {
		return false
	}
	var raw [16]byte
	binary.LittleEndian.PutUint64(raw[0:8], uint64(seconds))
	binary.LittleEndian.PutUint64(raw[8:16], uint64(nanos))
	return memory.Write(corecpu.Address64(address), raw[:]) == nil
}

func gettimeofday64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if args[0] != 0 {
		now := time.Now()
		var value [16]byte
		binary.LittleEndian.PutUint64(value[0:8], uint64(now.Unix()))
		binary.LittleEndian.PutUint64(value[8:16], uint64(now.Nanosecond()/1000))
		if err := ctx.Memory.Write(corecpu.Address64(args[0]), value[:]); err != nil {
			return int64(EFAULT)
		}
	}
	if args[1] != 0 {
		var timezone [8]byte
		if err := ctx.Memory.Write(corecpu.Address64(args[1]), timezone[:]); err != nil {
			return int64(EFAULT)
		}
	}
	return 0
}

func clockGettime64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[1] == 0 {
		return int64(EFAULT)
	}
	var seconds int64
	var nanos int64
	switch args[0] {
	case clockRealtime64:
		now := time.Now()
		seconds, nanos = now.Unix(), int64(now.Nanosecond())
	case clockMonotonic64, clockProcess64, clockThread64:
		if ctx.StartTime.IsZero() {
			ctx.StartTime = time.Now()
		}
		elapsed := time.Since(ctx.StartTime)
		seconds, nanos = int64(elapsed/time.Second), int64(elapsed%time.Second)
	default:
		return int64(EINVAL)
	}
	if !writeTimespecGuest64(ctx.Memory, args[1], seconds, nanos) {
		return int64(EFAULT)
	}
	return 0
}

func clockGetres64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	switch args[0] {
	case clockRealtime64, clockMonotonic64, clockProcess64, clockThread64:
	default:
		return int64(EINVAL)
	}
	if args[1] != 0 && !writeTimespecGuest64(ctx.Memory, args[1], 0, 1) {
		return int64(EFAULT)
	}
	return 0
}

func getrusage64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[1] == 0 {
		return int64(EFAULT)
	}
	if args[0] != 0 && args[0] != 1 && args[0] != ^uint64(0) {
		return int64(EINVAL)
	}
	// struct rusage is 144 bytes on x86-64. The current guest does not expose
	// host accounting, so return a valid zeroed structure.
	if err := ctx.Memory.Write(corecpu.Address64(args[1]), make([]byte, 144)); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func times64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[0] == 0 {
		return int64(EFAULT)
	}
	// struct tms contains four clock_t values, each 64-bit in the x86-64 ABI.
	if err := ctx.Memory.Write(corecpu.Address64(args[0]), make([]byte, 32)); err != nil {
		return int64(EFAULT)
	}
	if ctx.StartTime.IsZero() {
		ctx.StartTime = time.Now()
	}
	return int64(time.Since(ctx.StartTime) / (time.Second / 100))
}

func rtSigaction64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[3] != sigSetSize64 {
		return int64(EINVAL)
	}
	signal := args[0]
	if signal == 0 || signal > sigMax64 || signal == sigKill64 || signal == sigStop64 {
		return int64(EINVAL)
	}
	if ctx.SignalActions == nil {
		ctx.SignalActions = make(map[uint64][32]byte)
	}
	if args[2] != 0 {
		old := ctx.SignalActions[signal]
		if err := ctx.Memory.Write(corecpu.Address64(args[2]), old[:]); err != nil {
			return int64(EFAULT)
		}
	}
	if args[1] != 0 {
		var action [32]byte
		if err := ctx.Memory.Read(corecpu.Address64(args[1]), action[:]); err != nil {
			return int64(EFAULT)
		}
		ctx.SignalActions[signal] = action
	}
	return 0
}

func rtSigprocmask64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[3] != sigSetSize64 {
		return int64(EINVAL)
	}
	if args[2] != 0 {
		var old [8]byte
		binary.LittleEndian.PutUint64(old[:], ctx.SignalMask)
		if err := ctx.Memory.Write(corecpu.Address64(args[2]), old[:]); err != nil {
			return int64(EFAULT)
		}
	}
	if args[1] == 0 {
		if args[0] != sigBlock64 && args[0] != sigUnblock64 && args[0] != sigSetMask64 {
			return int64(EINVAL)
		}
		return 0
	}
	var raw [8]byte
	if err := ctx.Memory.Read(corecpu.Address64(args[1]), raw[:]); err != nil {
		return int64(EFAULT)
	}
	set := binary.LittleEndian.Uint64(raw[:])
	set &^= (uint64(1) << (sigKill64 - 1)) | (uint64(1) << (sigStop64 - 1))
	switch args[0] {
	case sigBlock64:
		ctx.SignalMask |= set
	case sigUnblock64:
		ctx.SignalMask &^= set
	case sigSetMask64:
		ctx.SignalMask = set
	default:
		return int64(EINVAL)
	}
	return 0
}
