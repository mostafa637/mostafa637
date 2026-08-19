package loader

import (
	"errors"
	"fmt"
	"io"
	"math"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

var ErrUnalignedBias = errors.New("loader: load bias is not page-aligned")

type AddressSpace struct {
	Image    *coreelf.Image
	Bias     uint32
	Entry    corecpu.Address
	Start    corecpu.Address
	End      corecpu.Address
	Brk      corecpu.Address
	Segments []MappedSegment
}

type MappedSegment struct {
	Source coreelf.Segment
	Start  corecpu.Address
	End    corecpu.Address
	Flags  corecpu.Flags
}

func Load(r io.ReaderAt, size int64, image *coreelf.Image, memory *corecpu.Memory, bias uint32) (*AddressSpace, error) {
	if r == nil || image == nil || memory == nil || size <= 0 {
		return nil, fmt.Errorf("loader: invalid argument")
	}
	if bias&(coreelf.PageSize-1) != 0 {
		return nil, fmt.Errorf("%w: %#x", ErrUnalignedBias, bias)
	}

	loadable := image.LoadSegments()
	if len(loadable) == 0 {
		return nil, fmt.Errorf("loader: image has no PT_LOAD")
	}
	mapped := make([]MappedSegment, 0, len(loadable))
	var start, end uint32
	for index, segment := range loadable {
		segmentStart, segmentEnd, err := rebasedRange(segment, bias)
		if err != nil {
			return nil, fmt.Errorf("loader: segment %d: %w", index, err)
		}
		if index == 0 || segmentStart < start {
			start = segmentStart &^ (coreelf.PageSize - 1)
		}
		segmentEnd, err = alignUpChecked(segmentEnd)
		if err != nil {
			return nil, fmt.Errorf("loader: segment %d: %w", index, err)
		}
		if index == 0 || segmentEnd > end {
			end = segmentEnd
		}
		mapped = append(mapped, MappedSegment{
			Source: segment,
			Start:  corecpu.Address(segmentStart),
			End:    corecpu.Address(segmentEnd),
			Flags:  memoryFlags(uint32(segment.Flags)),
		})
	}
	if end <= start {
		return nil, fmt.Errorf("loader: empty address range")
	}

	for pageAddress := start; pageAddress < end; pageAddress += coreelf.PageSize {
		pageData := make([]byte, corecpu.PageSize)
		flags := corecpu.Flags(0)
		for _, segment := range mapped {
			segmentStart := uint32(segment.Start)
			segmentEnd := uint32(segment.End)
			pageEnd := pageAddress + coreelf.PageSize
			if pageEnd <= segmentStart || pageAddress >= segmentEnd {
				continue
			}
			flags |= segment.Flags
			if err := copyFileData(r, size, pageAddress, pageData, segment); err != nil {
				return nil, err
			}
		}
		if flags == 0 {
			continue
		}
		if err := memory.MapBytes(corecpu.Page(pageAddress>>corecpu.PageBits), 1, pageData, 0, flags); err != nil {
			return nil, fmt.Errorf("loader: map page %#x: %w", pageAddress, err)
		}
	}

	entry, err := addAddress(image.Header.Entry, bias)
	if err != nil {
		return nil, fmt.Errorf("loader: entry: %w", err)
	}
	return &AddressSpace{
		Image:    image,
		Bias:     bias,
		Entry:    corecpu.Address(entry),
		Start:    corecpu.Address(start),
		End:      corecpu.Address(end),
		Brk:      corecpu.Address(end),
		Segments: mapped,
	}, nil
}

func rebasedRange(segment coreelf.Segment, bias uint32) (uint32, uint32, error) {
	start, err := addAddress(segment.Vaddr, bias)
	if err != nil {
		return 0, 0, err
	}
	end, err := segment.End()
	if err != nil {
		return 0, 0, err
	}
	end, err = addAddress(end, bias)
	if err != nil {
		return 0, 0, err
	}
	if end < start {
		return 0, 0, fmt.Errorf("segment range wraps")
	}
	return start, end, nil
}

func addAddress(address, bias uint32) (uint32, error) {
	if bias > math.MaxUint32-address {
		return 0, fmt.Errorf("address overflow")
	}
	return address + bias, nil
}

func alignUpChecked(address uint32) (uint32, error) {
	if address == 0 {
		return 0, nil
	}
	if address > math.MaxUint32-(coreelf.PageSize-1) {
		return 0, fmt.Errorf("address alignment overflow")
	}
	return (address + coreelf.PageSize - 1) &^ (coreelf.PageSize - 1), nil
}

func memoryFlags(flags uint32) corecpu.Flags {
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

func copyFileData(r io.ReaderAt, size int64, pageAddress uint32, pageData []byte, segment MappedSegment) error {
	source := segment.Source
	segmentStart := uint32(segment.Start)
	fileStart := segmentStart
	fileEnd := fileStart + source.FileSize
	pageEnd := pageAddress + coreelf.PageSize
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
	fileOffset := uint64(source.Offset) + uint64(start-fileStart)
	length := int(end - start)
	if fileOffset+uint64(length) > uint64(size) {
		return fmt.Errorf("loader: segment file range exceeds image")
	}
	reader := io.NewSectionReader(r, int64(fileOffset), int64(length))
	if _, err := io.ReadFull(reader, pageData[int(start-pageAddress):int(end-pageAddress)]); err != nil {
		return fmt.Errorf("loader: read segment at %#x: %w", fileOffset, err)
	}
	return nil
}
