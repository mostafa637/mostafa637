package syscall

import (
	"encoding/binary"
	"path"
	"strings"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

const (
	stat64GuestSize     = 96
	atSymlinkNoFollow64 = 0x100
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
	if args[0] != atFDCWD64 {
		return int64(EBADF)
	}
	if args[3]&^uint64(atSymlinkNoFollow64) != 0 {
		return int64(EINVAL)
	}
	name, ok := readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	resolved, ok := resolveGuestPath64(ctx, name)
	if !ok || ctx == nil || ctx.FS == nil {
		return int64(ENOENT)
	}
	var info corefs.FileInfo
	var err error
	if args[3]&atSymlinkNoFollow64 != 0 {
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
