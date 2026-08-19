package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	archSetGS64    = 0x1001
	archSetFS64    = 0x1002
	archGetFS64    = 0x1003
	archGetGS64    = 0x1004
	archGetCPUID64 = 0x1011
	archSetCPUID64 = 0x1012
)

func archPrctl64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(EFAULT)
	}
	switch args[0] {
	case archSetGS64:
		ctx.GSBase = args[1]
	case archSetFS64:
		ctx.FSBase = args[1]
	case archGetFS64:
		return writeArchValue64(ctx, args[1], ctx.FSBase)
	case archGetGS64:
		return writeArchValue64(ctx, args[1], ctx.GSBase)
	case archGetCPUID64:
		if ctx.CPUIDEnabled {
			return writeArchValue64(ctx, args[1], 1)
		}
		return writeArchValue64(ctx, args[1], 0)
	case archSetCPUID64:
		if args[1] > 1 {
			return int64(EINVAL)
		}
		ctx.CPUIDEnabled = args[1] != 0
	default:
		return int64(EINVAL)
	}
	return 0
}

func writeArchValue64(ctx *Context64, address uint64, value uint64) int64 {
	if ctx.Memory == nil || address == 0 {
		return int64(EFAULT)
	}
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], value)
	if err := ctx.Memory.Write(corecpu.Address64(address), raw[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}
