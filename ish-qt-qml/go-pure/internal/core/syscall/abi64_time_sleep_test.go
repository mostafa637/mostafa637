package syscall

import (
	"encoding/binary"
	"testing"
	"time"
)

func TestDispatcher64ClockNanosleep(t *testing.T) {
	ctx, dispatcher, state, area := newRuntime64TestContext(t)
	ctx.StartTime = time.Now()

	var request [16]byte
	binary.LittleEndian.PutUint64(request[8:], uint64(2*time.Millisecond))
	if err := ctx.Memory.Write(area+0x100, request[:]); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if got := dispatch64Runtime(t, dispatcher, state, Sys64ClockNanosleep, clockMonotonic64, 0, uint64(area+0x100), uint64(area+0x120)); got != 0 {
		t.Fatalf("relative clock_nanosleep = %d", got)
	}
	if elapsed := time.Since(started); elapsed < time.Millisecond || elapsed > 200*time.Millisecond {
		t.Fatalf("relative clock_nanosleep elapsed=%v", elapsed)
	}
	var remaining [16]byte
	if err := ctx.Memory.Read(area+0x120, remaining[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint64(remaining[:8]) != 0 || binary.LittleEndian.Uint64(remaining[8:]) != 0 {
		t.Fatalf("relative remaining timespec=%#x", remaining)
	}

	absolute := time.Since(ctx.StartTime) - time.Millisecond
	if absolute < 0 {
		absolute = 0
	}
	binary.LittleEndian.PutUint64(request[:8], uint64(absolute/time.Second))
	binary.LittleEndian.PutUint64(request[8:], uint64(absolute%time.Second))
	if err := ctx.Memory.Write(area+0x100, request[:]); err != nil {
		t.Fatal(err)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64ClockNanosleep, clockMonotonic64, timerAbstime64, uint64(area+0x100), 0); got != 0 {
		t.Fatalf("absolute clock_nanosleep = %d", got)
	}

	if got := dispatch64Runtime(t, dispatcher, state, Sys64ClockNanosleep, 99, 0, uint64(area+0x100), 0); got != int64(EINVAL) {
		t.Fatalf("invalid clock ID = %d", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64ClockNanosleep, clockMonotonic64, 2, uint64(area+0x100), 0); got != int64(EINVAL) {
		t.Fatalf("invalid clock_nanosleep flags = %d", got)
	}
}
