package syscall

import corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

func ioUringExecuteVector64(ctx *Context64, ring *ioUring64, fd int32, sqeFlags uint8, iovecAddress corecpu.Address64, iovecCount uint64, offset uint64, write bool) int32 {
	if iovecCount == 0 || iovecCount > 1024 {
		return EINVAL
	}
	iovecs, result := readGuestIOVecs64(ctx, uint64(iovecAddress), iovecCount)
	if result != 0 {
		return int32(result)
	}
	file, result := ioUringResolveFile64(ctx, ring, fd, sqeFlags&ioUringSQEFixedFile64 != 0)
	if result != 0 {
		return int32(result)
	}
	var total int
	currentOffset := offset
	for _, iovec := range iovecs {
		if iovec.Len > 16<<20 || iovec.Len > uint64(^uint(0)>>1) || !ioUringValidateRange64(ctx, iovec.Base, iovec.Len) {
			if total > 0 {
				return int32(total)
			}
			return EFAULT
		}
		buffer := make([]byte, int(iovec.Len))
		var n int
		var err error
		if write {
			if err = ctx.Memory.Read(iovec.Base, buffer); err == nil {
				n, err = ioUringWriteAt64(file, buffer, currentOffset)
			}
		} else {
			n, err = ioUringReadAt64(file, buffer, currentOffset)
			if n > 0 {
				if writeErr := ctx.Memory.Write(iovec.Base, buffer[:n]); writeErr != nil {
					return EFAULT
				}
			}
		}
		if n > 0 {
			total += n
			if currentOffset != ^uint64(0) {
				if uint64(n) > ^uint64(0)-currentOffset {
					return int32(total)
				}
				currentOffset += uint64(n)
			}
		}
		if err != nil || n < int(iovec.Len) {
			if total > 0 {
				return int32(total)
			}
			if err != nil {
				return ioUringErrno64(err)
			}
			return 0
		}
	}
	return int32(total)
}
