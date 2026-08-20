package syscall

import corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

func ioUringPublishMap64(ctx *Context64, ring *ioUring64, kind ioUringMapKind64, base corecpu.Address64, length uint64) int64 {
	ring.mu.Lock()
	defer ring.mu.Unlock()
	if ioUringMapBase64(ring, kind) != 0 {
		_ = ctx.Memory.UnmapAlways(base, length)
		ctx.removeMappings64(base, length)
		return int64(EEXIST)
	}
	ioUringSetMapBase64(ring, kind, base)
	return int64(base)
}

func ioUringMapBase64(ring *ioUring64, kind ioUringMapKind64) corecpu.Address64 {
	_, _, base := ioUringMapSlot64(ring, mapKindOffset64(kind))
	return base
}

func ioUringSetMapBase64(ring *ioUring64, kind ioUringMapKind64, base corecpu.Address64) {
	switch kind {
	case ioUringMapSQ64:
		ring.sqRingBase = base
	case ioUringMapCQ64:
		ring.cqRingBase = base
	case ioUringMapSQE64:
		ring.sqesBase = base
	}
}

func mapKindOffset64(kind ioUringMapKind64) uint64 {
	if kind == ioUringMapCQ64 {
		return ioUringOffCQRing64
	}
	if kind == ioUringMapSQE64 {
		return ioUringOffSQES64
	}
	return ioUringOffSQRing64
}

func firstIoUringError64(result, fallback int64) int64 {
	if result != 0 {
		return result
	}
	return fallback
}
