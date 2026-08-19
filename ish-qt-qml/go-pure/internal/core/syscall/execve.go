package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	execPathLimit  = 4096
	execVectorSize = 4096
	execStringSize = 131072
)

func execve(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Memory == nil || context.Execve == nil {
		return ENOSYS
	}
	path, ok := readGuestString(context, state, corecpu.Address(args[0]), execPathLimit)
	if !ok || path == "" {
		return EFAULT
	}
	argv, ok := readGuestStringVector(context, state, corecpu.Address(args[1]))
	if !ok {
		return EFAULT
	}
	env, ok := readGuestStringVector(context, state, corecpu.Address(args[2]))
	if !ok {
		return EFAULT
	}
	return context.Execve(path, argv, env)
}

func readGuestStringVector(context *Context, state *corecpu.MachineState, address corecpu.Address) ([]string, bool) {
	if address == 0 {
		return nil, true
	}
	values := make([]string, 0, 8)
	for index := 0; index < execVectorSize; index++ {
		var raw [4]byte
		entry := address + corecpu.Address(index*4)
		if err := context.Memory.Read(entry, raw[:]); err != nil {
			return nil, false
		}
		pointer := corecpu.Address(binary.LittleEndian.Uint32(raw[:]))
		if pointer == 0 {
			return values, true
		}
		value, ok := readGuestString(context, state, pointer, execStringSize)
		if !ok {
			return nil, false
		}
		values = append(values, value)
	}
	return nil, false
}
