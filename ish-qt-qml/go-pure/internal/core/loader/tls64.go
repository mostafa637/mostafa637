package loader

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

const defaultTLSBase64 corecpu.Address64 = 0x00007fff00000000

type TLSBlock64 struct {
	Start    corecpu.Address64
	End      corecpu.Address64
	FileSize uint64
	MemSize  uint64
	Align    uint64
}

func LoadTLS64(r io.ReaderAt, size int64, image *coreelf.Image64, memory *corecpu.Memory64, bias, base corecpu.Address64) (*TLSBlock64, error) {
	if r == nil || image == nil || memory == nil || size <= 0 {
		return nil, fmt.Errorf("loader64: invalid TLS argument")
	}
	var segment *coreelf.Segment64
	for index := range image.Segments {
		if image.Segments[index].Type == elf.PT_TLS {
			segment = &image.Segments[index]
			break
		}
	}
	if segment == nil || segment.MemSize == 0 {
		return nil, nil
	}
	if segment.FileSize > segment.MemSize {
		return nil, fmt.Errorf("loader64: TLS filesz exceeds memsz")
	}
	if segment.Offset > uint64(size) || segment.FileSize > uint64(size)-segment.Offset {
		return nil, fmt.Errorf("loader64: TLS file range exceeds image")
	}
	start, ok := addNoOverflow64(uint64(base), uint64(bias))
	if !ok {
		return nil, fmt.Errorf("loader64: TLS base overflow")
	}
	align := segment.Align
	if align == 0 {
		align = 1
	}
	if align&(align-1) != 0 {
		return nil, fmt.Errorf("loader64: TLS alignment is not a power of two")
	}
	if start > math.MaxUint64-(align-1) {
		return nil, fmt.Errorf("loader64: TLS alignment overflow")
	}
	start = (start + align - 1) &^ (align - 1)
	end, ok := addNoOverflow64(start, segment.MemSize)
	if !ok || !canonicalRange64(start, end) {
		return nil, fmt.Errorf("loader64: TLS address range is invalid")
	}
	pageStart := start &^ (uint64(coreelf.PageSize) - 1)
	pageEnd, ok := alignUpNoOverflow64(end)
	if !ok || !canonicalRange64(pageStart, pageEnd) {
		return nil, fmt.Errorf("loader64: TLS page range is invalid")
	}
	data := make([]byte, int(pageEnd-pageStart))
	if segment.FileSize > 0 {
		fileData := make([]byte, int(segment.FileSize))
		if _, err := io.NewSectionReader(r, int64(segment.Offset), int64(segment.FileSize)).ReadAt(fileData, 0); err != nil && err != io.EOF {
			return nil, fmt.Errorf("loader64: read TLS template: %w", err)
		}
		copy(data[int(start-pageStart):], fileData)
	}
	if err := memory.MapBytes(corecpu.Address64(pageStart), data, corecpu.PRead|corecpu.PWrite); err != nil {
		return nil, fmt.Errorf("loader64: map TLS at %#x: %w", pageStart, err)
	}
	return &TLSBlock64{Start: corecpu.Address64(start), End: corecpu.Address64(end), FileSize: segment.FileSize, MemSize: segment.MemSize, Align: align}, nil
}

func DefaultTLSBase64() corecpu.Address64 { return defaultTLSBase64 }

func AttachTLS64(state *corecpu.MachineState64, layout *TLSLayout64) error {
	if state == nil || layout == nil || layout.ThreadPointer == 0 {
		return fmt.Errorf("loader64: invalid TLS state")
	}
	state.FSBase = uint64(layout.ThreadPointer)
	state.TLS = uint64(layout.ThreadPointer)
	return nil
}

type TLSModuleSpec64 struct {
	ID     uint64
	Name   string
	Reader io.ReaderAt
	Size   int64
	Image  *coreelf.Image64
	Bias   corecpu.Address64
}

type TLSModule64 struct {
	ID    uint64
	Name  string
	Block TLSBlock64
}

type TLSLayout64 struct {
	ThreadPointer corecpu.Address64
	DTVStart      corecpu.Address64
	DTVEnd        corecpu.Address64
	Modules       []TLSModule64
}

func LoadTLSModules64(memory *corecpu.Memory64, specs []TLSModuleSpec64, base, dtvBase corecpu.Address64) (*TLSLayout64, error) {
	if memory == nil {
		return nil, fmt.Errorf("loader64: nil TLS memory")
	}
	if len(specs) == 0 {
		return nil, nil
	}
	var maxID uint64
	for _, spec := range specs {
		if spec.ID == 0 || spec.Reader == nil || spec.Image == nil || spec.Size <= 0 {
			return nil, fmt.Errorf("loader64: invalid TLS module %d", spec.ID)
		}
		if spec.ID > 4096 {
			return nil, fmt.Errorf("loader64: TLS module id %d is too large", spec.ID)
		}
		if spec.ID > maxID {
			maxID = spec.ID
		}
	}
	modules := make([]TLSModule64, 0, len(specs))
	cursor := base
	for _, spec := range specs {
		block, err := LoadTLS64(spec.Reader, spec.Size, spec.Image, memory, spec.Bias, cursor)
		if err != nil {
			return nil, fmt.Errorf("loader64: TLS module %d (%s): %w", spec.ID, spec.Name, err)
		}
		if block == nil {
			return nil, fmt.Errorf("loader64: TLS module %d (%s) has no PT_TLS", spec.ID, spec.Name)
		}
		modules = append(modules, TLSModule64{ID: spec.ID, Name: spec.Name, Block: *block})
		aligned, err := alignTLSAddress64(uint64(block.End), uint64(coreelf.PageSize))
		if err != nil {
			return nil, err
		}
		cursor = corecpu.Address64(aligned)
	}
	if dtvBase == 0 {
		var err error
		aligned, err := alignTLSAddress64(uint64(cursor), uint64(coreelf.PageSize))
		if err != nil {
			return nil, err
		}
		dtvBase = corecpu.Address64(aligned)
	}
	if maxID > (math.MaxUint64-8)/8 {
		return nil, fmt.Errorf("loader64: TLS DTV size overflows 64-bit space")
	}
	dtvBytes := (maxID + 1) * 8
	dtvBaseValue := uint64(dtvBase)
	dtvEnd, ok := addNoOverflow64(dtvBaseValue, dtvBytes)
	if !ok || !canonicalRange64(dtvBaseValue, dtvEnd) {
		return nil, fmt.Errorf("loader64: TLS DTV address overflow")
	}
	dtvPageStart := dtvBaseValue &^ (uint64(coreelf.PageSize) - 1)
	dtvPageEnd, ok := alignUpNoOverflow64(dtvEnd)
	if !ok {
		return nil, fmt.Errorf("loader64: TLS DTV page overflow")
	}
	dtvData := make([]byte, int(dtvPageEnd-dtvPageStart))
	binary.LittleEndian.PutUint64(dtvData[int(dtvBaseValue-dtvPageStart):], maxID)
	for _, module := range modules {
		binary.LittleEndian.PutUint64(dtvData[int(dtvBaseValue-dtvPageStart+module.ID*8):], uint64(module.Block.Start))
	}
	if err := memory.MapBytes(corecpu.Address64(dtvPageStart), dtvData, corecpu.PRead|corecpu.PWrite); err != nil {
		return nil, fmt.Errorf("loader64: map TLS DTV at %#x: %w", dtvBase, err)
	}
	return &TLSLayout64{ThreadPointer: modules[0].Block.Start, DTVStart: dtvBase, DTVEnd: corecpu.Address64(dtvEnd), Modules: modules}, nil
}

func alignTLSAddress64(address, alignment uint64) (uint64, error) {
	if alignment == 0 || alignment&(alignment-1) != 0 {
		return 0, fmt.Errorf("loader64: invalid TLS alignment")
	}
	if address > math.MaxUint64-(alignment-1) {
		return 0, fmt.Errorf("loader64: TLS address alignment overflow")
	}
	return (address + alignment - 1) &^ (alignment - 1), nil
}

func alignUpNoOverflow64(address uint64) (uint64, bool) {
	page := uint64(coreelf.PageSize)
	if address > math.MaxUint64-(page-1) {
		return 0, false
	}
	return (address + page - 1) &^ (page - 1), true
}
