package kernel

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	coreloader "github.com/mostafa637/mostafa637/go-pure/internal/core/loader"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

func processELF() []byte {
	const headerSize = 52
	const programSize = 32
	const programOffset = headerSize
	const payloadOffset = 0x1000
	data := make([]byte, payloadOffset+0x20)
	copy(data[0:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4], data[5], data[6], data[7] = 1, 1, 1, 3
	binary.LittleEndian.PutUint16(data[16:], 2)
	binary.LittleEndian.PutUint16(data[18:], 3)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint32(data[24:], 0x08048100)
	binary.LittleEndian.PutUint32(data[28:], programOffset)
	binary.LittleEndian.PutUint16(data[40:], headerSize)
	binary.LittleEndian.PutUint16(data[42:], programSize)
	binary.LittleEndian.PutUint16(data[44:], 1)
	ph := data[programOffset : programOffset+programSize]
	binary.LittleEndian.PutUint32(ph[0:], 1)
	binary.LittleEndian.PutUint32(ph[4:], payloadOffset)
	binary.LittleEndian.PutUint32(ph[8:], 0x08048000)
	binary.LittleEndian.PutUint32(ph[12:], 0x08048000)
	binary.LittleEndian.PutUint32(ph[16:], 0x20)
	binary.LittleEndian.PutUint32(ph[20:], 0x3000)
	binary.LittleEndian.PutUint32(ph[24:], 7)
	binary.LittleEndian.PutUint32(ph[28:], 0x1000)
	copy(data[payloadOffset:], []byte("guest-code"))
	return data
}

func TestProcessLoadELFInitialState(t *testing.T) {
	process := NewProcess(77, nil)
	stack := coreloader.DefaultStackConfig()
	stack.Env = []string{"PATH=/bin"}
	loaded, err := process.LoadELF(bytes.NewReader(processELF()), int64(len(processELF())), "/bin/guest", 0, stack)
	if err != nil {
		t.Fatalf("LoadELF: %v", err)
	}
	if loaded.Space.Entry != 0x08048100 || process.CPU.EIP != uint32(loaded.Space.Entry) {
		t.Fatalf("entry = %#x/%#x", loaded.Space.Entry, process.CPU.EIP)
	}
	if process.CPU.Get(corecpu.ESP) != uint32(loaded.Stack.SP) || loaded.Stack.SP&0xf != 0 {
		t.Fatalf("stack pointer = %#x, layout = %#x", process.CPU.Get(corecpu.ESP), loaded.Stack.SP)
	}
	if process.Context.StartBrk != uint32(loaded.Space.Brk) || process.Context.Brk != process.Context.StartBrk {
		t.Fatalf("brk = %#x/%#x", process.Context.StartBrk, process.Context.Brk)
	}
	var word [4]byte
	if err := process.Memory.Read(loaded.Stack.SP, word[:]); err != nil {
		t.Fatalf("read argc: %v", err)
	}
	if binary.LittleEndian.Uint32(word[:]) != 1 {
		t.Fatalf("argc = %d", binary.LittleEndian.Uint32(word[:]))
	}
	if err := process.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func processExitELF() []byte {
	const (
		headerSize    = 52
		programSize   = 32
		programOffset = headerSize
		payloadOffset = 0x1000
		entry         = 0x08048000
	)
	code := []byte{
		0xb8, 0x01, 0x00, 0x00, 0x00, // mov eax, SYS_exit
		0xbb, 0x2a, 0x00, 0x00, 0x00, // mov ebx, 42
		0xcd, 0x80, // int 0x80
	}
	data := make([]byte, payloadOffset+len(code))
	copy(data[0:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4], data[5], data[6], data[7] = 1, 1, 1, 3
	binary.LittleEndian.PutUint16(data[16:], 2)
	binary.LittleEndian.PutUint16(data[18:], 3)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint32(data[24:], entry)
	binary.LittleEndian.PutUint32(data[28:], programOffset)
	binary.LittleEndian.PutUint16(data[40:], headerSize)
	binary.LittleEndian.PutUint16(data[42:], programSize)
	binary.LittleEndian.PutUint16(data[44:], 1)
	ph := data[programOffset : programOffset+programSize]
	binary.LittleEndian.PutUint32(ph[0:], 1)
	binary.LittleEndian.PutUint32(ph[4:], payloadOffset)
	binary.LittleEndian.PutUint32(ph[8:], entry)
	binary.LittleEndian.PutUint32(ph[12:], entry)
	binary.LittleEndian.PutUint32(ph[16:], uint32(len(code)))
	binary.LittleEndian.PutUint32(ph[20:], 0x1000)
	binary.LittleEndian.PutUint32(ph[24:], 5)
	binary.LittleEndian.PutUint32(ph[28:], 0x1000)
	copy(data[payloadOffset:], code)
	return data
}

func TestProcessLoadAndRunExitELF(t *testing.T) {
	process := NewProcess(88, nil)
	defer process.Close()
	image := processExitELF()
	if _, err := process.LoadELF(bytes.NewReader(image), int64(len(image)), "/bin/exit42", 0, coreloader.DefaultStackConfig()); err != nil {
		t.Fatalf("LoadELF: %v", err)
	}
	if err := process.Run(16); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code, exited := process.ExitCode(); !exited || code != 42 {
		t.Fatalf("exit state: exited=%v code=%d", exited, code)
	}
}

func processExecveELF() []byte {
	const (
		headerSize    = 52
		programSize   = 32
		programOffset = headerSize
		payloadOffset = 0x1000
		entry         = 0x08048000
		pathAddr      = entry + 0x100
		argvAddr      = entry + 0x120
		envAddr       = entry + 0x130
		arg0Addr      = entry + 0x140
		env0Addr      = entry + 0x150
	)
	code := []byte{
		0xb8, 0x0b, 0x00, 0x00, 0x00, // mov eax, SYS_execve
		0xbb, 0x00, 0x00, 0x00, 0x00, // mov ebx, filename
		0xb9, 0x00, 0x00, 0x00, 0x00, // mov ecx, argv
		0xba, 0x00, 0x00, 0x00, 0x00, // mov edx, envp
		0xcd, 0x80, // int 0x80
	}
	binary.LittleEndian.PutUint32(code[6:10], pathAddr)
	binary.LittleEndian.PutUint32(code[11:15], argvAddr)
	binary.LittleEndian.PutUint32(code[16:20], envAddr)
	data := make([]byte, payloadOffset+0x180)
	copy(data[0:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4], data[5], data[6], data[7] = 1, 1, 1, 3
	binary.LittleEndian.PutUint16(data[16:], 2)
	binary.LittleEndian.PutUint16(data[18:], 3)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint32(data[24:], entry)
	binary.LittleEndian.PutUint32(data[28:], programOffset)
	binary.LittleEndian.PutUint16(data[40:], headerSize)
	binary.LittleEndian.PutUint16(data[42:], programSize)
	binary.LittleEndian.PutUint16(data[44:], 1)
	ph := data[programOffset : programOffset+programSize]
	binary.LittleEndian.PutUint32(ph[0:], 1)
	binary.LittleEndian.PutUint32(ph[4:], payloadOffset)
	binary.LittleEndian.PutUint32(ph[8:], entry)
	binary.LittleEndian.PutUint32(ph[12:], entry)
	binary.LittleEndian.PutUint32(ph[16:], uint32(len(data)-payloadOffset))
	binary.LittleEndian.PutUint32(ph[20:], 0x1000)
	binary.LittleEndian.PutUint32(ph[24:], 7)
	binary.LittleEndian.PutUint32(ph[28:], 0x1000)
	copy(data[payloadOffset:], code)
	copy(data[payloadOffset+0x100:], []byte("/bin/second\x00"))
	binary.LittleEndian.PutUint32(data[payloadOffset+0x120:], arg0Addr)
	binary.LittleEndian.PutUint32(data[payloadOffset+0x124:], 0)
	binary.LittleEndian.PutUint32(data[payloadOffset+0x130:], env0Addr)
	binary.LittleEndian.PutUint32(data[payloadOffset+0x134:], 0)
	copy(data[payloadOffset+0x140:], []byte("second\x00"))
	copy(data[payloadOffset+0x150:], []byte("PATH=/bin\x00"))
	return data
}

func TestProcessExecveReloadsImage(t *testing.T) {
	db, err := storage.Open(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	fake, err := corefs.New(filepath.Join(t.TempDir(), "root"), db)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fake.Close() })
	if err := fake.Mkdir("/bin", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile("/bin/second", processExitELF(), 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}

	process := NewProcess(99, fake)
	defer process.Close()
	first := processExecveELF()
	if _, err := process.LoadELF(bytes.NewReader(first), int64(len(first)), "/bin/first", 0, coreloader.DefaultStackConfig()); err != nil {
		t.Fatalf("LoadELF: %v", err)
	}
	if err := process.Run(64); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if code, exited := process.ExitCode(); !exited || code != 42 {
		t.Fatalf("execve exit state: exited=%v code=%d", exited, code)
	}
}

func processDynamicMainELF(interpreter string) []byte {
	const (
		headerSize    = 52
		programSize   = 32
		programOffset = headerSize
		payloadOffset = 0x1000
		interpOffset  = 0x2000
		entry         = 0x08048000
	)
	code := []byte{0xcd, 0x80}
	data := make([]byte, interpOffset+len(interpreter)+1)
	copy(data[0:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4], data[5], data[6], data[7] = 1, 1, 1, 3
	binary.LittleEndian.PutUint16(data[16:], 2)
	binary.LittleEndian.PutUint16(data[18:], 3)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint32(data[24:], entry)
	binary.LittleEndian.PutUint32(data[28:], programOffset)
	binary.LittleEndian.PutUint16(data[40:], headerSize)
	binary.LittleEndian.PutUint16(data[42:], programSize)
	binary.LittleEndian.PutUint16(data[44:], 2)
	load := data[programOffset : programOffset+programSize]
	binary.LittleEndian.PutUint32(load[0:], 1)
	binary.LittleEndian.PutUint32(load[4:], payloadOffset)
	binary.LittleEndian.PutUint32(load[8:], entry)
	binary.LittleEndian.PutUint32(load[12:], entry)
	binary.LittleEndian.PutUint32(load[16:], uint32(len(code)))
	binary.LittleEndian.PutUint32(load[20:], 0x1000)
	binary.LittleEndian.PutUint32(load[24:], 5)
	binary.LittleEndian.PutUint32(load[28:], 0x1000)
	interp := data[programOffset+programSize : programOffset+2*programSize]
	binary.LittleEndian.PutUint32(interp[0:], 3)
	binary.LittleEndian.PutUint32(interp[4:], interpOffset)
	binary.LittleEndian.PutUint32(interp[16:], uint32(len(interpreter)+1))
	binary.LittleEndian.PutUint32(interp[20:], uint32(len(interpreter)+1))
	binary.LittleEndian.PutUint32(interp[24:], 4)
	binary.LittleEndian.PutUint32(interp[28:], 1)
	copy(data[payloadOffset:], code)
	copy(data[interpOffset:], interpreter+"\x00")
	return data
}

func processInterpreterELF() []byte {
	const (
		headerSize    = 52
		programSize   = 32
		programOffset = headerSize
		payloadOffset = 0x1000
		entry         = 0x1000
	)
	code := []byte{
		0xb8, 0x01, 0x00, 0x00, 0x00,
		0xbb, 0x2a, 0x00, 0x00, 0x00,
		0xcd, 0x80,
	}
	data := make([]byte, payloadOffset+len(code))
	copy(data[0:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4], data[5], data[6], data[7] = 1, 1, 1, 3
	binary.LittleEndian.PutUint16(data[16:], 3)
	binary.LittleEndian.PutUint16(data[18:], 3)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint32(data[24:], entry)
	binary.LittleEndian.PutUint32(data[28:], programOffset)
	binary.LittleEndian.PutUint16(data[40:], headerSize)
	binary.LittleEndian.PutUint16(data[42:], programSize)
	binary.LittleEndian.PutUint16(data[44:], 1)
	ph := data[programOffset : programOffset+programSize]
	binary.LittleEndian.PutUint32(ph[0:], 1)
	binary.LittleEndian.PutUint32(ph[4:], payloadOffset)
	binary.LittleEndian.PutUint32(ph[8:], payloadOffset)
	binary.LittleEndian.PutUint32(ph[12:], payloadOffset)
	binary.LittleEndian.PutUint32(ph[16:], uint32(len(code)))
	binary.LittleEndian.PutUint32(ph[20:], 0x2000)
	binary.LittleEndian.PutUint32(ph[24:], 5)
	binary.LittleEndian.PutUint32(ph[28:], 0x1000)
	copy(data[payloadOffset:], code)
	return data
}

func TestProcessLoadsPTInterpAndStartsInterpreter(t *testing.T) {
	db, err := storage.Open(t.Context(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	fake, err := corefs.New(filepath.Join(t.TempDir(), "root"), db)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fake.Close() })
	if err := fake.Mkdir("/lib", 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := fake.WriteFile("/lib/ld.so.1", processInterpreterELF(), 0o755, 0, 0); err != nil {
		t.Fatal(err)
	}
	process := NewProcess(101, fake)
	defer process.Close()
	main := processDynamicMainELF("/lib/ld.so.1")
	loaded, err := process.LoadELF(bytes.NewReader(main), int64(len(main)), "/bin/dynamic", 0, coreloader.DefaultStackConfig())
	if err != nil {
		t.Fatalf("LoadELF dynamic: %v", err)
	}
	if loaded.Interpreter == nil || loaded.InterpreterSpace == nil {
		t.Fatal("interpreter was not loaded")
	}
	if loaded.InterpreterSpace.Bias != 0x3ffff000 {
		t.Fatalf("interpreter bias = %#x", loaded.InterpreterSpace.Bias)
	}
	if process.CPU.EIP != uint32(loaded.InterpreterSpace.Entry) {
		t.Fatalf("EIP = %#x, interpreter entry = %#x", process.CPU.EIP, loaded.InterpreterSpace.Entry)
	}
	var auxBase, auxEntry uint32
	for cursor := uint32(loaded.Stack.Auxv); ; cursor += 8 {
		var pair [8]byte
		if err := process.Memory.Read(corecpu.Address(cursor), pair[:]); err != nil {
			t.Fatalf("read auxv: %v", err)
		}
		typ := binary.LittleEndian.Uint32(pair[:4])
		value := binary.LittleEndian.Uint32(pair[4:])
		if typ == coreloader.AT_NULL {
			break
		}
		switch typ {
		case coreloader.AT_BASE:
			auxBase = value
		case coreloader.AT_ENTRY:
			auxEntry = value
		}
	}
	if auxBase != loaded.InterpreterSpace.Bias || auxEntry != uint32(loaded.Space.Entry) {
		t.Fatalf("dynamic auxv: AT_BASE=%#x AT_ENTRY=%#x", auxBase, auxEntry)
	}
	if err := process.Run(32); err != nil {
		t.Fatalf("Run dynamic: %v", err)
	}
	if code, exited := process.ExitCode(); !exited || code != 42 {
		t.Fatalf("dynamic exit state: exited=%v code=%d", exited, code)
	}
}
