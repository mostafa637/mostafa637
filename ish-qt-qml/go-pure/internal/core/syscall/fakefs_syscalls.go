package syscall

import (
	"encoding/binary"
	"os"
	pathpkg "path"
	"strings"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefs "github.com/mostafa637/mostafa637/go-pure/internal/core/fs"
)

const (
	stat64Size = 96

	dTypeUnknown = 0
	dTypeFIFO    = 1
	dTypeChar    = 2
	dTypeDir     = 4
	dTypeBlock   = 6
	dTypeRegular = 8
	dTypeSymlink = 10
	dTypeSocket  = 12
)

func resolveGuestPath(context *Context, name string) (string, bool) {
	if context == nil || name == "" {
		return "", false
	}
	if strings.HasPrefix(name, "/") {
		return pathpkg.Clean(name), true
	}
	cwd := context.CWD
	if cwd == "" {
		cwd = "/"
	}
	return pathpkg.Join(cwd, name), true
}

func chdir(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FS == nil {
		return ENOSYS
	}
	name, ok := readGuestString(context, state, corecpu.Address(args[0]), 4096)
	if !ok {
		return EFAULT
	}
	path, ok := resolveGuestPath(context, name)
	if !ok {
		return ENOENT
	}
	info, err := context.FS.Stat(path)
	if err != nil {
		return errnoForOpen(err)
	}
	if !info.IsDir() {
		return ENOTDIR
	}
	context.CWD = path
	return 0
}

func getcwd(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil || args[1] == 0 {
		return EFAULT
	}
	cwd := context.CWD
	if cwd == "" {
		cwd = "/"
	}
	if uint64(len(cwd)+1) > uint64(args[1]) {
		return ENAMETOOLONG
	}
	data := append([]byte(cwd), 0)
	if err := writeMemory(context, state, corecpu.Address(args[0]), data); err != nil {
		return EFAULT
	}
	return int32(len(data))
}

func stat64(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FS == nil {
		return ENOSYS
	}
	name, ok := readGuestString(context, state, corecpu.Address(args[0]), 4096)
	if !ok {
		return EFAULT
	}
	path, ok := resolveGuestPath(context, name)
	if !ok {
		return ENOENT
	}
	info, err := context.FS.Stat(path)
	if err != nil {
		return errnoForOpen(err)
	}
	return writeStat64(context, state, corecpu.Address(args[1]), info)
}

func fstat64(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	file := context.file(args[0])
	if file == nil {
		return EBADF
	}
	if context.FS != nil && file.Path != "" {
		info, err := context.FS.Stat(file.Path)
		if err != nil {
			return errnoForOpen(err)
		}
		return writeStat64(context, state, corecpu.Address(args[1]), info)
	}
	for _, candidate := range []any{file.Reader, file.Writer} {
		if hostFile, ok := candidate.(*os.File); ok {
			info, err := hostFile.Stat()
			if err != nil {
				return EIO
			}
			return writeStat64(context, state, corecpu.Address(args[1]), hostFileInfo(info))
		}
	}
	return ENOTTY
}

func writeStat64(context *Context, state *corecpu.MachineState, address corecpu.Address, info corefs.FileInfo) int32 {
	buffer := make([]byte, stat64Size)
	binary.LittleEndian.PutUint64(buffer[0:8], info.Inode)
	binary.LittleEndian.PutUint32(buffer[12:16], uint32(info.Inode))
	binary.LittleEndian.PutUint32(buffer[16:20], info.Mode.Mode)
	if info.IsDir() {
		binary.LittleEndian.PutUint32(buffer[20:24], 2)
	} else {
		binary.LittleEndian.PutUint32(buffer[20:24], 1)
	}
	binary.LittleEndian.PutUint32(buffer[24:28], info.Mode.UID)
	binary.LittleEndian.PutUint32(buffer[28:32], info.Mode.GID)
	binary.LittleEndian.PutUint64(buffer[32:40], uint64(info.Mode.Rdev))
	binary.LittleEndian.PutUint64(buffer[44:52], uint64(info.Size))
	binary.LittleEndian.PutUint32(buffer[52:56], 4096)
	binary.LittleEndian.PutUint64(buffer[56:64], uint64((info.Size+511)/512))
	seconds := info.ModTime.Unix()
	nanos := uint32(info.ModTime.Nanosecond())
	binary.LittleEndian.PutUint32(buffer[64:68], uint32(seconds))
	binary.LittleEndian.PutUint32(buffer[68:72], nanos)
	binary.LittleEndian.PutUint32(buffer[72:76], uint32(seconds))
	binary.LittleEndian.PutUint32(buffer[76:80], nanos)
	binary.LittleEndian.PutUint32(buffer[80:84], uint32(seconds))
	binary.LittleEndian.PutUint32(buffer[84:88], nanos)
	binary.LittleEndian.PutUint64(buffer[88:96], info.Inode)
	if err := writeMemory(context, state, address, buffer); err != nil {
		return EFAULT
	}
	return 0
}

func hostFileInfo(info os.FileInfo) corefs.FileInfo {
	mode := uint32(corefs.ModeRegular)
	switch {
	case info.IsDir():
		mode = corefs.ModeDir
	case info.Mode()&os.ModeSymlink != 0:
		mode = corefs.ModeSymlink
	case info.Mode()&os.ModeNamedPipe != 0:
		mode = corefs.ModeFIFO
	case info.Mode()&os.ModeSocket != 0:
		mode = corefs.ModeSocket
	case info.Mode()&os.ModeDevice != 0:
		mode = corefs.ModeBlock
	}
	return corefs.FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    corefs.IshStat{Mode: mode | uint32(info.Mode().Perm())},
		Inode:   uint64(info.ModTime().UnixNano()),
		ModTime: info.ModTime(),
	}
}

type guestDirent struct {
	inode uint64
	name  string
	kind  uint8
}

func getdents64(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.FS == nil {
		return ENOSYS
	}
	file := context.file(args[0])
	if file == nil {
		return EBADF
	}
	if file.Path == "" {
		return ENOTDIR
	}
	info, err := context.FS.Stat(file.Path)
	if err != nil {
		return errnoForOpen(err)
	}
	if !info.IsDir() {
		return ENOTDIR
	}
	limit, ok := safeLength(args[2])
	if !ok || limit == 0 {
		return EINVAL
	}
	entries, err := context.FS.ReadDir(file.Path)
	if err != nil {
		return errnoForOpen(err)
	}
	directories := []guestDirent{{inode: info.Inode, name: ".", kind: dTypeDir}}
	parent := pathpkg.Dir(file.Path)
	if parentInfo, parentErr := context.FS.Stat(parent); parentErr == nil {
		directories = append(directories, guestDirent{inode: parentInfo.Inode, name: "..", kind: dTypeDir})
	} else {
		directories = append(directories, guestDirent{inode: info.Inode, name: "..", kind: dTypeDir})
	}
	for _, entry := range entries {
		directories = append(directories, guestDirent{inode: entry.Inode, name: entry.Name, kind: direntType(entry.Mode.Mode)})
	}
	start := file.DirPos
	if start < 0 {
		start = 0
	}
	if start >= len(directories) {
		return 0
	}
	buffer := make([]byte, 0, limit)
	next := start
	for next < len(directories) {
		entry := directories[next]
		recordLength := align8(19 + len(entry.name) + 1)
		if recordLength > limit && len(buffer) == 0 {
			return EINVAL
		}
		if len(buffer)+recordLength > limit {
			break
		}
		record := make([]byte, recordLength)
		binary.LittleEndian.PutUint64(record[0:8], entry.inode)
		binary.LittleEndian.PutUint64(record[8:16], uint64(next+1))
		binary.LittleEndian.PutUint16(record[16:18], uint16(recordLength))
		record[18] = entry.kind
		copy(record[19:], entry.name)
		buffer = append(buffer, record...)
		next++
	}
	if err := writeMemory(context, state, corecpu.Address(args[1]), buffer); err != nil {
		return EFAULT
	}
	file.DirPos = next
	return int32(len(buffer))
}

func align8(value int) int {
	return (value + 7) &^ 7
}

func direntType(mode uint32) uint8 {
	switch mode & corefs.ModeTypeMask {
	case corefs.ModeDir:
		return dTypeDir
	case corefs.ModeSymlink:
		return dTypeSymlink
	case corefs.ModeFIFO:
		return dTypeFIFO
	case corefs.ModeChar:
		return dTypeChar
	case corefs.ModeBlock:
		return dTypeBlock
	case corefs.ModeSocket:
		return dTypeSocket
	case corefs.ModeRegular:
		return dTypeRegular
	default:
		return dTypeUnknown
	}
}
