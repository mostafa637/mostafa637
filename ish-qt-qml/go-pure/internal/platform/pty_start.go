package platform

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

func validSize(cols, rows int) (int, int) {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	return cols, rows
}

func prepareCommand(base *exec.Cmd, ctx context.Context) *exec.Cmd {
	path, args := commandParts(base)
	if ctx != nil {
		return exec.CommandContext(ctx, path, args...)
	}
	return exec.Command(path, args...)
}

func commandParts(cmd *exec.Cmd) (string, []string) {
	if cmd == nil || cmd.Path == "" {
		return "/bin/sh", nil
	}
	return cmd.Path, append([]string(nil), cmd.Args[1:]...)
}

func (s *PTYSession) startLocked(cmd *exec.Cmd, cols, rows int) error {
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return fmt.Errorf("start PTY: %w", err)
	}
	s.cmd, s.ptmx = cmd, ptmx
	s.out, s.done, s.closed = make(chan []byte, 32), make(chan struct{}), false
	s.once = sync.Once{}
	s.wg.Add(2)
	go s.readLoop()
	go s.waitLoop()
	return nil
}
