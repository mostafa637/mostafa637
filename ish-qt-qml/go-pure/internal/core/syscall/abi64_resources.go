package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func getrlimit64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[1] == 0 {
		return int64(EFAULT)
	}
	if ctx.RLimits == nil {
		ctx.RLimits = defaultResourceLimits64()
	}
	limit, ok := ctx.RLimits[args[0]]
	if !ok {
		return int64(EINVAL)
	}
	var value [16]byte
	binary.LittleEndian.PutUint64(value[0:8], limit.Cur)
	binary.LittleEndian.PutUint64(value[8:16], limit.Max)
	if err := ctx.Memory.Write(corecpu.Address64(args[1]), value[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func setrlimit64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[1] == 0 {
		return int64(EFAULT)
	}
	if ctx.RLimits == nil {
		ctx.RLimits = defaultResourceLimits64()
	}
	if _, ok := ctx.RLimits[args[0]]; !ok {
		return int64(EINVAL)
	}
	var value [16]byte
	if err := ctx.Memory.Read(corecpu.Address64(args[1]), value[:]); err != nil {
		return int64(EFAULT)
	}
	limit := ResourceLimit64{Cur: binary.LittleEndian.Uint64(value[0:8]), Max: binary.LittleEndian.Uint64(value[8:16])}
	if limit.Cur > limit.Max {
		return int64(EINVAL)
	}
	ctx.RLimits[args[0]] = limit
	return 0
}

func getgroups64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	groups := ctx.Groups
	if groups == nil {
		groups = []uint32{0}
	}
	if args[0] == 0 {
		return int64(len(groups))
	}
	if args[0] < uint64(len(groups)) || args[1] == 0 {
		return int64(EINVAL)
	}
	buffer := make([]byte, len(groups)*4)
	for index, group := range groups {
		binary.LittleEndian.PutUint32(buffer[index*4:], group)
	}
	if err := ctx.Memory.Write(corecpu.Address64(args[1]), buffer); err != nil {
		return int64(EFAULT)
	}
	return int64(len(groups))
}

func setgroups64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if args[0] > 65536 {
		return int64(EINVAL)
	}
	count := int(args[0])
	groups := make([]uint32, count)
	if count > 0 {
		if args[1] == 0 {
			return int64(EFAULT)
		}
		buffer := make([]byte, count*4)
		if err := ctx.Memory.Read(corecpu.Address64(args[1]), buffer); err != nil {
			return int64(EFAULT)
		}
		for index := range groups {
			groups[index] = binary.LittleEndian.Uint32(buffer[index*4:])
		}
	}
	ctx.Groups = groups
	return 0
}

func signalStub64(*Context64, [6]uint64) int64 {
	return int64(ENOSYS)
}

func currentTID64(ctx *Context64) uint64 {
	if ctx.TID != 0 {
		return ctx.TID
	}
	return ctx.PID
}

func tkill64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	if args[0] != currentTID64(ctx) {
		return int64(ESRCH)
	}
	return kill64(ctx, [6]uint64{ctx.PID, args[1]})
}

func tgkill64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	if args[0] != ctx.PID || args[1] != currentTID64(ctx) {
		return int64(ESRCH)
	}
	return kill64(ctx, [6]uint64{ctx.PID, args[2]})
}
