package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func (r *ioUring64) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	r.closed = true
	r.registeredBuffers = nil
	r.registeredFiles = nil
	if r.wake != nil {
		r.wake.Broadcast()
	}
	r.mu.Unlock()
	return nil
}

func ioUringReadU32(ctx *Context64, address corecpu.Address64) (uint32, int64) {
	var raw [4]byte
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Read(address, raw[:]) != nil {
		return 0, int64(EFAULT)
	}
	return binary.LittleEndian.Uint32(raw[:]), 0
}

func ioUringWriteU32(ctx *Context64, address corecpu.Address64, value uint32) int64 {
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], value)
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Write(address, raw[:]) != nil {
		return int64(EFAULT)
	}
	return 0
}

func ioUringReadU64(ctx *Context64, address corecpu.Address64) (uint64, int64) {
	var raw [8]byte
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Read(address, raw[:]) != nil {
		return 0, int64(EFAULT)
	}
	return binary.LittleEndian.Uint64(raw[:]), 0
}

func ioUringWriteU64(ctx *Context64, address corecpu.Address64, value uint64) int64 {
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], value)
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Write(address, raw[:]) != nil {
		return int64(EFAULT)
	}
	return 0
}

func ioUringRoundPow2(value, maximum uint32, clamp bool) (uint32, int64) {
	if value == 0 {
		return 0, int64(EINVAL)
	}
	if value > maximum {
		if !clamp {
			return 0, int64(EINVAL)
		}
		return maximum, 0
	}
	rounded := uint32(1)
	for rounded < value {
		if rounded > maximum/2 {
			if !clamp {
				return 0, int64(EINVAL)
			}
			return maximum, 0
		}
		rounded <<= 1
	}
	return rounded, 0
}

func ioUringAlignPage64(length uint64) (uint64, bool) {
	if length == 0 || length > ^uint64(0)-(corecpu.Page64Size-1) {
		return 0, false
	}
	return (length + corecpu.Page64Size - 1) & ^(corecpu.Page64Size - 1), true
}
