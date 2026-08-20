package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCoreSessionGuestELF64WasmEcho(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "bin", "echo64")
	if err := os.WriteFile(path, guestEchoELF64(), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Rootfs: root, Shell: "/bin/echo64", UseGuest: true, UseWasm: true, Bootstrap: true})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Start(context.Background(), 80, 24); err != nil {
		t.Fatal(err)
	}
	if err := s.Write([]byte("pong")); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	deadline := time.After(3 * time.Second)
	for output.String() != "pong" {
		select {
		case chunk := <-s.Output():
			output.Write(chunk)
		case <-deadline:
			t.Fatalf("Wasm echo output: %q", output.String())
		}
	}
}
