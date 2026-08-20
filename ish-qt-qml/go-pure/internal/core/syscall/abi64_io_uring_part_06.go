package syscall

import (
	"errors"
	"io"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func ioUringRingFromFD64(ctx *Context64, fd uint64) (*ioUring64, int64) {
	if ctx == nil || fd > maxFD64 {
		return nil, int64(EBADF)
	}
	file, err := ctx.GetFile(fd)
	if err != nil || file == nil {
		return nil, int64(EBADF)
	}
	ring, ok := file.Opaque.(*ioUring64)
	if !ok || ring == nil {
		return nil, int64(EINVAL)
	}
	ring.mu.Lock()
	closed := ring.closed
	ring.mu.Unlock()
	if closed {
		return nil, int64(EBADF)
	}
	return ring, 0
}

func ioUringValidateRange64(ctx *Context64, address corecpu.Address64, length uint64) bool {
	if ctx == nil || ctx.Memory == nil {
		return false
	}
	if length == 0 {
		return true
	}
	if address > corecpu.Address64(^uint64(0)-length+1) || length > uint64(^uint(0)>>1) {
		return false
	}
	if err := ctx.Memory.Read(address, make([]byte, 1)); err != nil {
		return false
	}
	last := address + corecpu.Address64(length-1)
	return ctx.Memory.Read(last, make([]byte, 1)) == nil
}

func ioUringResolveFile64(ctx *Context64, ring *ioUring64, fd int32, fixed bool) (*corefd.File, int64) {
	if fixed {
		if fd < 0 || int(fd) >= len(ring.registeredFiles) || ring.registeredFiles[fd] < 0 {
			return nil, int64(EBADF)
		}
		fd = ring.registeredFiles[fd]
	}
	if fd < 0 {
		return nil, int64(EBADF)
	}
	file, err := ctx.GetFile(uint64(fd))
	if err != nil || file == nil {
		return nil, int64(EBADF)
	}
	return file, 0
}

func ioUringReadAt64(file *corefd.File, dst []byte, offset uint64) (int, error) {
	if file == nil || file.Reader == nil {
		return 0, corefd.ErrBadFD
	}
	if offset == ^uint64(0) {
		return file.Read(dst)
	}
	if offset > uint64(^uint64(0)>>1) {
		return 0, errors.New("io_uring: offset overflow")
	}
	if reader, ok := file.Reader.(io.ReaderAt); ok {
		return reader.ReadAt(dst, int64(offset))
	}
	if file.Seeker == nil {
		return 0, corefd.ErrNotSeek
	}
	current, err := file.Seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	if _, err = file.Seeker.Seek(int64(offset), io.SeekStart); err != nil {
		return 0, err
	}
	n, readErr := file.Read(dst)
	_, _ = file.Seeker.Seek(current, io.SeekStart)
	return n, readErr
}
