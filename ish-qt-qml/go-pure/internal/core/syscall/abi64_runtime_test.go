package syscall

import (
	"encoding/binary"
	"testing"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func newRuntime64TestContext(t *testing.T) (*Context64, *Dispatcher64, *corecpu.MachineState64, corecpu.Address64) {
	t.Helper()
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xb000
	if err := memory.Map(area, 3*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.PID = 41
	ctx.TID = 42
	return ctx, NewDispatcher64(ctx), corecpu.NewMachineState64(memory), area
}

func dispatch64Runtime(t *testing.T, dispatcher *Dispatcher64, state *corecpu.MachineState64, number Number64, args ...uint64) int64 {
	t.Helper()
	state.Set(corecpu.RAX, uint64(number))
	values := [6]corecpu.Reg64{corecpu.RDI, corecpu.RSI, corecpu.RDX, corecpu.R10, corecpu.R8, corecpu.R9}
	for index := range values {
		value := uint64(0)
		if index < len(args) {
			value = args[index]
		}
		state.Set(values[index], value)
	}
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume {
		t.Fatalf("dispatch %d: resume=%v err=%v", number, resume, err)
	}
	return int64(state.Get(corecpu.RAX))
}

func TestDispatcher64RuntimeIdentityAndSleep(t *testing.T) {
	ctx, dispatcher, state, area := newRuntime64TestContext(t)
	if got := dispatch64Runtime(t, dispatcher, state, Sys64GetTID); got != 42 {
		t.Fatalf("gettid = %d", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64SetTIDAddr, uint64(area)); got != 42 || ctx.TIDAddress != uint64(area) {
		t.Fatalf("set_tid_address = %d, address=%#x", got, ctx.TIDAddress)
	}

	var request [16]byte
	binary.LittleEndian.PutUint64(request[0:8], 0)
	binary.LittleEndian.PutUint64(request[8:16], 0)
	if err := ctx.Memory.Write(area+0x100, request[:]); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Nanosleep, uint64(area+0x100), uint64(area+0x120)); got != 0 {
		t.Fatalf("nanosleep = %d", got)
	}
	if elapsed := time.Since(started); elapsed > 200*time.Millisecond {
		t.Fatalf("zero nanosleep took %v", elapsed)
	}
	var remaining [16]byte
	if err := ctx.Memory.Read(area+0x120, remaining[:]); err != nil {
		t.Fatal(err)
	}
	if seconds := binary.LittleEndian.Uint64(remaining[:8]); seconds != 0 || binary.LittleEndian.Uint64(remaining[8:]) != 0 {
		t.Fatalf("remaining timespec = %#x", remaining)
	}

	binary.LittleEndian.PutUint64(request[8:16], uint64(time.Second))
	if err := ctx.Memory.Write(area+0x100, request[:]); err != nil {
		t.Fatal(err)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Nanosleep, uint64(area+0x100), 0); got != int64(EINVAL) {
		t.Fatalf("invalid nanosleep = %d", got)
	}
}

func TestDispatcher64FutexWaitWakeAndTimeout(t *testing.T) {
	ctx, dispatcher, state, area := newRuntime64TestContext(t)
	futexAddress := area + 0x200
	var value [4]byte
	binary.LittleEndian.PutUint32(value[:], 7)
	if err := ctx.Memory.Write(futexAddress, value[:]); err != nil {
		t.Fatal(err)
	}

	waitState := corecpu.NewMachineState64(ctx.Memory)
	waitResult := make(chan int64, 1)
	go func() {
		waitResult <- dispatch64Runtime(t, dispatcher, waitState, Sys64Futex, uint64(futexAddress), uint64(futexWait), 7, 0, 0, 0)
	}()
	select {
	case result := <-waitResult:
		t.Fatalf("futex wait returned before wake: %d", result)
	case <-time.After(20 * time.Millisecond):
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Futex, uint64(futexAddress), uint64(futexWake), 1, 0, 0, 0); got != 1 {
		t.Fatalf("futex wake = %d", got)
	}
	select {
	case got := <-waitResult:
		if got != 0 {
			t.Fatalf("futex wait = %d", got)
		}
	case <-time.After(time.Second):
		t.Fatal("futex waiter did not wake")
	}

	if got := dispatch64Runtime(t, dispatcher, state, Sys64Futex, uint64(futexAddress), uint64(futexWait), 6, 0, 0, 0); got != int64(EAGAIN) {
		t.Fatalf("futex mismatch = %d", got)
	}
	var timeout [16]byte
	binary.LittleEndian.PutUint64(timeout[0:8], 0)
	binary.LittleEndian.PutUint64(timeout[8:16], uint64(2*time.Millisecond))
	if err := ctx.Memory.Write(area+0x300, timeout[:]); err != nil {
		t.Fatal(err)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Futex, uint64(futexAddress), uint64(futexWait), 7, uint64(area+0x300), 0, 0); got != int64(ETIMEDOUT) {
		t.Fatalf("futex timeout = %d", got)
	}
}

func TestDispatcher64Rseq(t *testing.T) {
	ctx, dispatcher, state, area := newRuntime64TestContext(t)
	rseqAddress := area + 0x400
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Rseq, uint64(rseqAddress), uint64(rseqSize), 0, 0); got != 0 {
		t.Fatalf("rseq register = %d", got)
	}
	if ctx.RseqAddress != uint64(rseqAddress) || ctx.RseqLength != uint64(rseqSize) {
		t.Fatalf("rseq state = address %#x length %d", ctx.RseqAddress, ctx.RseqLength)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Rseq, uint64(rseqAddress), uint64(rseqSize), 0, 0); got != int64(EBUSY) {
		t.Fatalf("second rseq register = %d", got)
	}
	if got := dispatch64Runtime(t, dispatcher, state, Sys64Rseq, 0, 0, uint64(rseqUnregister), 0); got != 0 {
		t.Fatalf("rseq unregister = %d", got)
	}
	if ctx.RseqAddress != 0 || ctx.RseqLength != 0 || ctx.RseqSignature != 0 {
		t.Fatalf("rseq state after unregister = %#x/%d/%d", ctx.RseqAddress, ctx.RseqLength, ctx.RseqSignature)
	}
}
