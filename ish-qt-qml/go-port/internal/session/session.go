package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty/v2"
)

// Session is a small PTY-backed shell session for the Go/Gio port. It is an
// integration seam: the current implementation starts a host shell, while a
// future iSH backend can implement the same interface over Asbestos/fakefs.
type Session struct {
	cmd *exec.Cmd
	pty *os.File

	mu     sync.Mutex
	closed bool
	out    chan []byte
	done   chan error
}

// Start launches shell in a PTY. It intentionally does not invoke a shell
// through a string, so the program and arguments remain explicit.
func Start(ctx context.Context, shell string, args ...string) (*Session, error) {
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	file, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("session: start %s: %w", shell, err)
	}
	s := &Session{cmd: cmd, pty: file, out: make(chan []byte, 32), done: make(chan error, 1)}
	go s.readLoop()
	return s, nil
}

func (s *Session) readLoop() {
	buf := make([]byte, 32*1024)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			s.out <- chunk
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.done <- err
			} else {
				s.done <- nil
			}
			close(s.out)
			return
		}
	}
}

func (s *Session) Output() <-chan []byte { return s.out }
func (s *Session) Done() <-chan error    { return s.done }

func (s *Session) Write(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return io.ErrClosedPipe
	}
	_, err := s.pty.Write(data)
	return err
}

func (s *Session) Resize(cols, rows uint16) error {
	if cols == 0 || rows == 0 {
		return errors.New("session: PTY dimensions must be non-zero")
	}
	return pty.Setsize(s.pty, &pty.Winsize{Cols: cols, Rows: rows})
}

func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if err := s.pty.Close(); err != nil {
		_ = s.cmd.Process.Kill()
		return err
	}
	return s.cmd.Process.Kill()
}
