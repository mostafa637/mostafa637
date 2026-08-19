package syscall

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func TestDispatcher64SharedMemoryLifecycle(t *testing.T) {
	memory := corecpu.NewMemory64()
	const metadata corecpu.Address64 = 0x12000
	if err := memory.Map(metadata, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.PID = 77
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	set64Syscall(state, Sys64Shmget, 0x4455, corecpu.Page64Size, uint64(ipcCreat64|0o600))
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) == 0 {
		t.Fatalf("shmget: err=%v id=%d", err, int64(state.Get(corecpu.RAX)))
	}
	shmID := state.Get(corecpu.RAX)
	set64Syscall(state, Sys64Shmget, 0x4455, corecpu.Page64Size, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != shmID {
		t.Fatalf("shmget existing: err=%v id=%d want=%d", err, int64(state.Get(corecpu.RAX)), shmID)
	}

	set64Syscall(state, Sys64Shmat, shmID, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 0 {
		t.Fatalf("shmat: err=%v addr=%#x", err, state.Get(corecpu.RAX))
	}
	attached := corecpu.Address64(state.Get(corecpu.RAX))
	if err := memory.Write(attached, []byte("shared")); err != nil {
		t.Fatal(err)
	}

	set64Syscall(state, Sys64Shmctl, shmID, ipcStat64, uint64(metadata))
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("shmctl IPC_STAT: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var ds [shmDSSize64]byte
	if err := memory.Read(metadata, ds[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(ds[48:56]); got != corecpu.Page64Size {
		t.Fatalf("shm size=%d want=%d", got, corecpu.Page64Size)
	}
	if got := binary.LittleEndian.Uint64(ds[88:96]); got != 1 {
		t.Fatalf("shm nattch=%d want=1", got)
	}

	set64Syscall(state, Sys64Shmdt, uint64(attached))
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("shmdt: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Shmat, shmID, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) < 0 {
		t.Fatalf("shmat second: err=%v addr=%#x", err, state.Get(corecpu.RAX))
	}
	attached = corecpu.Address64(state.Get(corecpu.RAX))
	var restored [6]byte
	if err := memory.Read(attached, restored[:]); err != nil {
		t.Fatal(err)
	}
	if string(restored[:]) != "shared" {
		t.Fatalf("shared data=%q", restored)
	}
	set64Syscall(state, Sys64Shmctl, shmID, ipcRmid64, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("shmctl IPC_RMID: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Shmat, shmID, 0, 0)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != eidrm64 {
		t.Fatalf("shmat removed: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Shmdt, uint64(attached))
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("shmdt removed segment: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}

func TestDispatcher64SemaphoreLifecycle(t *testing.T) {
	memory := corecpu.NewMemory64()
	const buffer corecpu.Address64 = 0x14000
	if err := memory.Map(buffer, 2*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.PID = 88
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	set64Syscall(state, Sys64Semget, 0x7788, 2, uint64(ipcCreat64|0o600))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) <= 0 {
		t.Fatalf("semget: err=%v id=%d", err, int64(state.Get(corecpu.RAX)))
	}
	semID := state.Get(corecpu.RAX)
	set64Syscall(state, Sys64Semctl, semID, 0, semSetVal64, 3)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("semctl SETVAL: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Semctl, semID, 0, semGetVal64)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 3 {
		t.Fatalf("semctl GETVAL=%d", int64(state.Get(corecpu.RAX)))
	}

	var operation [6]byte
	binary.LittleEndian.PutUint16(operation[0:2], 0)
	binary.LittleEndian.PutUint16(operation[2:4], ^uint16(1))
	binary.LittleEndian.PutUint16(operation[4:6], 0)
	if err := memory.Write(buffer, operation[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Semop, semID, uint64(buffer), 1)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("semop decrement: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Semctl, semID, 0, semGetVal64)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 1 {
		t.Fatalf("semop GETVAL=%d", int64(state.Get(corecpu.RAX)))
	}

	binary.LittleEndian.PutUint16(operation[2:4], ^uint16(1))
	binary.LittleEndian.PutUint16(operation[4:6], semNowait64)
	if err := memory.Write(buffer, operation[:]); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Semop, semID, uint64(buffer), 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != int64(EAGAIN) {
		t.Fatalf("semop NOWAIT: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}

	var values [2]uint16
	if err := memory.Write(buffer, []byte{4, 0, 7, 0}); err != nil {
		t.Fatal(err)
	}
	set64Syscall(state, Sys64Semctl, semID, 0, semSetAll64, uint64(buffer))
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("semctl SETALL: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Semctl, semID, 0, semGetAll64, uint64(buffer))
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("semctl GETALL: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	var encodedValues [4]byte
	if err := memory.Read(buffer, encodedValues[:]); err != nil {
		t.Fatal(err)
	}
	values[0] = binary.LittleEndian.Uint16(encodedValues[0:2])
	values[1] = binary.LittleEndian.Uint16(encodedValues[2:4])
	if values != [2]uint16{4, 7} {
		t.Fatalf("semctl GETALL=%v", values)
	}

	set64Syscall(state, Sys64Semctl, semID, 0, semIPCRmid64)
	if _, err := dispatcher.Dispatch(state); err != nil || state.Get(corecpu.RAX) != 0 {
		t.Fatalf("semctl IPC_RMID: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	set64Syscall(state, Sys64Semop, semID, uint64(buffer), 1)
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != eidrm64 {
		t.Fatalf("semop removed: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
}
