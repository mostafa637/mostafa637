package syscall

import (
	"sync"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

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

	ioUringOpNOP64        = 0
	ioUringOpReadV64      = 1
	ioUringOpWriteV64     = 2
	ioUringOpFSync64      = 3
	ioUringOpPollAdd64    = 6
	ioUringOpPollRemove64 = 7
	ioUringOpTimeout64    = 11
	ioUringOpRead64       = 22
	ioUringOpWrite64      = 23

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
