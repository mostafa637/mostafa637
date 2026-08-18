package platform

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

// PTYSession is the host-shell implementation of session.Session.
// The iSH-core adapter can implement the same interface without changing the UI.
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
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	if s.cmd == nil || s.cmd.Path == "" {
		s.cmd = exec.Command("/bin/sh")
	}
	path := s.cmd.Path
	args := append([]string(nil), s.cmd.Args[1:]...)
	if ctx != nil {
		s.cmd = exec.CommandContext(ctx, path, args...)
	} else {
		s.cmd = exec.Command(path, args...)
	}
	s.cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	ptmx, err := pty.StartWithSize(s.cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return fmt.Errorf("start PTY: %w", err)
	}
	s.ptmx = ptmx
	s.out = make(chan []byte, 32)
	s.done = make(chan struct{})
	s.closed = false
	s.once = sync.Once{}
	s.wg.Add(2)
	go s.readLoop()
	go s.waitLoop()
	return nil
}

func (s *PTYSession) Output() <-chan []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.out
}

func (s *PTYSession) readLoop() {
	defer s.wg.Done()
	defer s.closeOutput()
	buf := make([]byte, 32*1024)
	for {
		s.mu.Lock()
		ptmx := s.ptmx
		out := s.out
		done := s.done
		s.mu.Unlock()
		if ptmx == nil || out == nil || done == nil {
			return
		}
		n, err := ptmx.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			select {
			case out <- data:
			case <-done:
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrClosed) {
				// Linux PTYs commonly return EIO after the slave exits. This is
				// a normal end-of-session condition, not an application crash.
			}
			signalStop(s)
			return
		}
	}
}

func (s *PTYSession) waitLoop() {
	defer s.wg.Done()
	_ = s.cmd.Wait()
	signalStop(s)
}

func (s *PTYSession) Write(p []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ptmx == nil {
		return errors.New("PTY session is closed")
	}
	_, err := s.ptmx.Write(p)
	return err
}

func (s *PTYSession) Resize(cols, rows int) error {
	if cols < 1 || rows < 1 {
		return errors.New("invalid PTY size")
	}
	s.mu.Lock()
	ptmx := s.ptmx
	s.mu.Unlock()
	if ptmx == nil {
		return errors.New("PTY session is not started")
	}
	return pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func (s *PTYSession) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	ptmx := s.ptmx
	cmd := s.cmd
	s.mu.Unlock()
	signalStop(s)
	if ptmx != nil {
		_ = ptmx.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	s.wg.Wait()
	return nil
}

func signalStop(s *PTYSession) {
	s.mu.Lock()
	done := s.done
	s.mu.Unlock()
	if done != nil {
		s.once.Do(func() { close(done) })
	}
}

func (s *PTYSession) closeOutput() {
	s.mu.Lock()
	out := s.out
	s.out = nil
	s.mu.Unlock()
	if out != nil {
		close(out)
	}
}
