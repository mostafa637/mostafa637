package syscall

import (
	"encoding/binary"
	"time"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

// Linux x86-64 exposes struct sysinfo as a 112-byte structure. The explicit
// four-byte alignment hole after procs/pad is part of the native long layout.
const (
	sysinfoSize64       = 112
	sysinfoLoadShift64  = 16
	sysinfoMemUnit64    = uint64(1)
	guestRAMBytes64     = uint64(256 * 1024 * 1024)
	sysinfoUptimeOffset = 0
	sysinfoLoadsOffset  = 8
	sysinfoTotalRAM     = 32
	sysinfoFreeRAM      = 40
	sysinfoSharedRAM    = 48
	sysinfoBufferRAM    = 56
	sysinfoTotalSwap    = 64
	sysinfoFreeSwap     = 72
	sysinfoProcs        = 80
	sysinfoTotalHigh    = 88
	sysinfoFreeHigh     = 96
	sysinfoMemUnit      = 104
)

func mappedGuestBytes64(ctx *Context64) (mapped, shared uint64) {
	if ctx == nil {
		return 0, 0
	}
	for _, mapping := range ctx.Mappings {
		if mapping.Length == 0 {
			continue
		}
		if ^uint64(0)-mapped < mapping.Length {
			mapped = ^uint64(0)
		} else {
			mapped += mapping.Length
		}
		if mapping.Shared {
			if ^uint64(0)-shared < mapping.Length {
				shared = ^uint64(0)
			} else {
				shared += mapping.Length
			}
		}
	}
	return mapped, shared
}

func putSysinfo64U64(raw []byte, offset int, value uint64) {
	binary.LittleEndian.PutUint64(raw[offset:offset+8], value)
}

func sysinfo64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || args[0] == 0 {
		return int64(EFAULT)
	}

	raw := make([]byte, sysinfoSize64)
	uptime := uint64(0)
	if !ctx.StartTime.IsZero() {
		elapsed := time.Since(ctx.StartTime)
		if elapsed > 0 {
			uptime = uint64(elapsed / time.Second)
		}
	}
	putSysinfo64U64(raw, sysinfoUptimeOffset, uptime)

	// A single guest scheduler is represented by a load of 1.0 in the
	// traditional fixed-point sysinfo scale. This is intentionally independent
	// of host load so Android and Linux expose identical guest semantics.
	load := uint64(1) << sysinfoLoadShift64
	for i := 0; i < 3; i++ {
		putSysinfo64U64(raw, sysinfoLoadsOffset+i*8, load)
	}

	mapped, shared := mappedGuestBytes64(ctx)
	free := uint64(0)
	if mapped < guestRAMBytes64 {
		free = guestRAMBytes64 - mapped
	}
	if shared > guestRAMBytes64 {
		shared = guestRAMBytes64
	}
	putSysinfo64U64(raw, sysinfoTotalRAM, guestRAMBytes64)
	putSysinfo64U64(raw, sysinfoFreeRAM, free)
	putSysinfo64U64(raw, sysinfoSharedRAM, shared)
	putSysinfo64U64(raw, sysinfoBufferRAM, 0)
	putSysinfo64U64(raw, sysinfoTotalSwap, 0)
	putSysinfo64U64(raw, sysinfoFreeSwap, 0)
	binary.LittleEndian.PutUint16(raw[sysinfoProcs:sysinfoProcs+2], 1)
	// raw[82:88] remains the ABI pad/alignment area.
	putSysinfo64U64(raw, sysinfoTotalHigh, 0)
	putSysinfo64U64(raw, sysinfoFreeHigh, 0)
	binary.LittleEndian.PutUint32(raw[sysinfoMemUnit:sysinfoMemUnit+4], uint32(sysinfoMemUnit64))

	if err := ctx.Memory.Write(corecpu.Address64(args[0]), raw); err != nil {
		return int64(EFAULT)
	}
	return 0
}
