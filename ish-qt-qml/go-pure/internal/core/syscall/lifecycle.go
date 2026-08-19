package syscall

import "encoding/binary"

import corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"

// forkStub and cloneStub intentionally fail closed until ProcessManager can
// clone address spaces, descriptor tables, and parent/child lifecycle state.
// Returning ENOSYS is safer than pretending a child exists.
func forkStub(*Context, *corecpu.MachineState, [6]uint32) int32 {
	return ENOSYS
}

func cloneStub(*Context, *corecpu.MachineState, [6]uint32) int32 {
	return ENOSYS
}

// wait4 consumes exited children from ChildRegistry and writes the standard
// low-byte exit status encoding into the guest status pointer.
func wait4(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil || context.Children == nil {
		return ECHILD
	}
	pid, status, err := context.Children.Wait(context.PID, int32(args[0]), args[2])
	if err != 0 {
		return err
	}
	if pid == 0 {
		return 0
	}
	if args[1] != 0 {
		if context.Memory == nil {
			return EFAULT
		}
		var encoded [4]byte
		binary.LittleEndian.PutUint32(encoded[:], uint32(status))
		if err := context.Memory.Write(corecpu.Address(args[1]), encoded[:]); err != nil {
			return EFAULT
		}
	}
	return pid
}

func kill(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil {
		return ESRCH
	}
	target := int32(args[0])
	if target == 0 || target == -1 || uint32(target) == context.PID {
		// Signal delivery is not installed yet. Treating a signal to the
		// current process as accepted keeps common probing code progressing.
		return 0
	}
	return ESRCH
}

func signalStub(*Context, *corecpu.MachineState, [6]uint32) int32 {
	return ENOSYS
}

func gettid(context *Context, _ *corecpu.MachineState, _ [6]uint32) int32 {
	if context == nil {
		return ESRCH
	}
	return int32(context.PID)
}

func setTIDAddress(context *Context, _ *corecpu.MachineState, args [6]uint32) int32 {
	if context == nil {
		return ESRCH
	}
	context.TIDAddress = args[0]
	return int32(context.PID)
}
