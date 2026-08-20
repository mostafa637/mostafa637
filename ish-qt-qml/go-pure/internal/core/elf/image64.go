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
	ErrInvalidImage64       = errors.New("elf64: invalid image")
	ErrUnsupportedClass64   = errors.New("elf64: unsupported ELF class")
	ErrUnsupportedEndian64  = errors.New("elf64: unsupported ELF byte order")
	ErrUnsupportedMachine64 = errors.New("elf64: unsupported machine")
	ErrUnsupportedType64    = errors.New("elf64: unsupported file type")
	ErrInvalidSegment64     = errors.New("elf64: invalid load segment")
)

type Header64 struct {
	Type       elf.Type
	Machine    elf.Machine
	Entry      uint64
	ProgramOff uint64
	ProgramNum uint16
	ProgramEnt uint16
}

type Segment64 struct {
	Type     elf.ProgType
	Offset   uint64
	Vaddr    uint64
	Paddr    uint64
	FileSize uint64
	MemSize  uint64
	Flags    elf.ProgFlag
	Align    uint64
}

func (s Segment64) Loadable() bool   { return s.Type == elf.PT_LOAD }
func (s Segment64) Readable() bool   { return s.Flags&elf.PF_R != 0 }
func (s Segment64) Writable() bool   { return s.Flags&elf.PF_W != 0 }
func (s Segment64) Executable() bool { return s.Flags&elf.PF_X != 0 }

func (s Segment64) End() (uint64, error) {
	if s.MemSize > math.MaxUint64-s.Vaddr {
		return 0, ErrInvalidSegment64
	}
	return s.Vaddr + s.MemSize, nil
}

func (s Segment64) FileEnd() (uint64, error) {
	if s.FileSize > math.MaxUint64-s.Offset {
		return 0, ErrInvalidSegment64
	}
	return s.Offset + s.FileSize, nil
}

func (s Segment64) PageStart() uint64 { return s.Vaddr &^ (uint64(PageSize) - 1) }

func (s Segment64) PageEnd() (uint64, error) {
	end, err := s.End()
	if err != nil {
		return 0, err
	}
	if end > math.MaxUint64-(uint64(PageSize)-1) {
		return 0, ErrInvalidSegment64
	}
	return (end + uint64(PageSize) - 1) &^ (uint64(PageSize) - 1), nil
}

type Image64 struct {
	Header   Header64
	Segments []Segment64
	Interp   string
	Dynamic  *DynamicInfo64
	TLS      *Segment64
}

func (i *Image64) LoadSegments() []Segment64 {
	if i == nil {
		return nil
	}
	segments := make([]Segment64, 0, len(i.Segments))
	for _, segment := range i.Segments {
		if segment.Loadable() {
			segments = append(segments, segment)
		}
	}
	return segments
}

func (i *Image64) LoadRange() (start, end uint64, err error) {
	loadable := i.LoadSegments()
	if len(loadable) == 0 {
		return 0, 0, ErrInvalidImage64
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

func Parse64(r io.ReaderAt, size int64) (*Image64, error) {
	if r == nil || size <= 0 {
		return nil, ErrInvalidImage64
	}
	ident := make([]byte, 16)
	if _, err := io.ReadFull(io.NewSectionReader(r, 0, int64(len(ident))), ident); err != nil {
		return nil, fmt.Errorf("%w: ident: %v", ErrInvalidImage64, err)
	}
	if !bytes.Equal(ident[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return nil, ErrInvalidImage64
	}
	if ident[4] != byte(elf.ELFCLASS64) {
		return nil, ErrUnsupportedClass64
	}
	if ident[5] != byte(elf.ELFDATA2LSB) {
		return nil, ErrUnsupportedEndian64
	}
	if ident[6] != byte(elf.EV_CURRENT) {
		return nil, fmt.Errorf("%w: version=%d", ErrInvalidImage64, ident[6])
	}
	file, err := elf.NewFile(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidImage64, err)
	}
	defer file.Close()
	if file.Class != elf.ELFCLASS64 {
		return nil, ErrUnsupportedClass64
	}
	if file.Data != elf.ELFDATA2LSB {
		return nil, ErrUnsupportedEndian64
	}
	if file.Machine != elf.EM_X86_64 {
		return nil, ErrUnsupportedMachine64
	}
	if file.Type != elf.ET_EXEC && file.Type != elf.ET_DYN {
		return nil, ErrUnsupportedType64
	}
	if file.Version != elf.EV_CURRENT {
		return nil, fmt.Errorf("%w: version=%d", ErrInvalidImage64, file.Version)
	}
	if len(file.Progs) == 0 {
		return nil, fmt.Errorf("%w: no program headers", ErrInvalidImage64)
	}
	if size < 64 {
		return nil, ErrInvalidImage64
	}
	headerBytes := make([]byte, 64)
	if _, err := io.ReadFull(io.NewSectionReader(r, 0, int64(len(headerBytes))), headerBytes); err != nil {
		return nil, fmt.Errorf("%w: header: %v", ErrInvalidImage64, err)
	}
	programOff := binary.LittleEndian.Uint64(headerBytes[32:40])
	programEnt := binary.LittleEndian.Uint16(headerBytes[54:56])
	programNum := binary.LittleEndian.Uint16(headerBytes[56:58])
	if programOff > uint64(size) || programEnt == 0 || uint64(programNum)*uint64(programEnt) > uint64(size)-programOff {
		return nil, ErrInvalidImage64
	}

	image := &Image64{Header: Header64{Type: file.Type, Machine: file.Machine, Entry: file.Entry, ProgramOff: programOff, ProgramNum: programNum, ProgramEnt: programEnt}, Segments: make([]Segment64, 0, len(file.Progs))}
	for _, program := range file.Progs {
		segment := Segment64{Type: program.Type, Offset: program.Off, Vaddr: program.Vaddr, Paddr: program.Paddr, FileSize: program.Filesz, MemSize: program.Memsz, Flags: program.Flags, Align: program.Align}
		if segment.Loadable() {
			if segment.MemSize < segment.FileSize {
				return nil, fmt.Errorf("%w: memsz < filesz", ErrInvalidSegment64)
			}
			if segment.Align != 0 && segment.Align&(segment.Align-1) != 0 {
				return nil, fmt.Errorf("%w: alignment is not power of two", ErrInvalidSegment64)
			}
			if segment.Offset%uint64(PageSize) != segment.Vaddr%uint64(PageSize) {
				return nil, fmt.Errorf("%w: offset/vaddr page mismatch", ErrInvalidSegment64)
			}
			if _, err := segment.End(); err != nil {
				return nil, err
			}
			fileEnd, err := segment.FileEnd()
			if err != nil || fileEnd > uint64(size) {
				return nil, ErrInvalidSegment64
			}
		}
		image.Segments = append(image.Segments, segment)
		if segment.Type == elf.PT_TLS && segment.MemSize != 0 {
			tls := segment
			image.TLS = &tls
		}
		if segment.Type == elf.PT_INTERP && segment.FileSize > 0 {
			reader := io.NewSectionReader(r, int64(segment.Offset), int64(segment.FileSize))
			data, readErr := io.ReadAll(reader)
			if readErr != nil {
				return nil, fmt.Errorf("%w: interpreter: %v", ErrInvalidImage64, readErr)
			}
			for len(data) > 0 && data[len(data)-1] == 0 {
				data = data[:len(data)-1]
			}
			if len(data) == 0 {
				return nil, fmt.Errorf("%w: empty interpreter", ErrInvalidImage64)
			}
			image.Interp = string(data)
		}
	}
	if len(image.LoadSegments()) == 0 {
		return nil, fmt.Errorf("%w: no PT_LOAD", ErrInvalidImage64)
	}
	if dynamic, dynamicErr := parseDynamic64(r, size, image.Segments); dynamicErr != nil {
		return nil, fmt.Errorf("%w: dynamic: %v", ErrInvalidImage64, dynamicErr)
	} else {
		image.Dynamic = dynamic
	}
	return image, nil
}
