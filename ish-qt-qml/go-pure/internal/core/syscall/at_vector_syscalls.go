package syscall

import (
	"encoding/binary"
	"io"
	pathpkg "path"
	"strings"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

const (
	atFDCwd           uint32 = ^uint32(99)
	atSymlinkNoFollow uint32 = 0x100
	atEmptyPath       uint32 = 0x1000

	maxGuestIOVecs  = 1024
	maxGuestIOBytes = 16 << 20
)

type guestIOVec struct {
	Base uint32
	Len  uint32
}

func openat(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FS == nil {
		return ENOSYS
	}
	name, ok := readGuestString(context, state, corecpu.Address(args[1]), 4096)
	if !ok || name == "" {
		return ENOENT
	}
	path, result := resolveAtPath(context, args[0], name)
	if result != 0 {
		return result
	}
	return openResolvedPath(context, path, args[2], args[3])
}

func fstatat64(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FS == nil {
		return ENOSYS
	}
	flags := args[3]
	if flags&^(atSymlinkNoFollow|atEmptyPath) != 0 {
		return EINVAL
	}
	name, ok := readGuestString(context, state, corecpu.Address(args[1]), 4096)
	if !ok {
		return EFAULT
	}

	var path string
	if name == "" && flags&atEmptyPath != 0 {
		if args[0] == atFDCwd {
			path = context.CWD
		} else {
			file := context.file(args[0])
			if file == nil {
				return EBADF
			}
			path = file.Path
			if path == "" {
				return ENOTDIR
			}
		}
	} else {
		if name == "" {
			return ENOENT
		}
		var result int32
		path, result = resolveAtPath(context, args[0], name)
		if result != 0 {
			return result
		}
	}

	var fileInfo corefs.FileInfo
	var err error
	if flags&atSymlinkNoFollow != 0 {
		fileInfo, err = context.FS.Lstat(path)
	} else {
		fileInfo, err = context.FS.Stat(path)
	}
	if err != nil {
		return errnoForOpen(err)
	}
	return writeStat64(context, state, corecpu.Address(args[2]), fileInfo)
}

func resolveAtPath(context *Context, dirfd uint32, name string) (string, int32) {
	if context == nil || context.FS == nil || name == "" {
		return "", ENOENT
	}
	if strings.HasPrefix(name, "/") {
		return pathpkg.Clean(name), 0
	}
	if dirfd == atFDCwd {
		path, ok := resolveGuestPath(context, name)
		if !ok {
			return "", ENOENT
		}
		return path, 0
	}
	file := context.file(dirfd)
	if file == nil {
		return "", EBADF
	}
	if file.Path == "" {
		return "", ENOTDIR
	}
	info, err := context.FS.Stat(file.Path)
	if err != nil {
		return "", errnoForOpen(err)
	}
	if !info.IsDir() {
		return "", ENOTDIR
	}
	return pathpkg.Join(file.Path, name), 0
}

func readv(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	file := context.file(args[0])
	if file == nil || file.Reader == nil {
		return EBADF
	}
	iovecs, result := readGuestIOVecs(context, args[1], args[2])
	if result != 0 {
		return result
	}
	var total int32
	for _, iovec := range iovecs {
		if iovec.Len == 0 {
			continue
		}
		length, ok := safeLength(iovec.Len)
		if !ok {
			return EINVAL
		}
		buffer := make([]byte, length)
		n, err := file.Reader.Read(buffer)
		if n > 0 {
			if writeMemory(context, state, corecpu.Address(iovec.Base), buffer[:n]) != nil {
				return EFAULT
			}
			total += int32(n)
		}
		if err != nil && err != io.EOF {
			if total != 0 {
				return total
			}
			return EIO
		}
		if n < length || err == io.EOF {
			break
		}
	}
	return total
}

func writev(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	file := context.file(args[0])
	if file == nil || file.Writer == nil {
		return EBADF
	}
	iovecs, result := readGuestIOVecs(context, args[1], args[2])
	if result != 0 {
		return result
	}
	var total int32
	for _, iovec := range iovecs {
		if iovec.Len == 0 {
			continue
		}
		length, ok := safeLength(iovec.Len)
		if !ok {
			return EINVAL
		}
		buffer := make([]byte, length)
		if context.Memory == nil || state == nil || context.Memory.Read(corecpu.Address(iovec.Base), buffer) != nil {
			return EFAULT
		}
		n, err := file.Writer.Write(buffer)
		if n > 0 {
			total += int32(n)
		}
		if err != nil {
			if total != 0 {
				return total
			}
			return EIO
		}
		if n < length {
			break
		}
	}
	return total
}

func readGuestIOVecs(context *Context, address, count uint32) ([]guestIOVec, int32) {
	if context == nil || context.Memory == nil {
		return nil, EFAULT
	}
	if count > maxGuestIOVecs {
		return nil, EINVAL
	}
	if count == 0 {
		return nil, 0
	}
	byteCount := uint64(count) * 8
	if byteCount > uint64(^uint(0)>>1) || uint64(address)+byteCount > uint64(^uint32(0))+1 {
		return nil, EFAULT
	}
	raw := make([]byte, int(byteCount))
	if err := context.Memory.Read(corecpu.Address(address), raw); err != nil {
		return nil, EFAULT
	}
	iovecs := make([]guestIOVec, count)
	var total uint64
	for index := range iovecs {
		offset := index * 8
		base := binary.LittleEndian.Uint32(raw[offset : offset+4])
		length := binary.LittleEndian.Uint32(raw[offset+4 : offset+8])
		if uint64(base)+uint64(length) > uint64(^uint32(0))+1 {
			return nil, EFAULT
		}
		total += uint64(length)
		if total > maxGuestIOBytes {
			return nil, EINVAL
		}
		iovecs[index] = guestIOVec{Base: base, Len: length}
	}
	return iovecs, 0
}
