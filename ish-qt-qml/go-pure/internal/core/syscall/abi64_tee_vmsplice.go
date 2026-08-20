package syscall

import "io"

const (
	teeFlags64      = spliceMove64 | spliceNonblock64 | spliceMore64 | spliceGift64
	vmspliceFlags64 = spliceNonblock64 | spliceMore64 | spliceGift64
)

func tee64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if args[3]&^teeFlags64 != 0 {
		return int64(EINVAL)
	}
	if args[0] == args[1] {
		return int64(EINVAL)
	}
	source, err := ctx.GetFile(args[0])
	if err != nil || source == nil {
		return int64(EBADF)
	}
	destination, err := ctx.GetFile(args[1])
	if err != nil || destination == nil {
		return int64(EBADF)
	}
	sourceReader, sourceOK := source.Reader.(*pipeReader)
	destinationWriter, destinationOK := destination.Writer.(*pipeWriter)
	if !sourceOK || !destinationOK || sourceReader.pipe == nil || destinationWriter.pipe == nil {
		return int64(EINVAL)
	}
	if sourceReader.pipe == destinationWriter.pipe {
		return int64(EINVAL)
	}
	count, ok := transferLength64(args[2])
	if !ok {
		return int64(EINVAL)
	}
	if count == 0 {
		return 0
	}

	data, readErr := sourceReader.pipe.peek(count, args[3]&spliceNonblock64 != 0)
	if len(data) != 0 {
		written, writeErr := destinationWriter.pipe.write(data)
		if written != 0 {
			if writeErr != nil && written == len(data) {
				return int64(written)
			}
			return int64(written)
		}
		if writeErr != nil {
			return transferError64(writeErr)
		}
	}
	if readErr != nil {
		if readErr == io.EOF {
			return 0
		}
		return transferError64(readErr)
	}
	return 0
}

func vmsplice64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if args[3]&^vmspliceFlags64 != 0 {
		return int64(EINVAL)
	}
	destination, err := ctx.GetFile(args[0])
	if err != nil || destination == nil {
		return int64(EBADF)
	}
	destinationWriter, ok := destination.Writer.(*pipeWriter)
	if !ok || destinationWriter.pipe == nil {
		return int64(EINVAL)
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
		if err := ctx.Memory.Read(iovec.Base, buffer); err != nil {
			if total != 0 {
				return int64(total)
			}
			return int64(EFAULT)
		}
		written, writeErr := destinationWriter.pipe.write(buffer)
		if written > 0 {
			total += uint64(written)
		}
		if writeErr != nil {
			if total != 0 {
				return int64(total)
			}
			return transferError64(writeErr)
		}
		if written < len(buffer) {
			break
		}
	}
	return int64(total)
}
