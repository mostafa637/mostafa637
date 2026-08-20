package syscall

import "testing"

func TestDispatcher64WasmHandler(t *testing.T) {
	ctx := NewContext64(nil)
	ctx.PID = 321
	dispatcher := NewDispatcher64(ctx)
	got := dispatcher.WasmHandler()(uint64(Sys64GetPID), [6]uint64{})
	if got != 321 {
		t.Fatalf("got pid %d", got)
	}
}
