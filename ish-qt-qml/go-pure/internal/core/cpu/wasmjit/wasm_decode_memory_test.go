package wasmjit

import (
	"testing"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/cpu/machinecode"
)

func TestDecodeBaseIndexScale(t *testing.T) {
	items, err := decodeX86([]byte{0x48, 0x8b, 0x44, 0x8b, 0x08, 0xc3}, 0x5000)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("instructions=%d", len(items))
	}
	item := items[0]
	if item.Op != machinecode.OpLoad64 || item.MemBase != 3 || item.MemIndex != 1 || item.MemScale != 4 || item.Imm != 8 {
		t.Fatalf("memory instruction=%+v", item)
	}
}

func TestDecodeRIPRelative(t *testing.T) {
	items, err := decodeX86([]byte{0x48, 0x8b, 0x05, 0x20, 0, 0, 0, 0xc3}, 0x7000)
	if err != nil {
		t.Fatal(err)
	}
	item := items[0]
	if !item.MemRIP || item.NextPC != 0x7007 || item.Imm != 0x20 {
		t.Fatalf("rip instruction=%+v", item)
	}
}
