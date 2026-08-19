package session

import (
	"context"
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
