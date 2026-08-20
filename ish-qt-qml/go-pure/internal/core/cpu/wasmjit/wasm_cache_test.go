package wasmjit

import (
	"context"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
	"testing"
)

func TestCompilerCached(t *testing.T) {
	root := t.TempDir()
	compiler, err := NewCompiler(context.Background(), root+"/compiled")
	if err != nil {
		t.Fatal(err)
	}
	defer compiler.Close(context.Background())
	cache, err := NewBlockCache(root + "/blocks")
	if err != nil {
		t.Fatal(err)
	}
	block := GuestBlock{PC: 11, Arch: "amd64", Bytes: mustBytes(t)}
	host, err := compiler.CompileCached(context.Background(), block, cache)
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close(context.Background())
	var regs [16]uint64
	got, err := host.Run(context.Background(), regs)
	if err != nil || got != 9 {
		t.Fatalf("got=%d err=%v", got, err)
	}
}

func TestBlockCache(t *testing.T) {
	cache, err := NewBlockCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := KeyForBlock(GuestBlock{PC: 7, Arch: "amd64", Bytes: []byte{1, 2}})
	want := []byte{0, 97, 115, 109}
	if err := cache.Store(key, want); err != nil {
		t.Fatal(err)
	}
	got, err := cache.Load(key)
	if err != nil || string(got) != string(want) {
		t.Fatalf("got=%x err=%v", got, err)
	}
}

func mustBytes(t *testing.T) []byte {
	bytes, err := machinecode.EmitX86(testInstructions())
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}
