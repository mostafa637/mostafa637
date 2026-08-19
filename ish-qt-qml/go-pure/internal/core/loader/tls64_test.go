package loader

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

func tls64Fixture() (*coreelf.Image64, []byte) {
	data := make([]byte, 0x103)
	copy(data[0x100:], []byte{1, 2, 3})
	return &coreelf.Image64{Segments: []coreelf.Segment64{{Type: elf.PT_TLS, Offset: 0x100, FileSize: 3, MemSize: 8, Align: 8}}}, data
}

func TestLoadTLS64(t *testing.T) {
	image, data := tls64Fixture()
	memory := corecpu.NewMemory64()
	block, err := LoadTLS64(bytes.NewReader(data), int64(len(data)), image, memory, 0, 0x8000)
	if err != nil {
		t.Fatal(err)
	}
	if block == nil || block.Start != 0x8000 || block.End != 0x8008 {
		t.Fatalf("block=%+v", block)
	}
	var got [8]byte
	if err := memory.Read(block.Start, got[:]); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got[:], []byte{1, 2, 3, 0, 0, 0, 0, 0}) {
		t.Fatalf("TLS bytes=%v", got)
	}
}

func TestLoadTLSModules64DTV(t *testing.T) {
	image, data := tls64Fixture()
	memory := corecpu.NewMemory64()
	layout, err := LoadTLSModules64(memory, []TLSModuleSpec64{{ID: 1, Name: "main", Reader: bytes.NewReader(data), Size: int64(len(data)), Image: image}}, 0x9000, 0xa000)
	if err != nil {
		t.Fatal(err)
	}
	if layout == nil || layout.ThreadPointer != 0x9000 || layout.DTVStart != 0xa000 || layout.DTVEnd != 0xa010 {
		t.Fatalf("layout=%+v", layout)
	}
	var dtv [16]byte
	if err := memory.Read(layout.DTVStart, dtv[:]); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(dtv[0:8]); got != 1 {
		t.Fatalf("DTV count=%d", got)
	}
	if got := binary.LittleEndian.Uint64(dtv[8:16]); got != uint64(layout.ThreadPointer) {
		t.Fatalf("DTV module pointer=%#x", got)
	}
	state := corecpu.NewMachineState64(memory)
	if err := AttachTLS64(state, layout); err != nil {
		t.Fatal(err)
	}
	if state.FSBase != uint64(layout.ThreadPointer) || state.TLS != uint64(layout.ThreadPointer) {
		t.Fatalf("TLS state fs=%#x tls=%#x", state.FSBase, state.TLS)
	}
}
