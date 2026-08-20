package syscall

import (
	"encoding/binary"
	"errors"
	"io"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

const (
	spliceMove64     uint64 = 1
	spliceNonblock64 uint64 = 2
	spliceMore64     uint64 = 4
	spliceGift64     uint64 = 8
	spliceFlags64           = spliceMove64 | spliceNonblock64 | spliceMore64 | spliceGift64
	espipe64         int64  = -29
	maxTransfer64           = 1 << 20
)

func pipe264(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FDs == nil || args[0] == 0 {
		return int64(EFAULT)
	}
	if args[1]&^(uint64(pipe2Cloexec)|uint64(pipe2Nonblock)) != 0 {
		return int64(EINVAL)
	}

	pipe := newGuestPipe(args[1]&uint64(pipe2Nonblock) != 0)
	cloexec := args[1]&uint64(pipe2Cloexec) != 0
	statusFlags := uint64(0)
	if args[1]&uint64(pipe2Nonblock) != 0 {
		statusFlags = uint64(guestOpenNonblock)
	}
	reader := &corefd.File{
		Reader:      &pipeReader{pipe: pipe},
		Closer:      &pipeReader{pipe: pipe},
		Poll:        func(events uint16) uint16 { return pipe.ready(events) },
		Cloexec:     cloexec,
		StatusFlags: statusFlags,
	}
	writer := &corefd.File{
		Writer:      &pipeWriter{pipe: pipe},
		Closer:      &pipeWriter{pipe: pipe},
		Poll:        func(events uint16) uint16 { return pipe.ready(events) },
		Cloexec:     cloexec,
		StatusFlags: statusFlags,
	}
	readFD, err := ctx.FDs.Open(reader)
	if err != nil {
		return int64(EMFILE)
	}
	writeFD, err := ctx.FDs.Open(writer)
	if err != nil {
		_ = ctx.FDs.Close(readFD)
		return int64(EMFILE)
	}
	var result [8]byte
	binary.LittleEndian.PutUint32(result[0:4], uint32(readFD))
	binary.LittleEndian.PutUint32(result[4:8], uint32(writeFD))
	if err := ctx.Memory.Write(corecpu.Address64(args[0]), result[:]); err != nil {
		_ = ctx.FDs.Close(readFD)
		_ = ctx.FDs.Close(writeFD)
		return int64(EFAULT)
	}
	return 0
}

func sendfile64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if args[0] == args[1] {
		return int64(EINVAL)
	}
	outFile, err := ctx.GetFile(args[0])
	if err != nil || outFile.Writer == nil {
		return int64(EBADF)
	}
	inFile, err := ctx.GetFile(args[1])
	if err != nil || inFile.Reader == nil {
		return int64(EBADF)
	}
	count, ok := transferLength64(args[3])
	if !ok {
		return int64(EINVAL)
	}

	offset, errCode := prepareOffset64(ctx, inFile, args[2])
	if errCode != 0 {
		return errCode
	}

	total, transferErr := copyBetween64(inFile.Reader, outFile.Writer, count)
	if offset != nil {
		if finishErr := finishOffset64(ctx, offset); finishErr != nil && total == 0 {
			return offsetError64(finishErr)
		}
	}
	if transferErr != nil && total == 0 {
		return transferError64(transferErr)
	}
	return int64(total)
}

func splice64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if args[5]&^spliceFlags64 != 0 {
		return int64(EINVAL)
	}
	if args[0] == args[2] {
		return int64(EINVAL)
	}
	inFile, err := ctx.GetFile(args[0])
	if err != nil || inFile.Reader == nil {
		return int64(EBADF)
	}
	outFile, err := ctx.GetFile(args[2])
	if err != nil || outFile.Writer == nil {
		return int64(EBADF)
	}
	count, ok := transferLength64(args[4])
	if !ok {
		return int64(EINVAL)
	}

	inOffset, errCode := prepareOffset64(ctx, inFile, args[1])
	if errCode != 0 {
		return errCode
	}
	outOffset, errCode := prepareOffset64(ctx, outFile, args[3])
	if errCode != 0 {
		return errCode
	}

	total, transferErr := copyBetween64(inFile.Reader, outFile.Writer, count)
	if inOffset != nil {
		if finishErr := finishOffset64(ctx, inOffset); finishErr != nil && total == 0 {
			return offsetError64(finishErr)
		}
	}
	if outOffset != nil {
		if finishErr := finishOffset64(ctx, outOffset); finishErr != nil && total == 0 {
			return offsetError64(finishErr)
		}
	}
	if transferErr != nil && total == 0 {
		return transferError64(transferErr)
	}
	return int64(total)
}

func transferLength64(value uint64) (int, bool) {
	if value > maxTransfer64 {
		return 0, false
	}
	return int(value), true
}

func copyBetween64(src io.Reader, dst io.Writer, count int) (int, error) {
	if count == 0 {
		return 0, nil
	}
	buffer := make([]byte, 32*1024)
	total := 0
	for total < count {
		want := count - total
		if want > len(buffer) {
			want = len(buffer)
		}
		n, readErr := src.Read(buffer[:want])
		if n > 0 {
			written := 0
			for written < n {
				m, writeErr := dst.Write(buffer[written:n])
				if m > 0 {
					written += m
					total += m
				}
				if writeErr != nil {
					return total, writeErr
				}
				if m == 0 {
					return total, io.ErrShortWrite
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF && total != 0 {
				return total, nil
			}
			return total, readErr
		}
		if n == 0 {
			return total, io.ErrNoProgress
		}
	}
	return total, nil
}

type offset64State struct {
	file    *corefd.File
	address uint64
	origin  int64
}

func prepareOffset64(ctx *Context64, file *corefd.File, address uint64) (*offset64State, int64) {
	if address == 0 {
		return nil, 0
	}
	if file == nil || file.Seeker == nil {
		return nil, espipe64
	}
	value, err := readSigned64(ctx, address)
	if err != nil {
		return nil, int64(EFAULT)
	}
	if value < 0 {
		return nil, int64(EINVAL)
	}
	origin, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, espipe64
	}
	if _, err := file.Seek(value, io.SeekStart); err != nil {
		return nil, int64(EINVAL)
	}
	return &offset64State{file: file, address: address, origin: origin}, 0
}

func finishOffset64(ctx *Context64, state *offset64State) error {
	if state == nil || state.file == nil {
		return nil
	}
	position, err := state.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	writeErr := writeSigned64(ctx, state.address, position)
	_, restoreErr := state.file.Seek(state.origin, io.SeekStart)
	if writeErr != nil {
		return writeErr
	}
	return restoreErr
}

func offsetError64(err error) int64 {
	if err == nil {
		return 0
	}
	if errors.Is(err, errGuestMemory64) {
		return int64(EFAULT)
	}
	return int64(EIO)
}

func readSigned64(ctx *Context64, address uint64) (int64, error) {
	var raw [8]byte
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Read(corecpu.Address64(address), raw[:]) != nil {
		return 0, errors.New("guest offset read failed")
	}
	return int64(binary.LittleEndian.Uint64(raw[:])), nil
}

var errGuestMemory64 = errors.New("guest offset memory access failed")

func writeSigned64(ctx *Context64, address uint64, value int64) error {
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], uint64(value))
	if ctx == nil || ctx.Memory == nil {
		return errGuestMemory64
	}
	if err := ctx.Memory.Write(corecpu.Address64(address), raw[:]); err != nil {
		return errGuestMemory64
	}
	return nil
}

func transferError64(err error) int64 {
	if err == nil {
		return 0
	}
	if errors.Is(err, errWouldBlock64) {
		return int64(EAGAIN)
	}
	if errors.Is(err, io.ErrClosedPipe) {
		return int64(EPIPE)
	}
	return int64(EIO)
}
