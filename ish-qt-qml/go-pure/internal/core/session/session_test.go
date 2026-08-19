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
