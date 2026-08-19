package loader

import (
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
