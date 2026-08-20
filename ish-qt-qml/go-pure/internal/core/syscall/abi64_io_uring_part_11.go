package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func ioUringQueueCQE64(ctx *Context64, ring *ioUring64, userData uint64, result int32, flags uint32) int64 {
	cqHead, ret := ioUringReadU32(ctx, ring.cqRingBase)
	if ret != 0 {
		return ret
	}
	cqTail, ret := ioUringReadU32(ctx, ring.cqRingBase+4)
	if ret != 0 {
		return ret
	}
	if uint32(cqTail-cqHead) >= ring.cqEntries {
		overflow, overflowResult := ioUringReadU32(ctx, ring.cqRingBase+16)
		if overflowResult != 0 {
			return overflowResult
		}
		return ioUringWriteU32(ctx, ring.cqRingBase+16, overflow+1)
	}
	mask, ret := ioUringReadU32(ctx, ring.cqRingBase+8)
	if ret != 0 {
		return ret
	}
	slot := cqTail & mask
	entry := ring.cqRingBase + ioUringCQEOffset64 + corecpu.Address64(slot*ioUringCQESize64)
	var cqe [ioUringCQESize64]byte
	binary.LittleEndian.PutUint64(cqe[0:8], userData)
	binary.LittleEndian.PutUint32(cqe[8:12], uint32(result))
	binary.LittleEndian.PutUint32(cqe[12:16], flags)
	if err := ctx.Memory.Write(entry, cqe[:]); err != nil {
		return int64(EFAULT)
	}
	if ret = ioUringWriteU32(ctx, ring.cqRingBase+4, cqTail+1); ret != 0 {
		return ret
	}
	if ring.wake != nil {
		ring.wake.Broadcast()
	}
	return 0
}

func ioUringReleaseMapping64(ctx *Context64, base corecpu.Address64) {
	if ctx == nil {
		return
	}
	for _, mapping := range ctx.Mappings {
		if mapping.Base != base {
			continue
		}
		ring, ok := mapping.Special.(*ioUring64)
		if !ok || ring == nil {
			continue
		}
		ring.mu.Lock()
		switch mapping.Offset {
		case ioUringOffSQRing64:
			ring.sqRingBase = 0
		case ioUringOffCQRing64:
			ring.cqRingBase = 0
		case ioUringOffSQES64:
			ring.sqesBase = 0
		}
		ring.mu.Unlock()
	}
}
