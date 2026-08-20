package syscall

import corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

type ioUringMapKind64 uint8

const (
	ioUringMapSQ64 ioUringMapKind64 = iota + 1
	ioUringMapCQ64
	ioUringMapSQE64
)

func mmapIoUring64(ctx *Context64, ring *ioUring64, args [6]uint64) int64 {
	addr, length, prot, flags, offset, result := ioUringMmapArgs64(args)
	if result != 0 || ctx == nil || ctx.Memory == nil || ring == nil {
		return firstIoUringError64(result, int64(EBADF))
	}
	kind, required, result := ioUringMapInfo64(ring, offset, length)
	if result != 0 {
		return result
	}
	pages, result := ioUringPrepareMmap64(ctx, addr, length, flags)
	if result != 0 || length < required {
		return firstIoUringError64(result, int64(EINVAL))
	}
	base, result := ioUringAllocateMmap64(ctx, addr, pages, flags)
	if result != 0 {
		return result
	}
	if result = ioUringInitMap64(ctx, ring, kind, base, pages); result != 0 {
		return result
	}
	ctx.addMapping64(GuestMapping64{Base: base, Length: length, Pages: pages, Offset: offset, Prot: prot, Shared: true, Special: ring})
	return ioUringPublishMap64(ctx, ring, kind, base, length)
}

func ioUringMmapArgs64(args [6]uint64) (uint64, uint64, uint64, uint64, uint64, int64) {
	addr, length, prot, flags, offset := args[0], args[1], args[2], args[3], args[5]
	if length == 0 || prot&^uint64(ProtRead|ProtWrite) != 0 || prot&uint64(ProtRead|ProtWrite) != uint64(ProtRead|ProtWrite) {
		return 0, 0, 0, 0, 0, int64(EINVAL)
	}
	if flags&uint64(MapShared) == 0 || flags&uint64(MapPrivate|MapAnonymous) != 0 {
		return 0, 0, 0, 0, 0, int64(EINVAL)
	}
	return addr, length, prot, flags, offset, 0
}

func ioUringMapInfo64(ring *ioUring64, offset, length uint64) (ioUringMapKind64, uint64, int64) {
	ring.mu.Lock()
	defer ring.mu.Unlock()
	if ring.closed {
		return 0, 0, int64(EBADF)
	}
	kind, required, base := ioUringMapSlot64(ring, offset)
	if kind == 0 || length < required {
		return 0, 0, int64(EINVAL)
	}
	if base != 0 {
		return 0, 0, int64(EEXIST)
	}
	return kind, required, 0
}

func ioUringMapSlot64(ring *ioUring64, offset uint64) (ioUringMapKind64, uint64, corecpu.Address64) {
	switch offset {
	case ioUringOffSQRing64:
		return ioUringMapSQ64, ring.sqRingLength, ring.sqRingBase
	case ioUringOffCQRing64:
		return ioUringMapCQ64, ring.cqRingLength, ring.cqRingBase
	case ioUringOffSQES64:
		return ioUringMapSQE64, ring.sqesLength, ring.sqesBase
	default:
		return 0, 0, 0
	}
}
