package syscall

import (
	"encoding/binary"
	"testing"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func writeTimevalTest64(t *testing.T, ctx *Context64, address uint64, seconds, microseconds int64) {
	t.Helper()
	var raw [16]byte
	binary.LittleEndian.PutUint64(raw[0:8], uint64(seconds))
	binary.LittleEndian.PutUint64(raw[8:16], uint64(microseconds))
	if err := ctx.Memory.Write(corecpu.Address64(address), raw[:]); err != nil {
		t.Fatal(err)
	}
}

func readTimevalTest64(t *testing.T, ctx *Context64, address uint64) (uint64, uint64) {
	t.Helper()
	var raw [16]byte
	if err := ctx.Memory.Read(corecpu.Address64(address), raw[:]); err != nil {
		t.Fatal(err)
	}
	return binary.LittleEndian.Uint64(raw[0:8]), binary.LittleEndian.Uint64(raw[8:16])
}

func TestDispatcher64IntervalTimerValidation(t *testing.T) {
	ctx, dispatcher, state, area := newRuntime64TestContext(t)
	value := uint64(area) + 0x100
	old := uint64(area) + 0x140
	writeTimevalTest64(t, ctx, value, 0, 0)
	writeTimevalTest64(t, ctx, value+16, 0, 0)

	if got := dispatch64Runtime(t, dispatcher, state, Sys64Getitimer, 3, old); got != int64(EINVAL) {
		t.Fatalf("getitimer invalid which = %d", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Getitimer, itimerReal64, 0); got != int64(EFAULT) {
		t.Fatalf("getitimer null value = %d", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Setitimer, itimerReal64, old, 0); got != int64(EFAULT) {
		t.Fatalf("setitimer null new value = %d", got)
	}

	writeTimevalTest64(t, ctx, value+16, 0, 1_000_000)
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Setitimer, itimerReal64, old, value); got != int64(EINVAL) {
		t.Fatalf("setitimer invalid usec = %d", got)
	}
}

func TestDispatcher64IntervalTimerOneShotAndSignal(t *testing.T) {
	ctx, dispatcher, state, area := newRuntime64TestContext(t)
	value := uint64(area) + 0x100
	old := uint64(area) + 0x140
	out := uint64(area) + 0x180
	writeTimevalTest64(t, ctx, value, 0, 20_000)
	writeTimevalTest64(t, ctx, value+16, 0, 0)
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Setitimer, itimerReal64, old, value); got != 0 {
		t.Fatalf("setitimer one-shot = %d", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Getitimer, itimerReal64, out); got != 0 {
		t.Fatalf("getitimer active = %d", got)
	}
	seconds, microseconds := readTimevalTest64(t, ctx, out)
	if seconds != 0 || microseconds == 0 || microseconds > 20_000 {
		t.Fatalf("initial remaining timeval = %d.%06d", seconds, microseconds)
	}

	time.Sleep(35 * time.Millisecond)
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Getitimer, itimerReal64, out); got != 0 {
		t.Fatalf("getitimer expired = %d", got)
	}
	seconds, microseconds = readTimevalTest64(t, ctx, out)
	if seconds != 0 || microseconds != 0 {
		t.Fatalf("expired remaining timeval = %d.%06d", seconds, microseconds)
	}
	if ctx.PendingSignals&signalBit64(sigAlarm64) == 0 {
		t.Fatalf("SIGALRM was not queued: %#x", ctx.PendingSignals)
	}
}

func TestDispatcher64IntervalTimerPeriodicAndAlarm(t *testing.T) {
	ctx, dispatcher, state, area := newRuntime64TestContext(t)
	value := uint64(area) + 0x100
	old := uint64(area) + 0x140
	out := uint64(area) + 0x180
	writeTimevalTest64(t, ctx, value, 0, 10_000)
	writeTimevalTest64(t, ctx, value+16, 0, 10_000)
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Setitimer, itimerReal64, old, value); got != 0 {
		t.Fatalf("setitimer periodic = %d", got)
	}
	time.Sleep(25 * time.Millisecond)
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Getitimer, itimerReal64, out); got != 0 {
		t.Fatalf("getitimer periodic = %d", got)
	}
	seconds, microseconds := readTimevalTest64(t, ctx, out)
	if seconds != 0 || microseconds == 0 || microseconds > 10_000 {
		t.Fatalf("periodic remaining timeval = %d.%06d", seconds, microseconds)
	}
	intervalSeconds, intervalMicroseconds := readTimevalTest64(t, ctx, out+16)
	if intervalSeconds != 0 || intervalMicroseconds != 10_000 {
		t.Fatalf("periodic interval timeval = %d.%06d", intervalSeconds, intervalMicroseconds)
	}
	if ctx.PendingSignals&signalBit64(sigAlarm64) == 0 {
		t.Fatalf("periodic SIGALRM was not queued: %#x", ctx.PendingSignals)
	}

	if got := dispatch64Runtime(t, dispatcher, state, Sys64Alarm, 1); got != 1 {
		t.Fatalf("alarm previous seconds = %d, want 1", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Alarm, 0); got < 1 {
		t.Fatalf("alarm cancellation previous seconds = %d", got)
	}
}
