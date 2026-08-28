//go:build ishcore && cgo

package session

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"
)

func TestStartAlpineRunsShell(t *testing.T) {
	root := os.Getenv("ISH_TEST_ROOTFS")
	if root == "" {
		t.Skip("ISH_TEST_ROOTFS is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s, err := StartAlpine(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Resize(80, 24); err != nil {
		t.Fatal(err)
	}
	if err := s.Write([]byte("printf 'GO_ALPINE_CORE_OK\\n'; cat /etc/alpine-release; exit\n")); err != nil {
		t.Fatal(err)
	}

	var got []byte
	deadline := time.After(10 * time.Second)
	for !bytes.Contains(got, []byte("GO_ALPINE_CORE_OK")) || !bytes.Contains(got, []byte("3.19.0")) {
		select {
		case chunk, ok := <-s.Output():
			if !ok {
				t.Fatalf("session closed before expected Alpine output: %q", got)
			}
			got = append(got, chunk...)
		case <-deadline:
			t.Fatalf("timed out waiting for Alpine output: %q", got)
		}
	}
}
