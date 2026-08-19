package syscall

import (
	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

// GuestMapping records the backing information needed to implement MAP_SHARED
// write-back without making the CPU memory package depend on a filesystem.
type GuestMapping struct {
	Base     uint32
	Length   uint32
	Pages    corecpu.Pages
	Path     string
	Offset   uint64
	FileSize int64
	Prot     uint32
	Shared   bool
}

func (context *Context) addMapping(mapping GuestMapping) {
	if context == nil || mapping.Length == 0 {
		return
	}
	context.Mappings = append(context.Mappings, mapping)
}

func (context *Context) flushMappings(base, length uint32) int32 {
	if context == nil || context.Memory == nil || context.FS == nil || length == 0 {
		return 0
	}
	start := uint64(base)
	end := start + uint64(length)
	for _, mapping := range context.Mappings {
		if !mapping.Shared || mapping.Path == "" || mapping.Prot&ProtWrite == 0 {
			continue
		}
		mappingStart := uint64(mapping.Base)
		mappingEnd := mappingStart + uint64(mapping.Length)
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
		writeLength := int(writeEnd - writeStart)
		// A write-only mapping is still readable for the purpose of backing-store
		// synchronization. Temporarily grant read access, then restore PROT.
		page := corecpu.Page(mapping.Base >> corecpu.PageBits)
		readFlags := corecpu.PRead | corecpu.PWrite
		if err := context.Memory.SetFlags(page, mapping.Pages, readFlags|corecpu.PShared); err != nil {
			return EFAULT
		}
		data := make([]byte, writeLength)
		readErr := context.Memory.Read(corecpu.Address(writeStart), data)
		protectedFlags := memoryFlags(mapping.Prot)
		if mapping.Shared {
			protectedFlags |= corecpu.PShared
		}
		_ = context.Memory.SetFlags(page, mapping.Pages, protectedFlags)
		if readErr != nil {
			return EFAULT
		}
		offset := int64(mapping.Offset + (writeStart - mappingStart))
		if _, err := context.FS.WriteAt(mapping.Path, data, offset); err != nil {
			return EIO
		}
	}
	return 0
}

func (context *Context) removeMappings(base, length uint32) {
	if context == nil || length == 0 {
		return
	}
	start := uint64(base)
	end := start + uint64(length)
	kept := context.Mappings[:0]
	for _, mapping := range context.Mappings {
		mappingStart := uint64(mapping.Base)
		mappingEnd := mappingStart + uint64(mapping.Length)
		if end <= mappingStart || start >= mappingEnd {
			kept = append(kept, mapping)
		}
	}
	context.Mappings = kept
}
