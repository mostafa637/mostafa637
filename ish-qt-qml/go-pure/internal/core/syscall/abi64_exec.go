package syscall

import (
	"encoding/binary"
	"math"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	execPathLimit64  = 4096
	execVectorSize64 = 4096
	execStringSize64 = 131072
)

// execve64 reads the Linux x86-64 execve argument vectors from guest memory.
// Successful image replacement is delegated to Context64.Execve so the session
// layer can preserve process identity and install the new ELF image.
func execve64(ctx *Context64, args [6]uint64) int64 {
	if ctx == nil || ctx.Memory == nil || ctx.Execve == nil {
		return int64(ENOSYS)
	}
	path, ok := readGuestString64(ctx, corecpu.Address64(args[0]), execPathLimit64)
	if !ok || path == "" {
		return int64(EFAULT)
	}
	argv, ok := readGuestStringVector64(ctx, corecpu.Address64(args[1]))
	if !ok {
		return int64(EFAULT)
	}
	env, ok := readGuestStringVector64(ctx, corecpu.Address64(args[2]))
	if !ok {
		return int64(EFAULT)
	}
	return ctx.Execve(path, argv, env)
}

func readGuestStringVector64(ctx *Context64, address corecpu.Address64) ([]string, bool) {
	if address == 0 {
		return nil, true
	}
	values := make([]string, 0, 8)
	for index := 0; index < execVectorSize64; index++ {
		if uint64(index) > math.MaxUint64/8 {
			return nil, false
		}
		entry := uint64(address) + uint64(index)*8
		if entry < uint64(address) {
			return nil, false
		}
		var raw [8]byte
		if err := ctx.Memory.Read(corecpu.Address64(entry), raw[:]); err != nil {
			return nil, false
		}
		pointer := binary.LittleEndian.Uint64(raw[:])
		if pointer == 0 {
			return values, true
		}
		value, ok := readGuestString64(ctx, corecpu.Address64(pointer), execStringSize64)
		if !ok {
			return nil, false
		}
		values = append(values, value)
	}
	return nil, false
}
