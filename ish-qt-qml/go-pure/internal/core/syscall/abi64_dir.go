package syscall

import (
	"encoding/binary"
	pathpkg "path"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

// getdents64Guest writes Linux x86-64 linux_dirent64 records into guest memory.
// The descriptor owns DirPos, so duplicated descriptors share the same stream
// position through the retained corefd.File object, as Linux does for dup().
func getdents64Guest(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	file, err := ctx.GetFile(args[0])
	if err != nil || file == nil {
		return int64(EBADF)
	}
	if file.Path == "" {
		return int64(ENOTDIR)
	}
	info, err := ctx.FS.Stat(file.Path)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	if !info.IsDir() {
		return int64(ENOTDIR)
	}
	if args[2] == 0 || args[2] > uint64(^uint32(0)>>1) {
		return int64(EINVAL)
	}
	limit := int(args[2])

	entries, err := ctx.FS.ReadDir(file.Path)
	if err != nil {
		return int64(errnoForOpen(err))
	}
	directories := make([]guestDirent, 0, len(entries)+2)
	directories = append(directories, guestDirent{inode: info.Inode, name: ".", kind: dTypeDir})
	parent := pathpkg.Dir(file.Path)
	if parentInfo, parentErr := ctx.FS.Stat(parent); parentErr == nil {
		directories = append(directories, guestDirent{inode: parentInfo.Inode, name: "..", kind: dTypeDir})
	} else {
		directories = append(directories, guestDirent{inode: info.Inode, name: "..", kind: dTypeDir})
	}
	for _, entry := range entries {
		directories = append(directories, guestDirent{
			inode: entry.Inode,
			name:  entry.Name,
			kind:  direntType(entry.Mode.Mode),
		})
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
		if recordLength > int(^uint16(0)) {
			return int64(EINVAL)
		}
		if recordLength > limit && len(buffer) == 0 {
			return int64(EINVAL)
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
	if err := ctx.Memory.Write(corecpu.Address64(args[1]), buffer); err != nil {
		return int64(EFAULT)
	}
	file.DirPos = next
	return int64(len(buffer))
}
