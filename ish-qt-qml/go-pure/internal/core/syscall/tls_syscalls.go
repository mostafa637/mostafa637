package syscall

import (
	"encoding/binary"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
)

const (
	userDescSize       = 16
	userDescEntry      = 0
	userDescBase       = 4
	userDescLimit      = 8
	userDescFlags      = 12
	userDescEmptyEntry = ^uint32(0)
	defaultTLSSelector = uint32(6)
)

// setThreadArea implements the i386 user_desc ABI for the single guest thread.
// The emulator stores the base directly in MachineState.TLS; segment descriptor
// emulation is intentionally deferred until the CPU executes a segment load.
func setThreadArea(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || state == nil || context.Memory == nil {
		return EFAULT
	}
	address := corecpu.Address(args[0])
	var raw [userDescSize]byte
	if err := context.Memory.Read(address, raw[:]); err != nil {
		return EFAULT
	}
	entry := binary.LittleEndian.Uint32(raw[userDescEntry : userDescEntry+4])
	if entry == userDescEmptyEntry {
		entry = defaultTLSSelector
		binary.LittleEndian.PutUint32(raw[userDescEntry:userDescEntry+4], entry)
	} else if entry != defaultTLSSelector {
		return EINVAL
	}
	base := binary.LittleEndian.Uint32(raw[userDescBase : userDescBase+4])
	limit := binary.LittleEndian.Uint32(raw[userDescLimit : userDescLimit+4])
	if limit == 0 {
		return EINVAL
	}
	context.TLSBase = base
	state.GSBase = base
	state.TLS = base
	if err := context.Memory.Write(address, raw[:]); err != nil {
		return EFAULT
	}
	return 0
}

func getThreadArea(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || state == nil || context.Memory == nil {
		return EFAULT
	}
	if args[0] != defaultTLSSelector {
		return EINVAL
	}
	var raw [userDescSize]byte
	binary.LittleEndian.PutUint32(raw[userDescEntry:userDescEntry+4], defaultTLSSelector)
	binary.LittleEndian.PutUint32(raw[userDescBase:userDescBase+4], context.TLSBase)
	binary.LittleEndian.PutUint32(raw[userDescLimit:userDescLimit+4], 0xfffff)
	binary.LittleEndian.PutUint32(raw[userDescFlags:userDescFlags+4], 0x51) // 32-bit, usable data descriptor.
	if err := context.Memory.Write(corecpu.Address(args[1]), raw[:]); err != nil {
		return EFAULT
	}
	return 0
}

func setRobustList(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil {
		return EFAULT
	}
	context.RobustListHead = args[0]
	context.RobustListLen = args[1]
	return 0
}

func getRobustList(context *Context, state *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || state == nil || context.Memory == nil {
		return EFAULT
	}
	pid := int32(args[0])
	if pid != 0 && uint32(pid) != context.PID {
		return ESRCH
	}
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], context.RobustListHead)
	if err := context.Memory.Write(corecpu.Address(args[1]), raw[:]); err != nil {
		return EFAULT
	}
	binary.LittleEndian.PutUint32(raw[:], context.RobustListLen)
	if err := context.Memory.Write(corecpu.Address(args[2]), raw[:]); err != nil {
		return EFAULT
	}
	return 0
}
