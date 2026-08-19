package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcher64CapabilityABI(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xe000
	if err := memory.Map(area, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.PID = 55
	ctx.CapEffective = 0x0000001200000034
	ctx.CapPermitted = 0x0000005600000078
	ctx.CapInheritable = 0x0000009a000000bc
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	var header [capUserHeaderSize64]byte
	binary.LittleEndian.PutUint32(header[:4], capVersion64)
	binary.LittleEndian.PutUint32(header[4:], ^uint32(0))
	if err := memory.Write(area, header[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Capget, uint64(area), uint64(area+0x20))
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("capget: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	if err := memory.Read(area, header[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(header[:4]) != capVersion64 || binary.LittleEndian.Uint32(header[4:]) != 55 {
		t.Fatalf("capget header=%#x", header)
	}
	var data [capDataSize64]byte
	if err := memory.Read(area+0x20, data[:]); err != nil {
		t.Fatal(err)
	}
	if got := capabilityPair64(&data, 0); got != ctx.CapEffective {
		t.Fatalf("effective capabilities=%#x", got)
	}
	if got := capabilityPair64(&data, 4); got != ctx.CapPermitted {
		t.Fatalf("permitted capabilities=%#x", got)
	}
	if got := capabilityPair64(&data, 8); got != ctx.CapInheritable {
		t.Fatalf("inheritable capabilities=%#x", got)
	}

	binary.LittleEndian.PutUint32(data[0:4], 0x11111111)
	binary.LittleEndian.PutUint32(data[4:8], 0x22222222)
	binary.LittleEndian.PutUint32(data[8:12], 0x33333333)
	binary.LittleEndian.PutUint32(data[12:16], 0x44444444)
	binary.LittleEndian.PutUint32(data[16:20], 0x55555555)
	binary.LittleEndian.PutUint32(data[20:24], 0x66666666)
	if err := memory.Write(area+0x20, data[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Capset, uint64(area), uint64(area+0x20))
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("capset: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	if ctx.CapEffective != 0x4444444411111111 || ctx.CapPermitted != 0x5555555522222222 || ctx.CapInheritable != 0x6666666633333333 {
		t.Fatalf("capset state: effective=%#x permitted=%#x inheritable=%#x", ctx.CapEffective, ctx.CapPermitted, ctx.CapInheritable)
	}

	ctx.EffectiveUID = 1000
	set64Syscall(state, Sys64Capset, uint64(area), uint64(area+0x20))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EPERM) {
		t.Fatalf("non-root capset: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	ctx.EffectiveUID = 0
	binary.LittleEndian.PutUint32(header[:4], 0xdeadbeef)
	if err := memory.Write(area, header[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Capget, uint64(area), uint64(area+0x20))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("invalid cap version: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestDispatcher64MemoryLocking(t *testing.T) {
	memory := corecpu.NewMemory64()
	const area corecpu.Address64 = 0xf000
	if err := memory.Map(area, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	set64Syscall(state, Sys64Mlock, uint64(area)+corecpu.Page64Size-1, 2)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("mlock mapped crossing range: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	set64Syscall(state, Sys64Munlock, uint64(area)+corecpu.Page64Size-1, 2)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("munlock mapped crossing range: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	set64Syscall(state, Sys64Mlock, uint64(area)+2*corecpu.Page64Size, 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ENOMEM) {
		t.Fatalf("mlock unmapped range: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Mlock, ^uint64(0)-1, 4)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("mlock overflow: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Mlockall, mclCurrent64|mclOnFault64)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("mlockall valid flags: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
	set64Syscall(state, Sys64Mlockall, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EINVAL) {
		t.Fatalf("mlockall invalid flags: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Munlockall)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("munlockall: err=%v rax=%d", err, state.Get(corecpu.RAX))
	}
}
