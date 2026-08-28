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

type backend interface {
	Output() <-chan []byte
	Done() <-chan error
	Write([]byte) error
	Resize(cols, rows uint16) error
	Close() error
}

// Session is the stable terminal-session API used by Gio. The implementation
// can be the host PTY fallback or the native iSH/Asbestos backend.
type Session struct {
	impl backend
}

func newSession(impl backend) *Session { return &Session{impl: impl} }

// Start launches a host shell in a PTY. It remains useful for Linux development
// builds that do not link the native iSH core.
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
	p := &ptyBackend{cmd: cmd, pty: file, out: make(chan []byte, 32), done: make(chan error, 1)}
	go p.readLoop()
	return newSession(p), nil
}

type ptyBackend struct {
	cmd *exec.Cmd
	pty *os.File

	mu     sync.Mutex
	closed bool
	out    chan []byte
	done   chan error
}

func (p *ptyBackend) readLoop() {
	buf := make([]byte, 32*1024)
	for {
		n, err := p.pty.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			p.out <- chunk
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				p.done <- err
			} else {
				p.done <- nil
			}
			close(p.out)
			return
		}
	}
}

func (s *Session) Output() <-chan []byte          { return s.impl.Output() }
func (s *Session) Done() <-chan error             { return s.impl.Done() }
func (s *Session) Write(data []byte) error        { return s.impl.Write(data) }
func (s *Session) Resize(cols, rows uint16) error { return s.impl.Resize(cols, rows) }
func (s *Session) Close() error                   { return s.impl.Close() }

func (p *ptyBackend) Output() <-chan []byte { return p.out }
func (p *ptyBackend) Done() <-chan error    { return p.done }

func (p *ptyBackend) Write(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return io.ErrClosedPipe
	}
	_, err := p.pty.Write(data)
	return err
}

func (p *ptyBackend) Resize(cols, rows uint16) error {
	if cols == 0 || rows == 0 {
		return errors.New("session: PTY dimensions must be non-zero")
	}
	return pty.Setsize(p.pty, &pty.Winsize{Cols: cols, Rows: rows})
}

func (p *ptyBackend) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	if err := p.pty.Close(); err != nil {
		_ = p.cmd.Process.Kill()
		return err
	}
	return p.cmd.Process.Kill()
}
