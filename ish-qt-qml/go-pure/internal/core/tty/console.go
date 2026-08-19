// Package tty contains the platform-neutral guest console bridge.
// It does not open /dev/tty itself; callers inject the PTY or stream endpoints,
// which keeps Android and Linux lifecycle code outside the kernel package.
package tty

import (
	"io"

	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

type Console struct {
	Input  io.Reader
	Output io.Writer
	Error  io.Writer
}

func NewConsole(input io.Reader, output, errOut io.Writer) *Console {
	return &Console{Input: input, Output: output, Error: errOut}
}

func (c *Console) Install(table *corefd.Table) error {
	if c == nil || table == nil {
		return corefd.ErrBadFD
	}
	if err := table.InstallAt(0, &corefd.File{Reader: c.Input}, true); err != nil {
		return err
	}
	if err := table.InstallAt(1, &corefd.File{Writer: c.Output}, true); err != nil {
		return err
	}
	return table.InstallAt(2, &corefd.File{Writer: c.Error}, true)
}

type ReadWriter struct {
	io.Reader
	io.Writer
	io.Closer
}

func (r ReadWriter) Close() error {
	if r.Closer == nil {
		return nil
	}
	return r.Closer.Close()
}
