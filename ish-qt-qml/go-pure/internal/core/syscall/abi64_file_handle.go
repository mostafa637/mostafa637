package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

const (
	fileHandleType64       uint32 = 1
	fileHandleBytes64      uint32 = 8
	fileHandleStructSize64        = 8 + fileHandleBytes64
	fileHandleMountID64    int32  = 1
)

func nameToHandleAt64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil || ctx.Memory == nil {
		return int64(ENOSYS)
	}
	if args[4]&^uint64(atEmptyPath64|atSymlinkFollow64) != 0 {
		return int64(EINVAL)
	}
	if args[2] == 0 || args[3] == 0 {
		return int64(EFAULT)
	}

	var resolved string
	var result int64
	if args[1] == 0 && args[4]&atEmptyPath64 != 0 {
		resolved, result = resolveFDPath64(ctx, args[0])
	} else {
		name, ok := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
		if !ok {
			return int64(EFAULT)
		}
		if name == "" {
			return int64(ENOENT)
		}
		resolved, result = resolveAtPath64(ctx, args[0], name)
	}
	if result != 0 {
		return result
	}

	info, err := ctx.FS.Stat(resolved)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	if info.Inode == 0 {
		return int64(EIO)
	}

	var header [8]byte
	if err := ctx.Memory.Read(corecpu.Address64(args[2]), header[:]); err != nil {
		return int64(EFAULT)
	}
	provided := binary.LittleEndian.Uint32(header[0:4])
	if provided < fileHandleBytes64 {
		binary.LittleEndian.PutUint32(header[0:4], fileHandleBytes64)
		binary.LittleEndian.PutUint32(header[4:8], fileHandleType64)
		if err := ctx.Memory.Write(corecpu.Address64(args[2]), header[:]); err != nil {
			return int64(EFAULT)
		}
		return int64(EOVERFLOW)
	}
	if err := ctx.Memory.Write(corecpu.Address64(args[2]), encodeFileHandle64(info.Inode)); err != nil {
		return int64(EFAULT)
	}
	var mountID [4]byte
	binary.LittleEndian.PutUint32(mountID[:], uint32(fileHandleMountID64))
	if err := ctx.Memory.Write(corecpu.Address64(args[3]), mountID[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func openByHandleAt64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil || ctx.FDs == nil || ctx.Memory == nil {
		return int64(ENOSYS)
	}
	if args[2] > uint64(^uint32(0)) {
		return int64(EINVAL)
	}
	hostFlags, ok := hostOpenFlags(uint32(args[2]))
	if !ok {
		return int64(EINVAL)
	}
	mountFile, err := ctx.GetFile(args[0])
	if err != nil || mountFile == nil || mountFile.Path == "" {
		return int64(EBADF)
	}
	mountInfo, err := ctx.FS.Stat(mountFile.Path)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	if !mountInfo.IsDir() {
		return int64(ENOTDIR)
	}

	handle, err := readFileHandle64(ctx, corecpu.Address64(args[1]))
	if err != nil {
		return int64(EFAULT)
	}
	if handle.Type != fileHandleType64 || handle.Bytes != fileHandleBytes64 || handle.Inode == 0 {
		return int64(EINVAL)
	}
	paths, err := ctx.FS.PathsForInode(handle.Inode)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	if len(paths) == 0 {
		return int64(ENOENT)
	}
	path := string(paths[0])
	info, err := ctx.FS.Stat(path)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	file, err := ctx.FS.OpenFile(path, hostFlags, 0, info.Mode)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	guestFile := &corefd.File{
		Reader:  file,
		Writer:  file,
		Closer:  file,
		Seeker:  file,
		Path:    path,
		Cloexec: uint32(args[2])&guestOpenCloexec != 0,
	}
	fd, err := ctx.FDs.Open(guestFile)
	if err != nil {
		_ = file.Close()
		return int64(ENOMEM)
	}
	return int64(fd)
}

type fileHandle64 struct {
	Bytes uint32
	Type  uint32
	Inode uint64
}

func encodeFileHandle64(inode uint64) []byte {
	buffer := make([]byte, fileHandleStructSize64)
	binary.LittleEndian.PutUint32(buffer[0:4], fileHandleBytes64)
	binary.LittleEndian.PutUint32(buffer[4:8], fileHandleType64)
	binary.LittleEndian.PutUint64(buffer[8:16], inode)
	return buffer
}

func readFileHandle64(ctx *Context64, address corecpu.Address64) (fileHandle64, error) {
	var raw [fileHandleStructSize64]byte
	if address == 0 || ctx == nil || ctx.Memory == nil {
		return fileHandle64{}, errFault
	}
	if err := ctx.Memory.Read(address, raw[:]); err != nil {
		return fileHandle64{}, err
	}
	return fileHandle64{
		Bytes: binary.LittleEndian.Uint32(raw[0:4]),
		Type:  binary.LittleEndian.Uint32(raw[4:8]),
		Inode: binary.LittleEndian.Uint64(raw[8:16]),
	}, nil
}
