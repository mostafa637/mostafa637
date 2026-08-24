package platform

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
)

// PTYSession is the host-shell implementation of session.Session.
type PTYSession struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	ptmx   *os.File
	out    chan []byte
	done   chan struct{}
	closed bool
	once   sync.Once
	wg     sync.WaitGroup
}

func NewPTYSession(shell string) *PTYSession {
	if shell == "" {
		shell = "/bin/sh"
	}
	return &PTYSession{cmd: exec.Command(shell)}
}

func (s *PTYSession) Start(ctx context.Context, cols, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ptmx != nil {
		return errors.New("PTY session already started")
	}
	cols, rows = validSize(cols, rows)
	cmd := prepareCommand(s.cmd, ctx)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	return s.startLocked(cmd, cols, rows)
}

func (s *PTYSession) Output() <-chan []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.out
}
