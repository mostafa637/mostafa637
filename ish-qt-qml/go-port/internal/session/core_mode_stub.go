//go:build !ishcore || !cgo

package session

func nativeCoreAvailable() bool { return false }
