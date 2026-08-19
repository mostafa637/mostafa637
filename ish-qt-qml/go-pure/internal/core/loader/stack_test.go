package loader

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func readWord(t *testing.T, memory *corecpu.Memory, address corecpu.Address) uint32 {
	t.Helper()
	var data [4]byte
	if err := memory.Read(address, data[:]); err != nil {
		t.Fatalf("read %#x: %v", address, err)
	}
	return binary.LittleEndian.Uint32(data[:])
}

func readCString(t *testing.T, memory *corecpu.Memory, address corecpu.Address) string {
	t.Helper()
	data := make([]byte, 0, 32)
	for index := 0; index < 128; index++ {
		var one [1]byte
		if err := memory.Read(address+corecpu.Address(index), one[:]); err != nil {
			t.Fatalf("read string %#x: %v", address+corecpu.Address(index), err)
		}
		if one[0] == 0 {
			return string(data)
		}
		data = append(data, one[0])
	}
	t.Fatalf("unterminated string at %#x", address)
	return ""
}

func TestBuildStack(t *testing.T) {
	memory := corecpu.NewMemory()
	config := DefaultStackConfig()
	config.Argv = []string{"/bin/sh", "-c", "echo hi"}
	config.Env = []string{"PATH=/bin", "HOME=/"}
	config.ExecFilename = "/bin/sh"
	config.Auxv = []AuxEntry{
		{Type: AT_PAGESZ, Value: corecpu.PageSize},
		{Type: AT_RANDOM},
		{Type: AT_PLATFORM},
		{Type: AT_EXECFN},
	}
	layout, err := BuildStack(memory, config)
	if err != nil {
		t.Fatalf("BuildStack: %v", err)
	}
	if uint32(layout.SP)&0xf != 0 {
		t.Fatalf("SP %#x is not 16-byte aligned", layout.SP)
	}
	if readWord(t, memory, layout.SP) != 3 {
		t.Fatalf("argc = %d", readWord(t, memory, layout.SP))
	}
	argv := layout.SP + 4
	if got := readCString(t, memory, corecpu.Address(readWord(t, memory, argv))); got != "/bin/sh" {
		t.Fatalf("argv[0] = %q", got)
	}
	if got := readCString(t, memory, corecpu.Address(readWord(t, memory, argv+4))); got != "-c" {
		t.Fatalf("argv[1] = %q", got)
	}
	if readWord(t, memory, argv+12) != 0 {
		t.Fatalf("argv terminator = %#x", readWord(t, memory, argv+12))
	}
	envp := argv + 16
	if got := readCString(t, memory, corecpu.Address(readWord(t, memory, envp))); got != "PATH=/bin" {
		t.Fatalf("envp[0] = %q", got)
	}
	if got := readCString(t, memory, corecpu.Address(readWord(t, memory, envp+4))); got != "HOME=/" {
		t.Fatalf("envp[1] = %q", got)
	}
	if readWord(t, memory, envp+8) != 0 {
		t.Fatalf("envp terminator = %#x", readWord(t, memory, envp+8))
	}
	aux := envp + 12
	seenRandom, seenPlatform, seenExec := false, false, false
	for index := 0; index < 8; index++ {
		typ := readWord(t, memory, aux)
		value := readWord(t, memory, aux+4)
		aux += 8
		switch typ {
		case AT_NULL:
			index = 8
		case AT_RANDOM:
			seenRandom = value == uint32(layout.Random)
		case AT_PLATFORM:
			seenPlatform = value == uint32(layout.Platform)
		case AT_EXECFN:
			seenExec = value != 0 && readCString(t, memory, corecpu.Address(value)) == "/bin/sh"
		}
	}
	if !seenRandom || !seenPlatform || !seenExec {
		t.Fatalf("auxv random=%v platform=%v exec=%v", seenRandom, seenPlatform, seenExec)
	}
}

func TestBuildStackGrowsDown(t *testing.T) {
	memory := corecpu.NewMemory()
	config := DefaultStackConfig()
	config.Pages = 1
	config.Argv = []string{"guest"}
	config.Env = []string{"LONG=" + string(make([]byte, 5000))}
	if _, err := BuildStack(memory, config); err != nil {
		t.Fatalf("BuildStack with grows-down stack: %v", err)
	}
	startPage := corecpu.Page(config.Top>>corecpu.PageBits) - 1
	if _, ok := memory.Page(startPage - 1); !ok {
		t.Fatalf("grows-down did not allocate page %#x", startPage-1)
	}
}
