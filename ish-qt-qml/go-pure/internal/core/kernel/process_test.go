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
