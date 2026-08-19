package syscall

import (
	"bytes"
	"encoding/binary"
	"io"
	"path/filepath"
	"testing"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
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

func TestDescriptorSyscalls(t *testing.T) {
	memory := cpu.NewMemory()
	context := NewContext(memory)
	input := bytes.NewReader([]byte("hello"))
	if err := context.InstallFile(3, &File{Reader: input, Seeker: input}); err != nil {
		t.Fatal(err)
	}
	state := cpu.NewMachineState(memory)
	dispatcher := NewDispatcher(context)
	if got := dispatcher.Dispatch(state, SysDup, 3); got < 0 {
		t.Fatalf("dup = %d", got)
	}
	dupfd := uint32(state.EAXValue())
	if got := dispatcher.Dispatch(state, SysLseek, dupfd, 2, uint32(io.SeekStart)); got != 2 {
		t.Fatalf("lseek = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysDup2, dupfd, 9); got != 9 {
		t.Fatalf("dup2 = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysClose, 9); got != 0 {
		t.Fatalf("close = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysClose, 9); got != EBADF {
		t.Fatalf("second close = %d", got)
	}
}

func TestFakeFSSyscalls(t *testing.T) {
	db, err := storage.Open(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake, err := corefs.New(filepath.Join(t.TempDir(), "root"), db)
	if err != nil {
		t.Fatal(err)
	}
	if err := fake.Mkdir("/etc", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile("/etc/name", []byte("pure-go"), 0o644, 0, 0); err != nil {
		t.Fatal(err)
	}

	memory := cpu.NewMemory()
	if err := memory.Map(1, 9, cpu.PRead|cpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context := NewContext(memory)
	context.FS = fake
	dispatcher := NewDispatcher(context)
	state := cpu.NewMachineState(memory)

	writeGuest := func(address cpu.Address, value string) {
		t.Helper()
		if err := memory.Write(address, append([]byte(value), 0)); err != nil {
			t.Fatal(err)
		}
	}
	readGuest := func(address cpu.Address, size int) []byte {
		t.Helper()
		value := make([]byte, size)
		if err := memory.Read(address, value); err != nil {
			t.Fatal(err)
		}
		return value
	}

	writeGuest(0x1000, "/etc/name")
	if got := dispatcher.Dispatch(state, SysStat64, 0x1000, 0x2000); got != 0 {
		t.Fatalf("stat64 = %d", got)
	}
	stat := readGuest(0x2000, stat64Size)
	if mode := binary.LittleEndian.Uint32(stat[16:20]); mode != corefs.ModeRegular|0o644 {
		t.Fatalf("stat64 mode = %#o", mode)
	}
	if size := binary.LittleEndian.Uint64(stat[44:52]); size != uint64(len("pure-go")) {
		t.Fatalf("stat64 size = %d", size)
	}

	writeGuest(0x3000, "/etc")
	if got := dispatcher.Dispatch(state, SysOpen, 0x3000, 0, 0); got < 3 {
		t.Fatalf("open directory = %d", got)
	}
	dirFD := uint32(state.EAXValue())
	if got := dispatcher.Dispatch(state, SysGetdents64, dirFD, 0x4000, 256); got <= 0 {
		t.Fatalf("getdents64 = %d", got)
	}
	dirents := string(readGuest(0x4000, 256))
	for _, name := range []string{".", "..", "name"} {
		if !bytes.Contains([]byte(dirents), []byte(name)) {
			t.Fatalf("getdents64 missing %q in %q", name, dirents)
		}
	}

	writeGuest(0x5000, "/etc")
	if got := dispatcher.Dispatch(state, SysChdir, 0x5000); got != 0 {
		t.Fatalf("chdir = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysGetCWD, 0x6000, 64); got != int32(len("/etc")+1) {
		t.Fatalf("getcwd = %d", got)
	}
	if cwd := string(readGuest(0x6000, len("/etc")+1)); cwd != "/etc\x00" {
		t.Fatalf("cwd = %q", cwd)
	}

	writeGuest(0x7000, "name")
	if got := dispatcher.Dispatch(state, SysOpen, 0x7000, 0, 0); got < 3 {
		t.Fatalf("open relative file = %d", got)
	}
	fileFD := uint32(state.EAXValue())
	if got := dispatcher.Dispatch(state, SysFstat64, fileFD, 0x8000); got != 0 {
		t.Fatalf("fstat64 = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysRead, fileFD, 0x9000, 7); got != 7 {
		t.Fatalf("read relative file = %d", got)
	}
	if data := string(readGuest(0x9000, 7)); data != "pure-go" {
		t.Fatalf("relative file data = %q", data)
	}
}

func TestTaskLifecycleStubs(t *testing.T) {
	memory := cpu.NewMemory()
	context := NewContext(memory)
	context.PID = 77
	dispatcher := NewDispatcher(context)
	state := cpu.NewMachineState(memory)

	if got := dispatcher.Dispatch(state, SysFork); got != ENOSYS {
		t.Fatalf("fork = %d, want %d", got, ENOSYS)
	}
	if got := dispatcher.Dispatch(state, SysClone, 0, 0, 0, 0, 0); got != ENOSYS {
		t.Fatalf("clone = %d, want %d", got, ENOSYS)
	}
	if got := dispatcher.Dispatch(state, SysWait4, ^uint32(0), 0, 0, 0); got != ECHILD {
		t.Fatalf("wait4 = %d, want %d", got, ECHILD)
	}
	if got := dispatcher.Dispatch(state, SysKill, context.PID, 15); got != 0 {
		t.Fatalf("kill self = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysKill, 999, 15); got != ESRCH {
		t.Fatalf("kill unknown = %d, want %d", got, ESRCH)
	}
	if got := dispatcher.Dispatch(state, SysGetTID); got != int32(context.PID) {
		t.Fatalf("gettid = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysSetTIDAddress, 0x1234); got != int32(context.PID) || context.TIDAddress != 0x1234 {
		t.Fatalf("set_tid_address = %d address=%#x", got, context.TIDAddress)
	}
}

func TestWait4ChildRegistry(t *testing.T) {
	memory := cpu.NewMemory()
	if err := memory.Map(1, 1, cpu.PRead|cpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context := NewContext(memory)
	context.PID = 77
	if !context.Children.AddChild(context.PID, 88) {
		t.Fatal("AddChild failed")
	}
	dispatcher := NewDispatcher(context)
	state := cpu.NewMachineState(memory)

	if got := dispatcher.Dispatch(state, SysWait4, 88, 0x1000, WaitNoHang, 0); got != 0 {
		t.Fatalf("wait4 WNOHANG before exit = %d", got)
	}
	if !context.Children.MarkExited(88, 42) {
		t.Fatal("MarkExited failed")
	}
	if got := dispatcher.Dispatch(state, SysWait4, 88, 0x1000, 0, 0); got != 88 {
		t.Fatalf("wait4 exited child = %d", got)
	}
	var status [4]byte
	if err := memory.Read(0x1000, status[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(status[:]); got != 42<<8 {
		t.Fatalf("wait status = %#x", got)
	}
	if got := dispatcher.Dispatch(state, SysWait4, 88, 0, WaitNoHang, 0); got != ECHILD {
		t.Fatalf("wait4 reaped child = %d", got)
	}
}

func TestShellProbeSyscalls(t *testing.T) {
	db, err := storage.Open(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake, err := corefs.New(filepath.Join(t.TempDir(), "root"), db)
	if err != nil {
		t.Fatal(err)
	}
	if err := fake.Mkdir("/bin", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile("/bin/sh", []byte("shell"), 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.Symlink("sh", "/bin/link", 0, 0); err != nil {
		t.Fatal(err)
	}
	memory := cpu.NewMemory()
	if err := memory.Map(1, 4, cpu.PRead|cpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context := NewContext(memory)
	context.FS = fake
	context.PID = 123
	if err := context.InstallFile(3, &File{Reader: bytes.NewBuffer(nil), Writer: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDispatcher(context)
	state := cpu.NewMachineState(memory)
	writeGuest := func(address cpu.Address, value string) {
		t.Helper()
		if err := memory.Write(address, append([]byte(value), 0)); err != nil {
			t.Fatal(err)
		}
	}

	writeGuest(0x1000, "/bin/sh")
	if got := dispatcher.Dispatch(state, SysAccess, 0x1000, 0); got != 0 {
		t.Fatalf("access = %d", got)
	}
	writeGuest(0x1100, "/bin/link")
	if got := dispatcher.Dispatch(state, SysReadlink, 0x1100, 0x1200, 32); got != 2 {
		t.Fatalf("readlink = %d", got)
	}
	linkTarget := make([]byte, 2)
	if err := memory.Read(0x1200, linkTarget); err != nil {
		t.Fatal(err)
	}
	if string(linkTarget) != "sh" {
		t.Fatalf("readlink target = %q", linkTarget)
	}
	if got := dispatcher.Dispatch(state, SysUname, 0x2000); got != 0 {
		t.Fatalf("uname = %d", got)
	}
	uname := make([]byte, 65)
	if err := memory.Read(0x2000, uname); err != nil {
		t.Fatal(err)
	}
	if string(uname[:5]) != "Linux" {
		t.Fatalf("uname sysname = %q", uname[:5])
	}
	if got := dispatcher.Dispatch(state, SysFcntl64, 3, 3); got != 0 {
		t.Fatalf("fcntl F_GETFL = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysFcntl64, 3, 0); got < 4 {
		t.Fatalf("fcntl F_DUPFD = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysIoctl, 3, 0x5413, 0x3000); got != 0 {
		t.Fatalf("ioctl TIOCGWINSZ = %d", got)
	}
	var winsize [8]byte
	if err := memory.Read(0x3000, winsize[:]); err != nil {
		t.Fatal(err)
	}
	if rows, cols := binary.LittleEndian.Uint16(winsize[0:2]), binary.LittleEndian.Uint16(winsize[2:4]); rows != 24 || cols != 80 {
		t.Fatalf("winsize = %dx%d", cols, rows)
	}
	if got := dispatcher.Dispatch(state, SysGetUID); got != 0 {
		t.Fatalf("getuid = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysGetPPID); got != 1 {
		t.Fatalf("getppid = %d", got)
	}
}

func TestDispatcherExecveReadsGuestVectors(t *testing.T) {
	memory := cpu.NewMemory()
	if err := memory.Map(4, 1, cpu.PRead|cpu.PWrite); err != nil {
		t.Fatal(err)
	}
	context := NewContext(memory)
	var gotPath string
	var gotArgv, gotEnv []string
	context.Execve = func(path string, argv, env []string) int32 {
		gotPath = path
		gotArgv = append([]string(nil), argv...)
		gotEnv = append([]string(nil), env...)
		return 0
	}
	dispatcher := NewDispatcher(context)
	state := cpu.NewMachineState(memory)

	const base = uint32(4 * cpu.PageSize)
	pathAddr := base + 0x100
	argvAddr := base + 0x120
	envAddr := base + 0x130
	arg0Addr := base + 0x140
	env0Addr := base + 0x150
	write := func(address uint32, data []byte) {
		if err := memory.Write(cpu.Address(address), data); err != nil {
			t.Fatal(err)
		}
	}
	write(pathAddr, []byte("/bin/echo\x00"))
	write(arg0Addr, []byte("echo\x00"))
	write(env0Addr, []byte("PATH=/bin\x00"))
	var vector [8]byte
	binary.LittleEndian.PutUint32(vector[0:4], arg0Addr)
	write(argvAddr, vector[:])
	binary.LittleEndian.PutUint32(vector[0:4], env0Addr)
	write(envAddr, vector[:])

	if got := dispatcher.Dispatch(state, SysExecve, pathAddr, argvAddr, envAddr); got != 0 {
		t.Fatalf("execve result = %d", got)
	}
	if gotPath != "/bin/echo" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(gotArgv) != 1 || gotArgv[0] != "echo" {
		t.Fatalf("argv = %#v", gotArgv)
	}
	if len(gotEnv) != 1 || gotEnv[0] != "PATH=/bin" {
		t.Fatalf("env = %#v", gotEnv)
	}
}

func TestDispatcherPipePollAndReadWrite(t *testing.T) {
	memory := cpu.NewMemory()
	for page := cpu.Page(4); page <= 6; page++ {
		if err := memory.Map(page, 1, cpu.PRead|cpu.PWrite); err != nil {
			t.Fatal(err)
		}
	}
	context := NewContext(memory)
	dispatcher := NewDispatcher(context)
	state := cpu.NewMachineState(memory)
	const (
		pipeAddress = uint32(4 * cpu.PageSize)
		pollAddress = uint32(5 * cpu.PageSize)
		dataAddress = uint32(6 * cpu.PageSize)
	)
	if got := dispatcher.Dispatch(state, SysPipe, pipeAddress); got != 0 {
		t.Fatalf("pipe = %d", got)
	}
	var raw [8]byte
	if err := memory.Read(cpu.Address(pipeAddress), raw[:]); err != nil {
		t.Fatal(err)
	}
	readFD := binary.LittleEndian.Uint32(raw[0:4])
	writeFD := binary.LittleEndian.Uint32(raw[4:8])
	if readFD == writeFD {
		t.Fatalf("pipe descriptors are equal: %d", readFD)
	}
	if err := memory.Write(cpu.Address(dataAddress), []byte("ping")); err != nil {
		t.Fatal(err)
	}
	if got := dispatcher.Dispatch(state, SysWrite, writeFD, dataAddress, 4); got != 4 {
		t.Fatalf("pipe write = %d", got)
	}
	var pollfd [8]byte
	binary.LittleEndian.PutUint32(pollfd[0:4], readFD)
	binary.LittleEndian.PutUint16(pollfd[4:6], pollIn)
	if err := memory.Write(cpu.Address(pollAddress), pollfd[:]); err != nil {
		t.Fatal(err)
	}
	if got := dispatcher.Dispatch(state, SysPoll, pollAddress, 1, 0); got != 1 {
		t.Fatalf("poll = %d", got)
	}
	if err := memory.Read(cpu.Address(pollAddress+6), pollfd[6:8]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(pollfd[6:8])&pollIn == 0 {
		t.Fatalf("poll revents = %#x", binary.LittleEndian.Uint16(pollfd[6:8]))
	}
	if got := dispatcher.Dispatch(state, SysRead, readFD, dataAddress+0x100, 4); got != 4 {
		t.Fatalf("pipe read = %d", got)
	}
	var result [4]byte
	if err := memory.Read(cpu.Address(dataAddress+0x100), result[:]); err != nil {
		t.Fatal(err)
	}
	if string(result[:]) != "ping" {
		t.Fatalf("pipe data = %q", result[:])
	}
	if got := dispatcher.Dispatch(state, SysClose, writeFD); got != 0 {
		t.Fatalf("close writer = %d", got)
	}
	if got := dispatcher.Dispatch(state, SysPoll, pollAddress, 1, 0); got != 1 {
		t.Fatalf("poll EOF = %d", got)
	}
}
