package session

import (
	"testing"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coresyscall "github.com/mostafa637/mostafa637/go-pure/internal/core/syscall"
)

func TestGuest64ChildRuntimeRunsAndPublishesExit(t *testing.T) {
	memory := corecpu.NewMemory64()
	code := []byte{
		0xb8, 0x3c, 0x00, 0x00, 0x00, // mov eax, SYS_exit
		0xbf, 0x07, 0x00, 0x00, 0x00, // mov edi, 7
		0x0f, 0x05, // syscall
	}
	if err := memory.MapBytes(0x4000, code, corecpu.PRead|corecpu.PWrite|corecpu.PExec); err != nil {
		t.Fatal(err)
	}
	parent := coresyscall.NewContext64(memory)
	parent.PID = 100
	parent.TID = 100
	parentState := corecpu.NewMachineState64(memory)
	parentState.RIP = 0x4000
	parent.Machine = parentState
	transport := newGuestTransport("")
	request := coresyscall.CloneRequest64{Fork: true}
	childPID := transport.createChild64(parent, request)
	if childPID <= 0 {
		t.Fatalf("create child = %d", childPID)
	}
	if !parent.Children.AddChild(uint32(parent.PID), uint32(childPID)) {
		t.Fatal("failed to register child")
	}
	transport.startChild64(parent, childPID, request)

	deadline := time.After(2 * time.Second)
	for {
		pid, status, err := parent.Children.Wait(uint32(parent.PID), int32(childPID), coresyscall.WaitNoHang)
		if err == 0 && pid != 0 {
			if int64(pid) != childPID || status != 7<<8 {
				t.Fatalf("wait child: pid=%d status=%d", pid, status)
			}
			return
		}
		if err == 0 && pid == 0 {
			select {
			case <-deadline:
				t.Fatal("child did not exit")
			case <-time.After(time.Millisecond):
			}
			continue
		}
		if err != coresyscall.EINTR {
			t.Fatalf("wait child err=%d", err)
		}
		select {
		case <-deadline:
			t.Fatal("child did not exit")
		case <-time.After(time.Millisecond):
		}
	}
}
