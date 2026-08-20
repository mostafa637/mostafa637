package syscall

import corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

func ioUringPrepareMmap64(ctx *Context64, addr, length, flags uint64) (uint64, int64) {
	pages, ok := pagesFor64(length)
	if !ok {
		return 0, int64(EINVAL)
	}
	fixed, replace := flags&uint64(MapFixed) != 0, flags&mapFixedNoReplace64 != 0
	if !fixed && !replace {
		return pages, 0
	}
	if addr == 0 || addr&(corecpu.Page64Size-1) != 0 || !range64MmapValid(corecpu.Address64(addr), pages) {
		return 0, int64(EINVAL)
	}
	free := mappingRangeFree64(ctx.Memory, corecpu.Address64(addr), pages)
	if replace && !free {
		return 0, int64(EEXIST)
	}
	if fixed && !free && ctx.Memory.UnmapAlways(corecpu.Address64(addr), pages*corecpu.Page64Size) != nil {
		return 0, int64(EINVAL)
	}
	return pages, 0
}

func ioUringAllocateMmap64(ctx *Context64, addr uint64, pages uint64, flags uint64) (corecpu.Address64, int64) {
	fixed := flags&uint64(MapFixed) != 0 || flags&mapFixedNoReplace64 != 0
	base, result := findHole64(ctx.Memory, corecpu.Address64(addr), pages, fixed)
	if result != 0 {
		return 0, result
	}
	length := pages * corecpu.Page64Size
	if err := ctx.Memory.Map(base, length, corecpu.PRead|corecpu.PWrite|corecpu.PShared); err != nil {
		return 0, int64(ENOMEM)
	}
	return base, 0
}

func ioUringInitMap64(ctx *Context64, ring *ioUring64, kind ioUringMapKind64, base corecpu.Address64, pages uint64) int64 {
	length := pages * corecpu.Page64Size
	if kind == ioUringMapSQ64 && ioUringInitSQMap64(ctx, ring, base) != 0 {
		_ = ctx.Memory.UnmapAlways(base, length)
		return int64(EFAULT)
	}
	if kind == ioUringMapCQ64 && ioUringInitCQMap64(ctx, ring, base) != 0 {
		_ = ctx.Memory.UnmapAlways(base, length)
		return int64(EFAULT)
	}
	if ctx.Memory.SetFlags(base, length, corecpu.PRead|corecpu.PWrite|corecpu.PShared) != nil {
		_ = ctx.Memory.UnmapAlways(base, length)
		return int64(ENOMEM)
	}
	return 0
}

func ioUringInitSQMap64(ctx *Context64, ring *ioUring64, base corecpu.Address64) int64 {
	values := []uint32{0, 0, ring.sqEntries - 1, ring.sqEntries, 0, 0}
	return ioUringWriteMapHeader64(ctx, base, values)
}

func ioUringInitCQMap64(ctx *Context64, ring *ioUring64, base corecpu.Address64) int64 {
	values := []uint32{0, 0, ring.cqEntries - 1, ring.cqEntries, 0, ioUringCQEOffset64}
	return ioUringWriteMapHeader64(ctx, base, values)
}

func ioUringWriteMapHeader64(ctx *Context64, base corecpu.Address64, values []uint32) int64 {
	for index, value := range values {
		if result := ioUringWriteU32(ctx, base+corecpu.Address64(index*4), value); result != 0 {
			return result
		}
	}
	return 0
}
