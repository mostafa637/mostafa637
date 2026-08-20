package syscall

import (
	"errors"
	"io"

	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

func ioUringWriteAt64(file *corefd.File, src []byte, offset uint64) (int, error) {
	if file == nil || file.Writer == nil {
		return 0, corefd.ErrBadFD
	}
	if offset == ^uint64(0) {
		return file.Write(src)
	}
	if offset > uint64(^uint64(0)>>1) {
		return 0, errors.New("io_uring: offset overflow")
	}
	if writer, ok := file.Writer.(io.WriterAt); ok {
		return writer.WriteAt(src, int64(offset))
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
	n, writeErr := file.Write(src)
	_, _ = file.Seeker.Seek(current, io.SeekStart)
	return n, writeErr
}

func ioUringErrno64(err error) int32 {
	if err == nil || errors.Is(err, io.EOF) {
		return 0
	}
	if errors.Is(err, corefd.ErrBadFD) {
		return EBADF
	}
	if errors.Is(err, corefd.ErrNotSeek) {
		return -29 // ESPIPE: descriptor is not seekable.
	}
	return EIO
}
