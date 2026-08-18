package session

import (
	"context"
	"errors"
)

var ErrClosed = errors.New("session is closed")

// Session is the transport boundary between the terminal model and a shell/core.
// Implementations may wrap a host PTY, the iSH C core, or an Android backend.
type Session interface {
	Start(ctx context.Context, cols, rows int) error
	Output() <-chan []byte
	Write([]byte) error
	Resize(cols, rows int) error
	Close() error
}
