package syscall

import (
	"encoding/binary"
	"testing"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestSysinfo64LayoutAndGuestValues(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0x10000
	if err := memory.Map(area, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.StartTime = time.Now().Add(-7 * time.Second)
	ctx.Mappings = []GuestMapping64{
		{Base: 0x400000, Length: 3 * corecpu.Page64Size, Shared: true},
		{Base: 0x500000, Length: 2 * corecpu.Page64Size},
	}
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)
	set64Syscall(state, Sys64Sysinfo, uint64(area))
	if resume, err := dispatcher.Dispatch(state); err != nil || !resume || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("sysinfo: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}

	raw := make([]byte, sysinfoSize64)
	if err := memory.Read(area, raw); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(raw[sysinfoUptimeOffset:]); got < 6 {
		t.Fatalf("uptime=%d, want at least 6 seconds", got)
	}
	for i := 0; i < 3; i++ {
		got := binary.LittleEndian.Uint64(raw[sysinfoLoadsOffset+i*8:])
		if got != uint64(1)<<sysinfoLoadShift64 {
			t.Fatalf("load[%d]=%d, want %d", i, got, uint64(1)<<sysinfoLoadShift64)
		}
	}
	if got := binary.LittleEndian.Uint64(raw[sysinfoTotalRAM:]); got != guestRAMBytes64 {
		t.Fatalf("totalram=%d, want %d", got, guestRAMBytes64)
	}
	wantFree := guestRAMBytes64 - 5*corecpu.Page64Size
	if got := binary.LittleEndian.Uint64(raw[sysinfoFreeRAM:]); got != wantFree {
		t.Fatalf("freeram=%d, want %d", got, wantFree)
	}
	if got := binary.LittleEndian.Uint64(raw[sysinfoSharedRAM:]); got != 3*corecpu.Page64Size {
		t.Fatalf("sharedram=%d, want %d", got, 3*corecpu.Page64Size)
	}
	if got := binary.LittleEndian.Uint16(raw[sysinfoProcs:]); got != 1 {
		t.Fatalf("procs=%d, want 1", got)
	}
	if got := binary.LittleEndian.Uint32(raw[sysinfoMemUnit:]); got != 1 {
		t.Fatalf("mem_unit=%d, want 1", got)
	}
	if len(raw) != 112 {
		t.Fatalf("sysinfo size=%d, want 112", len(raw))
	}
}

func TestSysinfo64MemorySaturationAndFault(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0x12000
	if err := memory.Map(area, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.Mappings = []GuestMapping64{{Base: 0x400000, Length: guestRAMBytes64 + corecpu.Page64Size, Shared: true}}
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	set64Syscall(state, Sys64Sysinfo, uint64(area))
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("saturated sysinfo: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	var raw [sysinfoSize64]byte
	if err := memory.Read(area, raw[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(raw[sysinfoFreeRAM:]); got != 0 {
		t.Fatalf("saturated freeram=%d, want 0", got)
	}
	if got := binary.LittleEndian.Uint64(raw[sysinfoSharedRAM:]); got != guestRAMBytes64 {
		t.Fatalf("saturated sharedram=%d, want %d", got, guestRAMBytes64)
	}

	set64Syscall(state, Sys64Sysinfo, 0x7f0000000000)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("invalid sysinfo pointer: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
