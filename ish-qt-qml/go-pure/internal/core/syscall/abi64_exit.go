package syscall

func exit64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(EFAULT)
	}
	ctx.ExitCode = int32(args[0])
	ctx.Exited = true
	return int64(args[0])
}

func exitGroup64(ctx *Context64, args [6]uint64) int64 {
	return exit64(ctx, args)
}
