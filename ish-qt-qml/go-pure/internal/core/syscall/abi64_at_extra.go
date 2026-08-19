package syscall

import "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

func readlinkat64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	name, ok := readGuestString64(ctx, cpu.Address64(args[1]), 4096)
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
	target, err := ctx.FS.Readlink(resolved)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	if args[3] == 0 {
		return 0
	}
	data := []byte(target)
	if uint64(len(data)) > args[3] {
		data = data[:int(args[3])]
	}
	if err := ctx.Memory.Write(cpu.Address64(args[2]), data); err != nil {
		return int64(EFAULT)
	}
	return int64(len(data))
}

func mkdirat64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	name, ok := readGuestString64(ctx, cpu.Address64(args[1]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	if name == "" {
		return int64(ENOENT)
	}
	if args[2] > uint64(^uint32(0)) {
		return int64(EINVAL)
	}
	resolved, result := resolveAtPath64(ctx, args[0], name)
	if result != 0 {
		return result
	}
	if err := ctx.FS.Mkdir(resolved, uint32(args[2]), 0, 0); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}
