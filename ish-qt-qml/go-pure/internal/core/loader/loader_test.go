package loader

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

func loaderELF() []byte {
	const headerSize = 52
	const programSize = 32
	const programOffset = headerSize
	const payloadOffset = 0x1000
	data := make([]byte, payloadOffset+0x20)
	copy(data[0:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4], data[5], data[6], data[7] = 1, 1, 1, 3
	binary.LittleEndian.PutUint16(data[16:], 2)
	binary.LittleEndian.PutUint16(data[18:], 3)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint32(data[24:], 0x08048100)
	binary.LittleEndian.PutUint32(data[28:], programOffset)
	binary.LittleEndian.PutUint16(data[40:], headerSize)
	binary.LittleEndian.PutUint16(data[42:], programSize)
	binary.LittleEndian.PutUint16(data[44:], 1)
	ph := data[programOffset : programOffset+programSize]
	binary.LittleEndian.PutUint32(ph[0:], 1)
	binary.LittleEndian.PutUint32(ph[4:], payloadOffset)
	binary.LittleEndian.PutUint32(ph[8:], 0x08048000)
	binary.LittleEndian.PutUint32(ph[12:], 0x08048000)
	binary.LittleEndian.PutUint32(ph[16:], 0x20)
	binary.LittleEndian.PutUint32(ph[20:], 0x3000)
	binary.LittleEndian.PutUint32(ph[24:], 7)
	binary.LittleEndian.PutUint32(ph[28:], 0x1000)
	copy(data[payloadOffset:], []byte("guest-code"))
	return data
}

func TestLoadMapsSegmentsAndBSS(t *testing.T) {
	data := loaderELF()
	image, err := coreelf.Parse(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	memory := corecpu.NewMemory()
	space, err := Load(bytes.NewReader(data), int64(len(data)), image, memory, 0)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if space.Entry != 0x08048100 || space.Start != 0x08048000 || space.End != 0x0804b000 || space.Brk != space.End {
		t.Fatalf("space = %#v", space)
	}
	mapping, ok := memory.Page(corecpu.Page(0x08048000 >> corecpu.PageBits))
	if !ok || mapping.Flags&(corecpu.PRead|corecpu.PWrite|corecpu.PExec) != (corecpu.PRead|corecpu.PWrite|corecpu.PExec) {
		t.Fatalf("mapping = %#v, ok=%v", mapping, ok)
	}
	code := make([]byte, 10)
	if err := memory.Read(0x08048000, code); err != nil {
		t.Fatalf("Read code: %v", err)
	}
	if string(code) != "guest-code" {
		t.Fatalf("code = %q", code)
	}
	bss := make([]byte, corecpu.PageSize)
	if err := memory.Read(0x0804a000, bss); err != nil {
		t.Fatalf("Read BSS: %v", err)
	}
	for index, value := range bss {
		if value != 0 {
			t.Fatalf("BSS byte %d = %#x", index, value)
		}
	}
}

func TestLoadRejectsUnalignedBias(t *testing.T) {
	data := loaderELF()
	image, err := coreelf.Parse(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Load(bytes.NewReader(data), int64(len(data)), image, corecpu.NewMemory(), 1)
	if !errors.Is(err, ErrUnalignedBias) {
		t.Fatalf("Load error = %v, want ErrUnalignedBias", err)
	}
}

func TestApplyRelativeRelocation(t *testing.T) {
	const base = uint32(0x40000000)
	memory := corecpu.NewMemory()
	if err := memory.MapNothing(corecpu.Page(base>>corecpu.PageBits), 3, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatalf("MapNothing: %v", err)
	}
	const (
		relTable = uint32(0x1000)
		target   = uint32(0x2000)
		addend   = uint32(0x1234)
	)
	var rel [8]byte
	binary.LittleEndian.PutUint32(rel[0:], target)
	binary.LittleEndian.PutUint32(rel[4:], R386Relative)
	if err := memory.Write(corecpu.Address(base+relTable), rel[:]); err != nil {
		t.Fatalf("write relocation: %v", err)
	}
	var initial [4]byte
	binary.LittleEndian.PutUint32(initial[:], addend)
	if err := memory.Write(corecpu.Address(base+target), initial[:]); err != nil {
		t.Fatalf("write addend: %v", err)
	}
	space := &AddressSpace{
		Bias: base,
		Image: &coreelf.Image{Dynamic: &coreelf.DynamicInfo{
			Rel:    relTable,
			RelSz:  8,
			RelEnt: 8,
		}},
	}
	if err := ApplyRelocations(memory, space); err != nil {
		t.Fatalf("ApplyRelocations: %v", err)
	}
	var result [4]byte
	if err := memory.Read(corecpu.Address(base+target), result[:]); err != nil {
		t.Fatalf("read relocation result: %v", err)
	}
	if got, want := binary.LittleEndian.Uint32(result[:]), base+addend; got != want {
		t.Fatalf("relocation result = %#x, want %#x", got, want)
	}
}

func TestApplyRelocationsRejectsSymbolType(t *testing.T) {
	const base = uint32(0x41000000)
	memory := corecpu.NewMemory()
	if err := memory.MapNothing(corecpu.Page(base>>corecpu.PageBits), 2, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatalf("MapNothing: %v", err)
	}
	var rel [8]byte
	binary.LittleEndian.PutUint32(rel[0:], 0x1000)
	binary.LittleEndian.PutUint32(rel[4:], 1) // R_386_32
	if err := memory.Write(corecpu.Address(base+0x1000), rel[:]); err != nil {
		t.Fatalf("write relocation: %v", err)
	}
	space := &AddressSpace{
		Bias: base,
		Image: &coreelf.Image{Dynamic: &coreelf.DynamicInfo{
			Rel:    0x1000,
			RelSz:  8,
			RelEnt: 8,
		}},
	}
	if err := ApplyRelocations(memory, space); err == nil {
		t.Fatal("ApplyRelocations unexpectedly accepted symbol relocation")
	}
}

func TestLoadTLSInitialImage(t *testing.T) {
	data := []byte{1, 2, 3, 4, 0, 0, 0, 0}
	image := &coreelf.Image{Segments: []coreelf.Segment{{
		Type:     7, // PT_TLS
		Offset:   0,
		Vaddr:    0,
		FileSize: 4,
		MemSize:  8,
		Align:    16,
	}}}
	memory := corecpu.NewMemory()
	tls, err := LoadTLS(bytes.NewReader(data), int64(len(data)), image, memory, 0, 0x5000)
	if err != nil {
		t.Fatalf("LoadTLS: %v", err)
	}
	if tls == nil || tls.Start != 0x5000 || tls.End != 0x5008 {
		t.Fatalf("TLS = %#v", tls)
	}
	got := make([]byte, 8)
	if err := memory.Read(tls.Start, got); err != nil {
		t.Fatalf("read TLS: %v", err)
	}
	want := []byte{1, 2, 3, 4, 0, 0, 0, 0}
	if !bytes.Equal(got, want) {
		t.Fatalf("TLS bytes = %#v, want %#v", got, want)
	}
}

func TestApplyLocalSymbolRelocations(t *testing.T) {
	const base = uint32(0x40000000)
	memory := corecpu.NewMemory()
	if err := memory.MapNothing(corecpu.Page(base>>corecpu.PageBits), 4, corecpu.PRead|corecpu.PWrite); err != nil {
		t.Fatalf("MapNothing: %v", err)
	}
	if err := memory.Write(corecpu.Address(base+0x1000), []byte{0, 'f', 'o', 'o', 0}); err != nil {
		t.Fatalf("write string table: %v", err)
	}
	var symbol [16]byte
	binary.LittleEndian.PutUint32(symbol[0:4], 1)
	binary.LittleEndian.PutUint32(symbol[4:8], 0x1800)
	binary.LittleEndian.PutUint32(symbol[8:12], 4)
	binary.LittleEndian.PutUint16(symbol[14:16], 1)
	if err := memory.Write(corecpu.Address(base+0x1100+16), symbol[:]); err != nil {
		t.Fatalf("write symbol: %v", err)
	}
	var relocations [16]byte
	binary.LittleEndian.PutUint32(relocations[0:4], 0x2000)
	binary.LittleEndian.PutUint32(relocations[4:8], (1<<8)|R38632)
	binary.LittleEndian.PutUint32(relocations[8:12], 0x2004)
	binary.LittleEndian.PutUint32(relocations[12:16], (1<<8)|R386PC32)
	if err := memory.Write(corecpu.Address(base+0x1200), relocations[:]); err != nil {
		t.Fatalf("write relocations: %v", err)
	}
	var addends [8]byte
	binary.LittleEndian.PutUint32(addends[0:4], 5)
	if err := memory.Write(corecpu.Address(base+0x2000), addends[:]); err != nil {
		t.Fatalf("write addends: %v", err)
	}
	space := &AddressSpace{
		Bias: base,
		Image: &coreelf.Image{Dynamic: &coreelf.DynamicInfo{
			StrTab: 0x1000,
			StrSz:  5,
			SymTab: 0x1100,
			SymEnt: 16,
			Rel:    0x1200,
			RelSz:  16,
			RelEnt: 8,
		}},
	}
	if err := ApplyRelocations(memory, space); err != nil {
		t.Fatalf("ApplyRelocations: %v", err)
	}
	var result [8]byte
	if err := memory.Read(corecpu.Address(base+0x2000), result[:]); err != nil {
		t.Fatalf("read relocation results: %v", err)
	}
	if got, want := binary.LittleEndian.Uint32(result[0:4]), base+0x1800+5; got != want {
		t.Fatalf("R_386_32 = %#x, want %#x", got, want)
	}
	if got, want := binary.LittleEndian.Uint32(result[4:8]), uint32(0xfffff7fc); got != want {
		t.Fatalf("R_386_PC32 = %#x, want %#x", got, want)
	}
}
