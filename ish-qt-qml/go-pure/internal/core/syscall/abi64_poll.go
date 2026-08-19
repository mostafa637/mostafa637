package syscall

import (
	"encoding/binary"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

const (
	pollIn64          uint16 = 0x001
	pollPri64         uint16 = 0x002
	pollOut64         uint16 = 0x004
	pollErr64         uint16 = 0x008
	pollHup64         uint16 = 0x010
	pollNval64        uint16 = 0x020
	pollRdNorm64      uint16 = 0x040
	pollRdBand64      uint16 = 0x080
	pollWrNorm64      uint16 = 0x100
	pollWrBand64      uint16 = 0x200
	pollMsg64         uint16 = 0x400
	pollRemove64      uint16 = 0x1000
	pollRdHup64       uint16 = 0x2000
	pollFdSize64             = 8
	selectFDSetSize64        = 128
	selectMaxFD64            = selectFDSetSize64 * 8
)

func pollReady64(file *corefd.File, events uint16) uint16 {
	if file == nil {
		return pollNval64
	}
	if file.Poll != nil {
		return file.Poll(events)
	}
	var ready uint16
	if file.Reader != nil {
		ready |= pollIn64
	}
	if file.Writer != nil {
		ready |= pollOut64
	}
	return ready & events
}

func pollScan64(ctx *Context64, address corecpu.Address64, count uint64) (int64, bool) {
	if ctx == nil || ctx.Memory == nil || ctx.FDs == nil {
		return int64(ENOSYS), false
	}
	if count > 1<<20 {
		return int64(EINVAL), false
	}
	readyCount := int64(0)
	for index := uint64(0); index < count; index++ {
		offset := address + corecpu.Address64(index*pollFdSize64)
		var entry [pollFdSize64]byte
		if err := ctx.Memory.Read(offset, entry[:]); err != nil {
			return int64(EFAULT), false
		}
		fd := int32(binary.LittleEndian.Uint32(entry[0:4]))
		events := binary.LittleEndian.Uint16(entry[4:6])
		var revents uint16
		file, err := ctx.FDs.Get(fd)
		if err != nil || file == nil {
			revents |= pollNval64
		} else {
			revents = pollReady64(file, events)
		}
		binary.LittleEndian.PutUint16(entry[6:8], revents)
		if err := ctx.Memory.Write(offset+6, entry[6:8]); err != nil {
			return int64(EFAULT), false
		}
		if revents != 0 {
			readyCount++
		}
	}
	return readyCount, readyCount != 0
}

func pollWait64(ctx *Context64, address corecpu.Address64, count uint64, timeout time.Duration) int64 {
	if timeout == 0 {
		result, _ := pollScan64(ctx, address, count)
		return result
	}
	deadline := time.Time{}
	if timeout >= 0 {
		deadline = time.Now().Add(timeout)
	}
	for {
		result, ready := pollScan64(ctx, address, count)
		if result < 0 || ready {
			return result
		}
		if timeout >= 0 && !time.Now().Before(deadline) {
			return result
		}
		remaining := time.Millisecond
		if timeout >= 0 {
			remaining = time.Until(deadline)
			if remaining <= 0 {
				return result
			}
			if remaining > time.Millisecond {
				remaining = time.Millisecond
			}
		}
		time.Sleep(remaining)
	}
}

func poll64(ctx *Context64, args [6]uint64) int64 {
	timeout := int64(args[2])
	if timeout < -1 {
		return int64(EINVAL)
	}
	duration := time.Duration(-1)
	if timeout >= 0 {
		if timeout > int64((time.Duration(1<<63-1))/time.Millisecond) {
			duration = time.Duration(1<<63 - 1)
		} else {
			duration = time.Duration(timeout) * time.Millisecond
		}
	}
	return pollWait64(ctx, corecpu.Address64(args[0]), args[1], duration)
}

func readPollTimespec64(ctx *Context64, address corecpu.Address64) (time.Duration, int64) {
	if address == 0 {
		return -1, 0
	}
	var value [16]byte
	if err := ctx.Memory.Read(address, value[:]); err != nil {
		return 0, int64(EFAULT)
	}
	seconds := int64(binary.LittleEndian.Uint64(value[0:8]))
	nanoseconds := int64(binary.LittleEndian.Uint64(value[8:16]))
	if seconds < 0 || nanoseconds < 0 || nanoseconds >= int64(time.Second) {
		return 0, int64(EINVAL)
	}
	if seconds > int64((time.Duration(1<<63-1))/time.Second) {
		return time.Duration(1<<63 - 1), 0
	}
	return time.Duration(seconds)*time.Second + time.Duration(nanoseconds), 0
}

func ppoll64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	duration, result := readPollTimespec64(ctx, corecpu.Address64(args[2]))
	if result != 0 {
		return result
	}
	if args[4] != 0 && args[4] != 8 {
		return int64(EINVAL)
	}
	return pollWait64(ctx, corecpu.Address64(args[0]), args[1], duration)
}

func selectBit64(ctx *Context64, address corecpu.Address64, fd uint64) (bool, int64) {
	if address == 0 {
		return false, 0
	}
	if fd >= selectMaxFD64 {
		return false, int64(EINVAL)
	}
	var byteValue [1]byte
	byteAddress := address + corecpu.Address64(fd/8)
	if err := ctx.Memory.Read(byteAddress, byteValue[:]); err != nil {
		return false, int64(EFAULT)
	}
	return byteValue[0]&(1<<uint(fd%8)) != 0, 0
}

func setSelectBit64(ctx *Context64, address corecpu.Address64, fd uint64, set bool) int64 {
	if address == 0 {
		return 0
	}
	if fd >= selectMaxFD64 {
		return int64(EINVAL)
	}
	var byteValue [1]byte
	byteAddress := address + corecpu.Address64(fd/8)
	if err := ctx.Memory.Read(byteAddress, byteValue[:]); err != nil {
		return int64(EFAULT)
	}
	mask := byte(1 << uint(fd%8))
	if set {
		byteValue[0] |= mask
	} else {
		byteValue[0] &^= mask
	}
	if err := ctx.Memory.Write(byteAddress, byteValue[:]); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func clearSelectSet64(ctx *Context64, address corecpu.Address64, nfds uint64) int64 {
	if address == 0 {
		return 0
	}
	if nfds > selectMaxFD64 {
		return int64(EINVAL)
	}
	bytes := (nfds + 7) / 8
	if bytes == 0 {
		return 0
	}
	if err := ctx.Memory.Write(address, make([]byte, bytes)); err != nil {
		return int64(EFAULT)
	}
	return 0
}

func selectTimeout64(ctx *Context64, address corecpu.Address64, timespec bool) (time.Duration, int64) {
	if address == 0 {
		return -1, 0
	}
	if ctx == nil || ctx.Memory == nil {
		return 0, int64(EFAULT)
	}
	if timespec {
		return readPollTimespec64(ctx, address)
	}
	var value [16]byte
	if err := ctx.Memory.Read(address, value[:]); err != nil {
		return 0, int64(EFAULT)
	}
	seconds := int64(binary.LittleEndian.Uint64(value[0:8]))
	microseconds := int64(binary.LittleEndian.Uint64(value[8:16]))
	if seconds < 0 || microseconds < 0 || microseconds >= 1_000_000 {
		return 0, int64(EINVAL)
	}
	if seconds > int64((time.Duration(1<<63-1))/time.Second) {
		return time.Duration(1<<63 - 1), 0
	}
	return time.Duration(seconds)*time.Second + time.Duration(microseconds)*time.Microsecond, 0
}

func select64(ctx *Context64, args [6]uint64, timespec bool) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FDs == nil {
		return int64(ENOSYS)
	}
	nfds := args[0]
	if nfds > selectMaxFD64 {
		return int64(EINVAL)
	}
	readAddress := corecpu.Address64(args[1])
	writeAddress := corecpu.Address64(args[2])
	exceptAddress := corecpu.Address64(args[3])
	var readInput, writeInput, exceptInput [selectMaxFD64 / 8]byte
	for _, item := range []struct {
		address corecpu.Address64
		buffer  []byte
	}{
		{readAddress, readInput[:]},
		{writeAddress, writeInput[:]},
		{exceptAddress, exceptInput[:]},
	} {
		if item.address == 0 {
			continue
		}
		if err := ctx.Memory.Read(item.address, item.buffer); err != nil {
			return int64(EFAULT)
		}
	}
	for _, address := range []corecpu.Address64{readAddress, writeAddress, exceptAddress} {
		if result := clearSelectSet64(ctx, address, nfds); result != 0 {
			return result
		}
	}
	readyCount := int64(0)
	for fd := uint64(0); fd < nfds; fd++ {
		wantRead := readInput[fd/8]&(1<<uint(fd%8)) != 0
		wantWrite := writeInput[fd/8]&(1<<uint(fd%8)) != 0
		wantExcept := exceptInput[fd/8]&(1<<uint(fd%8)) != 0
		if !wantRead && !wantWrite && !wantExcept {
			continue
		}
		file, err := ctx.FDs.Get(int32(fd))
		if err != nil || file == nil {
			return int64(EBADF)
		}
		var events uint16
		if wantRead {
			events |= pollIn64
		}
		if wantWrite {
			events |= pollOut64
		}
		if wantExcept {
			events |= pollPri64
		}
		ready := pollReady64(file, events)
		if ready&pollIn64 != 0 && wantRead {
			if result := setSelectBit64(ctx, readAddress, fd, true); result != 0 {
				return result
			}
		}
		if ready&pollOut64 != 0 && wantWrite {
			if result := setSelectBit64(ctx, writeAddress, fd, true); result != 0 {
				return result
			}
		}
		if ready&pollPri64 != 0 && wantExcept {
			if result := setSelectBit64(ctx, exceptAddress, fd, true); result != 0 {
				return result
			}
		}
		if (ready&pollIn64 != 0 && wantRead) || (ready&pollOut64 != 0 && wantWrite) || (ready&pollPri64 != 0 && wantExcept) {
			readyCount++
		}
	}
	if readyCount != 0 {
		return readyCount
	}
	duration, result := selectTimeout64(ctx, corecpu.Address64(args[4]), timespec)
	if result != 0 {
		return result
	}
	if duration == 0 {
		return 0
	}
	if duration < 0 {
		duration = time.Millisecond
	}
	time.Sleep(duration)
	return select64(ctx, [6]uint64{args[0], args[1], args[2], args[3], args[4], args[5]}, timespec)
}

func selectSyscall64(ctx *Context64, args [6]uint64) int64 {
	return select64(ctx, args, false)
}

func pselect6Syscall64(ctx *Context64, args [6]uint64) int64 {
	return select64(ctx, args, true)
}
