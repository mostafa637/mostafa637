package core

import (
	"context"

	"github.com/mostafa637/mostafa637/go-pure/internal/session"
)

// BackendFactory is the seam for the iSH C core adapter.
// The first Linux prototype uses platform.PTYSession; the production backend
// will implement this factory with a cgo bridge to CoreSession.c.
type BackendFactory interface {
	New(ctx context.Context) (session.Session, error)
}

// NoopFactory is intentionally explicit: it prevents the UI from silently
// starting a host shell when the iSH core backend is expected.
type NoopFactory struct{}

func (NoopFactory) New(context.Context) (session.Session, error) {
	return nil, session.ErrClosed
}
