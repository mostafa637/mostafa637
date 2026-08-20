package main

import (
	"context"
	"fmt"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/wasmjit"
	"github.com/tetratelabs/wazero"
)

func main() {
	cases := map[string][]byte{
		"mov": {0x48, 0xc7, 0xc0, 5, 0, 0, 0, 0xc3},
		"add": {0x48, 0xc7, 0xc0, 5, 0, 0, 0, 0x48, 0x83, 0xc0, 3, 0xc3},
		"sub": {0x48, 0xc7, 0xc0, 5, 0, 0, 0, 0x48, 0x83, 0xe8, 3, 0xc3},
		"cmp": {0x48, 0xc7, 0xc0, 5, 0, 0, 0, 0x48, 0x83, 0xf8, 5, 0xc3},
	}
	ctx := context.Background()
	for name, code := range cases {
		wasm, err := wasmjit.EmitBlock(wasmjit.GuestBlock{Arch: "amd64", PC: 0x1000, Bytes: code})
		if err == nil {
			r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
			_, err = r.CompileModule(ctx, wasm)
			r.Close(ctx)
		}
		fmt.Printf("%s: %v\n", name, err)
	}
}
