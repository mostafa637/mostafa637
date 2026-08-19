package syscall

import corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

const guestInvalidID64 uint32 = ^uint32(0)

func currentPIDForGroups64(ctx *Context64) uint64 {
	if ctx == nil {
		return 0
	}
	return ctx.PID
}

func currentPGID64(ctx *Context64) uint64 {
	if ctx == nil {
		return 0
	}
	if ctx.PGID != 0 {
		return ctx.PGID
	}
	return currentPIDForGroups64(ctx)
}

func currentSID64(ctx *Context64) uint64 {
	if ctx == nil {
		return 0
	}
	if ctx.SID != 0 {
		return ctx.SID
	}
	return currentPIDForGroups64(ctx)
}

func getuid64(ctx *Context64, _ [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	return int64(ctx.RealUID)
}

func geteuid64(ctx *Context64, _ [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	return int64(ctx.EffectiveUID)
}

func getgid64(ctx *Context64, _ [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	return int64(ctx.RealGID)
}

func getegid64(ctx *Context64, _ [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	return int64(ctx.EffectiveGID)
}

func setuid64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	uid := uint32(args[0])
	if ctx.EffectiveUID == 0 {
		ctx.RealUID = uid
		ctx.EffectiveUID = uid
		ctx.SavedUID = uid
		return 0
	}
	if uid != ctx.RealUID && uid != ctx.SavedUID {
		return int64(EPERM)
	}
	ctx.EffectiveUID = uid
	return 0
}

func setgid64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	gid := uint32(args[0])
	if ctx.EffectiveUID == 0 {
		ctx.RealGID = gid
		ctx.EffectiveGID = gid
		ctx.SavedGID = gid
		return 0
	}
	if gid != ctx.RealGID && gid != ctx.SavedGID {
		return int64(EPERM)
	}
	ctx.EffectiveGID = gid
	return 0
}

func idRequested64(value uint64) (uint32, bool) {
	id := uint32(value)
	return id, id != guestInvalidID64
}

func allowedUnprivilegedID64(value uint32, real, effective, saved uint32) bool {
	return value == guestInvalidID64 || value == real || value == effective || value == saved
}

func setresuid64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	real, realSet := idRequested64(args[0])
	effective, effectiveSet := idRequested64(args[1])
	saved, savedSet := idRequested64(args[2])
	if ctx.EffectiveUID != 0 && (!allowedUnprivilegedID64(real, ctx.RealUID, ctx.EffectiveUID, ctx.SavedUID) ||
		!allowedUnprivilegedID64(effective, ctx.RealUID, ctx.EffectiveUID, ctx.SavedUID) ||
		!allowedUnprivilegedID64(saved, ctx.RealUID, ctx.EffectiveUID, ctx.SavedUID)) {
		return int64(EPERM)
	}
	if realSet {
		ctx.RealUID = real
	}
	if effectiveSet {
		ctx.EffectiveUID = effective
	}
	if savedSet {
		ctx.SavedUID = saved
	}
	return 0
}

func setresgid64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	real, realSet := idRequested64(args[0])
	effective, effectiveSet := idRequested64(args[1])
	saved, savedSet := idRequested64(args[2])
	if ctx.EffectiveUID != 0 && (!allowedUnprivilegedID64(real, ctx.RealGID, ctx.EffectiveGID, ctx.SavedGID) ||
		!allowedUnprivilegedID64(effective, ctx.RealGID, ctx.EffectiveGID, ctx.SavedGID) ||
		!allowedUnprivilegedID64(saved, ctx.RealGID, ctx.EffectiveGID, ctx.SavedGID)) {
		return int64(EPERM)
	}
	if realSet {
		ctx.RealGID = real
	}
	if effectiveSet {
		ctx.EffectiveGID = effective
	}
	if savedSet {
		ctx.SavedGID = saved
	}
	return 0
}

func setIDVector64(ctx *Context64, address uint64, values ...uint32) int64 {
	if ctx == nil || ctx.Memory == nil || address == 0 {
		return int64(EFAULT)
	}
	buffer := make([]byte, len(values)*4)
	for index, value := range values {
		buffer[index*4] = byte(value)
		buffer[index*4+1] = byte(value >> 8)
		buffer[index*4+2] = byte(value >> 16)
		buffer[index*4+3] = byte(value >> 24)
	}
	if err := ctx.Memory.Write(corecpu.Address64(address), buffer); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func getresuid64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	return setIDVector64(ctx, args[0], ctx.RealUID, ctx.EffectiveUID, ctx.SavedUID)
}

func getresgid64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	return setIDVector64(ctx, args[0], ctx.RealGID, ctx.EffectiveGID, ctx.SavedGID)
}

func setpgid64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	pid := args[0]
	if pid == 0 {
		pid = currentPIDForGroups64(ctx)
	}
	if pid != currentPIDForGroups64(ctx) {
		return int64(ESRCH)
	}
	pgid := args[1]
	if pgid == 0 {
		pgid = pid
	}
	if pgid == 0 {
		return int64(EINVAL)
	}
	ctx.PGID = pgid
	return 0
}

func getpgid64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	pid := args[0]
	if pid != 0 && pid != currentPIDForGroups64(ctx) {
		return int64(ESRCH)
	}
	return int64(currentPGID64(ctx))
}

func setsid64(ctx *Context64, _ [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	pid := currentPIDForGroups64(ctx)
	if pid == 0 {
		return int64(ESRCH)
	}
	if ctx.PGID != 0 && ctx.PGID == pid {
		return int64(EPERM)
	}
	ctx.PGID = pid
	ctx.SID = pid
	return int64(pid)
}

func getsid64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ESRCH)
	}
	pid := args[0]
	if pid != 0 && pid != currentPIDForGroups64(ctx) {
		return int64(ESRCH)
	}
	return int64(currentSID64(ctx))
}
