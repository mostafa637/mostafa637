package syscall

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcherReadWriteAndPID(t *testing.T) {
	memory := cpu.NewMemory()
	if err := memory.Map(2, 1, cpu.PRead|cpu.PWrite); err != nil {
		t.Fatal(err)
	}
	if err := memory.Map(3, 1, cpu.PRead|cpu.PWrite); err != nil {
		t.Fatal(err)
	}
	input := bytes.NewBufferString("hello")
	var output bytes.Buffer
	context := NewContext(memory)
	context.PID = 123
	context.Files[0] = &File{Reader: input, Writer: &output}
	dispatcher := NewDispatcher(context)
	state := cpu.NewMachineState(memory)

	if got := dispatcher.Dispatch(state, SysRead, 0, 2*cpu.PageSize, 5); got != 5 {
		t.Fatalf("read result = %d", got)
	}
	readBack := make([]byte, 5)
	if err := memory.Read(2*cpu.PageSize, readBack); err != nil {
		t.Fatal(err)
	}
	if string(readBack) != "hello" {
		t.Fatalf("guest read buffer = %q", readBack)
	}
	if got := dispatcher.Dispatch(state, SysWrite, 0, 2*cpu.PageSize, 5); got != 5 {
		t.Fatalf("write result = %d", got)
	}
	if output.String() != "hello" {
		t.Fatalf("host output = %q", output.String())
	}
	if got := dispatcher.Dispatch(state, SysGetPID); got != 123 {
		t.Fatalf("getpid result = %d", got)
	}
}

func TestDispatcherMemorySyscalls(t *testing.T) {
	memory := cpu.NewMemory()
	context := NewContext(memory)
	dispatcher := NewDispatcher(context)
	state := cpu.NewMachineState(memory)

	fixed := uint32(0x10000)
	got := dispatcher.Dispatch(state, SysMmap2, fixed, cpu.PageSize*2, ProtRead|ProtWrite, MapPrivate|MapAnonymous|MapFixed, ^uint32(0), 0)
	if uint32(got) != fixed {
		t.Fatalf("mmap2 result = %#x, want %#x", uint32(got), fixed)
	}
	if err := memory.Write(cpu.Address(fixed), []byte("mapped")); err != nil {
		t.Fatal(err)
	}
	if got := dispatcher.Dispatch(state, SysMprotect, fixed, cpu.PageSize, ProtRead); got != 0 {
		t.Fatalf("mprotect result = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysMunmap, fixed, cpu.PageSize*2); got != 0 {
		t.Fatalf("munmap result = %d", got)
	}
	if _, ok := memory.Page(cpu.Page(fixed >> cpu.PageBits)); ok {
		t.Fatal("mmap page remains after munmap")
	}
}

func TestDispatcherBrkLegacyMmapAndExit(t *testing.T) {
	memory := cpu.NewMemory()
	context := NewContext(memory)
	dispatcher := NewDispatcher(context)
	state := cpu.NewMachineState(memory)

	if got := dispatcher.Dispatch(state, SysBrk, 0x2000); uint32(got) != 0x2000 {
		t.Fatalf("initial brk = %#x", uint32(got))
	}
	if got := dispatcher.Dispatch(state, SysBrk, 0x5000); uint32(got) != 0x5000 {
		t.Fatalf("expanded brk = %#x", uint32(got))
	}
	if _, ok := memory.Page(1); !ok {
		t.Fatal("brk did not map heap page")
	}
	if got := dispatcher.Dispatch(state, SysBrk, 0x1000); uint32(got) != 0x1000 {
		t.Fatalf("brk shrink changed value to %#x", uint32(got))
	}

	if err := memory.Map(8, 1, cpu.PRead|cpu.PWrite); err != nil {
		t.Fatal(err)
	}
	args := make([]byte, 24)
	values := [6]uint32{0x30000, cpu.PageSize, ProtRead | ProtWrite, MapPrivate | MapAnonymous | MapFixed, ^uint32(0), 0}
	for i, value := range values {
		binary.LittleEndian.PutUint32(args[i*4:], value)
	}
	if err := memory.Write(8*cpu.PageSize, args); err != nil {
		t.Fatal(err)
	}
	if got := dispatcher.Dispatch(state, SysMmap, 8*cpu.PageSize); uint32(got) != values[0] {
		t.Fatalf("legacy mmap result = %#x", uint32(got))
	}
	if got := dispatcher.Dispatch(state, 9999); got != ENOSYS {
		t.Fatalf("unknown syscall result = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysExit, 7); got != 0 || !context.Exited || context.ExitCode != 7 {
		t.Fatalf("exit state: result=%d exited=%v code=%d", got, context.Exited, context.ExitCode)
	}
}
