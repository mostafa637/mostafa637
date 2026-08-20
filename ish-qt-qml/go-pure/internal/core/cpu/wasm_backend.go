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
	return NewWasmJITWithSyscallAndMemory(ctx, root, nil, nil)
}

func NewWasmJITWithSyscall(ctx context.Context, root string, handler wasmjit.SyscallHandler) (*WasmJIT, error) {
	return NewWasmJITWithSyscallAndMemory(ctx, root, handler, nil)
}

func NewWasmJITWithMemory(ctx context.Context, root string, memory *Memory64) (*WasmJIT, error) {
	return NewWasmJITWithSyscallAndMemory(ctx, root, nil, memory)
}

func NewWasmJITWithSyscallAndMemory(ctx context.Context, root string, handler wasmjit.SyscallHandler, memory *Memory64) (*WasmJIT, error) {
	load, store := memoryHandlers(memory)
	compiler, err := wasmjit.NewCompilerWithMemory(ctx, root+"/compiled", handler, load, store)
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
