package syscall

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

// wait4 has no child registry yet. ECHILD matches Linux's result when the
// calling task has no waitable children, while preserving the ABI shape.
func wait4(context *Context, _ *corecpu.MachineState, _ [6]uint32) int32 {
	if context == nil {
		return ECHILD
	}
	// A future ProcessManager will write the encoded wait status here. Until
	// then, no child is waitable from this single-process execution context.
	return ECHILD
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
