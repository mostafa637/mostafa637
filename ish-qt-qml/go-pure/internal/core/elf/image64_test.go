package elf_test

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"testing"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
	"github.com/mostafa637/mostafa637/go-pure/internal/core/loader"
)

func minimalELF64(t *testing.T) []byte {
	t.Helper()
	const fileOffset = 0x1000
	data := make([]byte, fileOffset+4)
	copy(data[fileOffset:], []byte{0x90, 0x0f, 0x05, 0xc3})
	ident := data[:16]
	copy(ident[:4], []byte{0x7f, 'E', 'L', 'F'})
	ident[4] = byte(elf.ELFCLASS64)
	ident[5] = byte(elf.ELFDATA2LSB)
	ident[6] = byte(elf.EV_CURRENT)
	binary.LittleEndian.PutUint16(data[16:18], uint16(elf.ET_EXEC))
	binary.LittleEndian.PutUint16(data[18:20], uint16(elf.EM_X86_64))
	binary.LittleEndian.PutUint32(data[20:24], uint32(elf.EV_CURRENT))
	binary.LittleEndian.PutUint64(data[24:32], 0x400000)
	binary.LittleEndian.PutUint64(data[32:40], 64)
	binary.LittleEndian.PutUint16(data[52:54], 64)
	binary.LittleEndian.PutUint16(data[54:56], 56)
	binary.LittleEndian.PutUint16(data[56:58], 1)
	// elf.PT_LOAD: R|X, file offset 0x1000, virtual address 0x400000.
	ph := data[64 : 64+56]
	binary.LittleEndian.PutUint32(ph[0:4], uint32(elf.PT_LOAD))
	binary.LittleEndian.PutUint32(ph[4:8], uint32(elf.PF_R|elf.PF_X))
	binary.LittleEndian.PutUint64(ph[8:16], fileOffset)
	binary.LittleEndian.PutUint64(ph[16:24], 0x400000)
	binary.LittleEndian.PutUint64(ph[32:40], 4)
	binary.LittleEndian.PutUint64(ph[40:48], uint64(coreelf.PageSize))
	binary.LittleEndian.PutUint64(ph[48:56], uint64(coreelf.PageSize))
	return data
}

func TestParse64AndLoad64(t *testing.T) {
	data := minimalELF64(t)
	image, err := coreelf.Parse64(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if image.Header.Machine != elf.EM_X86_64 || image.Header.Entry != 0x400000 {
		t.Fatalf("header=%+v", image.Header)
	}
	memory := corecpu.NewMemory64()
	space, err := loader.Load64(bytes.NewReader(data), int64(len(data)), image, memory, 0)
	if err != nil {
		t.Fatal(err)
	}
	if space.Entry != 0x400000 || space.Start != 0x400000 || space.End != 0x401000 {
		t.Fatalf("space=%+v", space)
	}
	var code [4]byte
	if err := memory.Read(space.Entry, code[:]); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(code[:], data[0x1000:0x1004]) {
		t.Fatalf("code=%x", code)
	}
	flags, ok := memory.MappingFlags(corecpu.Page64(0x400000 >> corecpu.Page64Bits))
	if !ok || flags&(corecpu.PRead|corecpu.PExec) != corecpu.PRead|corecpu.PExec || flags&corecpu.PWrite != 0 {
		t.Fatalf("flags=%#x mapped=%v", flags, ok)
	}
}
