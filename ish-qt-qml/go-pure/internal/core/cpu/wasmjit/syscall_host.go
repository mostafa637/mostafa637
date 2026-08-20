package wasmjit

import (
	"context"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func installSyscall(ctx context.Context, rt wazero.Runtime, handler SyscallHandler) (api.Module, error) {
	if handler == nil {
		return nil, nil
	}
	return rt.NewHostModuleBuilder("env").NewFunctionBuilder().WithFunc(syscallFunc(handler)).Export("syscall64").Instantiate(ctx)
}

func syscallFunc(handler SyscallHandler) func(context.Context, api.Module, uint64, uint64, uint64, uint64, uint64, uint64, uint64) uint64 {
	return func(_ context.Context, _ api.Module, number, a1, a2, a3, a4, a5, a6 uint64) uint64 {
		return handler(number, [6]uint64{a1, a2, a3, a4, a5, a6})
	}
}
