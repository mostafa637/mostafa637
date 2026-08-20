package wasmjit

import (
	"context"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type MemoryLoadHandler func(address uint64) uint64
type MemoryStoreHandler func(address, value uint64)

func installSyscall(ctx context.Context, rt wazero.Runtime, handler SyscallHandler) (api.Module, error) {
	return installHost(ctx, rt, handler, nil, nil)
}

func installHost(ctx context.Context, rt wazero.Runtime, syscall SyscallHandler, load MemoryLoadHandler, store MemoryStoreHandler) (api.Module, error) {
	builder := rt.NewHostModuleBuilder("env")
	builder.NewFunctionBuilder().WithFunc(syscallFunc(syscall)).Export("syscall64")
	builder.NewFunctionBuilder().WithFunc(loadFunc(load)).Export("load64")
	builder.NewFunctionBuilder().WithFunc(storeFunc(store)).Export("store64")
	return builder.Instantiate(ctx)
}

func syscallFunc(handler SyscallHandler) func(context.Context, api.Module, uint64, uint64, uint64, uint64, uint64, uint64, uint64) uint64 {
	return func(_ context.Context, _ api.Module, number, a1, a2, a3, a4, a5, a6 uint64) uint64 {
		if handler == nil {
			return 0
		}
		return handler(number, [6]uint64{a1, a2, a3, a4, a5, a6})
	}
}

func loadFunc(handler MemoryLoadHandler) func(context.Context, api.Module, uint64) uint64 {
	return func(_ context.Context, _ api.Module, address uint64) uint64 {
		if handler == nil {
			return 0
		}
		return handler(address)
	}
}

func storeFunc(handler MemoryStoreHandler) func(context.Context, api.Module, uint64, uint64) {
	return func(_ context.Context, _ api.Module, address, value uint64) {
		if handler != nil {
			handler(address, value)
		}
	}
}
