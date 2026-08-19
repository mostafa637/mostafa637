package core

import (
	"context"

	coresession "github.com/mostafa637/mostafa637/go-pure/internal/core/session"
	"github.com/mostafa637/mostafa637/go-pure/internal/session"
)

// BackendFactory is the seam between the UI/application layer and a core
// implementation. The Go implementation below owns fakefs and session
// lifecycle; the C core remains only as an upstream comparison reference.
type BackendFactory interface {
	New(ctx context.Context) (session.Session, error)
}

// NoopFactory is intentionally explicit: it prevents the UI from silently
// starting a host shell when the iSH core backend is expected.
type NoopFactory struct{}

func (NoopFactory) New(context.Context) (session.Session, error) {
	return nil, session.ErrClosed
}

// GoFactory constructs the current Pure Go core session.
type GoFactory struct {
	Config coresession.Config
}

func (f GoFactory) New(context.Context) (session.Session, error) {
	return coresession.New(f.Config)
}
