package syscall

import corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

const uname64FieldSize = 65

func chdir64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	name, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	path, ok := resolveGuestPath64(ctx, name)
	if !ok {
		return int64(ENOENT)
	}
	info, err := ctx.FS.Stat(path)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	if !info.IsDir() {
		return int64(ENOTDIR)
	}
	ctx.CWD = path
	return 0
}

func getcwd64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[0] == 0 || args[1] == 0 {
		return int64(EFAULT)
	}
	cwd := ctx.CWD
	if cwd == "" {
		cwd = "/"
	}
	data := append([]byte(cwd), 0)
	if uint64(len(data)) > args[1] {
		return int64(ENAMETOOLONG)
	}
	if err := ctx.Memory.Write(corecpu.Address64(args[0]), data); err != nil {
		return int64(EFAULT)
	}
	return int64(len(data))
}

func readlink64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	pathName, ok := readGuestString64(ctx, corecpu.Address64(args[0]), 4096)
	if !ok {
		return int64(EFAULT)
	}
	path, ok := resolveGuestPath64(ctx, pathName)
	if !ok {
		return int64(ENOENT)
	}
	target, err := ctx.FS.Readlink(path)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	if args[2] == 0 {
		return 0
	}
	data := []byte(target)
	if uint64(len(data)) > args[2] {
		data = data[:int(args[2])]
	}
	if err := ctx.Memory.Write(corecpu.Address64(args[1]), data); err != nil {
		return int64(EFAULT)
	}
	return int64(len(data))
}

func uname64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[0] == 0 {
		return int64(EFAULT)
	}
	fields := [...]string{
		"Linux",
		"ish-go",
		"6.1.0-ish",
		"#1 Pure Go",
		"x86_64",
		"(none)",
	}
	buffer := make([]byte, uname64FieldSize*len(fields))
	for index, value := range fields {
		copy(buffer[index*uname64FieldSize:(index+1)*uname64FieldSize-1], value)
	}
	if err := ctx.Memory.Write(corecpu.Address64(args[0]), buffer); err != nil {
		return int64(EFAULT)
	}
	return 0
}
