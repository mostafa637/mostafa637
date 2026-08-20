package wasmjit

import "errors"

var (
	ErrUnsupported  = errors.New("wasmjit: unsupported guest instruction")
	ErrInvalidBlock = errors.New("wasmjit: invalid guest block")
)
