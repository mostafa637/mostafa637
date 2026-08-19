package loader

import (
	"fmt"
	"io"
	"math"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

var ErrUnalignedBias64 = fmt.Errorf("loader64: load bias is not page-aligned")

type AddressSpace64 struct {
	Image    *coreelf.Image64
	Bias     corecpu.Address64
	Entry    corecpu.Address64
	Start    corecpu.Address64
	End      corecpu.Address64
	Brk      corecpu.Address64
	Segments []MappedSegment64
}

type MappedSegment64 struct {
	Source coreelf.Segment64
	Start  corecpu.Address64
	End    corecpu.Address64
	Flags  corecpu.Flags
}

func Load64(r io.ReaderAt, size int64, image *coreelf.Image64, memory *corecpu.Memory64, bias corecpu.Address64) (*AddressSpace64, error) {
	if r == nil || image == nil || memory == nil || size <= 0 {
		return nil, fmt.Errorf("loader64: invalid argument")
	}
	if uint64(bias)&(uint64(coreelf.PageSize)-1) != 0 {
		return nil, fmt.Errorf("%w: %#x", ErrUnalignedBias64, bias)
	}
	loadable := image.LoadSegments()
	if len(loadable) == 0 {
		return nil, fmt.Errorf("loader64: image has no PT_LOAD")
	}
	mapped := make([]MappedSegment64, 0, len(loadable))
	var start, end uint64
	for index, segment := range loadable {
		segmentStart, segmentEnd, err := rebasedRange64(segment, uint64(bias))
		if err != nil {
			return nil, fmt.Errorf("loader64: segment %d: %w", index, err)
		}
		segmentEnd, err = alignUp64Checked(segmentEnd)
		if err != nil {
			return nil, fmt.Errorf("loader64: segment %d: %w", index, err)
		}
		if index == 0 || segmentStart < start {
			start = segmentStart &^ (uint64(coreelf.PageSize) - 1)
		}
		if index == 0 || segmentEnd > end {
			end = segmentEnd
		}
		mapped = append(mapped, MappedSegment64{Source: segment, Start: corecpu.Address64(segmentStart), End: corecpu.Address64(segmentEnd), Flags: memoryFlags64(uint32(segment.Flags))})
	}
	if end <= start || !canonicalRange64(start, end) {
		return nil, fmt.Errorf("loader64: empty or non-canonical address range")
	}
	for pageAddress := start; pageAddress < end; pageAddress += uint64(coreelf.PageSize) {
		pageData := make([]byte, corecpu.Page64Size)
		flags := corecpu.Flags(0)
		for _, segment := range mapped {
			segmentStart := uint64(segment.Start)
			segmentEnd := uint64(segment.End)
			pageEnd := pageAddress + uint64(coreelf.PageSize)
			if pageEnd <= segmentStart || pageAddress >= segmentEnd {
				continue
			}
			flags |= segment.Flags
			if err := copyFileData64(r, size, pageAddress, pageData, segment); err != nil {
				return nil, err
			}
		}
		if flags == 0 {
			continue
		}
		if err := memory.MapBytes(corecpu.Address64(pageAddress), pageData, flags); err != nil {
			return nil, fmt.Errorf("loader64: map page %#x: %w", pageAddress, err)
		}
	}
	entry, err := addAddress64(image.Header.Entry, uint64(bias))
	if err != nil {
		return nil, fmt.Errorf("loader64: entry: %w", err)
	}
	return &AddressSpace64{Image: image, Bias: bias, Entry: corecpu.Address64(entry), Start: corecpu.Address64(start), End: corecpu.Address64(end), Brk: corecpu.Address64(end), Segments: mapped}, nil
}

func rebasedRange64(segment coreelf.Segment64, bias uint64) (uint64, uint64, error) {
	start, err := addAddress64(segment.Vaddr, bias)
	if err != nil {
		return 0, 0, err
	}
	end, err := segment.End()
	if err != nil {
		return 0, 0, err
	}
	end, err = addAddress64(end, bias)
	if err != nil {
		return 0, 0, err
	}
	if end < start {
		return 0, 0, fmt.Errorf("segment range wraps")
	}
	return start, end, nil
}

func addAddress64(address, bias uint64) (uint64, error) {
	if bias > math.MaxUint64-address {
		return 0, fmt.Errorf("address overflow")
	}
	return address + bias, nil
}

func alignUp64Checked(address uint64) (uint64, error) {
	if address == 0 {
		return 0, nil
	}
	page := uint64(coreelf.PageSize)
	if address > math.MaxUint64-(page-1) {
		return 0, fmt.Errorf("address alignment overflow")
	}
	return (address + page - 1) &^ (page - 1), nil
}

func canonicalRange64(start, end uint64) bool {
	if end <= start {
		return false
	}
	prefix := (end - 1) >> 47
	return prefix == 0 || prefix == 0x1ffff
}

func memoryFlags64(flags uint32) corecpu.Flags {
	var result corecpu.Flags
	if flags&4 != 0 {
		result |= corecpu.PRead
	}
	if flags&2 != 0 {
		result |= corecpu.PWrite
	}
	if flags&1 != 0 {
		result |= corecpu.PExec
	}
	return result
}

func copyFileData64(r io.ReaderAt, size int64, pageAddress uint64, pageData []byte, segment MappedSegment64) error {
	source := segment.Source
	segmentStart := uint64(segment.Start)
	fileStart := segmentStart
	fileEnd, ok := addNoOverflow64(fileStart, source.FileSize)
	if !ok {
		return fmt.Errorf("loader64: file range overflow")
	}
	pageEnd, ok := addNoOverflow64(pageAddress, uint64(coreelf.PageSize))
	if !ok {
		return fmt.Errorf("loader64: page range overflow")
	}
	start := pageAddress
	if start < fileStart {
		start = fileStart
	}
	end := pageEnd
	if end > fileEnd {
		end = fileEnd
	}
	if start >= end {
		return nil
	}
	fileOffset, ok := addNoOverflow64(source.Offset, start-fileStart)
	if !ok || fileOffset+end-start > uint64(size) {
		return fmt.Errorf("loader64: segment file range exceeds image")
	}
	reader := io.NewSectionReader(r, int64(fileOffset), int64(end-start))
	if _, err := io.ReadFull(reader, pageData[int(start-pageAddress):int(end-pageAddress)]); err != nil {
		return fmt.Errorf("loader64: read segment at %#x: %w", fileOffset, err)
	}
	return nil
}

func addNoOverflow64(left, right uint64) (uint64, bool) {
	result := left + right
	return result, result >= left
}
