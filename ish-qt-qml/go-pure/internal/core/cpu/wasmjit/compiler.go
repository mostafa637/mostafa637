package wasmjit

import (
	"context"
	"errors"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"os"
)

type Compiler struct {
	rt    wazero.Runtime
	cache wazero.CompilationCache
	host  api.Module
}

func NewCompiler(ctx context.Context, dir string) (*Compiler, error) {
	return NewCompilerWithMemory(ctx, dir, nil, nil, nil)
}

func NewCompilerWithSyscall(ctx context.Context, dir string, handler SyscallHandler) (*Compiler, error) {
	return NewCompilerWithMemory(ctx, dir, handler, nil, nil)
}

func NewCompilerWithMemory(ctx context.Context, dir string, syscall SyscallHandler, load MemoryLoadHandler, store MemoryStoreHandler) (*Compiler, error) {
	cache, err := wazero.NewCompilationCacheWithDir(dir)
	if err != nil {
		return nil, err
	}
	cfg := wazero.NewRuntimeConfigCompiler().WithCompilationCache(cache)
	rt := wazero.NewRuntimeWithConfig(ctx, cfg)
	host, err := installHost(ctx, rt, syscall, load, store)
	if err != nil {
		rt.Close(ctx)
		cache.Close(ctx)
		return nil, err
	}
	return &Compiler{rt: rt, cache: cache, host: host}, nil
}

func (c *Compiler) Compile(ctx context.Context, block GuestBlock) (*HostBlock, error) {
	insts, err := decodeGuest(block)
	if err != nil {
		return nil, err
	}
	return c.compileWASM(ctx, emitModule(insts), KeyForBlock(block), insts)
}

func (c *Compiler) CompileCached(ctx context.Context, block GuestBlock, cache *BlockCache) (*HostBlock, error) {
	key := KeyForBlock(block)
	insts, err := decodeGuest(block)
	if err != nil {
		return nil, err
	}
	wasm, err := cachedWASM(block, key, cache)
	if err != nil {
		return nil, err
	}
	return c.compileWASM(ctx, wasm, key, insts)
}

func (c *Compiler) compileWASM(ctx context.Context, wasm []byte, key BlockKey, insts []machinecode.Instruction) (*HostBlock, error) {
	host, mod, err := compileOnRuntime(ctx, c.rt, c.host, wasm, key)
	if err != nil {
		return nil, err
	}
	host.stop = func(ctx context.Context) error { return mod.Close(ctx) }
	host.flow, host.hasFlow = lastFlow(insts)
	return host, nil
}

func (c *Compiler) Close(ctx context.Context) error {
	err := c.rt.Close(ctx)
	cacheErr := c.cache.Close(ctx)
	if err != nil {
		return err
	}
	return cacheErr
}

func cachedWASM(block GuestBlock, key BlockKey, cache *BlockCache) ([]byte, error) {
	wasm, err := cache.Load(key)
	if err == nil {
		return wasm, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	wasm, err = EmitBlock(block)
	if err != nil {
		return nil, err
	}
	return wasm, cache.Store(key, wasm)
}
