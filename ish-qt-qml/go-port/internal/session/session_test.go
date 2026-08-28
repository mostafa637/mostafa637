package session

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestStartWriteAndRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, err := Start(ctx, "/bin/sh", "-i")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Resize(80, 24); err != nil {
		t.Fatal(err)
	}
	if err := s.Write([]byte("printf 'go-port-ok\\n'\nexit\n")); err != nil {
		t.Fatal(err)
	}

	var got []byte
	deadline := time.After(3 * time.Second)
	for !bytes.Contains(got, []byte("go-port-ok")) {
		select {
		case chunk, ok := <-s.Output():
			if !ok {
				t.Fatalf("session closed before output: %q", got)
			}
			got = append(got, chunk...)
		case <-deadline:
			t.Fatalf("timed out waiting for output: %q", got)
		}
	}
}
