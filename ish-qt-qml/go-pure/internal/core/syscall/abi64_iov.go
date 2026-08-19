package syscall

import (
	"encoding/binary"
	"io"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

const (
	iovec64Size   = 16
	maxIOVecs64   = 1024
	maxIOBytes64  = uint64(16 << 20)
	maxInt64Bytes = uint64(^uint(0) >> 1)
)

type guestIOVec64 struct {
	Base corecpu.Address64
	Len  uint64
}

func readv64(ctx *Context64, args [6]uint64) int64 {
	file, err := context64File(ctx, args[0])
	if err != 0 {
		return err
	}
	if file.Reader == nil {
		return int64(EBADF)
	}
	iovecs, result := readGuestIOVecs64(ctx, args[1], args[2])
	if result != 0 {
		return result
	}
	var total uint64
	for _, iovec := range iovecs {
		if iovec.Len == 0 {
			continue
		}
		if iovec.Len > maxInt64Bytes {
			return int64(EINVAL)
		}
		buffer := make([]byte, int(iovec.Len))
		n, readErr := file.Reader.Read(buffer)
		if n > 0 {
			if ctx.Memory == nil || ctx.Memory.Write(iovec.Base, buffer[:n]) != nil {
				if total != 0 {
					return int64(total)
				}
				return int64(EFAULT)
			}
			total += uint64(n)
		}
		if readErr != nil && readErr != io.EOF {
			if total != 0 {
				return int64(total)
			}
			return int64(EIO)
		}
		if n < len(buffer) || readErr == io.EOF {
			break
		}
	}
	return int64(total)
}

func writev64(ctx *Context64, args [6]uint64) int64 {
	file, err := context64File(ctx, args[0])
	if err != 0 {
		return err
	}
	if file.Writer == nil {
		return int64(EBADF)
	}
	iovecs, result := readGuestIOVecs64(ctx, args[1], args[2])
	if result != 0 {
		return result
	}
	var total uint64
	for _, iovec := range iovecs {
		if iovec.Len == 0 {
			continue
		}
		if iovec.Len > maxInt64Bytes {
			return int64(EINVAL)
		}
		buffer := make([]byte, int(iovec.Len))
		if ctx.Memory == nil || ctx.Memory.Read(iovec.Base, buffer) != nil {
			if total != 0 {
				return int64(total)
			}
			return int64(EFAULT)
		}
		n, writeErr := file.Writer.Write(buffer)
		if n > 0 {
			total += uint64(n)
		}
		if writeErr != nil {
			if total != 0 {
				return int64(total)
			}
			return int64(EIO)
		}
		if n < len(buffer) {
			break
		}
	}
	return int64(total)
}

func context64File(ctx *Context64, fd uint64) (*corefd.File, int64) {
	if ctx == nil {
		return nil, int64(EBADF)
	}
	file, err := ctx.GetFile(fd)
	if err != nil || file == nil {
		return nil, int64(EBADF)
	}
	return file, 0
}

func readGuestIOVecs64(ctx *Context64, address, count uint64) ([]guestIOVec64, int64) {
	if ctx == nil || ctx.Memory == nil {
		return nil, int64(EFAULT)
	}
	if count > maxIOVecs64 {
		return nil, int64(EINVAL)
	}
	if count == 0 {
		return nil, 0
	}
	if count > ^uint64(0)/iovec64Size {
		return nil, int64(EFAULT)
	}
	byteCount := count * iovec64Size
	if address > ^uint64(0)-byteCount || byteCount > maxInt64Bytes {
		return nil, int64(EFAULT)
	}
	raw := make([]byte, int(byteCount))
	if err := ctx.Memory.Read(corecpu.Address64(address), raw); err != nil {
		return nil, int64(EFAULT)
	}
	iovecs := make([]guestIOVec64, int(count))
	var total uint64
	for index := range iovecs {
		offset := index * iovec64Size
		base := binary.LittleEndian.Uint64(raw[offset : offset+8])
		length := binary.LittleEndian.Uint64(raw[offset+8 : offset+16])
		if base > ^uint64(0)-length {
			return nil, int64(EFAULT)
		}
		if length > maxIOBytes64 || total > maxIOBytes64-length {
			return nil, int64(EINVAL)
		}
		total += length
		iovecs[index] = guestIOVec64{Base: corecpu.Address64(base), Len: length}
	}
	return iovecs, 0
}
