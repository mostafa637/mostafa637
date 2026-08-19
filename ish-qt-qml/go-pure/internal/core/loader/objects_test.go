package loader

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

func TestObjectRegistryResolvesCrossObjectSymbol(t *testing.T) {
	const (
		mainBase    = uint32(0x40000000)
		libraryBase = uint32(0x50000000)
	)
	memory := corecpu.NewMemory()
	for _, base := range []uint32{mainBase, libraryBase} {
		if err := memory.MapNothing(corecpu.Page(base>>corecpu.PageBits), 3, corecpu.PRead|corecpu.PWrite); err != nil {
			t.Fatalf("MapNothing(%#x): %v", base, err)
		}
	}

	writeString := func(base uint32) {
		if err := memory.Write(corecpu.Address(base+0x1000), []byte{0, 'f', 'o', 'o', 0}); err != nil {
			t.Fatalf("write string table: %v", err)
		}
	}
	writeSymbol := func(base, value uint32, section uint16) {
		var symbol [16]byte
		binary.LittleEndian.PutUint32(symbol[0:4], 1)
		binary.LittleEndian.PutUint32(symbol[4:8], value)
		binary.LittleEndian.PutUint16(symbol[14:16], section)
		if err := memory.Write(corecpu.Address(base+0x1100+16), symbol[:]); err != nil {
			t.Fatalf("write symbol: %v", err)
		}
	}
	writeString(mainBase)
	writeString(libraryBase)
	// The main image has an undefined foo at symbol index 1.
	writeSymbol(mainBase, 0, shnUndef)
	// The shared object defines foo at image-relative address 0x1800.
	writeSymbol(libraryBase, 0x1800, 1)

	var relocation [8]byte
	binary.LittleEndian.PutUint32(relocation[0:4], 0x2000)
	binary.LittleEndian.PutUint32(relocation[4:8], (1<<8)|R386GlobDat)
	if err := memory.Write(corecpu.Address(mainBase+0x1200), relocation[:]); err != nil {
		t.Fatalf("write relocation: %v", err)
	}
	if err := memory.Write(corecpu.Address(mainBase+0x2000), make([]byte, 4)); err != nil {
		t.Fatalf("write relocation target: %v", err)
	}

	mainSpace := &AddressSpace{Bias: mainBase, Image: &coreelf.Image{Dynamic: &coreelf.DynamicInfo{
		StrTab: 0x1000, StrSz: 5, SymTab: 0x1100, SymSz: 32, SymEnt: 16,
		Rel: 0x1200, RelSz: 8, RelEnt: 8,
	}}}
	librarySpace := &AddressSpace{Bias: libraryBase, Image: &coreelf.Image{Dynamic: &coreelf.DynamicInfo{
		StrTab: 0x1000, StrSz: 5, SymTab: 0x1100, SymSz: 32, SymEnt: 16,
	}}}
	registry := NewObjectRegistry(memory)
	if err := registry.Register("/bin/app", mainSpace); err != nil {
		t.Fatalf("register main: %v", err)
	}
	if err := registry.Register("/lib/libfoo.so", librarySpace); err != nil {
		t.Fatalf("register library: %v", err)
	}
	if got, ok := registry.Resolve("foo"); !ok || got != libraryBase+0x1800 {
		t.Fatalf("Resolve(foo) = %#x, %v; want %#x, true", got, ok, libraryBase+0x1800)
	}
	if err := ApplyRelocationsWithRegistry(memory, mainSpace, registry); err != nil {
		t.Fatalf("ApplyRelocationsWithRegistry: %v", err)
	}
	var result [4]byte
	if err := memory.Read(corecpu.Address(mainBase+0x2000), result[:]); err != nil {
		t.Fatalf("read relocation result: %v", err)
	}
	if got, want := binary.LittleEndian.Uint32(result[:]), libraryBase+0x1800; got != want {
		t.Fatalf("cross-object relocation = %#x, want %#x", got, want)
	}
}
