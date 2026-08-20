package syscall

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

// The constants below mirror the stable part of the Linux io_uring userspace
// ABI. This implementation intentionally provides the mmap-backed ring mode;
// SQPOLL, IOPOLL, no-mmap rings, and 128-byte SQEs are rejected explicitly.
const (
	ioUringParamsSize64 = 120
	ioUringSQESize64    = 64
	ioUringCQESize64    = 16
	ioUringSQArray64    = 64
	ioUringRingHeader64 = ioUringSQArray64
	ioUringCQEOffset64  = 20

	ioUringSetupIOPoll64    = 1 << 0
	ioUringSetupSQPoll64    = 1 << 1
	ioUringSetupSQAff64     = 1 << 2
	ioUringSetupCQSize64    = 1 << 3
	ioUringSetupClamp64     = 1 << 4
	ioUringSetupSQE12864    = 1 << 10
	ioUringSetupCQE3264     = 1 << 11
	ioUringSetupNoMmap64    = 1 << 14
	ioUringSetupNoSQArray64 = 1 << 16

	ioUringEnterGetEvents64 = 1 << 0

	ioUringRegisterBuffers64   = 0
	ioUringUnregisterBuffers64 = 1
	ioUringRegisterFiles64     = 2
	ioUringUnregisterFiles64   = 3
	ioUringRegisterProbe64     = 8

	ioUringOpNOP64     = 0
	ioUringOpFSync64   = 3
	ioUringOpTimeout64 = 6
	ioUringOpRead64    = 22
	ioUringOpWrite64   = 23

	ioUringSQEFixedFile64 = 1 << 0

	ioUringOffSQRing64 = uint64(0)
	ioUringOffCQRing64 = uint64(0x08000000)
	ioUringOffSQES64   = uint64(0x10000000)

	ioUringFeatureSingleMMap64 = 1 << 0
	ioUringProbeOpSupported64  = 1 << 0
)

type ioUring64 struct {
	mu   sync.Mutex
	wake *sync.Cond

	closed bool

	sqEntries uint32
	cqEntries uint32

	sqRingBase corecpu.Address64
	cqRingBase corecpu.Address64
	sqesBase   corecpu.Address64

	sqRingLength uint64
	cqRingLength uint64
	sqesLength   uint64

	registeredBuffers []guestIOVec64
	registeredFiles   []int32
}

func (r *ioUring64) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	r.closed = true
	r.registeredBuffers = nil
	r.registeredFiles = nil
	if r.wake != nil {
		r.wake.Broadcast()
	}
	r.mu.Unlock()
	return nil
}

func ioUringReadU32(ctx *Context64, address corecpu.Address64) (uint32, int64) {
	var raw [4]byte
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Read(address, raw[:]) != nil {
		return 0, int64(EFAULT)
	}
	return binary.LittleEndian.Uint32(raw[:]), 0
}

func ioUringWriteU32(ctx *Context64, address corecpu.Address64, value uint32) int64 {
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], value)
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Write(address, raw[:]) != nil {
		return int64(EFAULT)
	}
	return 0
}

func ioUringReadU64(ctx *Context64, address corecpu.Address64) (uint64, int64) {
	var raw [8]byte
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Read(address, raw[:]) != nil {
		return 0, int64(EFAULT)
	}
	return binary.LittleEndian.Uint64(raw[:]), 0
}

func ioUringWriteU64(ctx *Context64, address corecpu.Address64, value uint64) int64 {
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], value)
	if ctx == nil || ctx.Memory == nil || ctx.Memory.Write(address, raw[:]) != nil {
		return int64(EFAULT)
	}
	return 0
}

func ioUringRoundPow2(value, maximum uint32, clamp bool) (uint32, int64) {
	if value == 0 {
		return 0, int64(EINVAL)
	}
	if value > maximum {
		if !clamp {
			return 0, int64(EINVAL)
		}
		return maximum, 0
	}
	rounded := uint32(1)
	for rounded < value {
		if rounded > maximum/2 {
			if !clamp {
				return 0, int64(EINVAL)
			}
			return maximum, 0
		}
		rounded <<= 1
	}
	return rounded, 0
}

func ioUringAlignPage64(length uint64) (uint64, bool) {
	if length == 0 || length > ^uint64(0)-(corecpu.Page64Size-1) {
		return 0, false
	}
	return (length + corecpu.Page64Size - 1) & ^(corecpu.Page64Size - 1), true
}

func ioUringRingLengths64(sqEntries, cqEntries uint32) (uint64, uint64, uint64, bool) {
	sqBytes := uint64(ioUringRingHeader64) + uint64(sqEntries)*4
	cqBytes := uint64(ioUringCQEOffset64) + uint64(cqEntries)*ioUringCQESize64
	sqLength, okSQ := ioUringAlignPage64(sqBytes)
	cqLength, okCQ := ioUringAlignPage64(cqBytes)
	sqesLength, okSQE := ioUringAlignPage64(uint64(sqEntries) * ioUringSQESize64)
	return sqLength, cqLength, sqesLength, okSQ && okCQ && okSQE
}

func ioUringSetup64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.FDs == nil {
		return int64(ENOMEM)
	}
	entries := uint32(args[0])
	params := corecpu.Address64(args[1])
	if args[0] == 0 || uint64(entries) != args[0] || params == 0 {
		return int64(EINVAL)
	}
	if err := ctx.Memory.Read(params, make([]byte, ioUringParamsSize64)); err != nil {
		return int64(EFAULT)
	}

	var raw [ioUringParamsSize64]byte
	if err := ctx.Memory.Read(params, raw[:]); err != nil {
		return int64(EFAULT)
	}
	requestedCQ := binary.LittleEndian.Uint32(raw[4:8])
	flags := binary.LittleEndian.Uint32(raw[8:12])
	unsupported := uint32(ioUringSetupIOPoll64 | ioUringSetupSQPoll64 | ioUringSetupSQAff64 | ioUringSetupSQE12864 | ioUringSetupCQE3264 | ioUringSetupNoMmap64 | ioUringSetupNoSQArray64)
	if flags&unsupported != 0 || flags&^(ioUringSetupCQSize64|ioUringSetupClamp64) != 0 {
		return int64(EINVAL)
	}
	clamp := flags&ioUringSetupClamp64 != 0
	sqEntries, result := ioUringRoundPow2(entries, 4096, clamp)
	if result != 0 {
		return result
	}
	cqEntries := sqEntries * 2
	if flags&ioUringSetupCQSize64 != 0 {
		if requestedCQ == 0 {
			return int64(EINVAL)
		}
		cqEntries, result = ioUringRoundPow2(requestedCQ, 8192, clamp)
		if result != 0 {
			return result
		}
	}
	if cqEntries < sqEntries {
		cqEntries = sqEntries
	}
	sqLength, cqLength, sqesLength, ok := ioUringRingLengths64(sqEntries, cqEntries)
	if !ok {
		return int64(EINVAL)
	}

	ring := &ioUring64{sqEntries: sqEntries, cqEntries: cqEntries, sqRingLength: sqLength, cqRingLength: cqLength, sqesLength: sqesLength}
	ring.wake = sync.NewCond(&ring.mu)
	file := &corefd.File{Opaque: ring, Closer: ring, Cloexec: true}
	fd, err := ctx.FDs.OpenWithCloexec(file, true)
	if err != nil {
		return int64(EMFILE)
	}

	for i := range raw {
		raw[i] = 0
	}
	binary.LittleEndian.PutUint32(raw[0:4], sqEntries)
	binary.LittleEndian.PutUint32(raw[4:8], cqEntries)
	binary.LittleEndian.PutUint32(raw[8:12], flags&ioUringSetupCQSize64|flags&ioUringSetupClamp64)
	binary.LittleEndian.PutUint32(raw[20:24], 0)
	// sq_off begins at byte 40 and cq_off at byte 80.
	binary.LittleEndian.PutUint32(raw[40:44], 0)
	binary.LittleEndian.PutUint32(raw[44:48], 4)
	binary.LittleEndian.PutUint32(raw[48:52], 8)
	binary.LittleEndian.PutUint32(raw[52:56], 12)
	binary.LittleEndian.PutUint32(raw[56:60], 16)
	binary.LittleEndian.PutUint32(raw[60:64], 20)
	binary.LittleEndian.PutUint32(raw[64:68], ioUringSQArray64)
	binary.LittleEndian.PutUint64(raw[72:80], 0)
	binary.LittleEndian.PutUint32(raw[80:84], 0)
	binary.LittleEndian.PutUint32(raw[84:88], 4)
	binary.LittleEndian.PutUint32(raw[88:92], 8)
	binary.LittleEndian.PutUint32(raw[92:96], 12)
	binary.LittleEndian.PutUint32(raw[96:100], 16)
	binary.LittleEndian.PutUint32(raw[100:104], ioUringCQEOffset64)
	binary.LittleEndian.PutUint32(raw[104:108], 24)
	binary.LittleEndian.PutUint64(raw[112:120], 0)
	if err := ctx.Memory.Write(params, raw[:]); err != nil {
		_ = ctx.FDs.Close(fd)
		return int64(EFAULT)
	}
	return int64(fd)
}

func mmapIoUring64(ctx *Context64, ring *ioUring64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ring == nil {
		return int64(EBADF)
	}
	addr, length, prot, flags, offset := args[0], args[1], args[2], args[3], args[5]
	if length == 0 || prot&^uint64(ProtRead|ProtWrite) != 0 || prot&uint64(ProtRead|ProtWrite) != uint64(ProtRead|ProtWrite) {
		return int64(EINVAL)
	}
	if flags&uint64(MapShared) == 0 || flags&uint64(MapPrivate) != 0 || flags&uint64(MapAnonymous) != 0 {
		return int64(EINVAL)
	}
	var required uint64
	var kind *corecpu.Address64
	ring.mu.Lock()
	if ring.closed {
		ring.mu.Unlock()
		return int64(EBADF)
	}
	switch offset {
	case ioUringOffSQRing64:
		required = ring.sqRingLength
		kind = &ring.sqRingBase
	case ioUringOffCQRing64:
		required = ring.cqRingLength
		kind = &ring.cqRingBase
	case ioUringOffSQES64:
		required = ring.sqesLength
		kind = &ring.sqesBase
	default:
		ring.mu.Unlock()
		return int64(EINVAL)
	}
	if length < required {
		ring.mu.Unlock()
		return int64(EINVAL)
	}
	if *kind != 0 {
		ring.mu.Unlock()
		return int64(EEXIST)
	}
	ring.mu.Unlock()

	pages, ok := pagesFor64(length)
	if !ok {
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
	mapLength := pages * corecpu.Page64Size
	if err := ctx.Memory.Map(base, mapLength, corecpu.PRead|corecpu.PWrite|corecpu.PShared); err != nil {
		return int64(ENOMEM)
	}
	finalFlags := corecpu.PRead | corecpu.PWrite | corecpu.PShared
	if offset == ioUringOffSQRing64 {
		if result := ioUringWriteU32(ctx, base, 0); result != 0 || ioUringWriteU32(ctx, base+4, 0) != 0 || ioUringWriteU32(ctx, base+8, ring.sqEntries-1) != 0 || ioUringWriteU32(ctx, base+12, ring.sqEntries) != 0 || ioUringWriteU32(ctx, base+16, 0) != 0 || ioUringWriteU32(ctx, base+20, 0) != 0 {
			_ = ctx.Memory.UnmapAlways(base, mapLength)
			return int64(EFAULT)
		}
	} else if offset == ioUringOffCQRing64 {
		if result := ioUringWriteU32(ctx, base, 0); result != 0 || ioUringWriteU32(ctx, base+4, 0) != 0 || ioUringWriteU32(ctx, base+8, ring.cqEntries-1) != 0 || ioUringWriteU32(ctx, base+12, ring.cqEntries) != 0 || ioUringWriteU32(ctx, base+16, 0) != 0 || ioUringWriteU32(ctx, base+20, ioUringCQEOffset64) != 0 {
			_ = ctx.Memory.UnmapAlways(base, mapLength)
			return int64(EFAULT)
		}
	}
	if err := ctx.Memory.SetFlags(base, mapLength, finalFlags); err != nil {
		_ = ctx.Memory.UnmapAlways(base, mapLength)
		return int64(ENOMEM)
	}
	ctx.addMapping64(GuestMapping64{Base: base, Length: length, Pages: pages, Offset: offset, Prot: prot, Shared: true, Special: ring})

	ring.mu.Lock()
	if *kind == 0 {
		*kind = base
	} else {
		// A racing duplicate mapping is not useful to the emulation because the
		// ring is represented by one guest address in the virtual machine.
		ring.mu.Unlock()
		_ = ctx.Memory.UnmapAlways(base, mapLength)
		ctx.removeMappings64(base, length)
		return int64(EEXIST)
	}
	ring.mu.Unlock()
	return int64(base)
}

func ioUringRingFromFD64(ctx *Context64, fd uint64) (*ioUring64, int64) {
	if ctx == nil || fd > maxFD64 {
		return nil, int64(EBADF)
	}
	file, err := ctx.GetFile(fd)
	if err != nil || file == nil {
		return nil, int64(EBADF)
	}
	ring, ok := file.Opaque.(*ioUring64)
	if !ok || ring == nil {
		return nil, int64(EINVAL)
	}
	ring.mu.Lock()
	closed := ring.closed
	ring.mu.Unlock()
	if closed {
		return nil, int64(EBADF)
	}
	return ring, 0
}

func ioUringValidateRange64(ctx *Context64, address corecpu.Address64, length uint64) bool {
	if ctx == nil || ctx.Memory == nil {
		return false
	}
	if length == 0 {
		return true
	}
	if address > corecpu.Address64(^uint64(0)-length+1) || length > uint64(^uint(0)>>1) {
		return false
	}
	if err := ctx.Memory.Read(address, make([]byte, 1)); err != nil {
		return false
	}
	last := address + corecpu.Address64(length-1)
	return ctx.Memory.Read(last, make([]byte, 1)) == nil
}

func ioUringResolveFile64(ctx *Context64, ring *ioUring64, fd int32, fixed bool) (*corefd.File, int64) {
	if fixed {
		if fd < 0 || int(fd) >= len(ring.registeredFiles) || ring.registeredFiles[fd] < 0 {
			return nil, int64(EBADF)
		}
		fd = ring.registeredFiles[fd]
	}
	if fd < 0 {
		return nil, int64(EBADF)
	}
	file, err := ctx.GetFile(uint64(fd))
	if err != nil || file == nil {
		return nil, int64(EBADF)
	}
	return file, 0
}

func ioUringReadAt64(file *corefd.File, dst []byte, offset uint64) (int, error) {
	if file == nil || file.Reader == nil {
		return 0, corefd.ErrBadFD
	}
	if offset == ^uint64(0) {
		return file.Read(dst)
	}
	if offset > uint64(^uint64(0)>>1) {
		return 0, errors.New("io_uring: offset overflow")
	}
	if reader, ok := file.Reader.(io.ReaderAt); ok {
		return reader.ReadAt(dst, int64(offset))
	}
	if file.Seeker == nil {
		return 0, corefd.ErrNotSeek
	}
	current, err := file.Seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	if _, err = file.Seeker.Seek(int64(offset), io.SeekStart); err != nil {
		return 0, err
	}
	n, readErr := file.Read(dst)
	_, _ = file.Seeker.Seek(current, io.SeekStart)
	return n, readErr
}

func ioUringWriteAt64(file *corefd.File, src []byte, offset uint64) (int, error) {
	if file == nil || file.Writer == nil {
		return 0, corefd.ErrBadFD
	}
	if offset == ^uint64(0) {
		return file.Write(src)
	}
	if offset > uint64(^uint64(0)>>1) {
		return 0, errors.New("io_uring: offset overflow")
	}
	if writer, ok := file.Writer.(io.WriterAt); ok {
		return writer.WriteAt(src, int64(offset))
	}
	if file.Seeker == nil {
		return 0, corefd.ErrNotSeek
	}
	current, err := file.Seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	if _, err = file.Seeker.Seek(int64(offset), io.SeekStart); err != nil {
		return 0, err
	}
	n, writeErr := file.Write(src)
	_, _ = file.Seeker.Seek(current, io.SeekStart)
	return n, writeErr
}

func ioUringErrno64(err error) int32 {
	if err == nil || errors.Is(err, io.EOF) {
		return 0
	}
	if errors.Is(err, corefd.ErrBadFD) {
		return EBADF
	}
	if errors.Is(err, corefd.ErrNotSeek) {
		return -29 // ESPIPE: descriptor is not seekable.
	}
	return EIO
}

func ioUringEnter64(ctx *Context64, args [6]uint64) int64 {
	ring, result := ioUringRingFromFD64(ctx, args[0])
	if result != 0 {
		return result
	}
	toSubmit, minComplete, enterFlags := args[1], args[2], args[3]
	if enterFlags&^uint64(ioUringEnterGetEvents64) != 0 || toSubmit > uint64(ring.sqEntries) {
		return int64(EINVAL)
	}
	if ring.sqRingBase == 0 || ring.cqRingBase == 0 || ring.sqesBase == 0 {
		return int64(EINVAL)
	}

	ring.mu.Lock()
	defer ring.mu.Unlock()
	if ring.closed {
		return int64(EBADF)
	}
	sqHead, result := ioUringReadU32(ctx, ring.sqRingBase)
	if result != 0 {
		return result
	}
	sqTail, result := ioUringReadU32(ctx, ring.sqRingBase+4)
	if result != 0 {
		return result
	}
	pending := uint32(sqTail - sqHead)
	if pending > ring.sqEntries {
		return int64(EOVERFLOW)
	}
	if uint64(pending) < toSubmit {
		toSubmit = uint64(pending)
	}
	sqMask, result := ioUringReadU32(ctx, ring.sqRingBase+8)
	if result != 0 {
		return result
	}
	processed := uint64(0)
	for processed < toSubmit {
		slot := (sqHead + uint32(processed)) & sqMask
		arrayEntry, result := ioUringReadU32(ctx, ring.sqRingBase+ioUringRingHeader64+corecpu.Address64(slot*4))
		if result != 0 {
			return result
		}
		if arrayEntry >= ring.sqEntries {
			if queueResult := ioUringQueueCQE64(ctx, ring, 0, int32(EINVAL), 0); queueResult != 0 {
				return queueResult
			}
			processed++
			continue
		}
		sqeAddress := ring.sqesBase + corecpu.Address64(arrayEntry*ioUringSQESize64)
		if !ioUringValidateRange64(ctx, sqeAddress, ioUringSQESize64) {
			return int64(EFAULT)
		}
		var sqe [ioUringSQESize64]byte
		if err := ctx.Memory.Read(sqeAddress, sqe[:]); err != nil {
			return int64(EFAULT)
		}
		userData := binary.LittleEndian.Uint64(sqe[32:40])
		cqeResult := ioUringExecuteSQE64(ctx, ring, sqe)
		if queueResult := ioUringQueueCQE64(ctx, ring, userData, cqeResult, 0); queueResult != 0 {
			return queueResult
		}
		processed++
	}
	if processed > 0 {
		if result := ioUringWriteU32(ctx, ring.sqRingBase, sqHead+uint32(processed)); result != 0 {
			return result
		}
	}
	if enterFlags&ioUringEnterGetEvents64 != 0 && minComplete != 0 {
		for {
			cqHead, headResult := ioUringReadU32(ctx, ring.cqRingBase)
			if headResult != 0 {
				return headResult
			}
			cqTail, tailResult := ioUringReadU32(ctx, ring.cqRingBase+4)
			if tailResult != 0 {
				return tailResult
			}
			if uint32(cqTail-cqHead) >= uint32(minComplete) || ring.closed {
				break
			}
			// This implementation executes supported SQEs synchronously. There
			// is no background SQPOLL worker that could create a future CQE, so
			// do not spin or deadlock when a caller asks for an unavailable event.
			return int64(EAGAIN)
		}
	}
	return int64(processed)
}

func ioUringExecuteSQE64(ctx *Context64, ring *ioUring64, sqe [ioUringSQESize64]byte) int32 {
	opcode := sqe[0]
	flags := sqe[1]
	fd := int32(binary.LittleEndian.Uint32(sqe[4:8]))
	offset := binary.LittleEndian.Uint64(sqe[8:16])
	address := corecpu.Address64(binary.LittleEndian.Uint64(sqe[16:24]))
	length := uint64(binary.LittleEndian.Uint32(sqe[24:28]))
	fsyncFlags := binary.LittleEndian.Uint32(sqe[28:32])

	switch opcode {
	case ioUringOpNOP64:
		return 0
	case ioUringOpRead64, ioUringOpWrite64:
		if length > 16<<20 || length > uint64(^uint(0)>>1) || !ioUringValidateRange64(ctx, address, length) {
			return EFAULT
		}
		file, result := ioUringResolveFile64(ctx, ring, fd, flags&ioUringSQEFixedFile64 != 0)
		if result != 0 {
			return int32(result)
		}
		buffer := make([]byte, int(length))
		if opcode == ioUringOpWrite64 {
			if err := ctx.Memory.Read(address, buffer); err != nil {
				return EFAULT
			}
			n, err := ioUringWriteAt64(file, buffer, offset)
			if n > 0 {
				return int32(n)
			}
			if err != nil {
				return ioUringErrno64(err)
			}
			return 0
		}
		n, err := ioUringReadAt64(file, buffer, offset)
		if n > 0 {
			if writeErr := ctx.Memory.Write(address, buffer[:n]); writeErr != nil {
				return EFAULT
			}
			return int32(n)
		}
		if err != nil {
			return ioUringErrno64(err)
		}
		return 0
	case ioUringOpFSync64:
		file, result := ioUringResolveFile64(ctx, ring, fd, flags&ioUringSQEFixedFile64 != 0)
		if result != 0 {
			return int32(result)
		}
		if fsyncFlags != 0 {
			// The Pure-Go fakefs has no range sync distinction; accepting the
			// documented fsync flags is sufficient for the ABI-level operation.
		}
		if syncer, ok := file.Closer.(guestSyncer64); ok {
			if err := syncer.Sync(); err != nil {
				return int32(errnoForOpen(err))
			}
			return 0
		}
		if ctx.FS != nil {
			if syncer, ok := any(ctx.FS).(guestSyncer64); ok {
				if err := syncer.Sync(); err != nil {
					return int32(errnoForOpen(err))
				}
				return 0
			}
		}
		return 0
	case ioUringOpTimeout64:
		if !ioUringValidateRange64(ctx, address, 16) {
			return EFAULT
		}
		var timespec [16]byte
		if err := ctx.Memory.Read(address, timespec[:]); err != nil {
			return EFAULT
		}
		seconds := binary.LittleEndian.Uint64(timespec[0:8])
		nanoseconds := binary.LittleEndian.Uint64(timespec[8:16])
		maxDurationSeconds := uint64(^uint64(0)>>1) / uint64(time.Second)
		if nanoseconds >= 1_000_000_000 || seconds > maxDurationSeconds {
			return EINVAL
		}
		duration := time.Duration(seconds)*time.Second + time.Duration(nanoseconds)
		if duration > 0 {
			time.Sleep(duration)
		}
		return 0
	default:
		return EOPNOTSUPP
	}
}

func ioUringQueueCQE64(ctx *Context64, ring *ioUring64, userData uint64, result int32, flags uint32) int64 {
	cqHead, ret := ioUringReadU32(ctx, ring.cqRingBase)
	if ret != 0 {
		return ret
	}
	cqTail, ret := ioUringReadU32(ctx, ring.cqRingBase+4)
	if ret != 0 {
		return ret
	}
	if uint32(cqTail-cqHead) >= ring.cqEntries {
		overflow, overflowResult := ioUringReadU32(ctx, ring.cqRingBase+16)
		if overflowResult != 0 {
			return overflowResult
		}
		return ioUringWriteU32(ctx, ring.cqRingBase+16, overflow+1)
	}
	mask, ret := ioUringReadU32(ctx, ring.cqRingBase+8)
	if ret != 0 {
		return ret
	}
	slot := cqTail & mask
	entry := ring.cqRingBase + ioUringCQEOffset64 + corecpu.Address64(slot*ioUringCQESize64)
	var cqe [ioUringCQESize64]byte
	binary.LittleEndian.PutUint64(cqe[0:8], userData)
	binary.LittleEndian.PutUint32(cqe[8:12], uint32(result))
	binary.LittleEndian.PutUint32(cqe[12:16], flags)
	if err := ctx.Memory.Write(entry, cqe[:]); err != nil {
		return int64(EFAULT)
	}
	if ret = ioUringWriteU32(ctx, ring.cqRingBase+4, cqTail+1); ret != 0 {
		return ret
	}
	if ring.wake != nil {
		ring.wake.Broadcast()
	}
	return 0
}

func ioUringReleaseMapping64(ctx *Context64, base corecpu.Address64) {
	if ctx == nil {
		return
	}
	for _, mapping := range ctx.Mappings {
		if mapping.Base != base {
			continue
		}
		ring, ok := mapping.Special.(*ioUring64)
		if !ok || ring == nil {
			continue
		}
		ring.mu.Lock()
		switch mapping.Offset {
		case ioUringOffSQRing64:
			ring.sqRingBase = 0
		case ioUringOffCQRing64:
			ring.cqRingBase = 0
		case ioUringOffSQES64:
			ring.sqesBase = 0
		}
		ring.mu.Unlock()
	}
}

func ioUringRegister64(ctx *Context64, args [6]uint64) int64 {
	ring, result := ioUringRingFromFD64(ctx, args[0])
	if result != 0 {
		return result
	}
	opcode, arg, nrArgs := args[1], corecpu.Address64(args[2]), args[3]
	switch opcode {
	case ioUringRegisterBuffers64:
		if nrArgs > 1024 {
			return int64(EINVAL)
		}
		iovecs, result := readGuestIOVecs64(ctx, uint64(arg), nrArgs)
		if result != 0 {
			return result
		}
		for _, iovec := range iovecs {
			if iovec.Len > 16<<20 || !ioUringValidateRange64(ctx, iovec.Base, iovec.Len) {
				return int64(EFAULT)
			}
		}
		ring.mu.Lock()
		ring.registeredBuffers = append([]guestIOVec64(nil), iovecs...)
		ring.mu.Unlock()
		return int64(nrArgs)
	case ioUringUnregisterBuffers64:
		if nrArgs != 0 {
			return int64(EINVAL)
		}
		ring.mu.Lock()
		ring.registeredBuffers = nil
		ring.mu.Unlock()
		return 0
	case ioUringRegisterFiles64:
		if nrArgs > 1024 || nrArgs > uint64(^uint(0)>>1) {
			return int64(EINVAL)
		}
		if nrArgs > 0 && !ioUringValidateRange64(ctx, arg, nrArgs*4) {
			return int64(EFAULT)
		}
		fds := make([]int32, int(nrArgs))
		var raw [4]byte
		for i := range fds {
			if err := ctx.Memory.Read(arg+corecpu.Address64(i*4), raw[:]); err != nil {
				return int64(EFAULT)
			}
			value := binary.LittleEndian.Uint32(raw[:])
			if value == ^uint32(0) {
				fds[i] = -1
				continue
			}
			if value > uint32(maxFD64) {
				return int64(EBADF)
			}
			if _, err := ctx.GetFile(uint64(value)); err != nil {
				return int64(EBADF)
			}
			fds[i] = int32(value)
		}
		ring.mu.Lock()
		ring.registeredFiles = fds
		ring.mu.Unlock()
		return int64(nrArgs)
	case ioUringUnregisterFiles64:
		if nrArgs != 0 {
			return int64(EINVAL)
		}
		ring.mu.Lock()
		ring.registeredFiles = nil
		ring.mu.Unlock()
		return 0
	case ioUringRegisterProbe64:
		return ioUringRegisterProbe64Guest(ctx, arg, nrArgs)
	default:
		return int64(EOPNOTSUPP)
	}
}

func ioUringRegisterProbe64Guest(ctx *Context64, address corecpu.Address64, nrArgs uint64) int64 {
	if address == 0 || nrArgs == 0 || nrArgs > 255 {
		return int64(EINVAL)
	}
	if nrArgs > (^uint64(0)-16)/8 || !ioUringValidateRange64(ctx, address, 16+nrArgs*8) {
		return int64(EFAULT)
	}
	var header [16]byte
	header[0] = ioUringOpWrite64
	header[1] = byte(nrArgs)
	if err := ctx.Memory.Write(address, header[:]); err != nil {
		return int64(EFAULT)
	}
	supported := map[uint8]bool{
		ioUringOpNOP64: true, ioUringOpFSync64: true, ioUringOpTimeout64: true,
		ioUringOpRead64: true, ioUringOpWrite64: true,
	}
	var op [8]byte
	for index := uint64(0); index < nrArgs; index++ {
		for i := range op {
			op[i] = 0
		}
		op[0] = byte(index)
		if supported[byte(index)] {
			binary.LittleEndian.PutUint16(op[2:4], ioUringProbeOpSupported64)
		}
		if err := ctx.Memory.Write(address+16+corecpu.Address64(index*8), op[:]); err != nil {
			return int64(EFAULT)
		}
	}
	return 0
}
