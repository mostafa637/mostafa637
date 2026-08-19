package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

// statfs64GuestSize is the native x86-64 Linux struct statfs size.
const statfs64GuestSize = 120

const statfs64Magic uint64 = 0x01021994 // TMPFS_MAGIC

func writeStatFS64Guest(ctx *Context64, address corecpu.Address64, info corefs.FileInfo) int64 {
	if ctx == nil || ctx.Memory == nil || address == 0 {
		return int64(EFAULT)
	}
	var value [statfs64GuestSize]byte
	binary.LittleEndian.PutUint64(value[0:8], statfs64Magic)
	binary.LittleEndian.PutUint64(value[8:16], uint64(corecpu.Page64Size))
	const blocks uint64 = 1 << 30
	binary.LittleEndian.PutUint64(value[16:24], blocks)
	binary.LittleEndian.PutUint64(value[24:32], blocks)
	binary.LittleEndian.PutUint64(value[32:40], blocks-1)
	binary.LittleEndian.PutUint64(value[40:48], 1<<32)
	binary.LittleEndian.PutUint64(value[48:56], 1<<32)
	binary.LittleEndian.PutUint32(value[56:60], uint32(info.Inode))
	binary.LittleEndian.PutUint32(value[60:64], uint32(info.Inode>>32))
	binary.LittleEndian.PutUint64(value[64:72], 255) // f_namelen
	binary.LittleEndian.PutUint64(value[72:80], uint64(corecpu.Page64Size))
	// f_flags and the four f_spare words remain zero for the virtual FS.
	if err := ctx.Memory.Write(address, value[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func statfs64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil || ctx.Memory == nil {
		return int64(ENOSYS)
	}
	if args[1] < statfs64GuestSize {
		return int64(EINVAL)
	}
	name, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	resolved, result := resolveAtPath64(ctx, atFDCWD64, name)
	if result != 0 {
		return result
	}
	info, err := ctx.FS.Stat(resolved)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	return writeStatFS64Guest(ctx, corecpu.Address64(args[2]), info)
}

func fstatfs64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil || ctx.Memory == nil {
		return int64(ENOSYS)
	}
	if args[1] < statfs64GuestSize {
		return int64(EINVAL)
	}
	file, err := ctx.GetFile(args[0])
	if err != nil || file == nil || file.Path == "" {
		return int64(EBADF)
	}
	info, statErr := ctx.FS.Stat(file.Path)
	if statErr != nil {
		return int64(errnoForOpen(statErr))
	}
	return writeStatFS64Guest(ctx, corecpu.Address64(args[2]), info)
}

func truncate64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	size, result := guestTruncateSize64(args[1])
	if result != 0 {
		return result
	}
	name, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	resolved, result := resolveAtPath64(ctx, atFDCWD64, name)
	if result != 0 {
		return result
	}
	if err := ctx.FS.Truncate(resolved, size); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func ftruncate64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	size, result := guestTruncateSize64(args[1])
	if result != 0 {
		return result
	}
	file, err := ctx.GetFile(args[0])
	if err != nil || file == nil {
		return int64(EBADF)
	}
	if file.Path == "" {
		return int64(EINVAL)
	}
	if truncateErr := ctx.FS.Truncate(file.Path, size); truncateErr != nil {
		return int64(errnoForOpen(truncateErr))
	}
	return 0
}

func guestTruncateSize64(raw uint64) (int64, int64) {
	if raw > uint64(^uint64(0)>>1) {
		return 0, int64(EINVAL)
	}
	return int64(raw), 0
}

type guestSyncer64 interface {
	Sync() error
}

func sync64(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil || file == nil {
		return int64(EBADF)
	}
	if syncer, ok := file.Closer.(guestSyncer64); ok {
		if syncErr := syncer.Sync(); syncErr != nil {
			return int64(errnoForOpen(syncErr))
		}
	}
	return 0
}

func fsync64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ENOSYS)
	}
	return sync64(ctx, args)
}

func fdatasync64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ENOSYS)
	}
	return sync64(ctx, args)
}
