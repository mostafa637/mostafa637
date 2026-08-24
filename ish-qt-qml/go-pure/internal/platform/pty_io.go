package platform

import (
	"errors"
	"io"
	"os"
)

func (s *PTYSession) readLoop() {
	defer s.wg.Done()
	defer s.closeOutput()
	buf := make([]byte, 32*1024)
	for {
		if !s.readOnce(buf) {
			return
		}
	}
}

func (s *PTYSession) readOnce(buf []byte) bool {
	ptmx, out, done := s.readState()
	if ptmx == nil || out == nil || done == nil {
		return false
	}
	n, err := ptmx.Read(buf)
	if n > 0 && !s.publish(out, done, buf[:n]) {
		return false
	}
	if err != nil {
		signalStop(s)
		return false
	}
	return true
}

func (s *PTYSession) readState() (*os.File, chan []byte, chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ptmx, s.out, s.done
}

func (s *PTYSession) publish(out chan []byte, done chan struct{}, data []byte) bool {
	copyData := append([]byte(nil), data...)
	select {
	case out <- copyData:
		return true
	case <-done:
		return false
	}
}

func (s *PTYSession) waitLoop() {
	defer s.wg.Done()
	_ = s.cmd.Wait()
	signalStop(s)
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

func isPTYReadEnd(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed)
}
