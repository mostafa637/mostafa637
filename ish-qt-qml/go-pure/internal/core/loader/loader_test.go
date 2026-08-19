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
