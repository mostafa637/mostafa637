package syscall

import (
	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/wasmjit"
)

func (d *Dispatcher64) WasmHandler() wasmjit.SyscallHandler {
	return func(number uint64, args [6]uint64) uint64 {
		return dispatchWasm64(d, number, args)
	}
}

func dispatchWasm64(d *Dispatcher64, number uint64, args [6]uint64) uint64 {
	if d == nil || d.Context == nil {
		return uint64(-int64(ENOSYS))
	}
	state := corecpu.NewMachineState64(d.Context.Memory)
	state.Set(corecpu.RAX, number)
	state.Set(corecpu.RDI, args[0])
	state.Set(corecpu.RSI, args[1])
	state.Set(corecpu.RDX, args[2])
	state.Set(corecpu.R10, args[3])
	state.Set(corecpu.R8, args[4])
	state.Set(corecpu.R9, args[5])
	if _, err := d.Dispatch(state); err != nil {
		return uint64(-int64(ENOSYS))
	}
	return state.Get(corecpu.RAX)
}
