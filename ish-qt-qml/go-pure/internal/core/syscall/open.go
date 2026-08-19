package syscall

import (
	"errors"
	"io/fs"
	"os"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

const (
	guestOpenAccessMask = 0x3
	guestOpenCreat      = 0x40
	guestOpenExcl       = 0x80
	guestOpenNoCTTY     = 0x100
	guestOpenTrunc      = 0x200
	guestOpenAppend     = 0x400
	guestOpenNonblock   = 0x800
	guestOpenNoFollow   = 0x20000
)

func readGuestString(context *Context, state *corecpu.MachineState, address corecpu.Address, limit int) (string, bool) {
	if context == nil || context.Memory == nil || limit <= 0 {
		return "", false
	}
	buffer := make([]byte, 0, limit)
	var one [1]byte
	for i := 0; i < limit; i++ {
		if err := context.Memory.Read(address+corecpu.Address(i), one[:]); err != nil {
			return "", false
		}
		if one[0] == 0 {
			return string(buffer), true
		}
		buffer = append(buffer, one[0])
	}
	return "", false
}

func hostOpenFlags(flags uint32) (int, bool) {
	known := uint32(guestOpenAccessMask | guestOpenCreat | guestOpenExcl | guestOpenNoCTTY | guestOpenTrunc | guestOpenAppend | guestOpenNonblock | guestOpenNoFollow)
	if flags&^known != 0 {
		return 0, false
	}
	result := 0
	switch flags & guestOpenAccessMask {
	case 0:
		result |= os.O_RDONLY
	case 1:
		result |= os.O_WRONLY
	case 2:
		result |= os.O_RDWR
	default:
		return 0, false
	}
	if flags&guestOpenCreat != 0 {
		result |= os.O_CREATE
	}
	if flags&guestOpenExcl != 0 {
		result |= os.O_EXCL
	}
	if flags&guestOpenTrunc != 0 {
		result |= os.O_TRUNC
	}
	if flags&guestOpenAppend != 0 {
		result |= os.O_APPEND
	}
	// O_NONBLOCK is a descriptor-status flag on Linux. Go's os.File
	// does not expose a portable os.O_NONBLOCK constant, so it is retained
	// in the guest ABI and handled by the fd/tty layer when needed.
	return result, true
}

func errnoForOpen(err error) int32 {
	switch {
	case errors.Is(err, corefs.ErrInvalidPath):
		return EINVAL
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, corefs.ErrNotFound):
		return ENOENT
	case errors.Is(err, fs.ErrExist):
		return EEXIST
	case errors.Is(err, fs.ErrPermission), errors.Is(err, os.ErrPermission):
		return EACCES
	default:
		return EIO
	}
}
