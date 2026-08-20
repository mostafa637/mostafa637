package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

func ioUringRegister64(ctx *Context64, args [6]uint64) int64 {
	ring, result := ioUringRingFromFD64(ctx, args[0])
	if result != 0 {
		return result
	}
	opcode, arg, nrArgs := args[1], corecpu.Address64(args[2]), args[3]
	switch opcode {
	case ioUringRegisterBuffers64:
		if nrArgs > 1024 {
			return int64(EINVAL)
		}
		iovecs, result := readGuestIOVecs64(ctx, uint64(arg), nrArgs)
		if result != 0 {
			return result
		}
		for _, iovec := range iovecs {
			if iovec.Len > 16<<20 || !ioUringValidateRange64(ctx, iovec.Base, iovec.Len) {
				return int64(EFAULT)
			}
		}
		ring.mu.Lock()
		ring.registeredBuffers = append([]guestIOVec64(nil), iovecs...)
		ring.mu.Unlock()
		return int64(nrArgs)
	case ioUringUnregisterBuffers64:
		if nrArgs != 0 {
			return int64(EINVAL)
		}
		ring.mu.Lock()
		ring.registeredBuffers = nil
		ring.mu.Unlock()
		return 0
	case ioUringRegisterFiles64:
		if nrArgs > 1024 || nrArgs > uint64(^uint(0)>>1) {
			return int64(EINVAL)
		}
		if nrArgs > 0 && !ioUringValidateRange64(ctx, arg, nrArgs*4) {
			return int64(EFAULT)
		}
		fds := make([]int32, int(nrArgs))
		var raw [4]byte
		for i := range fds {
			if err := ctx.Memory.Read(arg+corecpu.Address64(i*4), raw[:]); err != nil {
				return int64(EFAULT)
			}
			value := binary.LittleEndian.Uint32(raw[:])
			if value == ^uint32(0) {
				fds[i] = -1
				continue
			}
			if value > uint32(maxFD64) {
				return int64(EBADF)
			}
			if _, err := ctx.GetFile(uint64(value)); err != nil {
				return int64(EBADF)
			}
			fds[i] = int32(value)
		}
		ring.mu.Lock()
		ring.registeredFiles = fds
		ring.mu.Unlock()
		return int64(nrArgs)
	case ioUringUnregisterFiles64:
		if nrArgs != 0 {
			return int64(EINVAL)
		}
		ring.mu.Lock()
		ring.registeredFiles = nil
		ring.mu.Unlock()
		return 0
	case ioUringRegisterProbe64:
		return ioUringRegisterProbe64Guest(ctx, arg, nrArgs)
	default:
		return int64(EOPNOTSUPP)
	}
}
