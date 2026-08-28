//go:build !ishcore

package session

import "context"

func startAlpine(context.Context, string) (*Session, error) {
	return nil, ErrNativeCoreUnavailable
}
