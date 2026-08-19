package kernel

import (
	"bytes"
	"testing"

	coresyscall "github.com/mostafa637/mostafa637/go-pure/internal/core/syscall"
)

func TestProcessSyscallBoundary(t *testing.T) {
	process := NewProcess(77, nil)
	defer process.Close()
	if process.Memory == nil || process.CPU == nil || process.Syscalls == nil {
		t.Fatal("process did not initialize CPU, memory and syscall dispatcher")
	}
	if err := process.Memory.Map(4, 1, 3); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := process.AttachFile(1, nil, &output); err != nil {
		t.Fatal(err)
	}
	if err := process.Memory.Write(4*4096, []byte("kernel")); err != nil {
		t.Fatal(err)
	}
	if got := process.Syscall(coresyscall.SysWrite, 1, 4*4096, 6); got != 6 {
		t.Fatalf("write result = %d", got)
	}
	if output.String() != "kernel" {
		t.Fatalf("output = %q", output.String())
	}
	if got := process.Syscall(coresyscall.SysGetPID); got != 77 {
		t.Fatalf("getpid result = %d", got)
	}
}

func TestProcessCloseIsIdempotent(t *testing.T) {
	process := NewProcess(1, nil)
	if err := process.Close(); err != nil {
		t.Fatal(err)
	}
	if err := process.Close(); err != nil {
		t.Fatal(err)
	}
	if got := process.Syscall(coresyscall.SysGetPID); got != coresyscall.EFAULT {
		t.Fatalf("closed syscall = %d", got)
	}
}

func TestProcessRunsGuestInstructions(t *testing.T) {
	process := NewProcess(88, nil)
	defer process.Close()
	if err := process.Memory.Map(1, 1, 1|2|4); err != nil {
		t.Fatal(err)
	}
	if err := process.Memory.Map(2, 1, 1|2); err != nil {
		t.Fatal(err)
	}
	code := []byte{
		0xB8, 0x14, 0x00, 0x00, 0x00, // mov eax, getpid
		0xCD, 0x80, // int 0x80
		0xF4, // hlt
	}
	if err := process.Memory.Write(4096, code); err != nil {
		t.Fatal(err)
	}
	process.CPU.EIP = 4096
	process.CPU.Set(4, 2*4096)
	if err := process.Run(8); err != nil {
		t.Fatal(err)
	}
	if process.CPU.Get(0) != 88 {
		t.Fatalf("guest eax = %d", process.CPU.Get(0))
	}
}

func TestProcessStopsOnGuestExit(t *testing.T) {
	process := NewProcess(99, nil)
	defer process.Close()
	if err := process.Memory.Map(1, 1, 1|2|4); err != nil {
		t.Fatal(err)
	}
	if err := process.Memory.Write(4096, []byte{
		0xB8, 0x01, 0x00, 0x00, 0x00, // eax = SYS_EXIT
		0xBB, 0x07, 0x00, 0x00, 0x00, // ebx = 7
		0xCD, 0x80,
	}); err != nil {
		t.Fatal(err)
	}
	process.CPU.EIP = 4096
	if err := process.Run(8); err != nil {
		t.Fatal(err)
	}
	code, exited := process.ExitCode()
	if !exited || code != 7 {
		t.Fatalf("exit state: exited=%v code=%d", exited, code)
	}
}
