package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCoreSessionGuestELF64WasmExit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "bin", "exit64")
	if err := os.WriteFile(path, guestExitELF64(), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Rootfs: root, Shell: "/bin/exit64", UseGuest: true, UseWasm: true, Bootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Start(context.Background(), 80, 24); err != nil {
		t.Fatalf("Wasm ELF64 Start: %v", err)
	}
	select {
	case _, ok := <-s.Output():
		if ok {
			for range s.Output() {
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for Wasm ELF64 guest exit")
	}
	if code, exited := s.Kernel().ExitCode(); !exited || code != 42 {
		t.Fatalf("Wasm ELF64 guest exit: exited=%v code=%d", exited, code)
	}
}
