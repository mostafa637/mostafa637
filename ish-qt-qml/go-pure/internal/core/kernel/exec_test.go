package kernel

import (
	"bytes"
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreloader "github.com/mostafa637/mostafa637/go-pure/internal/core/loader"
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
