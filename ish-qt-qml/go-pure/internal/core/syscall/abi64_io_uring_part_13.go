package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

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
		ioUringOpNOP64: true, ioUringOpReadV64: true, ioUringOpWriteV64: true,
		ioUringOpFSync64: true, ioUringOpPollAdd64: true, ioUringOpTimeout64: true,
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
