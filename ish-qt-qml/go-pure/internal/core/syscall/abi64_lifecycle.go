package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func wait4_64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Children == nil {
		return int64(ECHILD)
	}
	pid, status, err := ctx.Children.Wait(uint32(ctx.PID), int32(args[0]), uint32(args[2]))
	if err != 0 {
		return int64(err)
	}
	if pid == 0 {
		return 0
	}
	if args[1] != 0 {
		if ctx.Memory == nil {
			return int64(EFAULT)
		}
		var encoded [4]byte
		binary.LittleEndian.PutUint32(encoded[:], uint32(status))
		if err := ctx.Memory.Write(corecpu.Address64(args[1]), encoded[:]); err != nil {
			return int64(EFAULT)
		}
	}
	return int64(pid)
}

func setRobustList64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(EFAULT)
	}
	ctx.RobustListHead = args[0]
	ctx.RobustListLen = args[1]
	return 0
}

func getRobustList64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	pid := int64(args[0])
	if pid != 0 && uint64(pid) != ctx.PID {
		return int64(ESRCH)
	}
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], ctx.RobustListHead)
	if err := ctx.Memory.Write(corecpu.Address64(args[1]), raw[:]); err != nil {
		return int64(EFAULT)
	}
	binary.LittleEndian.PutUint64(raw[:], ctx.RobustListLen)
	if err := ctx.Memory.Write(corecpu.Address64(args[2]), raw[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}
