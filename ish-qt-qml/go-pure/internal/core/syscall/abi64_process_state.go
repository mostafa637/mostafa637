package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	prSetPDeathSig64  = 1
	prGetPDeathSig64  = 2
	prGetDumpable64   = 3
	prSetDumpable64   = 4
	prSetName64       = 15
	prGetName64       = 16
	prSetNoNewPrivs64 = 38
	prGetNoNewPrivs64 = 39
	cpuSetSize64      = 8
)

func prctl64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	switch args[0] {
	case prSetPDeathSig64:
		if args[1] > sigMax64 || (args[1] != 0 && (args[1] == sigKill64 || args[1] == sigStop64)) {
			return int64(EINVAL)
		}
		ctx.ParentDeathSig = args[1]
		return 0
	case prGetPDeathSig64:
		return writeU32Guest64(ctx, args[1], uint32(ctx.ParentDeathSig))
	case prGetDumpable64:
		if ctx.Dumpable {
			return 1
		}
		return 0
	case prSetDumpable64:
		if args[1] != 0 && args[1] != 1 {
			return int64(EINVAL)
		}
		ctx.Dumpable = args[1] == 1
		return 0
	case prSetName64:
		if args[1] == 0 {
			return int64(EFAULT)
		}
		var name [16]byte
		if err := ctx.Memory.Read(corecpu.Address64(args[1]), name[:]); err != nil {
			return int64(EFAULT)
		}
		ctx.ProcessName = name
		return 0
	case prGetName64:
		if args[1] == 0 {
			return int64(EFAULT)
		}
		if err := ctx.Memory.Write(corecpu.Address64(args[1]), ctx.ProcessName[:]); err != nil {
			return int64(EFAULT)
		}
		return 0
	case prSetNoNewPrivs64:
		if args[1] != 1 {
			return int64(EINVAL)
		}
		ctx.NoNewPrivs = true
		return 0
	case prGetNoNewPrivs64:
		if ctx.NoNewPrivs {
			return 1
		}
		return 0
	default:
		return int64(ENOSYS)
	}
}

func writeU32Guest64(ctx *Context64, address uint64, value uint32) int64 {
	if ctx == nil || ctx.Memory == nil || address == 0 {
		return int64(EFAULT)
	}
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], value)
	if err := ctx.Memory.Write(corecpu.Address64(address), raw[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func affinityPID64(ctx *Context64, pid uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	if pid != 0 && pid != ctx.PID {
		return int64(ESRCH)
	}
	return 0
}

func schedSetAffinity64(ctx *Context64, args [6]uint64) int64 {
	if result := affinityPID64(ctx, args[0]); result != 0 {
		return result
	}
	if ctx.Memory == nil || args[1] < cpuSetSize64 || args[2] == 0 {
		return int64(EINVAL)
	}
	var raw [8]byte
	if err := ctx.Memory.Read(corecpu.Address64(args[2]), raw[:]); err != nil {
		return int64(EFAULT)
	}
	mask := binary.LittleEndian.Uint64(raw[:])
	if mask == 0 {
		return int64(EINVAL)
	}
	ctx.AffinityMask = mask
	return 0
}

func schedGetAffinity64(ctx *Context64, args [6]uint64) int64 {
	if result := affinityPID64(ctx, args[0]); result != 0 {
		return result
	}
	if ctx.Memory == nil || args[1] < cpuSetSize64 || args[2] == 0 {
		return int64(EINVAL)
	}
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], ctx.AffinityMask)
	if err := ctx.Memory.Write(corecpu.Address64(args[2]), raw[:]); err != nil {
		return int64(EFAULT)
	}
	return int64(cpuSetSize64)
}

func getcpu64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if args[0] != 0 {
		if err := writeU32Guest64(ctx, args[0], 0); err != 0 {
			return err
		}
	}
	if args[1] != 0 {
		if err := writeU32Guest64(ctx, args[1], 0); err != 0 {
			return err
		}
	}
	return 0
}
