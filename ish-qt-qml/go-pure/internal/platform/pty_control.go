package platform

import (
	"errors"
	"os"
	"os/exec"

	"github.com/creack/pty"
)

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
	ptmx, cmd := s.ptmx, s.cmd
	s.mu.Unlock()
	signalStop(s)
	closePTY(ptmx)
	killCommand(cmd)
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

func closePTY(ptmx *os.File) {
	if ptmx != nil {
		_ = ptmx.Close()
	}
}

func killCommand(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
