package syscall

import (
	"io"

	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

const (
	fallocateKeepSize64  = 0x01
	fallocatePunchHole64 = 0x02
	fallocateSupported64 = fallocateKeepSize64 | fallocatePunchHole64
	fallocateZeroChunk64 = 1 << 20
)

func fallocate64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	mode := args[1]
	if mode&^uint64(fallocateSupported64) != 0 {
		return int64(EOPNOTSUPP)
	}
	offset, ok := signedFallocate64(args[2])
	if !ok {
		return int64(EINVAL)
	}
	length, ok := signedFallocate64(args[3])
	if !ok {
		return int64(EINVAL)
	}
	if offset < 0 || length < 0 {
		return int64(EINVAL)
	}
	if offset > int64(^uint64(0)>>1)-length {
		return int64(EINVAL)
	}

	file, err := ctx.GetFile(args[0])
	if err != nil || file == nil {
		return int64(EBADF)
	}
	if file.Path == "" {
		return int64(EINVAL)
	}
	info, err := ctx.FS.Stat(file.Path)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	if info.Mode.Mode&corefs.ModeTypeMask != corefs.ModeRegular {
		return int64(EOPNOTSUPP)
	}
	writer, ok := file.Writer.(io.WriterAt)
	if !ok || writer == nil {
		return int64(EBADF)
	}
	if length == 0 {
		return 0
	}

	end := offset + length
	if mode&fallocatePunchHole64 != 0 {
		if mode&fallocateKeepSize64 == 0 {
			return int64(EOPNOTSUPP)
		}
		if offset >= info.Size {
			return 0
		}
		if end > info.Size {
			end = info.Size
		}
		if err := zeroFallocateRange64(writer, offset, end-offset); err != nil {
			return int64(errnoForOpen(err))
		}
		return 0
	}

	if mode&fallocateKeepSize64 != 0 || end <= info.Size {
		return 0
	}
	if err := ctx.FS.Truncate(file.Path, end); err != nil {
		return int64(errnoForOpen(err))
	}
	return 0
}

func signedFallocate64(raw uint64) (int64, bool) {
	if raw > uint64(^uint64(0)>>1) {
		return 0, false
	}
	return int64(raw), true
}

func zeroFallocateRange64(writer io.WriterAt, offset, length int64) error {
	if length <= 0 {
		return nil
	}
	zeros := make([]byte, fallocateZeroChunk64)
	for length > 0 {
		chunk := int64(len(zeros))
		if chunk > length {
			chunk = length
		}
		n, err := writer.WriteAt(zeros[:int(chunk)], offset)
		if n != int(chunk) {
			if err == nil {
				return io.ErrShortWrite
			}
			return err
		}
		if err != nil {
			return err
		}
		offset += chunk
		length -= chunk
	}
	return nil
}
