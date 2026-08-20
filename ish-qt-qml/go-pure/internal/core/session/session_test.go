package session

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	transport "github.com/mostafa637/mostafa637/go-pure/internal/session"
)

func TestCoreSessionBootstrapsRootfs(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "etc", "profile"), []byte("export PATH=/bin"), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := New(Config{Rootfs: root, Bootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	info, err := s.FS().Stat("/etc/profile")
	if err != nil {
		t.Fatal(err)
	}
	if info.Inode == 0 || info.Mode.Mode != 0o100644 || info.Mode.UID != 0 || info.Mode.GID != 0 {
		t.Fatalf("bootstrapped info = %#v", info)
	}
}

func TestCoreSessionPTYLifecycle(t *testing.T) {
	s, err := New(Config{Rootfs: t.TempDir(), Shell: "/bin/sh"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Write([]byte("echo before\n")); !errors.Is(err, transport.ErrClosed) {
		t.Fatalf("write before start = %v, want ErrClosed", err)
	}
	if err := s.Start(context.Background(), 80, 24); err != nil {
		t.Fatal(err)
	}
	if err := s.Write([]byte("printf 'core-ready\\n'\n")); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(3 * time.Second)
	var output strings.Builder
	for {
		select {
		case chunk, ok := <-s.Output():
			if !ok {
				t.Fatalf("session output closed before marker: %q", output.String())
			}
			output.Write(chunk)
			if strings.Contains(output.String(), "core-ready") {
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
				if err := s.Write([]byte("echo after\n")); !errors.Is(err, transport.ErrClosed) {
					t.Fatalf("write after close = %v, want ErrClosed", err)
				}
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for output, got %q", output.String())
		}
	}
}

func guestExitELF() []byte {
	const (
		headerSize    = 52
		programSize   = 32
		programOffset = headerSize
		payloadOffset = 0x1000
		entry         = 0x08048000
	)
	code := []byte{0xb8, 0x01, 0, 0, 0, 0xbb, 0x2a, 0, 0, 0, 0xcd, 0x80}
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

func TestCoreSessionGuestELFExit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "exit42"), guestExitELF(), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Rootfs: root, Shell: "/bin/exit42", UseGuest: true, Bootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Start(context.Background(), 80, 24); err != nil {
		t.Fatalf("guest Start: %v", err)
	}
	select {
	case _, ok := <-s.Output():
		if ok {
			for range s.Output() {
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for guest exit")
	}
	if code, exited := s.Kernel().ExitCode(); !exited || code != 42 {
		t.Fatalf("guest exit: exited=%v code=%d", exited, code)
	}
}

func guestEchoELF() []byte {
	const (
		headerSize    = 52
		programSize   = 32
		programOffset = headerSize
		payloadOffset = 0x1000
		entry         = 0x08048000
		buffer        = 0x08048100
	)
	code := []byte{
		0xb8, 0x03, 0, 0, 0, // mov eax, SYS_read
		0xbb, 0, 0, 0, 0, // mov ebx, 0
		0xb9, 0x00, 0x81, 0x04, 0x08, // mov ecx, buffer
		0xba, 0x04, 0, 0, 0, // mov edx, 4
		0xcd, 0x80,
		0xb8, 0x04, 0, 0, 0, // mov eax, SYS_write
		0xbb, 0x01, 0, 0, 0, // mov ebx, 1
		0xb9, 0x00, 0x81, 0x04, 0x08, // mov ecx, buffer
		0xba, 0x04, 0, 0, 0, // mov edx, 4
		0xcd, 0x80,
		0xb8, 0x01, 0, 0, 0, // mov eax, SYS_exit
		0xbb, 0, 0, 0, 0, // mov ebx, 0
		0xcd, 0x80,
	}
	data := make([]byte, payloadOffset+0x104)
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
	binary.LittleEndian.PutUint32(ph[20:], 0x2000)
	binary.LittleEndian.PutUint32(ph[24:], 7)
	binary.LittleEndian.PutUint32(ph[28:], 0x1000)
	copy(data[payloadOffset:], code)
	_ = buffer
	return data
}

func TestCoreSessionGuestStdinStdout(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "echo4"), guestEchoELF(), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Rootfs: root, Shell: "/bin/echo4", UseGuest: true, Bootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Start(context.Background(), 80, 24); err != nil {
		t.Fatalf("guest Start: %v", err)
	}
	if err := s.Resize(100, 30); err != nil {
		t.Fatalf("guest Resize: %v", err)
	}
	if got := s.Kernel().Context.WinCols; got != 100 {
		t.Fatalf("guest columns = %d", got)
	}
	if got := s.Kernel().Context.WinRows; got != 30 {
		t.Fatalf("guest rows = %d", got)
	}
	if err := s.Write([]byte("pong")); err != nil {
		t.Fatalf("guest Write: %v", err)
	}
	var output strings.Builder
	deadline := time.After(2 * time.Second)
	for {
		select {
		case chunk, ok := <-s.Output():
			if !ok {
				t.Fatalf("guest output closed before echo: %q", output.String())
			}
			output.Write(chunk)
			if output.String() == "pong" {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for guest echo: %q", output.String())
		}
	}
}

func guestExitELF64() []byte {
	const (
		headerSize    = 64
		programSize   = 56
		programOffset = headerSize
		payloadOffset = 0x1000
		entry         = 0x00400000
	)
	code := []byte{
		0xb8, 0x3c, 0x00, 0x00, 0x00, // mov eax, SYS_exit
		0xbf, 0x2a, 0x00, 0x00, 0x00, // mov edi, 42
		0x0f, 0x05, // syscall
	}
	data := make([]byte, payloadOffset+len(code))
	copy(data[0:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4], data[5], data[6], data[7] = 2, 1, 1, 0
	binary.LittleEndian.PutUint16(data[16:], 2)
	binary.LittleEndian.PutUint16(data[18:], 62)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint64(data[24:], entry)
	binary.LittleEndian.PutUint64(data[32:], programOffset)
	binary.LittleEndian.PutUint16(data[52:], headerSize)
	binary.LittleEndian.PutUint16(data[54:], programSize)
	binary.LittleEndian.PutUint16(data[56:], 1)
	ph := data[programOffset : programOffset+programSize]
	binary.LittleEndian.PutUint32(ph[0:], 1)
	binary.LittleEndian.PutUint32(ph[4:], 5)
	binary.LittleEndian.PutUint64(ph[8:], payloadOffset)
	binary.LittleEndian.PutUint64(ph[16:], entry)
	binary.LittleEndian.PutUint64(ph[24:], entry)
	binary.LittleEndian.PutUint64(ph[32:], uint64(len(code)))
	binary.LittleEndian.PutUint64(ph[40:], 0x1000)
	binary.LittleEndian.PutUint64(ph[48:], 0x1000)
	copy(data[payloadOffset:], code)
	return data
}

func TestCoreSessionGuestELF64Exit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "exit64"), guestExitELF64(), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Rootfs: root, Shell: "/bin/exit64", UseGuest: true, Bootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Start(context.Background(), 80, 24); err != nil {
		t.Fatalf("ELF64 Start: %v", err)
	}
	select {
	case _, ok := <-s.Output():
		if ok {
			for range s.Output() {
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ELF64 guest exit")
	}
	if code, exited := s.Kernel().ExitCode(); !exited || code != 42 {
		t.Fatalf("ELF64 guest exit: exited=%v code=%d", exited, code)
	}
}

func guestEchoELF64() []byte {
	const (
		headerSize    = 64
		programSize   = 56
		programOffset = headerSize
		payloadOffset = 0x1000
		entry         = 0x00400000
		buffer        = 0x00401000
	)
	code := []byte{
		0xb8, 0x00, 0x00, 0x00, 0x00, // mov eax, SYS_read
		0xbf, 0x00, 0x00, 0x00, 0x00, // mov edi, 0
		0xbe, byte(buffer & 0xff), byte((buffer >> 8) & 0xff), byte((buffer >> 16) & 0xff), byte((buffer >> 24) & 0xff), // mov esi, buffer
		0xba, 0x04, 0x00, 0x00, 0x00, // mov edx, 4
		0x0f, 0x05, // syscall
		0xb8, 0x01, 0x00, 0x00, 0x00, // mov eax, SYS_write
		0xbf, 0x01, 0x00, 0x00, 0x00, // mov edi, 1
		0xbe, byte(buffer & 0xff), byte((buffer >> 8) & 0xff), byte((buffer >> 16) & 0xff), byte((buffer >> 24) & 0xff), // mov esi, buffer
		0xba, 0x04, 0x00, 0x00, 0x00, // mov edx, 4
		0x0f, 0x05, // syscall
		0xb8, 0x3c, 0x00, 0x00, 0x00, // mov eax, SYS_exit
		0x31, 0xff, // xor edi, edi
		0x0f, 0x05, // syscall
	}
	data := make([]byte, payloadOffset+len(code))
	copy(data[0:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4], data[5], data[6], data[7] = 2, 1, 1, 0
	binary.LittleEndian.PutUint16(data[16:], 2)
	binary.LittleEndian.PutUint16(data[18:], 62)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint64(data[24:], entry)
	binary.LittleEndian.PutUint64(data[32:], programOffset)
	binary.LittleEndian.PutUint16(data[52:], headerSize)
	binary.LittleEndian.PutUint16(data[54:], programSize)
	binary.LittleEndian.PutUint16(data[56:], 1)
	ph := data[programOffset : programOffset+programSize]
	binary.LittleEndian.PutUint32(ph[0:], 1)
	binary.LittleEndian.PutUint32(ph[4:], 7)
	binary.LittleEndian.PutUint64(ph[8:], payloadOffset)
	binary.LittleEndian.PutUint64(ph[16:], entry)
	binary.LittleEndian.PutUint64(ph[24:], entry)
	binary.LittleEndian.PutUint64(ph[32:], uint64(len(code)))
	binary.LittleEndian.PutUint64(ph[40:], 0x2000)
	binary.LittleEndian.PutUint64(ph[48:], 0x1000)
	copy(data[payloadOffset:], code)
	return data
}

func TestCoreSessionGuestELF64StdinStdout(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "echo64"), guestEchoELF64(), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Rootfs: root, Shell: "/bin/echo64", UseGuest: true, Bootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Start(context.Background(), 80, 24); err != nil {
		t.Fatalf("ELF64 echo Start: %v", err)
	}
	if err := s.Write([]byte("pong")); err != nil {
		t.Fatalf("ELF64 echo Write: %v", err)
	}
	var output strings.Builder
	deadline := time.After(2 * time.Second)
	for {
		select {
		case chunk, ok := <-s.Output():
			if !ok {
				t.Fatalf("ELF64 output closed before echo: %q", output.String())
			}
			output.Write(chunk)
			if output.String() == "pong" {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for ELF64 echo: %q", output.String())
		}
	}
}

func dynamicGuestELF64(interp string, needed []string, code []byte, withTLS bool) []byte {
	const (
		headerSize    = 64
		programSize   = 56
		programOffset = headerSize
		loadVaddr     = 0x400000
		dynamicOffset = 0x1000
		stringOffset  = 0x1200
		interpOffset  = 0x1300
		tlsOffset     = 0x1400
		codeOffset    = 0x1500
		fileSize      = 0x1600
	)
	phnum := 2
	if interp != "" {
		phnum++
	}
	if withTLS {
		phnum++
	}
	data := make([]byte, fileSize)
	copy(data[:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4], data[5], data[6], data[7] = 2, 1, 1, 0
	binary.LittleEndian.PutUint16(data[16:], 3)
	binary.LittleEndian.PutUint16(data[18:], 62)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint64(data[24:], loadVaddr+codeOffset)
	binary.LittleEndian.PutUint64(data[32:], programOffset)
	binary.LittleEndian.PutUint16(data[52:], headerSize)
	binary.LittleEndian.PutUint16(data[54:], programSize)
	binary.LittleEndian.PutUint16(data[56:], uint16(phnum))
	putProgram := func(index int, typ uint32, flags uint32, offset, vaddr, filesz, memsz, align uint64) {
		ph := data[programOffset+index*programSize : programOffset+(index+1)*programSize]
		binary.LittleEndian.PutUint32(ph[0:], typ)
		binary.LittleEndian.PutUint32(ph[4:], flags)
		binary.LittleEndian.PutUint64(ph[8:], offset)
		binary.LittleEndian.PutUint64(ph[16:], vaddr)
		binary.LittleEndian.PutUint64(ph[32:], filesz)
		binary.LittleEndian.PutUint64(ph[40:], memsz)
		binary.LittleEndian.PutUint64(ph[48:], align)
	}
	putProgram(0, 1, 7, 0, loadVaddr, fileSize, fileSize+0x1000, 0x1000)
	stringsData := []byte{0}
	neededOffsets := make([]uint64, 0, len(needed))
	for _, name := range needed {
		neededOffsets = append(neededOffsets, uint64(len(stringsData)))
		stringsData = append(stringsData, []byte(name)...)
		stringsData = append(stringsData, 0)
	}
	copy(data[stringOffset:], stringsData)
	if interp != "" {
		interpData := append([]byte(interp), 0)
		copy(data[interpOffset:], interpData)
		putProgram(1, 3, 4, interpOffset, loadVaddr+interpOffset, uint64(len(interpData)), uint64(len(interpData)), 1)
	}
	if withTLS {
		tlsData := []byte{0x11, 0x22, 0x33, 0x44}
		copy(data[tlsOffset:], tlsData)
		index := 1
		if interp != "" {
			index++
		}
		putProgram(index, 7, 4, tlsOffset, loadVaddr+tlsOffset, uint64(len(tlsData)), 8, 8)
	}
	dynamicIndex := 1
	if interp != "" {
		dynamicIndex++
	}
	if withTLS {
		dynamicIndex++
	}
	dynamic := data[dynamicOffset : dynamicOffset+16*(3+len(needed))]
	putDynamic := func(index int, tag, value uint64) {
		binary.LittleEndian.PutUint64(dynamic[index*16:], tag)
		binary.LittleEndian.PutUint64(dynamic[index*16+8:], value)
	}
	putDynamic(0, 5, loadVaddr+stringOffset)
	putDynamic(1, 10, uint64(len(stringsData)))
	for index, offset := range neededOffsets {
		putDynamic(2+index, 1, offset)
	}
	putDynamic(2+len(needed), 0, 0)
	putProgram(dynamicIndex, 2, 4, dynamicOffset, loadVaddr+dynamicOffset, uint64(len(dynamic)), uint64(len(dynamic)), 8)
	copy(data[codeOffset:], code)
	return data
}

func TestLoadGuestImage64DynamicInterpreterNeededAndTLS(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lib64"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib64", "ld-mini.so"), guestExitELF64(), 0o755); err != nil {
		t.Fatal(err)
	}
	mainData := dynamicGuestELF64("/lib64/ld-mini.so", []string{"libneeded.so"}, nil, true)
	if err := os.WriteFile(filepath.Join(root, "bin", "dynamic64"), mainData, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "libneeded.so"), dynamicGuestELF64("", nil, nil, false), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Rootfs: root, Bootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	memory := corecpu.NewMemory64()
	loaded, layout, err := loadGuestImage64(s.FS(), "/bin/dynamic64", mainData, []string{"/bin/dynamic64"}, nil, memory)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Interpreter == nil || loaded.Interpreter.Space == nil {
		t.Fatal("dynamic image has no mapped interpreter")
	}
	if loaded.Entry != loaded.Interpreter.Space.Entry {
		t.Fatalf("entry=%#x interpreter=%#x", loaded.Entry, loaded.Interpreter.Space.Entry)
	}
	if loaded.Main.Image.Dynamic == nil || len(loaded.Main.Image.Dynamic.Needed) != 1 || loaded.Main.Image.Dynamic.Needed[0] != "libneeded.so" {
		t.Fatalf("main dynamic metadata=%+v", loaded.Main.Image.Dynamic)
	}
	if len(loaded.Registry.Objects()) != 3 {
		t.Fatalf("loaded objects=%d, want main/interpreter/needed", len(loaded.Registry.Objects()))
	}
	if loaded.TLS == nil || len(loaded.TLS.Modules) != 1 || loaded.TLS.ThreadPointer == 0 {
		t.Fatalf("TLS layout=%+v", loaded.TLS)
	}
	state := corecpu.NewMachineState64(memory)
	attachTLS64(state, loaded)
	if state.FSBase != uint64(loaded.TLS.ThreadPointer) || state.TLS != uint64(loaded.TLS.ThreadPointer) {
		t.Fatalf("TLS state FSBase=%#x TLS=%#x want=%#x", state.FSBase, state.TLS, loaded.TLS.ThreadPointer)
	}
	if layout.SP == 0 {
		t.Fatal("dynamic stack was not built")
	}
	auxv := dynamicAuxv64(loaded)
	var atBase, atEntry uint64
	for _, entry := range auxv {
		switch entry.Type {
		case 7:
			atBase = entry.Value
		case 9:
			atEntry = entry.Value
		}
	}
	if atBase != uint64(loaded.Interpreter.Space.Bias) || atEntry != uint64(loaded.Main.Space.Entry) {
		t.Fatalf("auxv AT_BASE=%#x AT_ENTRY=%#x", atBase, atEntry)
	}
}

func TestCoreSessionGuestELF64DynamicStartsInterpreter(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	mainData := dynamicGuestELF64("/lib64/ld-mini.so", []string{"libneeded.so"}, nil, true)
	if err := os.WriteFile(filepath.Join(root, "bin", "dynamic64"), mainData, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "libneeded.so"), dynamicGuestELF64("", nil, nil, false), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "lib64"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib64", "ld-mini.so"), guestExitELF64(), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Rootfs: root, Shell: "/bin/dynamic64", UseGuest: true, Bootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Start(context.Background(), 80, 24); err != nil {
		t.Fatalf("dynamic ELF64 Start: %v", err)
	}
	select {
	case _, ok := <-s.Output():
		if ok {
			for range s.Output() {
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for dynamic ELF64 interpreter exit")
	}
	if code, exited := s.Kernel().ExitCode(); !exited || code != 42 {
		t.Fatalf("dynamic ELF64 exit: exited=%v code=%d", exited, code)
	}
}

func TestDynamicGuestELF64RejectsMissingInterpreter(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	mainData := dynamicGuestELF64("/lib64/missing.so", nil, nil, false)
	if err := os.WriteFile(filepath.Join(root, "bin", "dynamic64"), mainData, 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Rootfs: root, Shell: "/bin/dynamic64", UseGuest: true, Bootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Start(context.Background(), 80, 24); err == nil {
		t.Fatal("missing interpreter unexpectedly succeeded")
	}
}
