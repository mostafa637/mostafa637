package elf

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

var (
	ErrInvalidImage       = errors.New("elf32: invalid image")
	ErrUnsupportedClass   = errors.New("elf32: unsupported ELF class")
	ErrUnsupportedEndian  = errors.New("elf32: unsupported ELF byte order")
	ErrUnsupportedMachine = errors.New("elf32: unsupported machine")
	ErrUnsupportedType    = errors.New("elf32: unsupported file type")
	ErrInvalidSegment     = errors.New("elf32: invalid load segment")
)

const (
	PageBits = 12
	PageSize = uint32(1 << PageBits)
)

type Header struct {
	Type       elf.Type
	Machine    elf.Machine
	Entry      uint32
	ProgramOff uint32
	ProgramNum uint16
	ProgramEnt uint16
}

type Segment struct {
	Type     elf.ProgType
	Offset   uint32
	Vaddr    uint32
	Paddr    uint32
	FileSize uint32
	MemSize  uint32
	Flags    elf.ProgFlag
	Align    uint32
}

func (s Segment) Loadable() bool   { return s.Type == elf.PT_LOAD }
func (s Segment) Readable() bool   { return s.Flags&elf.PF_R != 0 }
func (s Segment) Writable() bool   { return s.Flags&elf.PF_W != 0 }
func (s Segment) Executable() bool { return s.Flags&elf.PF_X != 0 }

func (s Segment) End() (uint32, error) {
	if s.MemSize > math.MaxUint32-s.Vaddr {
		return 0, ErrInvalidSegment
	}
	return s.Vaddr + s.MemSize, nil
}

func (s Segment) FileEnd() (uint32, error) {
	if s.FileSize > math.MaxUint32-s.Offset {
		return 0, ErrInvalidSegment
	}
	return s.Offset + s.FileSize, nil
}

func (s Segment) PageStart() uint32 { return s.Vaddr &^ (PageSize - 1) }
func (s Segment) PageEnd() (uint32, error) {
	end, err := s.End()
	if err != nil {
		return 0, err
	}
	if end > math.MaxUint32-(PageSize-1) {
		return 0, ErrInvalidSegment
	}
	return (end + PageSize - 1) &^ (PageSize - 1), nil
}

type Image struct {
	Header   Header
	Segments []Segment
	Interp   string
	Dynamic  *DynamicInfo
}

func (i *Image) LoadSegments() []Segment {
	if i == nil {
		return nil
	}
	segments := make([]Segment, 0, len(i.Segments))
	for _, segment := range i.Segments {
		if segment.Loadable() {
			segments = append(segments, segment)
		}
	}
	return segments
}

func (i *Image) LoadRange() (start, end uint32, err error) {
	loadable := i.LoadSegments()
	if len(loadable) == 0 {
		return 0, 0, ErrInvalidImage
	}
	start = loadable[0].PageStart()
	end, err = loadable[0].PageEnd()
	if err != nil {
		return 0, 0, err
	}
	for _, segment := range loadable[1:] {
		if segment.PageStart() < start {
			start = segment.PageStart()
		}
		segmentEnd, segmentErr := segment.PageEnd()
		if segmentErr != nil {
			return 0, 0, segmentErr
		}
		if segmentEnd > end {
			end = segmentEnd
		}
	}
	return start, end, nil
}

func Parse(r io.ReaderAt, size int64) (*Image, error) {
	if r == nil || size <= 0 {
		return nil, ErrInvalidImage
	}
	ident := make([]byte, 16)
	if _, err := io.ReadFull(io.NewSectionReader(r, 0, int64(len(ident))), ident); err != nil {
		return nil, fmt.Errorf("%w: ident: %v", ErrInvalidImage, err)
	}
	if !bytes.Equal(ident[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return nil, ErrInvalidImage
	}
	if ident[4] != byte(elf.ELFCLASS32) {
		return nil, ErrUnsupportedClass
	}
	if ident[5] != byte(elf.ELFDATA2LSB) {
		return nil, ErrUnsupportedEndian
	}
	if ident[6] != byte(elf.EV_CURRENT) {
		return nil, fmt.Errorf("%w: version=%d", ErrInvalidImage, ident[6])
	}
	file, err := elf.NewFile(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	defer file.Close()

	if file.Class != elf.ELFCLASS32 {
		return nil, ErrUnsupportedClass
	}
	if file.Data != elf.ELFDATA2LSB {
		return nil, ErrUnsupportedEndian
	}
	if file.Machine != elf.EM_386 {
		return nil, ErrUnsupportedMachine
	}
	if file.Type != elf.ET_EXEC && file.Type != elf.ET_DYN {
		return nil, ErrUnsupportedType
	}
	if file.Version != elf.EV_CURRENT {
		return nil, fmt.Errorf("%w: version=%d", ErrInvalidImage, file.Version)
	}
	headerBytes := make([]byte, 52)
	if _, err := io.ReadFull(io.NewSectionReader(r, 0, int64(len(headerBytes))), headerBytes); err != nil {
		return nil, fmt.Errorf("%w: header: %v", ErrInvalidImage, err)
	}
	programOff := binary.LittleEndian.Uint32(headerBytes[28:32])
	programEnt := binary.LittleEndian.Uint16(headerBytes[42:44])
	programNum := binary.LittleEndian.Uint16(headerBytes[44:46])
	if int64(programOff) > size || programEnt == 0 {
		return nil, ErrInvalidImage
	}
	if uint64(programNum)*uint64(programEnt) > uint64(size-int64(programOff)) {
		return nil, ErrInvalidImage
	}

	image := &Image{
		Header: Header{
			Type:       file.Type,
			Machine:    file.Machine,
			Entry:      uint32(file.Entry),
			ProgramOff: programOff,
			ProgramNum: programNum,
			ProgramEnt: programEnt,
		},
		Segments: make([]Segment, 0, len(file.Progs)),
	}
	for _, program := range file.Progs {
		if program.Off > math.MaxUint32 || program.Vaddr > math.MaxUint32 ||
			program.Paddr > math.MaxUint32 || program.Filesz > math.MaxUint32 ||
			program.Memsz > math.MaxUint32 || program.Align > math.MaxUint32 {
			return nil, ErrInvalidSegment
		}
		segment := Segment{
			Type:     program.Type,
			Offset:   uint32(program.Off),
			Vaddr:    uint32(program.Vaddr),
			Paddr:    uint32(program.Paddr),
			FileSize: uint32(program.Filesz),
			MemSize:  uint32(program.Memsz),
			Flags:    program.Flags,
			Align:    uint32(program.Align),
		}
		if segment.Loadable() {
			if segment.MemSize < segment.FileSize {
				return nil, fmt.Errorf("%w: memsz < filesz", ErrInvalidSegment)
			}
			if segment.Align != 0 && segment.Align&(segment.Align-1) != 0 {
				return nil, fmt.Errorf("%w: alignment is not power of two", ErrInvalidSegment)
			}
			if segment.Offset%PageSize != segment.Vaddr%PageSize {
				return nil, fmt.Errorf("%w: offset/vaddr page mismatch", ErrInvalidSegment)
			}
			if _, err := segment.End(); err != nil {
				return nil, err
			}
			fileEnd, err := segment.FileEnd()
			if err != nil || int64(fileEnd) > size {
				return nil, ErrInvalidSegment
			}
		}
		image.Segments = append(image.Segments, segment)
		if segment.Type == elf.PT_INTERP && segment.FileSize > 0 {
			reader := io.NewSectionReader(r, int64(segment.Offset), int64(segment.FileSize))
			data, readErr := io.ReadAll(reader)
			if readErr != nil {
				return nil, fmt.Errorf("%w: interpreter: %v", ErrInvalidImage, readErr)
			}
			for len(data) > 0 && data[len(data)-1] == 0 {
				data = data[:len(data)-1]
			}
			if len(data) == 0 {
				return nil, fmt.Errorf("%w: empty interpreter", ErrInvalidImage)
			}
			image.Interp = string(data)
		}
	}
	if dynamic, dynamicErr := parseDynamic(r, size, file.Progs); dynamicErr != nil {
		return nil, fmt.Errorf("%w: dynamic: %v", ErrInvalidImage, dynamicErr)
	} else {
		image.Dynamic = dynamic
	}
	if len(image.LoadSegments()) == 0 {
		return nil, fmt.Errorf("%w: no PT_LOAD", ErrInvalidImage)
	}
	return image, nil
}
