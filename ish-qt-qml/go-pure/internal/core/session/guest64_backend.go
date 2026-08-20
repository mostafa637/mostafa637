package session

import (
	"context"
	"os"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coresyscall "github.com/mostafa637/mostafa637/go-pure/internal/core/syscall"
)

type guest64Runner interface {
	RunToInterrupt(context.Context, *corecpu.MachineState64) uint64
	Poke()
	Close(context.Context) error
}

type directGuest64Runner struct{ jit *corecpu.JIT64 }

func newDirectGuest64Runner(memory *corecpu.Memory64, dispatcher *coresyscall.Dispatcher64) guest64Runner {
	jit := corecpu.NewJIT64(memory)
	jit.OnSyscall64 = dispatcher.Dispatch
	return &directGuest64Runner{jit: jit}
}

func (r *directGuest64Runner) RunToInterrupt(_ context.Context, state *corecpu.MachineState64) uint64 {
	return r.jit.RunToInterrupt(state)
}

func (r *directGuest64Runner) Poke()                       { r.jit.Poke() }
func (r *directGuest64Runner) Close(context.Context) error { return nil }

type wasmGuest64Runner struct {
	runner *corecpu.WasmRunner64
	root   string
}

func newGuest64Runner(ctx context.Context, useWasm bool, memory *corecpu.Memory64, dispatcher *coresyscall.Dispatcher64) (guest64Runner, error) {
	if !useWasm {
		return newDirectGuest64Runner(memory, dispatcher), nil
	}
	return newWasmGuest64Runner(ctx, memory, dispatcher)
}

func newWasmGuest64Runner(ctx context.Context, memory *corecpu.Memory64, dispatcher *coresyscall.Dispatcher64) (guest64Runner, error) {
	root, err := os.MkdirTemp("", "ish-go-wasm-")
	if err != nil {
		return nil, err
	}
	handler := dispatcher.WasmHandler()
	jit, err := corecpu.NewWasmJITWithSyscallAndMemory(ctx, root, handler, memory)
	if err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}
	runner := corecpu.NewWasmRunner64(jit, memory, 32)
	runner.SetSyscall(wasmResume(dispatcher.Context))
	return &wasmGuest64Runner{runner: runner, root: root}, nil
}

func wasmResume(ctx *coresyscall.Context64) func(*corecpu.MachineState64) (bool, error) {
	return func(state *corecpu.MachineState64) (bool, error) {
		if ctx != nil && ctx.Exited {
			state.Halted = true
			return false, nil
		}
		return true, nil
	}
}

func (r *wasmGuest64Runner) RunToInterrupt(ctx context.Context, state *corecpu.MachineState64) uint64 {
	return r.runner.RunToInterrupt(ctx, state)
}

func (r *wasmGuest64Runner) Poke() { r.runner.Poke() }

func (r *wasmGuest64Runner) Close(ctx context.Context) error {
	err := r.runner.Close(ctx)
	if removeErr := os.RemoveAll(r.root); err == nil {
		err = removeErr
	}
	return err
}
