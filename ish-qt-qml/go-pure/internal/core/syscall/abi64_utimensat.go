package syscall

import (
	"encoding/binary"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	utimeNow64              = int64(0x3fffffff)
	utimeOmit64             = int64(0x3ffffffe)
	utimensatTimespec64Size = 32
)

func readUtimensatTimes64(ctx *Context64, address corecpu.Address64) (*time.Time, *time.Time, int64) {
	if ctx == nil || ctx.Memory == nil || address == 0 {
		return nil, nil, int64(EFAULT)
	}
	buffer := make([]byte, utimensatTimespec64Size)
	if err := ctx.Memory.Read(address, buffer); err != nil {
		return nil, nil, int64(EFAULT)
	}
	decode := func(offset int) (*time.Time, int64) {
		seconds := int64(binary.LittleEndian.Uint64(buffer[offset : offset+8]))
		nanoseconds := int64(binary.LittleEndian.Uint64(buffer[offset+8 : offset+16]))
		switch nanoseconds {
		case utimeNow64:
			now := time.Now()
			return &now, 0
		case utimeOmit64:
			return nil, 0
		default:
			if nanoseconds < 0 || nanoseconds >= int64(time.Second) {
				return nil, int64(EINVAL)
			}
			value := time.Unix(seconds, nanoseconds)
			return &value, 0
		}
	}
	atime, result := decode(0)
	if result != 0 {
		return nil, nil, result
	}
	mtime, result := decode(16)
	if result != 0 {
		return nil, nil, result
	}
	return atime, mtime, 0
}

func utimensat64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	flags := args[3]
	if flags&^uint64(atSymlinkNoFollow64) != 0 {
		return int64(EINVAL)
	}
	name, ok := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	resolved, result := resolveAtPath64(ctx, args[0], name)
	if result != 0 {
		return result
	}
	noFollow := flags&uint64(atSymlinkNoFollow64) != 0
	if args[2] == 0 {
		now := time.Now()
		if err := ctx.FS.SetTimes(resolved, &now, &now, noFollow); err != nil {
			return int64(errnoForOpen(err))
		}
		return 0
	}
	atime, mtime, result := readUtimensatTimes64(ctx, corecpu.Address64(args[2]))
	if result != 0 {
		return result
	}
	if atime == nil && mtime == nil {
		if _, _, err := ctx.FS.Times(resolved, noFollow); err != nil {
			return int64(errnoForOpen(err))
		}
		return 0
	}
	if err := ctx.FS.SetTimes(resolved, atime, mtime, noFollow); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}
