package loader

import (
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

func TestApplyRelocations64RelativeAndDefinedSymbol(t *testing.T) {
	memory := corecpu.NewMemory64()
	const base = corecpu.Address64(0x400000)
	if err := memory.Map(base, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	image := &coreelf.Image64{Dynamic: &coreelf.DynamicInfo64{
		Rela: 0x200, RelaSz: 48, RelaEnt: 24,
		SymTab: 0x300, SymSz: 48, SymEnt: 24,
	}}
	image.DynamicSymbols = []coreelf.Symbol64{
		{},
		{Name: "foo", Value: 0x500, Section: 1},
	}
	space := &AddressSpace64{Image: image, Bias: base}
	registry := NewObjectRegistry64()
	object, err := registry.Add("main", space)
	if err != nil {
		t.Fatal(err)
	}
	writeRela64(t, memory, base+0x200, 0x100, 0, 8, 0x20)
	writeRela64(t, memory, base+0x218, 0x108, 1, 6, 3)
	write64Test(t, memory, base+0x100, 0x10)
	if err := ApplyRelocations64(memory, object, registry); err != nil {
		t.Fatal(err)
	}
	if got := read64Test(t, memory, base+0x100); got != uint64(base)+0x20 {
		t.Fatalf("relative relocation = %#x, want %#x", got, uint64(base)+0x20)
	}
	if got := read64Test(t, memory, base+0x108); got != uint64(base)+0x500+3 {
		t.Fatalf("defined symbol relocation = %#x, want %#x", got, uint64(base)+0x500+3)
	}
}

func TestApplyRelocations64ResolvesAcrossObjects(t *testing.T) {
	memory := corecpu.NewMemory64()
	const mainBase = corecpu.Address64(0x400000)
	const libBase = corecpu.Address64(0x500000)
	for _, address := range []corecpu.Address64{mainBase, libBase} {
		if err := memory.Map(address, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
			t.Fatal(err)
		}
	}
	mainImage := &coreelf.Image64{Dynamic: &coreelf.DynamicInfo64{Rela: 0x200, RelaSz: 24, RelaEnt: 24, SymTab: 0x300, SymSz: 48, SymEnt: 24}}
	mainImage.DynamicSymbols = []coreelf.Symbol64{{}, {Name: "shared", Section: 0}}
	libImage := &coreelf.Image64{Dynamic: &coreelf.DynamicInfo64{}}
	libImage.DynamicSymbols = []coreelf.Symbol64{{}, {Name: "shared", Value: 0x80, Section: 1}}
	registry := NewObjectRegistry64()
	mainObject, err := registry.Add("main", &AddressSpace64{Image: mainImage, Bias: mainBase})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Add("libshared.so", &AddressSpace64{Image: libImage, Bias: libBase}); err != nil {
		t.Fatal(err)
	}
	writeRela64(t, memory, mainBase+0x200, 0x100, 1, 6, 4)
	if err := ApplyRelocations64(memory, mainObject, registry); err != nil {
		t.Fatal(err)
	}
	want := uint64(libBase) + 0x80 + 4
	if got := read64Test(t, memory, mainBase+0x100); got != want {
		t.Fatalf("cross-object relocation = %#x, want %#x", got, want)
	}
}

func TestApplyRelocations64PC32Overflow(t *testing.T) {
	memory := corecpu.NewMemory64()
	const base = corecpu.Address64(0x400000)
	if err := memory.Map(base, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	image := &coreelf.Image64{Dynamic: &coreelf.DynamicInfo64{Rela: 0x200, RelaSz: 24, RelaEnt: 24}}
	image.DynamicSymbols = []coreelf.Symbol64{{}, {Name: "far", Value: 0x900000000000, Section: 1}}
	registry := NewObjectRegistry64()
	object, err := registry.Add("main", &AddressSpace64{Image: image, Bias: base})
	if err != nil {
		t.Fatal(err)
	}
	writeRela64(t, memory, base+0x200, 0x100, 1, 2, 0)
	if err := ApplyRelocations64(memory, object, registry); err == nil {
		t.Fatal("expected PC32 overflow")
	}
}

func writeRela64(t *testing.T, memory *corecpu.Memory64, address, offset corecpu.Address64, symbol uint64, relocation uint32, addend int64) {
	t.Helper()
	var raw [24]byte
	binary.LittleEndian.PutUint64(raw[:8], uint64(offset))
	binary.LittleEndian.PutUint64(raw[8:16], symbol<<32|uint64(relocation))
	binary.LittleEndian.PutUint64(raw[16:], uint64(addend))
	if err := memory.Write(address, raw[:]); err != nil {
		t.Fatal(err)
	}
}

func write64Test(t *testing.T, memory *corecpu.Memory64, address corecpu.Address64, value uint64) {
	t.Helper()
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], value)
	if err := memory.Write(address, raw[:]); err != nil {
		t.Fatal(err)
	}
}

func read64Test(t *testing.T, memory *corecpu.Memory64, address corecpu.Address64) uint64 {
	t.Helper()
	var raw [8]byte
	if err := memory.Read(address, raw[:]); err != nil {
		t.Fatal(err)
	}
	return binary.LittleEndian.Uint64(raw[:])
}

func TestApplyRelocations64TLSModels(t *testing.T) {
	memory := corecpu.NewMemory64()
	const base = corecpu.Address64(0x400000)
	if err := memory.Map(base, corecpu.Page64Size, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatal(err)
	}
	image := &coreelf.Image64{Dynamic: &coreelf.DynamicInfo64{Rela: 0x200, RelaSz: 72, RelaEnt: 24}}
	image.DynamicSymbols = []coreelf.Symbol64{{}, {Name: "tlsvar", Value: 0x20, Section: 1}}
	registry := NewObjectRegistry64()
	object, err := registry.Add("main", &AddressSpace64{Image: image, Bias: base})
	if err != nil {
		t.Fatal(err)
	}
	writeRela64(t, memory, base+0x200, 0x100, 1, rX8664DTPMod64, 0)
	writeRela64(t, memory, base+0x218, 0x108, 1, rX8664DTPOff64, 4)
	writeRela64(t, memory, base+0x230, 0x110, 1, rX8664TPOff64, 8)
	tls := &TLSLayout64{
		ThreadPointer: 0x700000,
		Modules:       []TLSModule64{{ID: 1, Name: "main", Block: TLSBlock64{Start: 0x700000, End: 0x700100}}},
	}
	if err := ApplyAllRelocations64WithTLS(memory, registry, tls); err != nil {
		t.Fatal(err)
	}
	if got := read64Test(t, memory, base+0x100); got != 1 {
		t.Fatalf("DTPMOD64=%#x, want 1", got)
	}
	if got := read64Test(t, memory, base+0x108); got != 0x24 {
		t.Fatalf("DTPOFF64=%#x, want %#x", got, uint64(0x24))
	}
	if got := read64Test(t, memory, base+0x110); got != 0x28 {
		t.Fatalf("TPOFF64=%#x, want %#x", got, uint64(0x28))
	}
	_ = object
}
