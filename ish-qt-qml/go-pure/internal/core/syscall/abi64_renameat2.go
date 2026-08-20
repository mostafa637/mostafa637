package syscall

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	renameNoReplace64 uint64 = 1
	renameExchange64  uint64 = 2
	renameWhiteout64  uint64 = 4
	renameFlags64            = renameNoReplace64 | renameExchange64 | renameWhiteout64
)

func renameat264(ctx *Context64, args [6]uint64) int64 {
	flags := args[4]
	if flags&^renameFlags64 != 0 || flags&(renameNoReplace64|renameExchange64) == (renameNoReplace64|renameExchange64) || flags&(renameExchange64|renameWhiteout64) == (renameExchange64|renameWhiteout64) {
		return int64(EINVAL)
	}
	if flags&renameWhiteout64 != 0 {
		return int64(EOPNOTSUPP)
	}
	oldName, ok := readGuestString64(ctx, cpu.Address64(args[1]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	newName, ok := readGuestString64(ctx, cpu.Address64(args[3]), 4096)
	if !ok {
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
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	var err error
	switch {
	case flags&renameExchange64 != 0:
		err = ctx.FS.RenameExchange(oldPath, newPath)
	case flags&renameNoReplace64 != 0:
		err = ctx.FS.RenameNoReplace(oldPath, newPath)
	default:
		err = ctx.FS.Rename(oldPath, newPath)
	}
	if err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}
