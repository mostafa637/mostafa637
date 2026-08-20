package wasmjit

import (
	"context"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

func CompileBlock(ctx context.Context, block GuestBlock) (*HostBlock, error) {
	wasm, err := EmitBlock(block)
	if err != nil {
		return nil, err
	}
	return CompileWASM(ctx, wasm, KeyForBlock(block))
}

func CompileWASM(ctx context.Context, wasm []byte, key BlockKey) (*HostBlock, error) {
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	hostModule, err := installHost(ctx, rt, nil, nil, nil)
	if err != nil {
		rt.Close(ctx)
		return nil, err
	}
	host, mod, err := compileOnRuntime(ctx, rt, hostModule, wasm, key)
	if err != nil {
		rt.Close(ctx)
		return nil, err
	}
	host.stop = closeRuntime(rt, mod)
	return host, nil
}

func compileOnRuntime(ctx context.Context, rt wazero.Runtime, hostModule api.Module, wasm []byte, key BlockKey) (*HostBlock, api.Module, error) {
	compiled, err := rt.CompileModule(ctx, wasm)
	if err != nil {
		return nil, nil, err
	}
	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, nil, err
	}
	host := &HostBlock{Code: wasm, Key: key, run: mod.ExportedFunction("run"), memory: mod.ExportedMemory("memory")}
	return host, mod, nil
}

func closeRuntime(rt wazero.Runtime, mod api.Module) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := mod.Close(ctx); err != nil {
			return err
		}
		return rt.Close(ctx)
	}
}
