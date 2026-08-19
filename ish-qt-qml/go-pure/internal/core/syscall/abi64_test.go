package syscall

import (
	"bytes"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcher64RegisterABI(t *testing.T) {
	memory := corecpu.NewMemory64()
	const buffer corecpu.Address64 = 0x6000
	if err := memory.Map(buffer, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context := NewContext64(memory)
	context.PID = 1234
	context.TID = 1235
	dispatcher := NewDispatcher64(context)
	state := corecpu.NewMachineState64(memory)

	state.Set(corecpu.RAX, uint64(Sys64GetPID))
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != 1234 {
		t.Fatalf("getpid: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}

	state.Set(corecpu.RAX, uint64(Sys64Getrandom))
	state.Set(corecpu.RDI, uint64(buffer))
	state.Set(corecpu.RSI, 32)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || state.Get(corecpu.RAX) != 32 {
		t.Fatalf("getrandom: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	var randomBytes [32]byte
	if err := memory.Read(buffer, randomBytes[:]); err != nil {
		t.Fatal(err)
	}

	state.Set(corecpu.RAX, uint64(Sys64Exit))
	state.Set(corecpu.RDI, 7)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || resume || !state.Halted || state.Get(corecpu.RAX) != 7 {
		t.Fatalf("exit: resume=%v err=%v halted=%v rax=%d", resume, err, state.Halted, state.Get(corecpu.RAX))
	}
}

func TestDispatcher64FDTableIO(t *testing.T) {
	memory := corecpu.NewMemory64()
	const buffer corecpu.Address64 = 0x7000
	if err := memory.Map(buffer, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	input := bytes.NewBufferString("ping")
	var output bytes.Buffer
	context := NewContext64(memory)
	if err := context.InstallFile(0, &File{Reader: input}); err != nil {
		t.Fatal(err)
	}
	if err := context.InstallFile(1, &File{Writer: &output}); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDispatcher64(context)
	state := corecpu.NewMachineState64(memory)

	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, 0)
	state.Set(corecpu.RSI, uint64(buffer))
	state.Set(corecpu.RDX, 4)
	resume, err := dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 4 {
		t.Fatalf("read: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	var got [4]byte
	if err := memory.Read(buffer, got[:]); err != nil {
		t.Fatal(err)
	}
	if string(got[:]) != "ping" {
		t.Fatalf("read data = %q", got[:])
	}

	state.Set(corecpu.RAX, uint64(Sys64Write))
	state.Set(corecpu.RDI, 1)
	state.Set(corecpu.RSI, uint64(buffer))
	state.Set(corecpu.RDX, 4)
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 4 || output.String() != "ping" {
		t.Fatalf("write: resume=%v err=%v rax=%d output=%q", resume, err, state.Get(corecpu.RAX), output.String())
	}

	state.Set(corecpu.RAX, uint64(Sys64Dup))
	state.Set(corecpu.RDI, 1)
	resume, err = dispatcher.Dispatch(state)
	dupFD := int64(state.Get(corecpu.RAX))
	if err != nil || !resume || dupFD < 3 {
		t.Fatalf("dup: resume=%v err=%v fd=%d", resume, err, dupFD)
	}
	state.Set(corecpu.RAX, uint64(Sys64Close))
	state.Set(corecpu.RDI, uint64(dupFD))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("close: resume=%v err=%v rax=%d", resume, err, state.Get(corecpu.RAX))
	}
	state.Set(corecpu.RAX, uint64(Sys64Read))
	state.Set(corecpu.RDI, uint64(dupFD))
	resume, err = dispatcher.Dispatch(state)
	if err != nil || !resume || int64(state.Get(corecpu.RAX)) != int64(EBADF) {
		t.Fatalf("closed fd: resume=%v err=%v rax=%d", resume, err, int64(state.Get(corecpu.RAX)))
	}
}
