package syscall

import (
	"encoding/binary"
	"sync"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	corefd "github.com/mostafa637/mostafa637/go-pure/internal/core/fd"
)

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
