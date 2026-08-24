package session

import (
	"context"
	"fmt"
	"sync"
)

type Factory func() Session

type Manager struct {
	mu      sync.Mutex
	factory Factory
	slots   map[int]Session
	nextID  int
}

func NewManager(factory Factory) *Manager {
	return &Manager{factory: factory, slots: make(map[int]Session)}
}

func (m *Manager) Create(ctx context.Context, cols, rows int) (int, Session, error) {
	if m == nil || m.factory == nil {
		return 0, nil, fmt.Errorf("session: nil factory")
	}
	s := m.factory()
	if s == nil {
		return 0, nil, fmt.Errorf("session: factory returned nil")
	}
	if err := s.Start(ctx, cols, rows); err != nil {
		return 0, nil, err
	}
	m.mu.Lock()
	id := m.nextID
	m.nextID++
	m.slots[id] = s
	m.mu.Unlock()
	return id, s, nil
}

func (m *Manager) Get(id int) (Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.slots[id]
	return s, ok
}

func (m *Manager) Close(id int) error {
	m.mu.Lock()
	s, ok := m.slots[id]
	if ok {
		delete(m.slots, id)
	}
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session: unknown id %d", id)
	}
	return s.Close()
}

func (m *Manager) Restart(ctx context.Context, id, cols, rows int) (Session, error) {
	if err := m.Close(id); err != nil {
		return nil, err
	}
	m.mu.Lock()
	factory := m.factory
	m.mu.Unlock()
	if factory == nil {
		return nil, fmt.Errorf("session: nil factory")
	}
	s := factory()
	if err := s.Start(ctx, cols, rows); err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.slots[id] = s
	m.mu.Unlock()
	return s, nil
}

func (m *Manager) CloseAll() error {
	m.mu.Lock()
	all := m.slots
	m.slots = make(map[int]Session)
	m.mu.Unlock()
	for _, s := range all {
		if err := s.Close(); err != nil {
			return err
		}
	}
	return nil
}
