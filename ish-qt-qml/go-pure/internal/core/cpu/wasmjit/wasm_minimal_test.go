package wasmjit

import (
	"context"
	"fmt"
	"github.com/tetratelabs/wazero"
	"testing"
)

func TestWasmMinimal(t *testing.T) {
	// Minimal valid WASM module: 17 i64 params → 17 i64 results, body = 17 local.get
	minimal := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
		// Type section: 1 type: (i64×17) → (i64×17)
		0x01, 0x26, 0x01, 0x60, 0x11,
		0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e,
		0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e,
		0x11,
		0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e,
		0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e, 0x7e,
		// Function section: 1 func → type 0
		0x03, 0x02, 0x01, 0x00,
		// Memory section: 1 memory, min 1 page
		0x05, 0x03, 0x01, 0x00, 0x01,
		// Export section: run → func 0, memory → mem 0
		0x07, 0x10, 0x02, 0x03, 'r', 'u', 'n', 0x00, 0x00, 0x06, 'm', 'e', 'm', 'o', 'r', 'y', 0x02, 0x00,
		// Code section: 1 func
		0x0a, 0x26, 0x01,
		// func body: size 35, 0 locals, body
		0x24, 0x00,
		0x20, 0x00, 0x20, 0x01, 0x20, 0x02, 0x20, 0x03,
		0x20, 0x04, 0x20, 0x05, 0x20, 0x06, 0x20, 0x07,
		0x20, 0x08, 0x20, 0x09, 0x20, 0x0a, 0x20, 0x0b,
		0x20, 0x0c, 0x20, 0x0d, 0x20, 0x0e, 0x20, 0x0f,
		0x20, 0x10,
		0x0b,
	}

	rt := wazero.NewRuntimeWithConfig(context.Background(), wazero.NewRuntimeConfigCompiler())
	defer rt.Close(context.Background())

	compiled, err := rt.CompileModule(context.Background(), minimal)
	if err != nil {
		t.Fatalf("compile minimal: %v", err)
	}
	fmt.Println("minimal module compiled OK")

	mod, err := rt.InstantiateModule(context.Background(), compiled, wazero.NewModuleConfig())
	if err != nil {
		t.Fatalf("instantiate minimal: %v", err)
	}
	defer mod.Close(context.Background())

	fn := mod.ExportedFunction("run")
	if fn == nil {
		t.Fatal("run not exported")
	}
	fmt.Println("minimal module instantiated OK, run found")

	// Now test with our actual module
	code := []byte{0x48, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0}
	block := GuestBlock{PC: 0, Bytes: code}
	wasmBytes, errBlock := EmitBlock(block)
	if errBlock != nil {
		t.Fatal(errBlock)
	}
	fmt.Printf("actual wasm len: %d\n", len(wasmBytes))

	_, herr := installHost(context.Background(), rt, nil, nil, nil)
	if herr != nil {
		t.Fatalf("installHost: %v", herr)
	}

	compiled2, err2 := rt.CompileModule(context.Background(), wasmBytes)
	if err2 != nil {
		t.Fatalf("compile actual: %v", err2)
	}
	fmt.Println("actual module compiled OK")

	// In wazero, simply instantiating the host module in the same runtime makes it available.
	mod2, err3 := rt.InstantiateModule(context.Background(), compiled2, wazero.NewModuleConfig().WithName("guest"))
	if err3 != nil {
		t.Fatalf("instantiate actual: %v", err3)
	}
	defer mod2.Close(context.Background())
	fmt.Println("actual module instantiated OK")
}
