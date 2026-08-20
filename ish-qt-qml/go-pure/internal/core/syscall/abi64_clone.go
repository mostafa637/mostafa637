package syscall

import corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

const (
	cloneSignalMask64 uint64 = 0xff
	cloneVM64         uint64 = 0x00000100
	cloneFS64         uint64 = 0x00000200
	cloneFiles64      uint64 = 0x00000400
	cloneSighand64    uint64 = 0x00000800
	cloneVFork64      uint64 = 0x00004000
	cloneParent64     uint64 = 0x00008000
	cloneThread64     uint64 = 0x00010000
	cloneSysvsem64    uint64 = 0x00040000
	cloneSetTLS64     uint64 = 0x00080000
	cloneParentTID64  uint64 = 0x00100000
	cloneChildClear64 uint64 = 0x00200000
	cloneChildTID64   uint64 = 0x01000000
	cloneSupported64  uint64 = cloneSignalMask64 | cloneVM64 | cloneFS64 | cloneFiles64 | cloneSighand64 | cloneVFork64 | cloneParent64 | cloneThread64 | cloneSysvsem64 | cloneSetTLS64 | cloneParentTID64 | cloneChildClear64 | cloneChildTID64
)

// Exported flag aliases used by the session runtime when materializing a
// child process from a validated CloneRequest64.
const (
	CloneVMFlag64       = cloneVM64
	CloneFilesFlag64    = cloneFiles64
	CloneSetTLSFlag64   = cloneSetTLS64
	CloneChildTIDFlag64 = cloneChildTID64
)

func validateCloneFlags64(flags uint64, childStack, parentTID, childTID uint64, fork, vfork bool) int64 {
	if flags&^cloneSupported64 != 0 {
		return int64(EINVAL)
	}
	if flags&cloneSighand64 != 0 && flags&cloneVM64 == 0 {
		return int64(EINVAL)
	}
	if flags&cloneThread64 != 0 && flags&(cloneVM64|cloneSighand64) != (cloneVM64|cloneSighand64) {
		return int64(EINVAL)
	}
	if flags&cloneParentTID64 != 0 && parentTID == 0 {
		return int64(EFAULT)
	}
	if flags&cloneChildTID64 != 0 && childTID == 0 {
		return int64(EFAULT)
	}
	if (fork || vfork) && (childStack != 0 || parentTID != 0 || childTID != 0) {
		return int64(EINVAL)
	}
	return 0
}

func cloneWithFactory64(ctx *Context64, request CloneRequest64) int64 {
	if ctx == nil {
		return int64(EFAULT)
	}
	if result := validateCloneFlags64(request.Flags, request.ChildStack, request.ParentTID, request.ChildTID, request.Fork, request.VFork); result != 0 {
		return result
	}
	if ctx.ProcessFactory == nil {
		return int64(ENOSYS)
	}
	if request.Flags&(cloneParentTID64|cloneChildTID64) != 0 {
		if ctx.Memory == nil {
			return int64(EFAULT)
		}
		var probe [8]byte
		if request.Flags&cloneParentTID64 != 0 {
			if err := ctx.Memory.Read(corecpu.Address64(request.ParentTID), probe[:]); err != nil {
				return int64(EFAULT)
			}
		}
		if request.Flags&cloneChildTID64 != 0 {
			if err := ctx.Memory.Read(corecpu.Address64(request.ChildTID), probe[:]); err != nil {
				return int64(EFAULT)
			}
		}
	}
	childPID := ctx.ProcessFactory(ctx, request)
	if childPID <= 0 {
		return childPID
	}
	if ctx.Children == nil || !ctx.Children.AddChild(uint32(ctx.PID), uint32(childPID)) {
		return int64(EAGAIN)
	}
	if request.Flags&cloneParentTID64 != 0 {
		if ctx.Memory == nil {
			return int64(EFAULT)
		}
		var raw [8]byte
		for i := range raw {
			raw[i] = byte(uint64(childPID) >> (8 * i))
		}
		if err := ctx.Memory.Write(corecpu.Address64(request.ParentTID), raw[:]); err != nil {
			return int64(EFAULT)
		}
	}
	if ctx.ChildStarter != nil {
		ctx.ChildStarter(ctx, childPID, request)
	}
	return childPID
}

func clone64(ctx *Context64, args [6]uint64) int64 {
	return cloneWithFactory64(ctx, CloneRequest64{
		Flags:      args[0],
		ChildStack: args[1],
		ParentTID:  args[2],
		ChildTID:   args[3],
		TLS:        args[4],
	})
}

func fork64(ctx *Context64, args [6]uint64) int64 {
	return cloneWithFactory64(ctx, CloneRequest64{Fork: true})
}

func vfork64(ctx *Context64, args [6]uint64) int64 {
	return cloneWithFactory64(ctx, CloneRequest64{VFork: true})
}
