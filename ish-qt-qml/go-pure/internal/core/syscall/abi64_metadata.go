package syscall

import (
	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const ownerIDUnused64 = uint64(^uint32(0))

func metadataMode64(value uint64) (uint32, int64) {
	if value > uint64(^uint32(0)) {
		return 0, int64(EINVAL)
	}
	return uint32(value), 0
}

func metadataOwner64(value uint64) (*uint32, int64) {
	if value == ownerIDUnused64 {
		return nil, 0
	}
	if value > uint64(^uint32(0)) {
		return nil, int64(EINVAL)
	}
	owner := uint32(value)
	return &owner, 0
}

func setPathAttributes64(ctx *Context64, name string, modeValue, uidValue, gidValue uint64) int64 {
	if ctx == nil || ctx.FS == nil || name == "" {
		return int64(ENOSYS)
	}
	mode, result := metadataMode64(modeValue)
	if result != 0 {
		return result
	}
	mode &= 0o7777
	uid, result := metadataOwner64(uidValue)
	if result != 0 {
		return result
	}
	gid, result := metadataOwner64(gidValue)
	if result != 0 {
		return result
	}
	if err := ctx.FS.SetAttr(name, &mode, uid, gid); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func setOwnerAttributes64(ctx *Context64, name string, uidValue, gidValue uint64) int64 {
	if ctx == nil || ctx.FS == nil || name == "" {
		return int64(ENOSYS)
	}
	uid, result := metadataOwner64(uidValue)
	if result != 0 {
		return result
	}
	gid, result := metadataOwner64(gidValue)
	if result != 0 {
		return result
	}
	if err := ctx.FS.SetAttr(name, nil, uid, gid); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func fchmod64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	file, err := ctx.GetFile(args[0])
	if err != nil || file == nil {
		return int64(EBADF)
	}
	if file.Path == "" {
		return int64(EBADF)
	}
	return setPathAttributes64(ctx, file.Path, args[1], ownerIDUnused64, ownerIDUnused64)
}

func fchmodat64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	if args[3]&^uint64(atSymlinkNoFollow64) != 0 {
		return int64(EINVAL)
	}
	name, ok := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	if name == "" {
		return int64(ENOENT)
	}
	resolved, result := resolveAtPath64(ctx, args[0], name)
	if result != 0 {
		return result
	}
	return setPathAttributes64(ctx, resolved, args[2], ownerIDUnused64, ownerIDUnused64)
}

func chown64(ctx *Context64, args [6]uint64) int64 {
	name, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	resolved, ok := resolveGuestPath64(ctx, name)
	if !ok {
		return int64(ENOENT)
	}
	return setOwnerAttributes64(ctx, resolved, args[1], args[2])
}

func lchown64(ctx *Context64, args [6]uint64) int64 {
	return chown64(ctx, args)
}

func fchown64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	file, err := ctx.GetFile(args[0])
	if err != nil || file == nil {
		return int64(EBADF)
	}
	if file.Path == "" {
		return int64(EBADF)
	}
	return setOwnerAttributes64(ctx, file.Path, args[1], args[2])
}

func fchownat64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	flags := args[4]
	if flags&^uint64(atSymlinkNoFollow64|atEmptyPath64) != 0 {
		return int64(EINVAL)
	}
	name, ok := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	var resolved string
	if name == "" && flags&atEmptyPath64 != 0 {
		var result int64
		resolved, result = resolveFDPath64(ctx, args[0])
		if result != 0 {
			return result
		}
	} else {
		if name == "" {
			return int64(ENOENT)
		}
		var result int64
		resolved, result = resolveAtPath64(ctx, args[0], name)
		if result != 0 {
			return result
		}
	}
	return setOwnerAttributes64(ctx, resolved, args[2], args[3])
}

func umask64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(EFAULT)
	}
	old := ctx.Umask & 0o777
	ctx.Umask = uint32(args[0]) & 0o777
	return int64(old)
}
