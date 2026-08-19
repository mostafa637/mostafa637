package kernel

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coresyscall "github.com/mostafa637/mostafa637/go-pure/internal/core/syscall"
)

func TestProcessManagerChildExitAndWait(t *testing.T) {
	manager := NewProcessManager(nil, 100)
	parent := manager.NewRoot()
	if parent.PID != 100 {
		t.Fatalf("parent PID = %d", parent.PID)
	}
	if err := parent.Memory.Map(1, 1, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	child, err := manager.CreateChild(parent)
	if err != nil {
		t.Fatal(err)
	}
	if child.PID != 101 {
		t.Fatalf("child PID = %d", child.PID)
	}
	if child.FS != parent.FS || child.Memory == parent.Memory {
		t.Fatal("child did not get an independent process object")
	}
	if got, ok := manager.Lookup(child.PID); !ok || got != child {
		t.Fatal("child lookup failed")
	}
	if got := child.Syscall(coresyscall.SysExit, 7); got != 0 {
		t.Fatalf("child exit = %d", got)
	}
	if !manager.MarkExited(child) {
		t.Fatal("MarkExited failed")
	}
	if got := parent.Syscall(coresyscall.SysWait4, child.PID, 0x1000, 0, 0); got != int32(child.PID) {
		t.Fatalf("wait4 = %d", got)
	}
	var status [4]byte
	if err := parent.Memory.Read(0x1000, status[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(status[:]); got != 7<<8 {
		t.Fatalf("wait status = %#x", got)
	}
	if got := parent.Syscall(coresyscall.SysWait4, child.PID, 0, coresyscall.WaitNoHang, 0); got != coresyscall.ECHILD {
		t.Fatalf("second wait4 = %d", got)
	}
}
