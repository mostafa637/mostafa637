package syscall

import (
	"encoding/binary"
	"math"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const timerAbstime64 uint64 = 1

func validClockID64(clockID uint64) bool {
	switch clockID {
	case clockRealtime64, clockMonotonic64, clockProcess64, clockThread64:
		return true
	default:
		return false
	}
}

func clockDuration64(ctx *Context64, clockID uint64) (time.Duration, bool) {
	switch clockID {
	case clockRealtime64:
		now := time.Now()
		return time.Duration(now.Unix())*time.Second + time.Duration(now.Nanosecond()), true
	case clockMonotonic64, clockProcess64, clockThread64:
		if ctx == nil {
			return 0, false
		}
		if ctx.StartTime.IsZero() {
			ctx.StartTime = time.Now()
		}
		return time.Since(ctx.StartTime), true
	default:
		return 0, false
	}
}

func clockNanosleep64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[2] == 0 {
		return int64(EFAULT)
	}
	if args[1]&^timerAbstime64 != 0 || !validClockID64(args[0]) {
		return int64(EINVAL)
	}
	var raw [16]byte
	if err := ctx.Memory.Read(corecpu.Address64(args[2]), raw[:]); err != nil {
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
	target := time.Duration(seconds)*time.Second + time.Duration(nanos)
	if args[1]&timerAbstime64 != 0 {
		now, ok := clockDuration64(ctx, args[0])
		if !ok {
			return int64(EINVAL)
		}
		target -= now
	}
	if target > 0 {
		time.Sleep(target)
	}
	if args[3] != 0 && args[1]&timerAbstime64 == 0 && !writeTimespec64(ctx, corecpu.Address64(args[3]), 0) {
		return int64(EFAULT)
	}
	return 0
}
