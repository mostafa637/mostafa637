package syscall

import (
	"errors"
	"io"

	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

const copyFileRangeFlags64 uint64 = 0

type copyRangeOffset64 struct {
	file     *corefd.File
	address  uint64
	explicit bool
	value    int64
	original int64
}

func prepareCopyRangeOffset64(ctx *Context64, file *corefd.File, address uint64) (copyRangeOffset64, int64) {
	state := copyRangeOffset64{file: file, address: address}
	if address == 0 {
		return state, 0
	}
	if file == nil || file.Seeker == nil {
		return state, int64(espipe64)
	}
	value, err := readSigned64(ctx, address)
	if err != nil {
		return state, int64(EFAULT)
	}
	if value < 0 {
		return state, int64(EINVAL)
	}
	original, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return state, int64(espipe64)
	}
	if _, err := file.Seek(value, io.SeekStart); err != nil {
		return state, int64(EINVAL)
	}
	state.explicit = true
	state.value = value
	state.original = original
	return state, 0
}

func finishCopyRangeOffset64(ctx *Context64, state copyRangeOffset64, total int) int64 {
	if !state.explicit {
		return 0
	}
	position, seekErr := state.file.Seek(0, io.SeekCurrent)
	if seekErr != nil {
		return int64(EIO)
	}
	writeErr := writeSigned64(ctx, state.address, position)
	_, restoreErr := state.file.Seek(state.original, io.SeekStart)
	if writeErr != nil {
		return int64(EFAULT)
	}
	if restoreErr != nil {
		return int64(EIO)
	}
	_ = total
	return 0
}

func copyFileRange64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FDs == nil {
		return int64(EFAULT)
	}
	if args[5]&^copyFileRangeFlags64 != 0 {
		return int64(EINVAL)
	}
	inFile, err := ctx.GetFile(args[0])
	if err != nil || inFile == nil || inFile.Reader == nil {
		return int64(EBADF)
	}
	outFile, err := ctx.GetFile(args[2])
	if err != nil || outFile == nil || outFile.Writer == nil {
		return int64(EBADF)
	}
	if args[0] == args[2] {
		return int64(EINVAL)
	}
	count, ok := transferLength64(args[4])
	if !ok {
		return int64(EINVAL)
	}
	inOffset, result := prepareCopyRangeOffset64(ctx, inFile, args[1])
	if result != 0 {
		return result
	}
	outOffset, result := prepareCopyRangeOffset64(ctx, outFile, args[3])
	if result != 0 {
		if inOffset.explicit {
			_, _ = inFile.Seek(inOffset.original, io.SeekStart)
		}
		return result
	}

	total, transferErr := copyBetween64(inFile.Reader, outFile.Writer, count)
	inFinish := finishCopyRangeOffset64(ctx, inOffset, total)
	outFinish := finishCopyRangeOffset64(ctx, outOffset, total)
	if inFinish != 0 && total == 0 {
		return inFinish
	}
	if outFinish != 0 && total == 0 {
		return outFinish
	}
	if transferErr != nil && !errors.Is(transferErr, io.EOF) && total == 0 {
		return transferError64(transferErr)
	}
	return int64(total)
}
