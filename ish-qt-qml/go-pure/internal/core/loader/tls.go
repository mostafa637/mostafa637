package loader

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

// TLSBlock is the initial i386 TLS image for one guest process. It contains
// the PT_TLS initialization bytes followed by zero-filled TLS BSS.
type TLSBlock struct {
	Start    corecpu.Address
	End      corecpu.Address
	FileSize uint32
	MemSize  uint32
	Align    uint32
}

const defaultTLSBase uint32 = 0xf7fe0000

// LoadTLS maps the main image's PT_TLS template into guest memory. Full
// multi-thread TLS module allocation is a later kernel concern; this provides
// the initial single-thread block needed by a musl-style guest.
func LoadTLS(r io.ReaderAt, size int64, image *coreelf.Image, memory *corecpu.Memory, bias uint32, base uint32) (*TLSBlock, error) {
	if r == nil || image == nil || memory == nil || size <= 0 {
		return nil, fmt.Errorf("loader: invalid TLS argument")
	}
	var segment *coreelf.Segment
	for index := range image.Segments {
		if image.Segments[index].Type == 7 { // PT_TLS; avoids depending on a missing debug/elf alias.
			segment = &image.Segments[index]
			break
		}
	}
	if segment == nil || segment.MemSize == 0 {
		return nil, nil
	}
	if segment.FileSize > segment.MemSize {
		return nil, fmt.Errorf("loader: TLS filesz exceeds memsz")
	}
	if uint64(segment.Offset)+uint64(segment.FileSize) > uint64(size) {
		return nil, fmt.Errorf("loader: TLS file range exceeds image")
	}
	align := segment.Align
	if align == 0 {
		align = 1
	}
	if align&(align-1) != 0 {
		return nil, fmt.Errorf("loader: TLS alignment is not a power of two")
	}
	if base&(align-1) != 0 {
		base = (base + align - 1) &^ (align - 1)
	}
	if base > math.MaxUint32-segment.MemSize {
		return nil, fmt.Errorf("loader: TLS address overflow")
	}
	end := base + segment.MemSize
	pageStart := base &^ (coreelf.PageSize - 1)
	pageEnd := (end + coreelf.PageSize - 1) &^ (coreelf.PageSize - 1)
	if pageEnd < end {
		return nil, fmt.Errorf("loader: TLS page range overflow")
	}
	pages := (pageEnd - pageStart) / coreelf.PageSize
	data := make([]byte, int(pages)*corecpu.PageSize)
	if segment.FileSize > 0 {
		fileData := make([]byte, segment.FileSize)
		if _, err := io.NewSectionReader(r, int64(segment.Offset), int64(segment.FileSize)).ReadAt(fileData, 0); err != nil && err != io.EOF {
			return nil, fmt.Errorf("loader: read TLS template: %w", err)
		}
		copy(data[int(base-pageStart):], fileData)
	}
	if err := memory.MapBytes(corecpu.Page(pageStart>>corecpu.PageBits), corecpu.Pages(pages), data, 0, corecpu.PRead|corecpu.PWrite|corecpu.PAnonymous); err != nil {
		return nil, fmt.Errorf("loader: map TLS at %#x: %w", pageStart, err)
	}
	return &TLSBlock{
		Start:    corecpu.Address(base),
		End:      corecpu.Address(end),
		FileSize: segment.FileSize,
		MemSize:  segment.MemSize,
		Align:    align,
	}, nil
}

func DefaultTLSBase() uint32 { return defaultTLSBase }

// TLSModuleSpec describes one loaded ELF object that contributes a PT_TLS
// template to the guest thread. IDs are the module IDs used by the DTV and
// must be unique and non-zero.
type TLSModuleSpec struct {
	ID     uint32
	Name   string
	Reader io.ReaderAt
	Size   int64
	Image  *coreelf.Image
	Bias   uint32
}

// TLSModule is one allocated module instance in a guest thread.
type TLSModule struct {
	ID    uint32
	Name  string
	Block TLSBlock
}

// TLSLayout is the initial thread-local storage layout. DTV contains a count
// at word zero and one guest address per module ID at subsequent words.
type TLSLayout struct {
	ThreadPointer corecpu.Address
	DTVStart      corecpu.Address
	DTVEnd        corecpu.Address
	Modules       []TLSModule
}

// LoadTLSModules maps all PT_TLS templates, allocates a DTV, and returns the
// thread pointer and module addresses. The layout is intentionally explicit so
// later clone/exit work can copy or release a thread's TLS without depending on
// Go pointers.
func LoadTLSModules(memory *corecpu.Memory, specs []TLSModuleSpec, base, dtvBase uint32) (*TLSLayout, error) {
	if memory == nil {
		return nil, fmt.Errorf("loader: nil TLS memory")
	}
	if len(specs) == 0 {
		return nil, nil
	}
	maxID := uint32(0)
	for _, spec := range specs {
		if spec.ID == 0 || spec.Reader == nil || spec.Image == nil || spec.Size <= 0 {
			return nil, fmt.Errorf("loader: invalid TLS module %d", spec.ID)
		}
		if spec.ID > 4096 {
			return nil, fmt.Errorf("loader: TLS module id %d is too large", spec.ID)
		}
		if spec.ID > maxID {
			maxID = spec.ID
		}
	}
	modules := make([]TLSModule, 0, len(specs))
	cursor := base
	for _, spec := range specs {
		block, err := LoadTLS(spec.Reader, spec.Size, spec.Image, memory, spec.Bias, cursor)
		if err != nil {
			return nil, fmt.Errorf("loader: TLS module %d (%s): %w", spec.ID, spec.Name, err)
		}
		if block == nil {
			return nil, fmt.Errorf("loader: TLS module %d (%s) has no PT_TLS", spec.ID, spec.Name)
		}
		modules = append(modules, TLSModule{ID: spec.ID, Name: spec.Name, Block: *block})
		cursor, err = alignTLSAddress(uint32(block.End), coreelf.PageSize)
		if err != nil {
			return nil, err
		}
	}
	if dtvBase == 0 {
		var err error
		dtvBase, err = alignTLSAddress(cursor, coreelf.PageSize)
		if err != nil {
			return nil, err
		}
	}
	dtvWords := maxID + 1
	if dtvWords > math.MaxUint32/4 {
		return nil, fmt.Errorf("loader: TLS DTV size overflows 32-bit space")
	}
	dtvBytes := dtvWords * 4
	dtvEnd64 := uint64(dtvBase) + uint64(dtvBytes)
	if dtvEnd64 > math.MaxUint32 {
		return nil, fmt.Errorf("loader: TLS DTV address overflow")
	}
	dtvPageStart := dtvBase &^ (coreelf.PageSize - 1)
	dtvPageEnd := (uint32(dtvEnd64) + coreelf.PageSize - 1) &^ (coreelf.PageSize - 1)
	if dtvPageEnd < uint32(dtvEnd64) {
		return nil, fmt.Errorf("loader: TLS DTV page range overflow")
	}
	dtvData := make([]byte, dtvPageEnd-dtvPageStart)
	binary.LittleEndian.PutUint32(dtvData[dtvBase-dtvPageStart:], maxID)
	for _, module := range modules {
		binary.LittleEndian.PutUint32(dtvData[dtvBase-dtvPageStart+module.ID*4:], uint32(module.Block.Start))
	}
	pages := (dtvPageEnd - dtvPageStart) / corecpu.PageSize
	if err := memory.MapBytes(corecpu.Page(dtvPageStart>>corecpu.PageBits), corecpu.Pages(pages), dtvData, 0, corecpu.PRead|corecpu.PWrite|corecpu.PAnonymous); err != nil {
		return nil, fmt.Errorf("loader: map TLS DTV at %#x: %w", dtvBase, err)
	}
	return &TLSLayout{
		ThreadPointer: modules[0].Block.Start,
		DTVStart:      corecpu.Address(dtvBase),
		DTVEnd:        corecpu.Address(dtvEnd64),
		Modules:       modules,
	}, nil
}

func alignTLSAddress(address, alignment uint32) (uint32, error) {
	if alignment == 0 || alignment&(alignment-1) != 0 {
		return 0, fmt.Errorf("loader: invalid TLS alignment")
	}
	if address > math.MaxUint32-(alignment-1) {
		return 0, fmt.Errorf("loader: TLS address alignment overflow")
	}
	return (address + alignment - 1) &^ (alignment - 1), nil
}
