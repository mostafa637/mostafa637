package syscall

import (
	"encoding/binary"
	"reflect"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func TestExecve64ReadsGuestVectors(t *testing.T) {
	memory := corecpu.NewMemory64()
	const base corecpu.Address64 = 0x1000
	if err := memory.Map(base, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	var gotPath string
	var gotArgv, gotEnv []string
	ctx.Execve = func(path string, argv, env []string) int64 {
		gotPath, gotArgv, gotEnv = path, append([]string(nil), argv...), append([]string(nil), env...)
		return 0
	}
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)

	const (
		pathAddr = base + 0x100
		argAddr  = base + 0x120
		envAddr  = base + 0x140
		argvAddr = base + 0x180
		envpAddr = base + 0x1a0
	)
	putGuestString64(t, memory, pathAddr, "/bin/demo")
	putGuestString64(t, memory, argAddr, "--version")
	putGuestString64(t, memory, envAddr, "PATH=/bin")
	putGuestPointerVector64(t, memory, argvAddr, pathAddr, argAddr)
	putGuestPointerVector64(t, memory, envpAddr, envAddr)

	state.Set(corecpu.RAX, uint64(Sys64Execve))
	state.Set(corecpu.RDI, uint64(pathAddr))
	state.Set(corecpu.RSI, uint64(argvAddr))
	state.Set(corecpu.RDX, uint64(envpAddr))
	resumed, err := dispatcher.Dispatch(state)
	if err != nil || !resumed {
		t.Fatalf("dispatch: resumed=%v err=%v", resumed, err)
	}
	if gotPath != "/bin/demo" {
		t.Fatalf("path=%q", gotPath)
	}
	if !reflect.DeepEqual(gotArgv, []string{"/bin/demo", "--version"}) {
		t.Fatalf("argv=%v", gotArgv)
	}
	if !reflect.DeepEqual(gotEnv, []string{"PATH=/bin"}) {
		t.Fatalf("env=%v", gotEnv)
	}
	if state.Get(corecpu.RAX) != 0 {
		t.Fatalf("rax=%#x", state.Get(corecpu.RAX))
	}
}

func TestExecve64RejectsInvalidVector(t *testing.T) {
	memory := corecpu.NewMemory64()
	const base corecpu.Address64 = 0x2000
	if err := memory.Map(base, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.Execve = func(string, []string, []string) int64 { return 0 }
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)
	putGuestString64(t, memory, base+0x20, "/bin/demo")
	state.Set(corecpu.RAX, uint64(Sys64Execve))
	state.Set(corecpu.RDI, uint64(base+0x20))
	state.Set(corecpu.RSI, uint64(base)+corecpu.Page64Size)
	state.Set(corecpu.RDX, 0)
	if _, err := dispatcher.Dispatch(state); err != nil {
		t.Fatal(err)
	}
	if int64(state.Get(corecpu.RAX)) != int64(EFAULT) {
		t.Fatalf("rax=%d, want %d", int64(state.Get(corecpu.RAX)), EFAULT)
	}
}

func putGuestString64(t *testing.T, memory *corecpu.Memory64, address corecpu.Address64, value string) {
	t.Helper()
	data := append([]byte(value), 0)
	if err := memory.Write(address, data); err != nil {
		t.Fatal(err)
	}
}

func putGuestPointerVector64(t *testing.T, memory *corecpu.Memory64, address corecpu.Address64, pointers ...corecpu.Address64) {
	t.Helper()
	data := make([]byte, 8*(len(pointers)+1))
	for index, pointer := range pointers {
		binary.LittleEndian.PutUint64(data[index*8:], uint64(pointer))
	}
	if err := memory.Write(address, data); err != nil {
		t.Fatal(err)
	}
}

func TestExecve64PreservesLifecycleAndClosesCLOEXEC(t *testing.T) {
	memory := corecpu.NewMemory64()
	const base corecpu.Address64 = 0x5000
	if err := memory.Map(base, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	ctx := NewContext64(memory)
	ctx.PID = 77
	ctx.ParentPID = 9
	if !ctx.Children.AddChild(uint32(ctx.PID), 100) {
		t.Fatal("add child failed")
	}
	file := &corefd.File{Cloexec: true}
	if err := ctx.InstallFile(3, file); err != nil {
		t.Fatal(err)
	}
	ctx.Execve = func(string, []string, []string) int64 {
		ctx.CloseOnExec()
		return 0
	}
	const pathAddress corecpu.Address64 = base + 0x100
	putGuestString64(t, memory, pathAddress, "/bin/demo")
	dispatcher := NewDispatcher64(ctx)
	state := corecpu.NewMachineState64(memory)
	state.Set(corecpu.RAX, uint64(Sys64Execve))
	state.Set(corecpu.RDI, uint64(pathAddress))
	if _, err := dispatcher.Dispatch(state); err != nil || int64(state.Get(corecpu.RAX)) != 0 {
		t.Fatalf("execve: err=%v rax=%d", err, int64(state.Get(corecpu.RAX)))
	}
	if ctx.PID != 77 || ctx.ParentPID != 9 {
		t.Fatalf("identity changed: pid=%d ppid=%d", ctx.PID, ctx.ParentPID)
	}
	if _, err := ctx.GetFile(3); err == nil {
		t.Fatal("CLOEXEC descriptor survived execve")
	}
	if !ctx.Children.MarkExited(100, 3) {
		t.Fatal("child registry did not survive execve")
	}
}
