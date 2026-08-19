package syscall

import (
	"encoding/binary"
	"path"
	"strings"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

const (
	stat64GuestSize     = 96
	statxGuestSize      = 256
	atSymlinkNoFollow64 = 0x100
	atEmptyPath64       = 0x1000
	atNoAutomount64     = 0x800
	atStatxForceSync64  = 0x2000
	atStatxDontSync64   = 0x4000
)

func resolveGuestPath64(ctx *Context64, name string) (string, bool) {
	if ctx == nil || name == "" {
		return "", false
	}
	if strings.HasPrefix(name, "/") {
		return path.Clean(name), true
	}
	cwd := ctx.CWD
	if cwd == "" {
		cwd = "/"
	}
	return path.Join(cwd, name), true
}

func writeStat64Guest(ctx *Context64, address corecpu.Address64, info corefs.FileInfo) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	buffer := make([]byte, stat64GuestSize)
	binary.LittleEndian.PutUint64(buffer[0:8], info.Inode)
	binary.LittleEndian.PutUint32(buffer[16:20], info.Mode.Mode)
	if info.IsDir() {
		binary.LittleEndian.PutUint32(buffer[20:24], 2)
	} else {
		binary.LittleEndian.PutUint32(buffer[20:24], 1)
	}
	binary.LittleEndian.PutUint32(buffer[24:28], info.Mode.UID)
	binary.LittleEndian.PutUint32(buffer[28:32], info.Mode.GID)
	binary.LittleEndian.PutUint64(buffer[32:40], uint64(info.Mode.Rdev))
	binary.LittleEndian.PutUint64(buffer[48:56], uint64(info.Size))
	binary.LittleEndian.PutUint32(buffer[56:60], 4096)
	binary.LittleEndian.PutUint64(buffer[64:72], uint64((info.Size+511)/512))
	seconds := info.ModTime.Unix()
	nanos := uint32(info.ModTime.Nanosecond())
	binary.LittleEndian.PutUint64(buffer[72:80], uint64(seconds))
	binary.LittleEndian.PutUint64(buffer[80:88], uint64(nanos))
	binary.LittleEndian.PutUint64(buffer[88:96], info.Inode)
	if err := ctx.Memory.Write(address, buffer); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func stat64Guest(ctx *Context64, args [6]uint64) int64 {
	name, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	resolved, ok := resolveGuestPath64(ctx, name)
	if !ok || ctx == nil || ctx.FS == nil {
		return int64(ENOENT)
	}
	info, err := ctx.FS.Stat(resolved)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	return writeStat64Guest(ctx, corecpu.Address64(args[1]), info)
}

func fstat64Guest(ctx *Context64, args [6]uint64) int64 {
	file, err := ctx.GetFile(args[0])
	if err != nil || file == nil {
		return int64(EBADF)
	}
	if ctx.FS == nil || file.Path == "" {
		return int64(ENOTTY)
	}
	info, err := ctx.FS.Stat(file.Path)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	return writeStat64Guest(ctx, corecpu.Address64(args[1]), info)
}

func fstatat64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	flags := args[3]
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
	var info corefs.FileInfo
	var err error
	if flags&atSymlinkNoFollow64 != 0 {
		info, err = ctx.FS.Lstat(resolved)
	} else {
		info, err = ctx.FS.Stat(resolved)
	}
	if err != nil {
		return int64(errnoForOpen(err))
	}
	return writeStat64Guest(ctx, corecpu.Address64(args[2]), info)
}

func mkdir64Guest(ctx *Context64, args [6]uint64) int64 {
	name, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	resolved, ok := resolveGuestPath64(ctx, name)
	if !ok || ctx == nil || ctx.FS == nil {
		return int64(ENOENT)
	}
	if args[1] > uint64(^uint32(0)) {
		return int64(EINVAL)
	}
	if err := ctx.FS.Mkdir(resolved, uint32(args[1]), 0, 0); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func rmdir64Guest(ctx *Context64, args [6]uint64) int64 {
	name, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	resolved, ok := resolveGuestPath64(ctx, name)
	if !ok || ctx == nil || ctx.FS == nil {
		return int64(ENOENT)
	}
	info, err := ctx.FS.Stat(resolved)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	if !info.IsDir() {
		return int64(ENOTDIR)
	}
	if err := ctx.FS.Unlink(resolved); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func unlink64Guest(ctx *Context64, args [6]uint64) int64 {
	name, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	resolved, ok := resolveGuestPath64(ctx, name)
	if !ok || ctx == nil || ctx.FS == nil {
		return int64(ENOENT)
	}
	if err := ctx.FS.Unlink(resolved); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func rename64Guest(ctx *Context64, args [6]uint64) int64 {
	oldName, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	newName, ok := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	oldPath, oldOK := resolveGuestPath64(ctx, oldName)
	newPath, newOK := resolveGuestPath64(ctx, newName)
	if !oldOK || !newOK || ctx == nil || ctx.FS == nil {
		return int64(ENOENT)
	}
	if err := ctx.FS.Rename(oldPath, newPath); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

const (
	atRemovedir64     = 0x200
	atSymlinkFollow64 = 0x400
	statxType64       = 0x00000001
	statxMode64       = 0x00000002
	statxNlink64      = 0x00000004
	statxUID64        = 0x00000008
	statxGID64        = 0x00000010
	statxAtime64      = 0x00000020
	statxMtime64      = 0x00000040
	statxCtime64      = 0x00000080
	statxIno64        = 0x00000100
	statxSize64       = 0x00000200
	statxBlocks64     = 0x00000400
	statxBasicStats64 = statxType64 | statxMode64 | statxNlink64 | statxUID64 | statxGID64 | statxAtime64 | statxMtime64 | statxCtime64 | statxIno64 | statxSize64 | statxBlocks64
)

func resolveFDPath64(ctx *Context64, dirfd uint64) (string, int64) {
	if ctx == nil || ctx.FS == nil {
		return "", int64(ENOSYS)
	}
	if dirfd == atFDCWD64 {
		cwd := ctx.CWD
		if cwd == "" {
			cwd = "/"
		}
		return path.Clean(cwd), 0
	}
	file, err := ctx.GetFile(dirfd)
	if err != nil || file == nil {
		return "", int64(EBADF)
	}
	if file.Path == "" {
		return "", int64(ENOTTY)
	}
	return path.Clean(file.Path), 0
}

func resolveAtPath64(ctx *Context64, dirfd uint64, name string) (string, int64) {
	if ctx == nil || ctx.FS == nil {
		return "", int64(ENOSYS)
	}
	if name == "" {
		return "", int64(ENOENT)
	}
	if strings.HasPrefix(name, "/") {
		return path.Clean(name), 0
	}
	base, result := resolveFDPath64(ctx, dirfd)
	if result != 0 {
		return "", result
	}
	if dirfd != atFDCWD64 {
		info, err := ctx.FS.Stat(base)
		if err != nil {
			return "", int64(errnoForOpen(err))
		}
		if !info.IsDir() {
			return "", int64(ENOTDIR)
		}
	}
	return path.Join(base, name), 0
}

func writeStatxTimestamp64(buffer []byte, offset int, value time.Time) {
	if value.IsZero() {
		return
	}
	binary.LittleEndian.PutUint64(buffer[offset:offset+8], uint64(value.Unix()))
	binary.LittleEndian.PutUint32(buffer[offset+8:offset+12], uint32(value.Nanosecond()))
}

func deviceMajorMinor64(device uint32) (uint32, uint32) {
	major := (device >> 8) & 0xfff
	minor := device & 0xff
	minor |= (device >> 12) & 0xffffff00
	return major, minor
}

func writeStatx64Guest(ctx *Context64, address corecpu.Address64, info corefs.FileInfo) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	buffer := make([]byte, statxGuestSize)
	binary.LittleEndian.PutUint32(buffer[0:4], statxBasicStats64)
	binary.LittleEndian.PutUint32(buffer[4:8], 4096)
	binary.LittleEndian.PutUint32(buffer[16:20], 1)
	if info.IsDir() {
		binary.LittleEndian.PutUint32(buffer[16:20], 2)
	}
	binary.LittleEndian.PutUint32(buffer[20:24], info.Mode.UID)
	binary.LittleEndian.PutUint32(buffer[24:28], info.Mode.GID)
	binary.LittleEndian.PutUint16(buffer[28:30], uint16(info.Mode.Mode))
	binary.LittleEndian.PutUint64(buffer[32:40], info.Inode)
	var size uint64
	if info.Size > 0 {
		size = uint64(info.Size)
	}
	binary.LittleEndian.PutUint64(buffer[40:48], size)
	binary.LittleEndian.PutUint64(buffer[48:56], (size+511)/512)
	writeStatxTimestamp64(buffer, 64, info.ModTime)
	writeStatxTimestamp64(buffer, 96, info.ModTime)
	writeStatxTimestamp64(buffer, 112, info.ModTime)
	writeStatxTimestamp64(buffer, 80, time.Time{})
	rdevMajor, rdevMinor := deviceMajorMinor64(info.Mode.Rdev)
	binary.LittleEndian.PutUint32(buffer[128:132], rdevMajor)
	binary.LittleEndian.PutUint32(buffer[132:136], rdevMinor)
	if err := ctx.Memory.Write(address, buffer); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func statx64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil || ctx.Memory == nil {
		return int64(ENOSYS)
	}
	flags := args[2]
	allowed := uint64(atSymlinkNoFollow64 | atEmptyPath64 | atNoAutomount64 | atStatxForceSync64 | atStatxDontSync64)
	if flags&^allowed != 0 || flags&atStatxForceSync64 != 0 && flags&atStatxDontSync64 != 0 {
		return int64(EINVAL)
	}
	var name string
	var ok bool
	if args[1] == 0 && flags&atEmptyPath64 != 0 {
		name = ""
		ok = true
	} else {
		name, ok = readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	}
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
	var info corefs.FileInfo
	var err error
	if flags&atSymlinkNoFollow64 != 0 {
		info, err = ctx.FS.Lstat(resolved)
	} else {
		info, err = ctx.FS.Stat(resolved)
	}
	if err != nil {
		return int64(errnoForOpen(err))
	}
	return writeStatx64Guest(ctx, corecpu.Address64(args[4]), info)
}

func unlinkat64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	if args[2]&^uint64(atRemovedir64) != 0 {
		return int64(EINVAL)
	}
	name, ok := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	if !ok || name == "" {
		return int64(EFAULT)
	}
	resolved, result := resolveAtPath64(ctx, args[0], name)
	if result != 0 {
		return result
	}
	info, err := ctx.FS.Stat(resolved)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	if args[2]&atRemovedir64 != 0 && !info.IsDir() {
		return int64(ENOTDIR)
	}
	if args[2]&atRemovedir64 == 0 && info.IsDir() {
		return int64(EINVAL)
	}
	if err := ctx.FS.Unlink(resolved); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func renameat64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	oldName, ok := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	if !ok || oldName == "" {
		return int64(EFAULT)
	}
	newName, ok := readGuestString64(ctx, corecpu.Address64(args[3]), 4096)
	if !ok || newName == "" {
		return int64(EFAULT)
	}
	oldPath, result := resolveAtPath64(ctx, args[0], oldName)
	if result != 0 {
		return result
	}
	newPath, result := resolveAtPath64(ctx, args[2], newName)
	if result != 0 {
		return result
	}
	if err := ctx.FS.Rename(oldPath, newPath); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func linkat64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	if args[4]&^uint64(atEmptyPath64|atSymlinkFollow64) != 0 {
		return int64(EINVAL)
	}
	oldName, ok := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	var oldPath string
	if oldName == "" && args[4]&atEmptyPath64 != 0 {
		var result int64
		oldPath, result = resolveFDPath64(ctx, args[0])
		if result != 0 {
			return result
		}
	} else {
		if oldName == "" {
			return int64(ENOENT)
		}
		var result int64
		oldPath, result = resolveAtPath64(ctx, args[0], oldName)
		if result != 0 {
			return result
		}
	}
	newName, ok := readGuestString64(ctx, corecpu.Address64(args[3]), 4096)
	if !ok || newName == "" {
		return int64(EFAULT)
	}
	newPath, result := resolveAtPath64(ctx, args[2], newName)
	if result != 0 {
		return result
	}
	if err := ctx.FS.Link(oldPath, newPath); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func symlinkat64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	target, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	name, ok := readGuestString64(ctx, corecpu.Address64(args[2]), 4096)
	if !ok || name == "" {
		return int64(EFAULT)
	}
	resolved, result := resolveAtPath64(ctx, args[1], name)
	if result != 0 {
		return result
	}
	if err := ctx.FS.Symlink(target, resolved, 0, 0); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func prlimit64_64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if args[0] != 0 && args[0] != ctx.PID {
		return int64(ESRCH)
	}
	if ctx.RLimits == nil {
		ctx.RLimits = defaultResourceLimits64()
	}
	resource := args[1]
	old, ok := ctx.RLimits[resource]
	if !ok {
		return int64(EINVAL)
	}
	if args[2] != 0 {
		var value [16]byte
		if err := ctx.Memory.Read(corecpu.Address64(args[2]), value[:]); err != nil {
			return int64(EFAULT)
		}
		limit := ResourceLimit64{Cur: binary.LittleEndian.Uint64(value[0:8]), Max: binary.LittleEndian.Uint64(value[8:16])}
		if limit.Cur > limit.Max {
			return int64(EINVAL)
		}
		ctx.RLimits[resource] = limit
	}
	if args[3] != 0 {
		var value [16]byte
		binary.LittleEndian.PutUint64(value[0:8], old.Cur)
		binary.LittleEndian.PutUint64(value[8:16], old.Max)
		if err := ctx.Memory.Write(corecpu.Address64(args[3]), value[:]); err != nil {
			return int64(EFAULT)
		}
	}
	return 0
}
