package elf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func testELF(withInterp bool) []byte {
	const headerSize = 52
	const programSize = 32
	const programOffset = headerSize
	const payloadOffset = 0x1000
	const interpOffset = 0x1200

	programCount := 1
	if withInterp {
		programCount = 2
	}
	length := payloadOffset + 0x20
	if withInterp {
		length = interpOffset + len("/lib/ld-musl-i386.so.1") + 1
	}
	data := make([]byte, length)
	copy(data[0:4], []byte{0x7f, 'E', 'L', 'F'})
	data[4] = byte(1)
	data[5] = byte(1)
	data[6] = 1
	data[7] = 3
	binary.LittleEndian.PutUint16(data[16:], 2)
	binary.LittleEndian.PutUint16(data[18:], 3)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint32(data[24:], 0x08048000)
	binary.LittleEndian.PutUint32(data[28:], programOffset)
	binary.LittleEndian.PutUint16(data[40:], headerSize)
	binary.LittleEndian.PutUint16(data[42:], programSize)
	binary.LittleEndian.PutUint16(data[44:], uint16(programCount))

	ph := data[programOffset : programOffset+programSize]
	binary.LittleEndian.PutUint32(ph[0:], 1)
	binary.LittleEndian.PutUint32(ph[4:], payloadOffset)
	binary.LittleEndian.PutUint32(ph[8:], 0x08048000)
	binary.LittleEndian.PutUint32(ph[12:], 0x08048000)
	binary.LittleEndian.PutUint32(ph[16:], 0x20)
	binary.LittleEndian.PutUint32(ph[20:], 0x3000)
	binary.LittleEndian.PutUint32(ph[24:], 5)
	binary.LittleEndian.PutUint32(ph[28:], 0x1000)
	copy(data[payloadOffset:], []byte("guest-code"))

	if withInterp {
		ph = data[programOffset+programSize : programOffset+2*programSize]
		binary.LittleEndian.PutUint32(ph[0:], 3)
		binary.LittleEndian.PutUint32(ph[4:], interpOffset)
		binary.LittleEndian.PutUint32(ph[16:], uint32(len("/lib/ld-musl-i386.so.1")+1))
		binary.LittleEndian.PutUint32(ph[20:], uint32(len("/lib/ld-musl-i386.so.1")+1))
		copy(data[interpOffset:], "/lib/ld-musl-i386.so.1\x00")
	}
	return data
}

func TestParseELF32(t *testing.T) {
	data := testELF(true)
	image, err := Parse(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if image.Header.Entry != 0x08048000 || image.Header.ProgramNum != 2 {
		t.Fatalf("header = %#v", image.Header)
	}
	if image.Interp != "/lib/ld-musl-i386.so.1" {
		t.Fatalf("Interp = %q", image.Interp)
	}
	segments := image.LoadSegments()
	if len(segments) != 1 || !segments[0].Readable() || !segments[0].Executable() || segments[0].Writable() {
		t.Fatalf("load segments = %#v", segments)
	}
	start, end, err := image.LoadRange()
	if err != nil || start != 0x08048000 || end != 0x0804b000 {
		t.Fatalf("LoadRange = %#x..%#x, %v", start, end, err)
	}
}

func TestParseRejectsUnsupportedImages(t *testing.T) {
	tests := []struct {
		name   string
		mutate func([]byte)
		err    error
	}{
		{"class", func(data []byte) { data[4] = 2 }, ErrUnsupportedClass},
		{"endian", func(data []byte) { data[5] = 2 }, ErrUnsupportedEndian},
		{"machine", func(data []byte) { binary.LittleEndian.PutUint16(data[18:], 62) }, ErrUnsupportedMachine},
		{"type", func(data []byte) { binary.LittleEndian.PutUint16(data[16:], 1) }, ErrUnsupportedType},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := testELF(false)
			test.mutate(data)
			_, err := Parse(bytes.NewReader(data), int64(len(data)))
			if !errors.Is(err, test.err) {
				t.Fatalf("error = %v, want %v", err, test.err)
			}
		})
	}
}

func TestParseRejectsInvalidSegment(t *testing.T) {
	data := testELF(false)
	binary.LittleEndian.PutUint32(data[52+20:], 0x10)
	_, err := Parse(bytes.NewReader(data), int64(len(data)))
	if !errors.Is(err, ErrInvalidSegment) {
		t.Fatalf("error = %v, want ErrInvalidSegment", err)
	}
}
