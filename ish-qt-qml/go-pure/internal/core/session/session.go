package session

import (
	"context"
	"fmt"
	"sync"

	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
	platform "github.com/mostafa637/mostafa637/go-pure/internal/platform"
	transport "github.com/mostafa637/mostafa637/go-pure/internal/session"
)

// Config describes the Pure Go rootfs/session boundary. The current Linux
// stage runs the configured host shell through PTY while fakefs owns rootfs
// metadata. A future CPU/kernel backend can replace the PTY field without
// changing the public transport contract.
type Config struct {
	Rootfs    string
	MetaDB    string
	Shell     string
	UID       uint32
	GID       uint32
	Bootstrap bool
}

func (c Config) withDefaults() Config {
	if c.Shell == "" {
		c.Shell = "/bin/sh"
	}
	if c.UID == 0 && c.GID == 0 {
		// iSH's initial root account is uid/gid 0; keep this explicit rather than
		// inheriting the host user while importing rootfs metadata.
	}
	return c
}

// CoreSession is the Go-owned lifecycle for fakefs plus the current host PTY
// execution bridge. It is deliberately independent from Gio and gritty.
type CoreSession struct {
	mu      sync.Mutex
	fs      *corefs.FS
	pty     *platform.PTYSession
	closed  bool
	started bool
}

func New(cfg Config) (*CoreSession, error) {
	cfg = cfg.withDefaults()
	if cfg.Rootfs == "" {
		return nil, fmt.Errorf("core session: Rootfs is required")
	}
	fake, err := corefs.Open(cfg.Rootfs, cfg.MetaDB)
	if err != nil {
		return nil, err
	}
	if cfg.Bootstrap {
		if err := fake.BootstrapMetadata(cfg.UID, cfg.GID); err != nil {
			_ = fake.Close()
			return nil, err
		}
	}
	return &CoreSession{fs: fake, pty: platform.NewPTYSession(cfg.Shell)}, nil
}

func NewWithFS(fake *corefs.FS, shell string) (*CoreSession, error) {
	if fake == nil {
		return nil, fmt.Errorf("core session: nil fakefs")
	}
	return &CoreSession{fs: fake, pty: platform.NewPTYSession(shell)}, nil
}

func (s *CoreSession) FS() *corefs.FS {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fs
}

func (s *CoreSession) Start(ctx context.Context, cols, rows int) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return transport.ErrClosed
	}
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("core session: already started")
	}
	pty := s.pty
	s.mu.Unlock()
	if err := pty.Start(ctx, cols, rows); err != nil {
		return err
	}
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()
	return nil
}

func (s *CoreSession) Output() <-chan []byte {
	s.mu.Lock()
	pty := s.pty
	s.mu.Unlock()
	if pty == nil {
		return nil
	}
	return pty.Output()
}

func (s *CoreSession) Write(p []byte) error {
	s.mu.Lock()
	if s.closed || !s.started {
		s.mu.Unlock()
		return transport.ErrClosed
	}
	pty := s.pty
	s.mu.Unlock()
	return pty.Write(p)
}

func (s *CoreSession) Resize(cols, rows int) error {
	s.mu.Lock()
	if s.closed || !s.started {
		s.mu.Unlock()
		return transport.ErrClosed
	}
	pty := s.pty
	s.mu.Unlock()
	return pty.Resize(cols, rows)
}

func (s *CoreSession) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	pty := s.pty
	fake := s.fs
	s.mu.Unlock()

	var first error
	if pty != nil {
		first = pty.Close()
	}
	if fake != nil {
		if err := fake.Close(); first == nil {
			first = err
		}
	}
	return first
}

var _ transport.Session = (*CoreSession)(nil)
