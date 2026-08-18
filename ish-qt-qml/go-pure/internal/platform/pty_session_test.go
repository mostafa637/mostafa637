package platform

import (
	"context"
	"testing"
	"time"
)

func TestPTYSessionRoundTrip(t *testing.T) {
	s := NewPTYSession("/bin/sh")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Start(ctx, 80, 24); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Close()
	if err := s.Write([]byte("printf 'ISH_GO_PTY_PASS\\n'\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	deadline := time.NewTimer(4 * time.Second)
	defer deadline.Stop()
	var got []byte
	for {
		select {
		case chunk, ok := <-s.Output():
			if !ok {
				t.Fatalf("PTY output closed before marker: %q", got)
			}
			got = append(got, chunk...)
			for i := 0; i+len("ISH_GO_PTY_PASS") <= len(got); i++ {
				if string(got[i:i+len("ISH_GO_PTY_PASS")]) == "ISH_GO_PTY_PASS" {
					return
				}
			}
		case <-deadline.C:
			t.Fatalf("timeout waiting for marker; output=%q", got)
		}
	}
}
