package syscall

import (
	"io"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

const (
	mapFixedNoReplace64 uint64            = 0x100000
	mmapBase64          corecpu.Address64 = 0x100000000
	mmapLimit64         corecpu.Address64 = 0x00007f0000000000
)

type GuestMapping64 struct {
	Base     corecpu.Address64
	Length   uint64
	Pages    uint64
	Path     string
	Offset   uint64
	FileSize int64
	Prot     uint64
	Shared   bool
}

func (ctx *Context64) addMapping64(mapping GuestMapping64) {
	if ctx == nil || mapping.Length == 0 {
		return
	}
	ctx.Mappings = append(ctx.Mappings, mapping)
}

func (ctx *Context64) flushMappings64(base corecpu.Address64, length uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FS == nil || length == 0 {
		return 0
	}
	start := uint64(base)
	end := start + length
	if end < start {
		return int64(EFAULT)
	}
	for _, mapping := range ctx.Mappings {
		if !mapping.Shared || mapping.Path == "" || mapping.Prot&uint64(ProtWrite) == 0 {
			continue
		}
		mappingStart := uint64(mapping.Base)
		mappingEnd := mappingStart + mapping.Length
		writeStart := start
		if writeStart < mappingStart {
			writeStart = mappingStart
		}
		writeEnd := end
		if writeEnd > mappingEnd {
			writeEnd = mappingEnd
		}
		fileEnd := mappingEnd
		if mapping.FileSize <= int64(mapping.Offset) {
			continue
		}
		availableEnd := mappingStart + uint64(mapping.FileSize-int64(mapping.Offset))
		if fileEnd > availableEnd {
			fileEnd = availableEnd
		}
		if writeEnd > fileEnd {
			writeEnd = fileEnd
		}
		if writeStart >= writeEnd {
			continue
		}
		writeLength := writeEnd - writeStart
		if writeLength > uint64(^uint(0)>>1) {
			return int64(EFAULT)
		}
		pageStart := corecpu.Address64(mappingStart & ^(corecpu.Page64Size - 1))
		readFlags := corecpu.PRead | corecpu.PWrite
		if err := ctx.Memory.SetFlags(pageStart, mapping.Pages*corecpu.Page64Size, readFlags|corecpu.PShared); err != nil {
			return int64(EFAULT)
		}
		data := make([]byte, int(writeLength))
		readErr := ctx.Memory.Read(corecpu.Address64(writeStart), data)
		protectedFlags := memoryFlags64(mapping.Prot)
		if mapping.Shared {
			protectedFlags |= corecpu.PShared
		}
		_ = ctx.Memory.SetFlags(pageStart, mapping.Pages*corecpu.Page64Size, protectedFlags)
		if readErr != nil {
			return int64(EFAULT)
		}
		offset := int64(mapping.Offset + (writeStart - mappingStart))
		if _, err := ctx.FS.WriteAt(mapping.Path, data, offset); err != nil {
			return int64(EIO)
		}
	}
	return 0
}

func (ctx *Context64) removeMappings64(base corecpu.Address64, length uint64) {
	if ctx == nil || length == 0 {
		return
	}
	start := uint64(base)
	end := start + length
	kept := ctx.Mappings[:0]
	for _, mapping := range ctx.Mappings {
		mappingStart := uint64(mapping.Base)
		mappingEnd := mappingStart + mapping.Length
		if end <= mappingStart || start >= mappingEnd {
			kept = append(kept, mapping)
		}
	}
	ctx.Mappings = kept
}

func pagesFor64(length uint64) (uint64, bool) {
	if length == 0 {
		return 0, false
	}
	pages := (length + corecpu.Page64Size - 1) / corecpu.Page64Size
	return pages, pages >= 1 && pages <= uint64(^uint64(0)>>corecpu.Page64Bits)
}

func mappingRangeFree64(memory *corecpu.Memory64, base corecpu.Address64, pages uint64) bool {
	if memory == nil || pages == 0 {
		return false
	}
	first := uint64(base) >> corecpu.Page64Bits
	for page := uint64(0); page < pages; page++ {
		if _, ok := memory.MappingFlags(corecpu.Page64(first + page)); ok {
			return false
		}
	}
	return true
}

func findHole64(memory *corecpu.Memory64, hint corecpu.Address64, pages uint64, fixed bool) (corecpu.Address64, int64) {
	if memory == nil || pages == 0 {
		return 0, int64(EINVAL)
	}
	if fixed {
		if !mappingRangeFree64(memory, hint, pages) {
			return hint, int64(EINVAL)
		}
		return hint, 0
	}
	if hint != 0 && uint64(hint)&(corecpu.Page64Size-1) == 0 && range64MmapValid(hint, pages) && mappingRangeFree64(memory, hint, pages) {
		return hint, 0
	}
	start := uint64(mmapBase64) >> corecpu.Page64Bits
	limit := uint64(mmapLimit64) >> corecpu.Page64Bits
	for start <= limit && pages <= limit-start {
		candidate := corecpu.Address64(start << corecpu.Page64Bits)
		if mappingRangeFree64(memory, candidate, pages) {
			return candidate, 0
		}
		start++
	}
	return 0, int64(ENOMEM)
}

func range64MmapValid(base corecpu.Address64, pages uint64) bool {
	if uint64(base)&(corecpu.Page64Size-1) != 0 || pages == 0 {
		return false
	}
	length := pages * corecpu.Page64Size
	if length/pages != corecpu.Page64Size {
		return false
	}
	return uint64(base) <= uint64(^uint64(0))-length+1
}

func memoryFlags64(prot uint64) corecpu.Flags {
	var flags corecpu.Flags
	if prot&uint64(ProtRead) != 0 {
		flags |= corecpu.PRead
	}
	if prot&uint64(ProtWrite) != 0 {
		flags |= corecpu.PWrite
	}
	if prot&uint64(ProtExec) != 0 {
		flags |= corecpu.PExec
	}
	return flags
}

func mmap64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(ENOMEM)
	}
	addr, length, prot, flags, rawFD, offset := args[0], args[1], args[2], args[3], args[4], args[5]
	if length == 0 || prot&^uint64(ProtRead|ProtWrite|ProtExec) != 0 {
		return int64(EINVAL)
	}
	if flags&uint64(MapShared) != 0 && flags&uint64(MapPrivate) != 0 {
		return int64(EINVAL)
	}
	if flags&(uint64(MapShared)|uint64(MapPrivate)) == 0 {
		return int64(EINVAL)
	}
	pages, ok := pagesFor64(length)
	if !ok || pages > (uint64(^uint64(0))>>corecpu.Page64Bits) {
		return int64(EINVAL)
	}
	if length > ^uint64(0)-(corecpu.Page64Size-1) {
		return int64(EINVAL)
	}
	anonymous := flags&uint64(MapAnonymous) != 0
	var backing *corefd.File
	var fileSize int64
	if !anonymous {
		if int64(rawFD) < 0 || offset&(corecpu.Page64Size-1) != 0 {
			return int64(EBADF)
		}
		var err error
		backing, err = ctx.GetFile(rawFD)
		if err != nil || backing == nil || backing.Path == "" || ctx.FS == nil {
			return int64(EBADF)
		}
		info, statErr := ctx.FS.Stat(backing.Path)
		if statErr != nil {
			return int64(errnoForOpen(statErr))
		}
		fileSize = info.Size
	} else if offset != 0 {
		return int64(EINVAL)
	}
	fixed := flags&uint64(MapFixed) != 0
	fixedNoReplace := flags&mapFixedNoReplace64 != 0
	if fixed || fixedNoReplace {
		if addr == 0 || addr&(corecpu.Page64Size-1) != 0 || !range64MmapValid(corecpu.Address64(addr), pages) {
			return int64(EINVAL)
		}
		if fixedNoReplace && !mappingRangeFree64(ctx.Memory, corecpu.Address64(addr), pages) {
			return int64(EEXIST)
		}
		if fixed && !mappingRangeFree64(ctx.Memory, corecpu.Address64(addr), pages) {
			if err := ctx.Memory.UnmapAlways(corecpu.Address64(addr), pages*corecpu.Page64Size); err != nil {
				return int64(EINVAL)
			}
		}
	}
	base, result := findHole64(ctx.Memory, corecpu.Address64(addr), pages, fixed || fixedNoReplace)
	if result != 0 {
		return result
	}
	finalFlags := memoryFlags64(prot)
	mapFlags := finalFlags | corecpu.PWrite
	if anonymous {
		mapFlags |= corecpu.PAnonymous
	}
	shared := flags&uint64(MapShared) != 0
	if shared {
		mapFlags |= corecpu.PShared
	}
	mapLength := pages * corecpu.Page64Size
	if err := ctx.Memory.Map(base, mapLength, mapFlags); err != nil {
		return int64(ENOMEM)
	}
	if backing != nil {
		available := uint64(0)
		if fileSize > 0 && offset < uint64(fileSize) {
			available = uint64(fileSize) - offset
		}
		if available > length {
			available = length
		}
		if available > uint64(^uint(0)>>1) {
			_ = ctx.Memory.UnmapAlways(base, mapLength)
			return int64(EFAULT)
		}
		if available > 0 {
			data := make([]byte, int(available))
			n, readErr := ctx.FS.ReadAt(backing.Path, data, int64(offset))
			if n > 0 {
				if err := ctx.Memory.Write(base, data[:n]); err != nil {
					_ = ctx.Memory.UnmapAlways(base, mapLength)
					return int64(EFAULT)
				}
			}
			if readErr != nil && n == 0 && readErr != io.EOF {
				_ = ctx.Memory.UnmapAlways(base, mapLength)
				return int64(EIO)
			}
		}
	}
	if err := ctx.Memory.SetFlags(base, mapLength, finalFlags|func() corecpu.Flags {
		if shared {
			return corecpu.PShared
		}
		return 0
	}()); err != nil {
		_ = ctx.Memory.UnmapAlways(base, mapLength)
		return int64(ENOMEM)
	}
	if backing != nil {
		ctx.addMapping64(GuestMapping64{Base: base, Length: length, Pages: pages, Path: backing.Path, Offset: offset, FileSize: fileSize, Prot: prot, Shared: shared})
	}
	return int64(base)
}

func munmap64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[0]&(corecpu.Page64Size-1) != 0 || args[1] == 0 {
		return int64(EINVAL)
	}
	pages, ok := pagesFor64(args[1])
	if !ok {
		return int64(EINVAL)
	}
	length := pages * corecpu.Page64Size
	if result := ctx.flushMappings64(corecpu.Address64(args[0]), args[1]); result != 0 {
		return result
	}
	if err := ctx.Memory.UnmapAlways(corecpu.Address64(args[0]), length); err != nil {
		return int64(EINVAL)
	}
	ctx.removeMappings64(corecpu.Address64(args[0]), length)
	return 0
}

func mprotect64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[0]&(corecpu.Page64Size-1) != 0 || args[1] == 0 || args[2]&^uint64(ProtRead|ProtWrite|ProtExec) != 0 {
		return int64(EINVAL)
	}
	pages, ok := pagesFor64(args[1])
	if !ok {
		return int64(EINVAL)
	}
	if err := ctx.Memory.SetFlags(corecpu.Address64(args[0]), pages*corecpu.Page64Size, memoryFlags64(args[2])); err != nil {
		return int64(ENOMEM)
	}
	start := args[0]
	end := start + args[1]
	for i := range ctx.Mappings {
		mappingStart := uint64(ctx.Mappings[i].Base)
		mappingEnd := mappingStart + ctx.Mappings[i].Length
		if start <= mappingStart && end >= mappingEnd {
			ctx.Mappings[i].Prot = args[2]
		}
	}
	return 0
}

func lseek64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil {
		return int64(EBADF)
	}
	file, err := ctx.GetFile(args[0])
	if err != nil || file == nil {
		return int64(EBADF)
	}
	if file.Seeker == nil {
		return int64(-29) // ESPIPE: descriptor is not seekable.
	}
	position, seekErr := file.Seek(int64(args[1]), int(args[2]))
	if seekErr != nil || position < 0 {
		return int64(EINVAL)
	}
	return position
}
