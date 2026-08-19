package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

const (
	accessRead  = 4
	accessWrite = 2
	accessExec  = 1

	fcntlDupFD    = 0
	fcntlGetFD    = 1
	fcntlSetFD    = 2
	fcntlGetFL    = 3
	fcntlSetFL    = 4
	fcntlDupFDClO = 1030
	fdCloexec     = 1

	tioCGWINSZ = 0x5413
	tioCSWINSZ = 0x5414
	tioCFLSH   = 0x540b
)

func access(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FS == nil {
		return ENOSYS
	}
	path, ok := readGuestString(context, state, corecpu.Address(args[0]), 4096)
	if !ok {
		return EFAULT
	}
	path, ok = resolveGuestPath(context, path)
	if !ok {
		return ENOENT
	}
	if _, err := context.FS.Stat(path); err != nil {
		return errnoForOpen(err)
	}
	// iSH starts the guest as root. Permission-bit enforcement belongs in the
	// metadata/credential layer, so existence is the useful access(2) result now.
	_ = args[1] & (accessRead | accessWrite | accessExec)
	return 0
}

func readlink(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FS == nil {
		return ENOSYS
	}
	path, ok := readGuestString(context, state, corecpu.Address(args[0]), 4096)
	if !ok {
		return EFAULT
	}
	path, ok = resolveGuestPath(context, path)
	if !ok {
		return ENOENT
	}
	target, err := context.FS.Readlink(path)
	if err != nil {
		return errnoForOpen(err)
	}
	if args[2] == 0 {
		return 0
	}
	data := []byte(target)
	if uint32(len(data)) > args[2] {
		data = data[:args[2]]
	}
	if err := context.Memory.Write(corecpu.Address(args[1]), data); err != nil {
		return EFAULT
	}
	return int32(len(data))
}

func uname(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil {
		return EFAULT
	}
	const fieldSize = 65
	fields := [...]string{
		"Linux",
		"ish-go",
		"6.1.0-ish",
		"#1 Pure Go",
		"i686",
		"(none)",
	}
	buffer := make([]byte, fieldSize*len(fields))
	for index, value := range fields {
		copy(buffer[index*fieldSize:(index+1)*fieldSize-1], value)
	}
	if err := context.Memory.Write(corecpu.Address(args[0]), buffer); err != nil {
		return EFAULT
	}
	return 0
}

func fcntl64(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	file, err := getFile(context, args[0])
	if err != 0 {
		return err
	}
	switch args[1] {
	case fcntlDupFD, fcntlDupFDClO:
		if context.FDs == nil {
			return EBADF
		}
		newFD, dupErr := context.FDs.Open(file)
		if dupErr != nil {
			return EBADF
		}
		if fcntlDupFDClO == args[1] {
			file.Cloexec = true
		}
		if context.Files != nil {
			context.Files[uint32(newFD)] = file
		}
		return newFD
	case fcntlGetFD:
		if file.Cloexec {
			return fdCloexec
		}
		return 0
	case fcntlSetFD:
		file.Cloexec = args[2]&fdCloexec != 0
		return 0
	case fcntlGetFL, fcntlSetFL:
		return 0

	default:
		return EINVAL
	}
}

func ioctl(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	file, err := getFile(context, args[0])
	if err != 0 {
		return err
	}
	_ = file
	switch args[1] {
	case tioCGWINSZ:
		if context.Memory == nil {
			return EFAULT
		}
		var size [8]byte
		binary.LittleEndian.PutUint16(size[0:2], context.WinRows)
		binary.LittleEndian.PutUint16(size[2:4], context.WinCols)
		if err := context.Memory.Write(corecpu.Address(args[2]), size[:]); err != nil {
			return EFAULT
		}
		return 0
	case tioCSWINSZ:
		if context.Memory == nil {
			return EFAULT
		}
		var size [8]byte
		if err := context.Memory.Read(corecpu.Address(args[2]), size[:]); err != nil {
			return EFAULT
		}
		context.WinRows = binary.LittleEndian.Uint16(size[0:2])
		context.WinCols = binary.LittleEndian.Uint16(size[2:4])
		return 0
	case tioCFLSH:
		return 0
	default:
		return ENOTTY
	}
}

func getuid(*Context, *corecpu.MachineState, [6]uint32) int32 { return 0 }
func getgid(*Context, *corecpu.MachineState, [6]uint32) int32 { return 0 }
func getppid(context *Context, _ *corecpu.MachineState, _ [6]uint32) int32 {
	if context == nil || context.ParentPID == 0 {
		return 1
	}
	return int32(context.ParentPID)
}

func getFile(context *Context, rawFD uint32) (*corefd.File, int32) {
	if context == nil || context.FDs == nil || rawFD > uint32(^uint32(0)>>1) {
		return nil, EBADF
	}
	file, err := context.FDs.Get(int32(rawFD))
	if err != nil {
		return nil, EBADF
	}
	return file, 0
}
