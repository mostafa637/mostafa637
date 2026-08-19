package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

const statfs64Size = 84

// writeStatFS64 emits the i386 Linux struct statfs64 layout. FakeFS has no
// quota/capacity backend, so the values describe a large virtual filesystem
// while preserving its block-size and name-length ABI properties.
func writeStatFS64(context *Context, address corecpu.Address, info corefs.FileInfo) int32 {
	if context == nil || context.Memory == nil || address == 0 {
		return EFAULT
	}
	var value [statfs64Size]byte
	binary.LittleEndian.PutUint32(value[0:4], 0x01021994) // TMPFS_MAGIC
	binary.LittleEndian.PutUint32(value[4:8], corecpu.PageSize)
	const blocks uint64 = 1 << 30
	binary.LittleEndian.PutUint64(value[8:16], blocks)
	binary.LittleEndian.PutUint64(value[16:24], blocks)
	binary.LittleEndian.PutUint64(value[24:32], blocks-1)
	binary.LittleEndian.PutUint64(value[32:40], 1<<32)
	binary.LittleEndian.PutUint64(value[40:48], 1<<32)
	binary.LittleEndian.PutUint32(value[48:52], uint32(info.Inode))
	binary.LittleEndian.PutUint32(value[52:56], 255)
	binary.LittleEndian.PutUint32(value[56:60], corecpu.PageSize)
	if err := context.Memory.Write(address, value[:]); err != nil {
		return EFAULT
	}
	return 0
}

func statfs64(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FS == nil || args[1] < statfs64Size {
		return EINVAL
	}
	path, ok := readGuestString(context, state, corecpu.Address(args[0]), 4096)
	if !ok {
		return EFAULT
	}
	path, ok = resolveGuestPath(context, path)
	if !ok {
		return ENOENT
	}
	info, err := context.FS.Stat(path)
	if err != nil {
		return errnoForOpen(err)
	}
	return writeStatFS64(context, corecpu.Address(args[2]), info)
}

func fstatfs64(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FS == nil || args[1] < statfs64Size {
		return EINVAL
	}
	file, result := getFile(context, args[0])
	if result != 0 || file == nil || file.Path == "" {
		return EBADF
	}
	info, err := context.FS.Stat(file.Path)
	if err != nil {
		return errnoForOpen(err)
	}
	return writeStatFS64(context, corecpu.Address(args[2]), info)
}
