package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestABI64PrctlProcessState(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	const address corecpu.Address64 = 0x10700
	name := make([]byte, 16)
	copy(name, "ish-go")
	if err := memory.Write(address, name); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Prctl, prSetName64, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("set name: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	for index := range name {
		name[index] = 0
	}
	if err := memory.Write(address, name); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Prctl, prGetName64, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("get name: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(address, name); err != nil {
		t.Fatal(err)
	}
	if string(name[:6]) != "ish-go" {
		t.Fatalf("process name=%q", name)
	}
	set64Syscall(state, Sys64Prctl, prSetPDeathSig64, 15)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("set pdeathsig: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Prctl, prGetPDeathSig64, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("get pdeathsig: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var signal [4]byte
	if err := memory.Read(address, signal[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(signal[:]) != 15 {
		t.Fatalf("pdeathsig=%d", binary.LittleEndian.Uint32(signal[:]))
	}
	set64Syscall(state, Sys64Prctl, prGetDumpable64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("get dumpable: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Prctl, prSetNoNewPrivs64, 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 || !ctx.NoNewPrivs {
		t.Fatalf("set no-new-privs: err=%v rax=%d state=%v", err, int64(state.Get(corecpu.RAX)), ctx.NoNewPrivs)
	}
	set64Syscall(state, Sys64Prctl, prGetNoNewPrivs64)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 1 {
		t.Fatalf("get no-new-privs: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestABI64AffinityAndGetCPU(t *testing.T) {
	_, memory, ctx, dispatcher, state := newABI64FilesystemTest(t)
	ctx.PID = 321
	const address corecpu.Address64 = 0x10800
	var mask [8]byte
	binary.LittleEndian.PutUint64(mask[:], 1)
	if err := memory.Write(address, mask[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64SchedSetAffinity, 0, cpuSetSize64, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 || ctx.AffinityMask != 1 {
		t.Fatalf("set affinity: err=%v rax=%d mask=%#x", err, int64(state.Get(corecpu.RAX)), ctx.AffinityMask)
	}
	set64Syscall(state, Sys64SchedGetAffinity, 0, cpuSetSize64, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(cpuSetSize64) {
		t.Fatalf("get affinity: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if err := memory.Read(address, mask[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint64(mask[:]) != 1 {
		t.Fatalf("affinity mask=%#x", binary.LittleEndian.Uint64(mask[:]))
	}
	set64Syscall(state, Sys64GetCPU, uint64(address), uint64(address+4), 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("getcpu: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var cpuNode [8]byte
	if err := memory.Read(address, cpuNode[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(cpuNode[0:4]) != 0 || binary.LittleEndian.Uint32(cpuNode[4:8]) != 0 {
		t.Fatalf("cpu/node=%d/%d", binary.LittleEndian.Uint32(cpuNode[0:4]), binary.LittleEndian.Uint32(cpuNode[4:8]))
	}
	set64Syscall(state, Sys64SchedGetAffinity, 999, cpuSetSize64, uint64(address))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(ESRCH) {
		t.Fatalf("foreign affinity: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
