package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	capVersion64        uint32 = 0x20080522
	capUserHeaderSize64        = 8
	capDataSize64              = 24

	mclCurrent64 uint64 = 1
	mclFuture64  uint64 = 2
	mclOnFault64 uint64 = 4
)

func capget64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[1] == 0 {
		return int64(EFAULT)
	}
	if args[0] != 0 {
		var header [capUserHeaderSize64]byte
		if err := ctx.Memory.Read(corecpu.Address64(args[0]), header[:]); err != nil {
			return int64(EFAULT)
		}
		version := binary.LittleEndian.Uint32(header[:4])
		if version != 0 && version != capVersion64 {
			return int64(EINVAL)
		}
		binary.LittleEndian.PutUint32(header[:4], capVersion64)
		binary.LittleEndian.PutUint32(header[4:8], uint32(ctx.PID))
		if err := ctx.Memory.Write(corecpu.Address64(args[0]), header[:]); err != nil {
			return int64(EFAULT)
		}
	}
	return writeCapabilityData64(ctx, corecpu.Address64(args[1]))
}

func capset64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[1] == 0 {
		return int64(EFAULT)
	}
	if ctx.EffectiveUID != 0 {
		return int64(EPERM)
	}
	if args[0] != 0 {
		var header [capUserHeaderSize64]byte
		if err := ctx.Memory.Read(corecpu.Address64(args[0]), header[:]); err != nil {
			return int64(EFAULT)
		}
		version := binary.LittleEndian.Uint32(header[:4])
		if version != capVersion64 {
			return int64(EINVAL)
		}
	}
	var data [capDataSize64]byte
	if err := ctx.Memory.Read(corecpu.Address64(args[1]), data[:]); err != nil {
		return int64(EFAULT)
	}
	ctx.CapEffective = capabilityPair64(&data, 0)
	ctx.CapPermitted = capabilityPair64(&data, 4)
	ctx.CapInheritable = capabilityPair64(&data, 8)
	return 0
}

func capabilityPair64(data *[capDataSize64]byte, fieldOffset int) uint64 {
	low := binary.LittleEndian.Uint32(data[fieldOffset : fieldOffset+4])
	high := binary.LittleEndian.Uint32(data[fieldOffset+12 : fieldOffset+16])
	return uint64(low) | uint64(high)<<32
}

func writeCapabilityData64(ctx *Context64, address corecpu.Address64) int64 {
	var data [capDataSize64]byte
	binary.LittleEndian.PutUint32(data[0:4], uint32(ctx.CapEffective))
	binary.LittleEndian.PutUint32(data[4:8], uint32(ctx.CapPermitted))
	binary.LittleEndian.PutUint32(data[8:12], uint32(ctx.CapInheritable))
	binary.LittleEndian.PutUint32(data[12:16], uint32(ctx.CapEffective>>32))
	binary.LittleEndian.PutUint32(data[16:20], uint32(ctx.CapPermitted>>32))
	binary.LittleEndian.PutUint32(data[20:24], uint32(ctx.CapInheritable>>32))
	if err := ctx.Memory.Write(address, data[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func validateMemoryLockRange64(ctx *Context64, address, length uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if length == 0 {
		return 0
	}
	if address+length < address {
		return int64(EINVAL)
	}
	pageOffset := address & (corecpu.Page64Size - 1)
	span := pageOffset + length
	if span < pageOffset {
		return int64(EINVAL)
	}
	pages, ok := pagesFor64(span)
	if !ok || !mappingRangeMapped64(ctx.Memory, corecpu.Address64(address), pages) {
		return int64(ENOMEM)
	}
	return 0
}

func mlock64(ctx *Context64, args [6]uint64) int64 {
	return validateMemoryLockRange64(ctx, args[0], args[1])
}

func munlock64(ctx *Context64, args [6]uint64) int64 {
	return validateMemoryLockRange64(ctx, args[0], args[1])
}

func mlockall64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if args[0] == 0 || args[0]&^(mclCurrent64|mclFuture64|mclOnFault64) != 0 {
		return int64(EINVAL)
	}
	return 0
}

func munlockall64(ctx *Context64, _ [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	return 0
}
