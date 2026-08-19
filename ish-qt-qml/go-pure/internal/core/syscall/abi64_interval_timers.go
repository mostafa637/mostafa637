package syscall

import (
	"encoding/binary"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	itimerReal64    = 0
	itimerVirtual64 = 1
	itimerProf64    = 2

	sigAlarm64   = 14
	sigVTAlarm64 = 26
	sigProf64    = 27

	maxTimerSeconds64 = uint64(^uint32(0))
)

// IntervalTimer64 models the guest interval timer independently of host
// process timers. Value is the next expiration deadline when Active is true.
type IntervalTimer64 struct {
	Value    time.Time
	Interval time.Duration
	Active   bool
}

func intervalTimerIndex64(which uint64) (int, uint64, bool) {
	switch which {
	case itimerReal64:
		return 0, sigAlarm64, true
	case itimerVirtual64:
		return 1, sigVTAlarm64, true
	case itimerProf64:
		return 2, sigProf64, true
	default:
		return 0, 0, false
	}
}

func readTimeval64(ctx *Context64, address uint64) (time.Duration, int64) {
	if ctx == nil || ctx.Memory == nil || address == 0 {
		return 0, int64(EFAULT)
	}
	var raw [16]byte
	if err := ctx.Memory.Read(corecpu.Address64(address), raw[:]); err != nil {
		return 0, int64(EFAULT)
	}
	seconds := int64(binary.LittleEndian.Uint64(raw[0:8]))
	microseconds := int64(binary.LittleEndian.Uint64(raw[8:16]))
	if seconds < 0 || microseconds < 0 || microseconds >= 1_000_000 {
		return 0, int64(EINVAL)
	}
	const maxDurationSeconds = int64(^uint64(0)>>1) / int64(time.Second)
	if seconds > maxDurationSeconds {
		return 0, int64(EINVAL)
	}
	duration := time.Duration(seconds)*time.Second + time.Duration(microseconds)*time.Microsecond
	if duration < 0 {
		return 0, int64(EINVAL)
	}
	return duration, 0
}

func writeTimeval64(ctx *Context64, address uint64, duration time.Duration) int64 {
	if ctx == nil || ctx.Memory == nil || address == 0 || duration < 0 {
		return int64(EFAULT)
	}
	seconds := duration / time.Second
	microseconds := (duration % time.Second) / time.Microsecond
	var raw [16]byte
	binary.LittleEndian.PutUint64(raw[0:8], uint64(seconds))
	binary.LittleEndian.PutUint64(raw[8:16], uint64(microseconds))
	if err := ctx.Memory.Write(corecpu.Address64(address), raw[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func refreshIntervalTimer64(ctx *Context64, index int, signal uint64) IntervalTimer64 {
	if ctx == nil || index < 0 || index >= len(ctx.Timers) {
		return IntervalTimer64{}
	}
	now := time.Now()
	ctx.TimerMu.Lock()
	timer := ctx.Timers[index]
	fire := timer.Active && !now.Before(timer.Value)
	if fire {
		if timer.Interval > 0 {
			// A delayed guest dispatch coalesces missed periodic expirations into
			// one pending bit and schedules the next deadline after now.
			timer.Value = now.Add(timer.Interval)
		} else {
			timer.Active = false
			timer.Value = time.Time{}
		}
		ctx.Timers[index] = timer
	}
	ctx.TimerMu.Unlock()
	if fire {
		queueSignal64(ctx, signal)
	}
	return timer
}

func intervalTimerRemaining64(timer IntervalTimer64, now time.Time) time.Duration {
	if !timer.Active || timer.Value.IsZero() || !timer.Value.After(now) {
		return 0
	}
	return timer.Value.Sub(now)
}

func getitimer64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[1] == 0 {
		return int64(EFAULT)
	}
	index, signal, ok := intervalTimerIndex64(args[0])
	if !ok {
		return int64(EINVAL)
	}
	timer := refreshIntervalTimer64(ctx, index, signal)
	return writeItimerValue64(ctx, args[1], timer)
}

func writeItimerValue64(ctx *Context64, address uint64, timer IntervalTimer64) int64 {
	var value time.Duration
	if timer.Active {
		value = intervalTimerRemaining64(timer, time.Now())
	}
	if result := writeTimeval64(ctx, address, value); result != 0 {
		return result
	}
	return writeTimeval64(ctx, address+16, timer.Interval)
}

func setitimer64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[2] == 0 {
		return int64(EFAULT)
	}
	index, signal, ok := intervalTimerIndex64(args[0])
	if !ok {
		return int64(EINVAL)
	}
	value, result := readTimeval64(ctx, args[2])
	if result != 0 {
		return result
	}
	interval, result := readTimeval64(ctx, args[2]+16)
	if result != 0 {
		return result
	}
	old := refreshIntervalTimer64(ctx, index, signal)
	if args[1] != 0 {
		if result := writeItimerValue64(ctx, args[1], old); result != 0 {
			return result
		}
	}
	ctx.TimerMu.Lock()
	if value == 0 {
		ctx.Timers[index] = IntervalTimer64{}
	} else {
		ctx.Timers[index] = IntervalTimer64{Value: time.Now().Add(value), Interval: interval, Active: true}
	}
	ctx.TimerMu.Unlock()
	return 0
}

func alarm64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(EFAULT)
	}
	if args[0] > maxTimerSeconds64 {
		return int64(EINVAL)
	}
	index, signal, _ := intervalTimerIndex64(itimerReal64)
	old := refreshIntervalTimer64(ctx, index, signal)
	remaining := intervalTimerRemaining64(old, time.Now())
	var previous uint64
	if remaining > 0 {
		previous = uint64((remaining + time.Second - 1) / time.Second)
		if previous > maxTimerSeconds64 {
			previous = maxTimerSeconds64
		}
	}
	ctx.TimerMu.Lock()
	if args[0] == 0 {
		ctx.Timers[index] = IntervalTimer64{}
	} else {
		ctx.Timers[index] = IntervalTimer64{Value: time.Now().Add(time.Duration(args[0]) * time.Second), Active: true}
	}
	ctx.TimerMu.Unlock()
	return int64(previous)
}
