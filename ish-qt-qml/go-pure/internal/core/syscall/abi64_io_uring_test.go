package syscall

import (
	"bytes"
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func mapTestPage64(t *testing.T, memory *corecpu.Memory64, address corecpu.Address64) {
	t.Helper()
	if err := memory.Map(address, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite|corecpu.PAnonymous); err != nil {
		t.Fatalf("map %#x: %v", address, err)
	}
}

func set64RingSyscall(state *corecpu.MachineState64, number Number64, args ...uint64) {
	state.Set(corecpu.RAX, uint64(number))
	registers := []corecpu.Reg64{corecpu.RDI, corecpu.RSI, corecpu.RDX, corecpu.R10, corecpu.R8, corecpu.R9}
	for i, register := range registers {
		value := uint64(0)
		if i < len(args) {
			value = args[i]
		}
		state.Set(register, value)
	}
}

func TestIoUring64ReadAndRegistration(t *testing.T) {
	memory := corecpu.NewMemory64()
	ctx := NewContext64(memory)
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	const paramsAddress corecpu.Address64 = 0x1000
	const bufferAddress corecpu.Address64 = 0x2000
	const filesAddress corecpu.Address64 = 0x3000
	const iovecAddress corecpu.Address64 = 0x4000
	const probeAddress corecpu.Address64 = 0x5000
	mapTestPage64(t, memory, paramsAddress)
	mapTestPage64(t, memory, bufferAddress)
	mapTestPage64(t, memory, filesAddress)
	mapTestPage64(t, memory, iovecAddress)
	mapTestPage64(t, memory, probeAddress)

	var params [ioUringParamsSize64]byte
	binary.LittleEndian.PutUint32(params[0:4], 4)
	if err := memory.Write(paramsAddress, params[:]); err != nil {
		t.Fatal(err)
	}
	set64RingSyscall(state, Sys64IoUringSetup, 4, uint64(paramsAddress))
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	ringFD := int64(state.Get(corecpu.RAX))
	if ringFD < 0 {
		t.Fatalf("io_uring_setup = %d", ringFD)
	}
	if got := binary.LittleEndian.Uint32(readMemory64(t, memory, paramsAddress, ioUringParamsSize64)[0:4]); got != 4 {
		t.Fatalf("sq_entries = %d, want 4", got)
	}

	set64RingSyscall(state, Sys64Mmap, 0, corecpu.Page64Size, uint64(ProtRead|ProtWrite), uint64(MapShared), uint64(ringFD), ioUringOffSQRing64)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	sqBase := corecpu.Address64(state.Get(corecpu.RAX))
	if sqBase == 0 {
		t.Fatalf("sq mmap returned %#x", sqBase)
	}
	set64RingSyscall(state, Sys64Mmap, 0, corecpu.Page64Size, uint64(ProtRead|ProtWrite), uint64(MapShared), uint64(ringFD), ioUringOffCQRing64)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	cqBase := corecpu.Address64(state.Get(corecpu.RAX))
	set64RingSyscall(state, Sys64Mmap, 0, corecpu.Page64Size, uint64(ProtRead|ProtWrite), uint64(MapShared), uint64(ringFD), ioUringOffSQES64)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	sqesBase := corecpu.Address64(state.Get(corecpu.RAX))
	if cqBase == 0 || sqesBase == 0 {
		t.Fatalf("ring mappings: sq=%#x cq=%#x sqes=%#x", sqBase, cqBase, sqesBase)
	}

	const inputFD uint64 = 7
	if err := ctx.InstallFile(inputFD, &corefd.File{Reader: bytes.NewReader([]byte("hello"))}); err != nil {
		t.Fatal(err)
	}
	var sqe [ioUringSQESize64]byte
	sqe[0] = ioUringOpRead64
	binary.LittleEndian.PutUint32(sqe[4:8], uint32(inputFD))
	binary.LittleEndian.PutUint64(sqe[8:16], ^uint64(0))
	binary.LittleEndian.PutUint64(sqe[16:24], uint64(bufferAddress))
	binary.LittleEndian.PutUint32(sqe[24:28], 5)
	binary.LittleEndian.PutUint64(sqe[32:40], 0xfeed)
	if err := memory.Write(sqesBase, sqe[:]); err != nil {
		t.Fatal(err)
	}
	if result := ioUringWriteU32(ctx, sqBase+ioUringRingHeader64, 0); result != 0 || ioUringWriteU32(ctx, sqBase+4, 1) != 0 {
		t.Fatalf("submit queue setup result=%d", result)
	}

	set64RingSyscall(state, Sys64IoUringEnter, uint64(ringFD), 1, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("io_uring_enter: err=%v result=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if got := string(readMemory64(t, memory, bufferAddress, 5)); got != "hello" {
		t.Fatalf("read buffer = %q", got)
	}
	cqe := readMemory64(t, memory, cqBase+ioUringCQEOffset64, ioUringCQESize64)
	if binary.LittleEndian.Uint64(cqe[0:8]) != 0xfeed || int32(binary.LittleEndian.Uint32(cqe[8:12])) != 5 {
		t.Fatalf("cqe user_data=%#x res=%d", binary.LittleEndian.Uint64(cqe[0:8]), int32(binary.LittleEndian.Uint32(cqe[8:12])))
	}

	var fileEntry [4]byte
	binary.LittleEndian.PutUint32(fileEntry[:], uint32(inputFD))
	if err := memory.Write(filesAddress, fileEntry[:]); err != nil {
		t.Fatal(err)
	}
	set64RingSyscall(state, Sys64IoUringRegister, uint64(ringFD), ioUringRegisterFiles64, uint64(filesAddress), 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("register files: err=%v result=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var iovec [16]byte
	binary.LittleEndian.PutUint64(iovec[0:8], uint64(bufferAddress))
	binary.LittleEndian.PutUint64(iovec[8:16], 5)
	if err := memory.Write(iovecAddress, iovec[:]); err != nil {
		t.Fatal(err)
	}
	set64RingSyscall(state, Sys64IoUringRegister, uint64(ringFD), ioUringRegisterBuffers64, uint64(iovecAddress), 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("register buffers: err=%v result=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64RingSyscall(state, Sys64IoUringRegister, uint64(ringFD), ioUringRegisterProbe64, uint64(probeAddress), 32)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("register probe: err=%v result=%d", err, int64(state.Get(corecpu.RAX)))
	}
	probe := readMemory64(t, memory, probeAddress, 16)
	if probe[0] != ioUringOpWrite64 || probe[1] != 32 {
		t.Fatalf("probe header last_op=%d ops_len=%d", probe[0], probe[1])
	}

	set64RingSyscall(state, Sys64IoUringRegister, uint64(ringFD), ioUringUnregisterBuffers64, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("unregister buffers: err=%v result=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64RingSyscall(state, Sys64IoUringRegister, uint64(ringFD), ioUringUnregisterFiles64, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("unregister files: err=%v result=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64RingSyscall(state, Sys64Close, uint64(ringFD))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("close ring: err=%v result=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func readMemory64(t *testing.T, memory *corecpu.Memory64, address corecpu.Address64, length uint64) []byte {
	t.Helper()
	data := make([]byte, int(length))
	if err := memory.Read(address, data); err != nil {
		t.Fatalf("read %#x/%d: %v", address, length, err)
	}
	return data
}
