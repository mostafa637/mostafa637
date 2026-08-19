package syscall

import (
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	termiosSize64 = 44

	tcgets64     = 0x5401
	tcsets64     = 0x5402
	tcsetsw64    = 0x5403
	tcsetsf64    = 0x5404
	tiocgwinsz64 = 0x5413
	tiocswinsz64 = 0x5414
	tiocflush64  = 0x540b
)

func fcntl64_64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FDs == nil || args[0] > maxFD64 {
		return int64(EBADF)
	}
	file, err := ctx.GetFile(args[0])
	if err != nil {
		return int64(EBADF)
	}
	if args[1] > uint64(^uint32(0)) {
		return int64(EINVAL)
	}
	switch uint32(args[1]) {
	case fcntlDupFD, fcntlDupFDClO:
		if args[2] > maxFD64 {
			return int64(EINVAL)
		}
		fd, dupErr := ctx.FDs.Open(file)
		if dupErr != nil {
			return int64(EMFILE)
		}
		if uint64(fd) < args[2] {
			// Table.Open normally starts at the next available descriptor. Keep
			// the current table API intact while honoring the common min-fd case.
			if err := ctx.FDs.Close(fd); err != nil {
				return int64(EBADF)
			}
			if err := ctx.FDs.InstallAt(int32(args[2]), file, false); err != nil {
				return int64(EMFILE)
			}
			fd = int32(args[2])
		}
		if uint32(args[1]) == fcntlDupFDClO {
			file.Cloexec = true
		}
		return int64(fd)
	case fcntlGetFD:
		if file.Cloexec {
			return fdCloexec
		}
		return 0
	case fcntlSetFD:
		if args[2]&^uint64(fdCloexec) != 0 {
			return int64(EINVAL)
		}
		file.Cloexec = args[2]&fdCloexec != 0
		return 0
	case fcntlGetFL:
		return int64(file.StatusFlags)
	case fcntlSetFL:
		const allowedStatusFlags = uint64(guestOpenNonblock | guestOpenAppend)
		if args[2]&^allowedStatusFlags != 0 {
			return int64(EINVAL)
		}
		file.StatusFlags = (file.StatusFlags &^ allowedStatusFlags) | (args[2] & allowedStatusFlags)
		setPipeNonblock64(file, args[2]&uint64(guestOpenNonblock) != 0)
		return 0
	default:
		return int64(EINVAL)
	}
}

func ioctl64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(EFAULT)
	}
	if _, err := ctx.GetFile(args[0]); err != nil {
		return int64(EBADF)
	}
	switch args[1] {
	case tiocgwinsz64:
		var size [8]byte
		putUint16(size[0:2], ctx.WinRows)
		putUint16(size[2:4], ctx.WinCols)
		if err := ctx.Memory.Write(cpu.Address64(args[2]), size[:]); err != nil {
			return int64(EFAULT)
		}
		return 0
	case tiocswinsz64:
		var size [8]byte
		if err := ctx.Memory.Read(cpu.Address64(args[2]), size[:]); err != nil {
			return int64(EFAULT)
		}
		ctx.WinRows = getUint16(size[0:2])
		ctx.WinCols = getUint16(size[2:4])
		return 0
	case tcgets64:
		if err := ctx.Memory.Write(cpu.Address64(args[2]), ctx.Termios[:]); err != nil {
			return int64(EFAULT)
		}
		return 0
	case tcsets64, tcsetsw64, tcsetsf64:
		var termios [termiosSize64]byte
		if err := ctx.Memory.Read(cpu.Address64(args[2]), termios[:]); err != nil {
			return int64(EFAULT)
		}
		copy(ctx.Termios[:], termios[:])
		return 0
	case tiocflush64:
		return 0
	default:
		return int64(ENOTTY)
	}
}

func putUint16(dst []byte, value uint16) {
	dst[0] = byte(value)
	dst[1] = byte(value >> 8)
}

func getUint16(src []byte) uint16 {
	return uint16(src[0]) | uint16(src[1])<<8
}
