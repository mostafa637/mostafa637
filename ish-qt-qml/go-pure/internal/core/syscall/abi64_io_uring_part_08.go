package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func ioUringEnter64(ctx *Context64, args [6]uint64) int64 {
	ring, result := ioUringRingFromFD64(ctx, args[0])
	if result != 0 {
		return result
	}
	toSubmit, minComplete, enterFlags := args[1], args[2], args[3]
	if enterFlags&^uint64(ioUringEnterGetEvents64) != 0 || toSubmit > uint64(ring.sqEntries) {
		return int64(EINVAL)
	}
	if ring.sqRingBase == 0 || ring.cqRingBase == 0 || ring.sqesBase == 0 {
		return int64(EINVAL)
	}

	ring.mu.Lock()
	defer ring.mu.Unlock()
	if ring.closed {
		return int64(EBADF)
	}
	sqHead, result := ioUringReadU32(ctx, ring.sqRingBase)
	if result != 0 {
		return result
	}
	sqTail, result := ioUringReadU32(ctx, ring.sqRingBase+4)
	if result != 0 {
		return result
	}
	pending := uint32(sqTail - sqHead)
	if pending > ring.sqEntries {
		return int64(EOVERFLOW)
	}
	if uint64(pending) < toSubmit {
		toSubmit = uint64(pending)
	}
	sqMask, result := ioUringReadU32(ctx, ring.sqRingBase+8)
	if result != 0 {
		return result
	}
	processed := uint64(0)
	for processed < toSubmit {
		slot := (sqHead + uint32(processed)) & sqMask
		arrayEntry, result := ioUringReadU32(ctx, ring.sqRingBase+ioUringRingHeader64+corecpu.Address64(slot*4))
		if result != 0 {
			return result
		}
		if arrayEntry >= ring.sqEntries {
			if queueResult := ioUringQueueCQE64(ctx, ring, 0, int32(EINVAL), 0); queueResult != 0 {
				return queueResult
			}
			processed++
			continue
		}
		sqeAddress := ring.sqesBase + corecpu.Address64(arrayEntry*ioUringSQESize64)
		if !ioUringValidateRange64(ctx, sqeAddress, ioUringSQESize64) {
			return int64(EFAULT)
		}
		var sqe [ioUringSQESize64]byte
		if err := ctx.Memory.Read(sqeAddress, sqe[:]); err != nil {
			return int64(EFAULT)
		}
		userData := binary.LittleEndian.Uint64(sqe[32:40])
		cqeResult := ioUringExecuteSQE64(ctx, ring, sqe)
		if queueResult := ioUringQueueCQE64(ctx, ring, userData, cqeResult, 0); queueResult != 0 {
			return queueResult
		}
		processed++
	}
	if processed > 0 {
		if result := ioUringWriteU32(ctx, ring.sqRingBase, sqHead+uint32(processed)); result != 0 {
			return result
		}
	}
	if enterFlags&ioUringEnterGetEvents64 != 0 && minComplete != 0 {
		for {
			cqHead, headResult := ioUringReadU32(ctx, ring.cqRingBase)
			if headResult != 0 {
				return headResult
			}
			cqTail, tailResult := ioUringReadU32(ctx, ring.cqRingBase+4)
			if tailResult != 0 {
				return tailResult
			}
			if uint32(cqTail-cqHead) >= uint32(minComplete) || ring.closed {
				break
			}
			// This implementation executes supported SQEs synchronously. There
			// is no background SQPOLL worker that could create a future CQE, so
			// do not spin or deadlock when a caller asks for an unavailable event.
			return int64(EAGAIN)
		}
	}
	return int64(processed)
}
