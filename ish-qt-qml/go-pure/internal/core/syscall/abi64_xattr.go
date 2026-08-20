package syscall

import (
	"errors"
	"strings"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corestorage "github.com/mostafa637/mostafa637/go-pure/internal/core/storage"
)

const (
	maxXattrName64  = 255
	maxXattrValue64 = 65536
)

func xattrErrno64(err error) int64 {
	switch {
	case errors.Is(err, corestorage.ErrNotFound):
		return int64(ENOENT)
	case errors.Is(err, corestorage.ErrExists):
		return int64(EEXIST)
	case errors.Is(err, corestorage.ErrNoData):
		return int64(ENODATA)
	case errors.Is(err, corestorage.ErrInvariant):
		return int64(EINVAL)
	default:
		return int64(errnoForOpen(err))
	}
}

func readXattrName64(ctx *Context64, address uint64) (string, int64) {
	name, ok := readGuestString64(ctx, corecpu.Address64(address), maxXattrName64+1)
	if !ok || name == "" || strings.IndexByte(name, 0) >= 0 {
		return "", int64(EINVAL)
	}
	return name, 0
}

func readXattrValue64(ctx *Context64, address, size uint64) ([]byte, int64) {
	if size > maxXattrValue64 {
		return nil, int64(E2BIG)
	}
	if size == 0 {
		return nil, 0
	}
	if address == 0 || size > uint64(^uint(0)>>1) {
		return nil, int64(EFAULT)
	}
	value := make([]byte, int(size))
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Read(corecpu.Address64(address), value) != nil {
		return nil, int64(EFAULT)
	}
	return value, 0
}

func resolveXattrPath64(ctx *Context64, address uint64) (string, int64) {
	name, ok := readGuestString64(ctx, corecpu.Address64(address), 4096)
	if !ok || name == "" {
		return "", int64(EFAULT)
	}
	resolved, result := resolveGuestPath64(ctx, name)
	if !result {
		return "", int64(ENOENT)
	}
	return resolved, 0
}

func xattrPathInode64(ctx *Context64, path string, noFollow bool) (uint64, int64) {
	if ctx == nil || ctx.FS == nil {
		return 0, int64(ENOSYS)
	}
	if noFollow {
		value, statErr := ctx.FS.Lstat(path)
		if statErr != nil {
			return 0, xattrErrno64(statErr)
		}
		return value.Inode, 0
	}
	value, statErr := ctx.FS.Stat(path)
	if statErr != nil {
		return 0, xattrErrno64(statErr)
	}
	return value.Inode, 0
}

func xattrFDInode64(ctx *Context64, fd uint64) (uint64, int64) {
	if ctx == nil || ctx.FS == nil {
		return 0, int64(ENOSYS)
	}
	file, err := ctx.GetFile(fd)
	if err != nil || file == nil {
		return 0, int64(EBADF)
	}
	if file.Path == "" {
		return 0, int64(EBADF)
	}
	info, statErr := ctx.FS.Stat(file.Path)
	if statErr != nil {
		return 0, xattrErrno64(statErr)
	}
	return info.Inode, 0
}

func setXattrInode64(ctx *Context64, inode uint64, args [6]uint64) int64 {
	if args[4]&^(uint64(corestorage.XattrCreate)|uint64(corestorage.XattrReplace)) != 0 {
		return int64(EINVAL)
	}
	attr, result := readXattrName64(ctx, args[1])
	if result != 0 {
		return result
	}
	value, result := readXattrValue64(ctx, args[2], args[3])
	if result != 0 {
		return result
	}
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	if err := ctx.FS.SetXattrByInode(inode, attr, value, uint32(args[4])); err != nil {
		return xattrErrno64(err)
	}
	return 0
}

func getXattrInode64(ctx *Context64, inode uint64, args [6]uint64) int64 {
	attr, result := readXattrName64(ctx, args[1])
	if result != 0 {
		return result
	}
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	value, err := ctx.FS.GetXattrByInode(inode, attr)
	if err != nil {
		return xattrErrno64(err)
	}
	if args[3] == 0 {
		return int64(len(value))
	}
	if args[3] < uint64(len(value)) {
		return int64(ERANGE)
	}
	if args[2] == 0 || ctx.Memory == nil || ctx.Memory.Write(corecpu.Address64(args[2]), value) != nil {
		return int64(EFAULT)
	}
	return int64(len(value))
}

func listXattrInode64(ctx *Context64, inode uint64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	value, err := ctx.FS.ListXattrByInode(inode)
	if err != nil {
		return xattrErrno64(err)
	}
	if args[2] == 0 {
		return int64(len(value))
	}
	if args[2] < uint64(len(value)) {
		return int64(ERANGE)
	}
	if args[1] == 0 || ctx.Memory == nil || ctx.Memory.Write(corecpu.Address64(args[1]), value) != nil {
		return int64(EFAULT)
	}
	return int64(len(value))
}

func removeXattrInode64(ctx *Context64, inode uint64, args [6]uint64) int64 {
	attr, result := readXattrName64(ctx, args[1])
	if result != 0 {
		return result
	}
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	if err := ctx.FS.RemoveXattrByInode(inode, attr); err != nil {
		return xattrErrno64(err)
	}
	return 0
}

func setxattr64(ctx *Context64, args [6]uint64) int64 {
	path, result := resolveXattrPath64(ctx, args[0])
	if result != 0 {
		return result
	}
	inode, result := xattrPathInode64(ctx, path, false)
	if result != 0 {
		return result
	}
	return setXattrInode64(ctx, inode, args)
}

func lsetxattr64(ctx *Context64, args [6]uint64) int64 {
	path, result := resolveXattrPath64(ctx, args[0])
	if result != 0 {
		return result
	}
	inode, result := xattrPathInode64(ctx, path, true)
	if result != 0 {
		return result
	}
	return setXattrInode64(ctx, inode, args)
}

func fsetxattr64(ctx *Context64, args [6]uint64) int64 {
	inode, result := xattrFDInode64(ctx, args[0])
	if result != 0 {
		return result
	}
	return setXattrInode64(ctx, inode, [6]uint64{0, args[1], args[2], args[3], args[4], args[5]})
}

func getxattr64(ctx *Context64, args [6]uint64) int64 {
	path, result := resolveXattrPath64(ctx, args[0])
	if result != 0 {
		return result
	}
	inode, result := xattrPathInode64(ctx, path, false)
	if result != 0 {
		return result
	}
	return getXattrInode64(ctx, inode, [6]uint64{0, args[1], args[2], args[3], args[4], args[5]})
}

func lgetxattr64(ctx *Context64, args [6]uint64) int64 {
	path, result := resolveXattrPath64(ctx, args[0])
	if result != 0 {
		return result
	}
	inode, result := xattrPathInode64(ctx, path, true)
	if result != 0 {
		return result
	}
	return getXattrInode64(ctx, inode, [6]uint64{0, args[1], args[2], args[3], args[4], args[5]})
}

func fgetxattr64(ctx *Context64, args [6]uint64) int64 {
	inode, result := xattrFDInode64(ctx, args[0])
	if result != 0 {
		return result
	}
	return getXattrInode64(ctx, inode, [6]uint64{0, args[1], args[2], args[3], args[4], args[5]})
}

func listxattr64(ctx *Context64, args [6]uint64) int64 {
	path, result := resolveXattrPath64(ctx, args[0])
	if result != 0 {
		return result
	}
	inode, result := xattrPathInode64(ctx, path, false)
	if result != 0 {
		return result
	}
	return listXattrInode64(ctx, inode, [6]uint64{0, args[1], args[2], args[3], args[4], args[5]})
}

func llistxattr64(ctx *Context64, args [6]uint64) int64 {
	path, result := resolveXattrPath64(ctx, args[0])
	if result != 0 {
		return result
	}
	inode, result := xattrPathInode64(ctx, path, true)
	if result != 0 {
		return result
	}
	return listXattrInode64(ctx, inode, [6]uint64{0, args[1], args[2], args[3], args[4], args[5]})
}

func flistxattr64(ctx *Context64, args [6]uint64) int64 {
	inode, result := xattrFDInode64(ctx, args[0])
	if result != 0 {
		return result
	}
	return listXattrInode64(ctx, inode, [6]uint64{0, args[1], args[2], args[3], args[4], args[5]})
}

func removexattr64(ctx *Context64, args [6]uint64) int64 {
	path, result := resolveXattrPath64(ctx, args[0])
	if result != 0 {
		return result
	}
	inode, result := xattrPathInode64(ctx, path, false)
	if result != 0 {
		return result
	}
	return removeXattrInode64(ctx, inode, [6]uint64{0, args[1], args[2], args[3], args[4], args[5]})
}

func lremovexattr64(ctx *Context64, args [6]uint64) int64 {
	path, result := resolveXattrPath64(ctx, args[0])
	if result != 0 {
		return result
	}
	inode, result := xattrPathInode64(ctx, path, true)
	if result != 0 {
		return result
	}
	return removeXattrInode64(ctx, inode, [6]uint64{0, args[1], args[2], args[3], args[4], args[5]})
}

func fremovexattr64(ctx *Context64, args [6]uint64) int64 {
	inode, result := xattrFDInode64(ctx, args[0])
	if result != 0 {
		return result
	}
	return removeXattrInode64(ctx, inode, [6]uint64{0, args[1], args[2], args[3], args[4], args[5]})
}
