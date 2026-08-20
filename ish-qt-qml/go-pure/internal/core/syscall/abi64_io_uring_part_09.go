package syscall

import (
	"encoding/binary"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func ioUringExecuteSQE64(ctx *Context64, ring *ioUring64, sqe [ioUringSQESize64]byte) int32 {
	opcode := sqe[0]
	flags := sqe[1]
	fd := int32(binary.LittleEndian.Uint32(sqe[4:8]))
	offset := binary.LittleEndian.Uint64(sqe[8:16])
	address := corecpu.Address64(binary.LittleEndian.Uint64(sqe[16:24]))
	length := uint64(binary.LittleEndian.Uint32(sqe[24:28]))
	fsyncFlags := binary.LittleEndian.Uint32(sqe[28:32])

	switch opcode {
	case ioUringOpNOP64:
		return 0
	case ioUringOpReadV64, ioUringOpWriteV64:
		return ioUringExecuteVector64(ctx, ring, fd, flags, address, length, offset, opcode == ioUringOpWriteV64)
	case ioUringOpRead64, ioUringOpWrite64:
		if length > 16<<20 || length > uint64(^uint(0)>>1) || !ioUringValidateRange64(ctx, address, length) {
			return EFAULT
		}
		file, result := ioUringResolveFile64(ctx, ring, fd, flags&ioUringSQEFixedFile64 != 0)
		if result != 0 {
			return int32(result)
		}
		buffer := make([]byte, int(length))
		if opcode == ioUringOpWrite64 {
			if err := ctx.Memory.Read(address, buffer); err != nil {
				return EFAULT
			}
			n, err := ioUringWriteAt64(file, buffer, offset)
			if n > 0 {
				return int32(n)
			}
			if err != nil {
				return ioUringErrno64(err)
			}
			return 0
		}
		n, err := ioUringReadAt64(file, buffer, offset)
		if n > 0 {
			if writeErr := ctx.Memory.Write(address, buffer[:n]); writeErr != nil {
				return EFAULT
			}
			return int32(n)
		}
		if err != nil {
			return ioUringErrno64(err)
		}
		return 0
	case ioUringOpPollAdd64:
		file, result := ioUringResolveFile64(ctx, ring, fd, flags&ioUringSQEFixedFile64 != 0)
		if result != 0 {
			return int32(result)
		}
		pollMask := binary.LittleEndian.Uint32(sqe[28:32])
		if file.Poll == nil {
			return EOPNOTSUPP
		}
		return int32(file.Poll(uint16(pollMask)))
	case ioUringOpPollRemove64:
		// Supported polls complete synchronously in this implementation, so
		// there is no pending request left to remove.
		return ENOENT
	case ioUringOpFSync64:

		file, result := ioUringResolveFile64(ctx, ring, fd, flags&ioUringSQEFixedFile64 != 0)
		if result != 0 {
			return int32(result)
		}
		if fsyncFlags != 0 {
			// The Pure-Go fakefs has no range sync distinction; accepting the
			// documented fsync flags is sufficient for the ABI-level operation.
		}
		if syncer, ok := file.Closer.(guestSyncer64); ok {
			if err := syncer.Sync(); err != nil {
				return int32(errnoForOpen(err))
			}
			return 0
		}
		if ctx.FS != nil {
			if syncer, ok := any(ctx.FS).(guestSyncer64); ok {
				if err := syncer.Sync(); err != nil {
					return int32(errnoForOpen(err))
				}
				return 0
			}
		}
		return 0
	case ioUringOpTimeout64:
		if !ioUringValidateRange64(ctx, address, 16) {
			return EFAULT
		}
		var timespec [16]byte
		if err := ctx.Memory.Read(address, timespec[:]); err != nil {
			return EFAULT
		}
		seconds := binary.LittleEndian.Uint64(timespec[0:8])
		nanoseconds := binary.LittleEndian.Uint64(timespec[8:16])
		maxDurationSeconds := uint64(^uint64(0)>>1) / uint64(time.Second)
		if nanoseconds >= 1_000_000_000 || seconds > maxDurationSeconds {
			return EINVAL
		}
		duration := time.Duration(seconds)*time.Second + time.Duration(nanoseconds)
		if duration > 0 {
			time.Sleep(duration)
		}
		return 0
	default:
		return EOPNOTSUPP
	}
}
