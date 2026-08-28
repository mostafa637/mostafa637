package session

import (
	"context"
	"errors"
)

var ErrNativeCoreUnavailable = errors.New("session: native iSH core is not enabled")

func NativeCoreAvailable() bool { return nativeCoreAvailable() }

// StartAlpine starts the bundled Alpine rootfs through the iSH/Asbestos core.
// Builds without the ishcore tag return ErrNativeCoreUnavailable so desktop
// development can explicitly choose the host-shell fallback.
func StartAlpine(ctx context.Context, rootPath string) (*Session, error) {
	return startAlpine(ctx, rootPath)
}
