package cpu

import "errors"

var (
	ErrARM64InvalidInstruction = errors.New("arm64: invalid instruction")
	ErrARM64Unsupported        = errors.New("arm64: unsupported instruction")
	ErrARM64StepLimit          = errors.New("arm64: step limit reached")
)
