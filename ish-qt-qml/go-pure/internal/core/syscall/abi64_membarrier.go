package syscall

import "sync/atomic"

const (
	membarrierCmdQuery64                    = 0
	membarrierCmdGlobal64                   = 1 << 0
	membarrierCmdGlobalExpedited64          = 1 << 1
	membarrierCmdRegisterGlobalExpedited64  = 1 << 2
	membarrierCmdPrivateExpedited64         = 1 << 3
	membarrierCmdRegisterPrivateExpedited64 = 1 << 4
	membarrierCmdPrivateExpeditedSyncCore64 = 1 << 5
	membarrierCmdRegisterPrivateSyncCore64  = 1 << 6
	membarrierCmdPrivateExpeditedRseq64     = 1 << 7
	membarrierCmdRegisterPrivateRseq64      = 1 << 8
	membarrierCmdGetRegistrations64         = 1 << 9
)

const membarrierSupportedCommands64 = membarrierCmdGlobal64 |
	membarrierCmdGlobalExpedited64 |
	membarrierCmdRegisterGlobalExpedited64 |
	membarrierCmdPrivateExpedited64 |
	membarrierCmdRegisterPrivateExpedited64

func membarrier64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(ENOSYS)
	}
	command := args[0]
	flags := args[1]
	if flags != 0 {
		return int64(EINVAL)
	}

	if command == membarrierCmdQuery64 {
		return int64(membarrierSupportedCommands64)
	}
	if command == membarrierCmdGetRegistrations64 {
		ctx.MembarrierMu.Lock()
		defer ctx.MembarrierMu.Unlock()
		var registered uint64
		if ctx.MembarrierGlobalRegistered {
			registered |= membarrierCmdRegisterGlobalExpedited64
		}
		if ctx.MembarrierPrivateRegistered {
			registered |= membarrierCmdRegisterPrivateExpedited64
		}
		return int64(registered)
	}

	switch command {
	case membarrierCmdRegisterGlobalExpedited64:
		ctx.MembarrierMu.Lock()
		ctx.MembarrierGlobalRegistered = true
		ctx.MembarrierMu.Unlock()
		return 0
	case membarrierCmdRegisterPrivateExpedited64:
		ctx.MembarrierMu.Lock()
		ctx.MembarrierPrivateRegistered = true
		ctx.MembarrierMu.Unlock()
		return 0
	case membarrierCmdGlobal64, membarrierCmdGlobalExpedited64:
		if command == membarrierCmdGlobalExpedited64 && !membarrierGlobalRegistered64(ctx) {
			return int64(EINVAL)
		}
		membarrierFence64(ctx)
		return 0
	case membarrierCmdPrivateExpedited64:
		if !membarrierPrivateRegistered64(ctx) {
			return int64(EINVAL)
		}
		membarrierFence64(ctx)
		return 0
	default:
		return int64(EINVAL)
	}
}

func membarrierGlobalRegistered64(ctx *Context64) bool {
	ctx.MembarrierMu.Lock()
	registered := ctx.MembarrierGlobalRegistered
	ctx.MembarrierMu.Unlock()
	return registered
}

func membarrierPrivateRegistered64(ctx *Context64) bool {
	ctx.MembarrierMu.Lock()
	registered := ctx.MembarrierPrivateRegistered
	ctx.MembarrierMu.Unlock()
	return registered
}

func membarrierFence64(ctx *Context64) {
	// Sequentially consistent atomic RMW is a full memory barrier in Go's
	// memory model and remains available with CGO_ENABLED=0.
	atomic.AddUint64(&ctx.MembarrierEpoch, 1)
}
