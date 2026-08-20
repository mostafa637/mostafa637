package cpu

import (
	"context"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/wasmjit"
)

type WasmJIT struct {
	compiler *wasmjit.Compiler
	cache    *wasmjit.BlockCache
}

func NewWasmJIT(ctx context.Context, root string) (*WasmJIT, error) {
	return NewWasmJITWithSyscall(ctx, root, nil)
}

func NewWasmJITWithSyscall(ctx context.Context, root string, handler wasmjit.SyscallHandler) (*WasmJIT, error) {
	compiler, err := wasmjit.NewCompilerWithSyscall(ctx, root+"/compiled", handler)
	if err != nil {
		return nil, err
	}
	cache, err := wasmjit.NewBlockCache(root + "/blocks")
	if err != nil {
		compiler.Close(ctx)
		return nil, err
	}
	return &WasmJIT{compiler: compiler, cache: cache}, nil
}

func (j *WasmJIT) Compile(ctx context.Context, pc uint64, bytes []byte) (*wasmjit.HostBlock, error) {
	block := wasmjit.GuestBlock{PC: pc, Bytes: bytes, Arch: "amd64"}
	return j.compiler.CompileCached(ctx, block, j.cache)
}

func (j *WasmJIT) Close(ctx context.Context) error { return j.compiler.Close(ctx) }
