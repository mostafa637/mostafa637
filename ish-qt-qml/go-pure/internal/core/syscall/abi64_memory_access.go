package syscall

import corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

const (
	mremapMayMove64   uint64 = 1
	mremapFixed64     uint64 = 2
	mremapDontUnmap64 uint64 = 4

	madviseNormal64        uint64 = 0
	madviseRandom64        uint64 = 1
	madviseSequential64    uint64 = 2
	madviseWillNeed64      uint64 = 3
	madviseDontNeed64      uint64 = 4
	madviseFree64          uint64 = 8
	madviseRemove64        uint64 = 9
	madviseDontFork64      uint64 = 10
	madviseDoFork64        uint64 = 11
	madviseMergeable64     uint64 = 12
	madviseUnmergeable64   uint64 = 13
	madviseHugePage64      uint64 = 14
	madviseNoHugePage64    uint64 = 15
	madviseDontDump64      uint64 = 16
	madviseDoDump64        uint64 = 17
	madviseWipeOnFork64    uint64 = 18
	madviseKeepOnFork64    uint64 = 19
	madviseCold64          uint64 = 20
	madvisePageOut64       uint64 = 21
	madvisePopulateRead64  uint64 = 22
	madvisePopulateWrite64 uint64 = 23

	faccessAtEaccess64         uint64 = 0x200
	faccessAtSymlinkNoFollow64        = atSymlinkNoFollow64
	faccessAtEmptyPath64              = atEmptyPath64
	faccessAtFlags64           uint64 = faccessAtEaccess64 | faccessAtSymlinkNoFollow64 | faccessAtEmptyPath64
	accessModeMask64           uint64 = accessRead | accessWrite | accessExec
	maxMremapBytes64           uint64 = 1 << 30
)

func mremap64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil {
		return int64(ENOMEM)
	}
	oldBase := corecpu.Address64(args[0])
	oldSize := args[1]
	newSize := args[2]
	flags := args[3]
	if oldSize == 0 || newSize == 0 || oldSize > maxMremapBytes64 || newSize > maxMremapBytes64 {
		return int64(EINVAL)
	}
	if uint64(oldBase)&(corecpu.Page64Size-1) != 0 {
		return int64(EINVAL)
	}
	if flags&^(mremapMayMove64|mremapFixed64|mremapDontUnmap64) != 0 || flags&mremapFixed64 != 0 && flags&mremapMayMove64 == 0 {
		return int64(EINVAL)
	}
	if flags&mremapDontUnmap64 != 0 {
		return int64(ENOSYS)
	}
	oldPages, ok := pagesFor64(oldSize)
	if !ok || !mappingRangeMapped64(ctx.Memory, oldBase, oldPages) {
		return int64(EINVAL)
	}
	newPages, ok := pagesFor64(newSize)
	if !ok {
		return int64(EINVAL)
	}
	oldLength := oldPages * corecpu.Page64Size
	newLength := newPages * corecpu.Page64Size
	baseFlags := mappingFlags64(ctx.Memory, oldBase, oldPages)
	if baseFlags == 0 && oldPages != 0 {
		// A PROT_NONE mapping is valid; retain an explicit anonymous page flag.
		baseFlags = corecpu.PAnonymous
	}

	if newPages <= oldPages {
		if newPages < oldPages {
			if err := ctx.Memory.UnmapAlways(oldBase+corecpu.Address64(newLength), oldLength-newLength); err != nil {
				return int64(EINVAL)
			}
		}
		ctx.updateMapping64(oldBase, oldLength, oldBase, newSize, newPages)
		return int64(oldBase)
	}

	if mappingRangeFree64(ctx.Memory, oldBase+corecpu.Address64(oldLength), newPages-oldPages) {
		if err := ctx.Memory.Map(oldBase+corecpu.Address64(oldLength), (newPages-oldPages)*corecpu.Page64Size, baseFlags); err != nil {
			return int64(ENOMEM)
		}
		ctx.updateMapping64(oldBase, oldLength, oldBase, newSize, newPages)
		return int64(oldBase)
	}
	if flags&mremapMayMove64 == 0 {
		return int64(ENOMEM)
	}

	newBase := corecpu.Address64(args[4])
	fixed := flags&mremapFixed64 != 0
	if fixed {
		if uint64(newBase)&(corecpu.Page64Size-1) != 0 {
			return int64(EINVAL)
		}
		if !mappingRangeFree64(ctx.Memory, newBase, newPages) {
			return int64(EINVAL)
		}
	} else {
		var result int64
		newBase, result = findHole64(ctx.Memory, newBase, newPages, false)
		if result != 0 {
			return result
		}
	}

	data, err := readRawRange64(ctx.Memory, oldBase, oldLength, oldPages)
	if err != nil {
		return int64(EFAULT)
	}
	mapFlags := baseFlags | corecpu.PWrite
	if err := ctx.Memory.Map(newBase, newLength, mapFlags); err != nil {
		return int64(ENOMEM)
	}
	if err := ctx.Memory.Write(newBase, data); err != nil {
		_ = ctx.Memory.UnmapAlways(newBase, newLength)
		return int64(EFAULT)
	}
	if err := ctx.Memory.SetFlags(newBase, newLength, baseFlags); err != nil {
		_ = ctx.Memory.UnmapAlways(newBase, newLength)
		return int64(EFAULT)
	}
	if err := ctx.Memory.UnmapAlways(oldBase, oldLength); err != nil {
		_ = ctx.Memory.UnmapAlways(newBase, newLength)
		return int64(EINVAL)
	}
	ctx.updateMapping64(oldBase, oldLength, newBase, newSize, newPages)
	return int64(newBase)
}

func mappingRangeMapped64(memory *corecpu.Memory64, base corecpu.Address64, pages uint64) bool {
	if memory == nil || pages == 0 {
		return false
	}
	first := uint64(base) >> corecpu.Page64Bits
	for page := uint64(0); page < pages; page++ {
		if _, ok := memory.MappingFlags(corecpu.Page64(first + page)); !ok {
			return false
		}
	}
	return true
}

func mappingFlags64(memory *corecpu.Memory64, base corecpu.Address64, pages uint64) corecpu.Flags {
	if memory == nil || pages == 0 {
		return 0
	}
	flags, ok := memory.MappingFlags(corecpu.Page64(uint64(base) >> corecpu.Page64Bits))
	if !ok {
		return 0
	}
	return flags
}

func readRawRange64(memory *corecpu.Memory64, base corecpu.Address64, length, pages uint64) ([]byte, error) {
	if memory == nil || length == 0 || length > uint64(^uint(0)>>1) {
		return nil, corecpu.ErrRange
	}
	flags := make([]corecpu.Flags, pages)
	first := uint64(base) >> corecpu.Page64Bits
	for page := uint64(0); page < pages; page++ {
		value, ok := memory.MappingFlags(corecpu.Page64(first + page))
		if !ok {
			return nil, corecpu.ErrUnmapped
		}
		flags[page] = value
	}
	if err := memory.SetFlags(base, pages*corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		return nil, err
	}
	data := make([]byte, int(length))
	readErr := memory.Read(base, data)
	for page := uint64(0); page < pages; page++ {
		pageBase := corecpu.Address64((first + page) << corecpu.Page64Bits)
		_ = memory.SetFlags(pageBase, corecpu.Page64Size, flags[page])
	}
	if readErr != nil {
		return nil, readErr
	}
	return data, nil
}

func (ctx *Context64) updateMapping64(oldBase corecpu.Address64, oldLength uint64, newBase corecpu.Address64, newLength uint64, newPages uint64) {
	if ctx == nil {
		return
	}
	for index := range ctx.Mappings {
		mapping := &ctx.Mappings[index]
		if mapping.Base != oldBase || mapping.Length > oldLength {
			continue
		}
		mapping.Base = newBase
		mapping.Length = newLength
		mapping.Pages = newPages
		return
	}
	ctx.removeMappings64(oldBase, oldLength)
}

func madvise64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[1] == 0 {
		return int64(EINVAL)
	}
	if args[0]&(corecpu.Page64Size-1) != 0 {
		return int64(EINVAL)
	}
	if args[2] > madvisePopulateWrite64 {
		return int64(EINVAL)
	}
	pages, ok := pagesFor64(args[1])
	if !ok || !mappingRangeMapped64(ctx.Memory, corecpu.Address64(args[0]), pages) {
		return int64(ENOMEM)
	}
	// Advisory state is intentionally a no-op in the sparse guest memory model.
	// Mapping and page contents remain unchanged for every accepted Linux advice.
	return 0
}

func faccessat2_64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.FS == nil {
		return int64(ENOSYS)
	}
	if args[2]&^accessModeMask64 != 0 || args[3]&^faccessAtFlags64 != 0 {
		return int64(EINVAL)
	}
	flags := args[3]
	name := ""
	if args[1] == 0 {
		if flags&faccessAtEmptyPath64 == 0 {
			return int64(EFAULT)
		}
	} else {
		var ok bool
		name, ok = readGuestString64(ctx, corecpu.Address64(args[1]), 4096)
		if !ok {
			return int64(EFAULT)
		}
	}
	var resolved string
	if name == "" && flags&faccessAtEmptyPath64 != 0 {
		var result int64
		resolved, result = resolveFDPath64(ctx, args[0])
		if result != 0 {
			return result
		}
	} else {
		if name == "" {
			return int64(ENOENT)
		}
		var result int64
		resolved, result = resolveAtPath64(ctx, args[0], name)
		if result != 0 {
			return result
		}
	}
	var err error
	if flags&faccessAtSymlinkNoFollow64 != 0 {
		_, err = ctx.FS.Lstat(resolved)
	} else {
		_, err = ctx.FS.Stat(resolved)
	}
	if err != nil {
		return int64(errnoForOpen(err))
	}
	// The guest currently runs with root credentials; permission enforcement is
	// deferred to the credential layer, matching the ABI32 access implementation.
	return 0
}
