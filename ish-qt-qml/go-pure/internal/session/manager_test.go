package session

import (
	"context"
	"testing"
)

type fakeSession struct{ started, closed bool }

func (s *fakeSession) Start(context.Context, int, int) error { s.started = true; return nil }
func (s *fakeSession) Output() <-chan []byte                 { return make(chan []byte) }
func (s *fakeSession) Write([]byte) error                    { return nil }
func (s *fakeSession) Resize(int, int) error                 { return nil }
func (s *fakeSession) Close() error                          { s.closed = true; return nil }

func TestManagerLifecycle(t *testing.T) {
	m := NewManager(func() Session { return &fakeSession{} })
	id, first, err := m.Create(context.Background(), 80, 24)
	if err != nil || id != 0 {
		t.Fatalf("Create = %d, %v", id, err)
	}
	if _, ok := m.Get(id); !ok || first == nil {
		t.Fatal("created session missing")
	}
	second, err := m.Restart(context.Background(), id, 100, 30)
	if err != nil || second == first {
		t.Fatalf("Restart = %v", err)
	}
	if err := m.CloseAll(); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Get(id); ok {
		t.Fatal("session remains after CloseAll")
	}
}
