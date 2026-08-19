package syscall

import (
	"encoding/binary"
	"path"
	"strings"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

const (
	openHowSize64 = 24
	openHowMax64  = 4096

	resolveNoXdev64     uint64 = 0x01
	resolveNoMagic64    uint64 = 0x02
	resolveNoSymlinks64 uint64 = 0x04
	resolveBeneath64    uint64 = 0x08
	resolveInRoot64     uint64 = 0x10
	resolveCached64     uint64 = 0x20
	resolveKnownFlags64        = resolveNoXdev64 | resolveNoMagic64 | resolveNoSymlinks64 | resolveBeneath64 | resolveInRoot64 | resolveCached64
)

func readOpenHow64(ctx *Context64, address, size uint64) (flags, mode, resolve uint64, result int64) {
	if size < openHowSize64 {
		return 0, 0, 0, int64(EINVAL)
	}
	if size > openHowMax64 {
		return 0, 0, 0, int64(E2BIG)
	}
	buffer := make([]byte, int(size))
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Read(corecpu.Address64(address), buffer) != nil {
		return 0, 0, 0, int64(EFAULT)
	}
	for _, value := range buffer[openHowSize64:] {
		if value != 0 {
			return 0, 0, 0, int64(E2BIG)
		}
	}
	return binary.LittleEndian.Uint64(buffer[0:8]), binary.LittleEndian.Uint64(buffer[8:16]), binary.LittleEndian.Uint64(buffer[16:24]), 0
}

func pathUnderOpenRoot64(base, name string, clamp bool) (string, int64) {
	base = path.Clean(base)
	if base == "" || !strings.HasPrefix(base, "/") {
		return "", int64(EINVAL)
	}
	parts := make([]string, 0)
	for _, part := range strings.Split(strings.TrimPrefix(name, "/"), "/") {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(parts) == 0 {
				if clamp {
					continue
				}
				return "", int64(EXDEV)
			}
			parts = parts[:len(parts)-1]
		default:
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return base, 0
	}
	return path.Join(append([]string{base}, parts...)...), 0
}

func resolveOpenat2Path64(ctx *Context64, dirfd uint64, name string, resolve uint64) (string, int64) {
	if resolve&resolveCached64 != 0 {
		return "", int64(EAGAIN)
	}
	base, result := resolveFDPath64(ctx, dirfd)
	if result != 0 {
		return "", result
	}
	if resolve&resolveInRoot64 != 0 {
		return pathUnderOpenRoot64(base, name, true)
	}
	if resolve&resolveBeneath64 != 0 {
		if strings.HasPrefix(name, "/") {
			return "", int64(EXDEV)
		}
		return pathUnderOpenRoot64(base, name, false)
	}
	return resolveAtPath64(ctx, dirfd, name)
}

func rejectSymlinkComponents64(ctx *Context64, resolved string) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	clean := path.Clean(resolved)
	if clean == "/" {
		return 0
	}
	current := ""
	for _, part := range strings.Split(strings.TrimPrefix(clean, "/"), "/") {
		current = path.Join(current, "/", part)
		info, err := ctx.FS.Lstat(current)
		if err != nil {
			// Let openResolvedPath64 report ENOENT or another parent error. A
			// missing final component is valid with O_CREAT.
			return 0
		}
		if info.Mode.Mode&corefs.ModeTypeMask == corefs.ModeSymlink {
			return int64(ELOOP)
		}
	}
	return 0
}

func openat264(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FS == nil || ctx.FDs == nil {
		return int64(ENOSYS)
	}
	name, ok := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	if name == "" {
		return int64(ENOENT)
	}
	flags, mode, resolve, result := readOpenHow64(ctx, args[2], args[3])
	if result != 0 {
		return result
	}
	if resolve&^resolveKnownFlags64 != 0 {
		return int64(EINVAL)
	}
	if mode&^uint64(0o7777) != 0 {
		return int64(EINVAL)
	}
	if mode != 0 && flags&guestOpenCreat == 0 {
		return int64(EINVAL)
	}
	if flags > uint64(^uint32(0)) {
		return int64(EINVAL)
	}
	resolved, result := resolveOpenat2Path64(ctx, args[0], name, resolve)
	if result != 0 {
		return result
	}
	if resolve&resolveNoSymlinks64 != 0 {
		if result = rejectSymlinkComponents64(ctx, resolved); result != 0 {
			return result
		}
	}
	if resolve&resolveBeneath64 != 0 && resolve&resolveNoSymlinks64 == 0 {
		// The fakefs has no mount points or magic links. Refusing symlink
		// components here preserves the safety guarantee of BENEATH rather
		// than allowing a host symlink to escape the dirfd root.
		if result = rejectSymlinkComponents64(ctx, resolved); result == int64(ELOOP) {
			return int64(EXDEV)
		}
	}
	return openResolvedPath64(ctx, resolved, flags, mode)
}
