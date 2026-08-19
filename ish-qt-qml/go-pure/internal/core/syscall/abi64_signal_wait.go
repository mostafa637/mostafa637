package syscall

import (
	"encoding/binary"
	"math/bits"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func signalBit64(signo uint64) uint64 {
	if signo == 0 || signo > sigMax64 {
		return 0
	}
	return uint64(1) << (signo - 1)
}

func ensureSignalWake64Locked(ctx *Context64) chan struct{} {
	if ctx.SignalWake == nil {
		ctx.SignalWake = make(chan struct{})
	}
	return ctx.SignalWake
}

// queueSignal64 records a guest signal and wakes all signal waiters. The
// guest signal queue is intentionally separate from host process signals so
// the emulator remains deterministic and CGo-free.
func queueSignal64(ctx *Context64, signo uint64) {
	if ctx == nil {
		return
	}
	bit := signalBit64(signo)
	if bit == 0 {
		return
	}
	ctx.SignalMu.Lock()
	ctx.PendingSignals |= bit
	wake := ensureSignalWake64Locked(ctx)
	ctx.SignalWake = make(chan struct{})
	close(wake)
	if ctx.SignalCond != nil {
		ctx.SignalCond.Broadcast()
	}
	ctx.SignalMu.Unlock()
}

func pendingSignal64Locked(ctx *Context64, set uint64) uint64 {
	pending := ctx.PendingSignals & set
	if pending == 0 {
		return 0
	}
	bit := pending & -pending
	ctx.PendingSignals &^= bit
	return uint64(bits.TrailingZeros64(bit)) + 1
}

func waitForSignal64(ctx *Context64, set uint64, timeout time.Duration, timed bool) (uint64, int64) {
	if ctx == nil {
		return 0, int64(EINVAL)
	}
	var deadline time.Time
	if timed {
		deadline = time.Now().Add(timeout)
	}
	for {
		ctx.SignalMu.Lock()
		if signo := pendingSignal64Locked(ctx, set); signo != 0 {
			ctx.SignalMu.Unlock()
			return signo, 0
		}
		wake := ensureSignalWake64Locked(ctx)
		ctx.SignalMu.Unlock()

		if !timed {
			<-wake
			continue
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, int64(EAGAIN)
		}
		timer := time.NewTimer(remaining)
		select {
		case <-wake:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
			return 0, int64(EAGAIN)
		}
	}
}

func readSignalSet64(ctx *Context64, address uint64, size uint64) (uint64, int64) {
	if ctx == nil || ctx.Memory == nil || size != sigSetSize64 || address == 0 {
		return 0, int64(EINVAL)
	}
	var raw [sigSetSize64]byte
	if err := ctx.Memory.Read(corecpu.Address64(address), raw[:]); err != nil {
		return 0, int64(EFAULT)
	}
	return binary.LittleEndian.Uint64(raw[:]), 0
}

func writeSignalSet64(ctx *Context64, address uint64, set uint64) int64 {
	if ctx == nil || ctx.Memory == nil || address == 0 {
		return int64(EFAULT)
	}
	var raw [sigSetSize64]byte
	binary.LittleEndian.PutUint64(raw[:], set)
	if err := ctx.Memory.Write(corecpu.Address64(address), raw[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func rtSigpending64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || args[1] != sigSetSize64 {
		return int64(EINVAL)
	}
	if args[0] == 0 || ctx.Memory == nil {
		return int64(EFAULT)
	}
	ctx.SignalMu.Lock()
	pending := ctx.PendingSignals
	ctx.SignalMu.Unlock()
	return writeSignalSet64(ctx, args[0], pending)
}

func readSignalTimeout64(ctx *Context64, address uint64) (time.Duration, int64) {
	if address == 0 {
		return 0, 0
	}
	if ctx == nil || ctx.Memory == nil {
		return 0, int64(EFAULT)
	}
	var raw [16]byte
	if err := ctx.Memory.Read(corecpu.Address64(address), raw[:]); err != nil {
		return 0, int64(EFAULT)
	}
	seconds := int64(binary.LittleEndian.Uint64(raw[:8]))
	nanos := int64(binary.LittleEndian.Uint64(raw[8:]))
	if seconds < 0 || nanos < 0 || nanos >= int64(time.Second) {
		return 0, int64(EINVAL)
	}
	if seconds > int64((time.Duration(1<<63-1))/time.Second) {
		return 0, int64(EINVAL)
	}
	return time.Duration(seconds)*time.Second + time.Duration(nanos), 0
}

func writeSignalInfo64(ctx *Context64, address uint64, signo uint64) int64 {
	if address == 0 {
		return 0
	}
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	var info [128]byte
	binary.LittleEndian.PutUint32(info[0:4], uint32(signo))
	binary.LittleEndian.PutUint32(info[12:16], uint32(ctx.PID))
	if err := ctx.Memory.Write(corecpu.Address64(address), info[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func rtSigtimedwait64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || args[3] != sigSetSize64 {
		return int64(EINVAL)
	}
	set, result := readSignalSet64(ctx, args[0], args[3])
	if result != 0 {
		return result
	}
	if set == 0 {
		return int64(EINVAL)
	}
	timeout, result := readSignalTimeout64(ctx, args[2])
	if result != 0 {
		return result
	}
	signo, result := waitForSignal64(ctx, set, timeout, args[2] != 0)
	if result != 0 {
		return result
	}
	if result = writeSignalInfo64(ctx, args[1], signo); result != 0 {
		return result
	}
	return int64(signo)
}

func rtSigsuspend64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || args[1] != sigSetSize64 {
		return int64(EINVAL)
	}
	mask, result := readSignalSet64(ctx, args[0], args[1])
	if result != 0 {
		return result
	}
	mask &^= signalBit64(sigKill64) | signalBit64(sigStop64)
	ctx.SignalMu.Lock()
	oldMask := ctx.SignalMask
	ctx.SignalMask = mask
	ctx.SignalMu.Unlock()
	_, _ = waitForSignal64(ctx, ^mask, 0, false)
	ctx.SignalMu.Lock()
	ctx.SignalMask = oldMask
	ctx.SignalMu.Unlock()
	return int64(EINTR)
}

func pause64(ctx *Context64, _ [6]uint64) int64 {
	if ctx == nil {
		return int64(EINVAL)
	}
	ctx.SignalMu.Lock()
	mask := ctx.SignalMask
	ctx.SignalMu.Unlock()
	_, _ = waitForSignal64(ctx, ^mask, 0, false)
	return int64(EINTR)
}
