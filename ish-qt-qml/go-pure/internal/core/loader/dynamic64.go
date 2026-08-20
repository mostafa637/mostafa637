package loader

import (
	"debug/elf"
	"fmt"
	"io"
	"math"
	"path"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

// OpenImage64 resolves an interpreter or DT_NEEDED name to an ELF reader.
// The caller owns the returned reader; LoadDynamic64 retains it through the
// Object64 registry for TLS template loading.
type OpenImage64 func(name string) (io.ReaderAt, int64, string, error)

type DynamicLoad64 struct {
	Main        *Object64
	Interpreter *Object64
	Registry    *ObjectRegistry64
	Entry       corecpu.Address64
	TLS         *TLSLayout64
}

// LoadDynamic64 maps the main image and its interpreter/dependencies, applies
// supported relocations, and builds the initial TLS module layout. The caller
// supplies an opener because filesystem policy belongs to the session layer.
func LoadDynamic64(r io.ReaderAt, size int64, filename string, bias corecpu.Address64, memory *corecpu.Memory64, open OpenImage64) (*DynamicLoad64, error) {
	if r == nil || size <= 0 || memory == nil {
		return nil, fmt.Errorf("loader64: invalid dynamic load argument")
	}
	mainImage, err := coreelf.Parse64(r, size)
	if err != nil {
		return nil, fmt.Errorf("loader64: parse main image: %w", err)
	}
	if mainImage.Header.Type == elf.ET_DYN && bias == 0 {
		bias = 0x0000000000400000
	}
	mainSpace, err := Load64(r, size, mainImage, memory, bias)
	if err != nil {
		return nil, fmt.Errorf("loader64: load main image: %w", err)
	}
	registry := NewObjectRegistry64()
	if filename == "" {
		filename = "<main>"
	}
	mainObject, err := registry.AddWithReader(filename, mainSpace, r, size)
	if err != nil {
		return nil, err
	}

	var interpreter *Object64
	entry := mainSpace.Entry
	if mainImage.Interp != "" {
		if open == nil {
			return nil, fmt.Errorf("loader64: PT_INTERP %q requires an opener", mainImage.Interp)
		}
		reader, interpreterSize, interpreterName, openErr := open(mainImage.Interp)
		if openErr != nil {
			return nil, fmt.Errorf("loader64: open interpreter %q: %w", mainImage.Interp, openErr)
		}
		if reader == nil || interpreterSize <= 0 {
			return nil, fmt.Errorf("loader64: invalid interpreter %q", mainImage.Interp)
		}
		interpreterImage, parseErr := coreelf.Parse64(reader, interpreterSize)
		if parseErr != nil {
			return nil, fmt.Errorf("loader64: parse interpreter %q: %w", mainImage.Interp, parseErr)
		}
		if interpreterImage.Interp != "" {
			return nil, fmt.Errorf("loader64: nested PT_INTERP is unsupported")
		}
		interpreterBias, _, biasErr := nextBias64(interpreterImage, 0x0000000040000000)
		if biasErr != nil {
			return nil, fmt.Errorf("loader64: interpreter bias: %w", biasErr)
		}
		interpreterSpace, loadErr := Load64(reader, interpreterSize, interpreterImage, memory, interpreterBias)
		if loadErr != nil {
			return nil, fmt.Errorf("loader64: load interpreter %q: %w", mainImage.Interp, loadErr)
		}
		if interpreterName == "" {
			interpreterName = mainImage.Interp
		}
		interpreter, err = registry.AddWithReader(interpreterName, interpreterSpace, reader, interpreterSize)
		if err != nil {
			return nil, err
		}
		entry = interpreterSpace.Entry
	}

	if err := loadNeeded64(registry, memory, open); err != nil {
		return nil, err
	}
	tls, err := loadTLSModules64(memory, registry)
	if err != nil {
		return nil, fmt.Errorf("loader64: load TLS modules: %w", err)
	}
	if err := ApplyAllRelocations64WithTLS(memory, registry, tls); err != nil {
		return nil, err
	}
	return &DynamicLoad64{Main: mainObject, Interpreter: interpreter, Registry: registry, Entry: entry, TLS: tls}, nil
}

func loadNeeded64(registry *ObjectRegistry64, memory *corecpu.Memory64, open OpenImage64) error {
	if registry == nil || open == nil {
		return nil
	}
	objects := registry.Objects()
	next := uint64(0x0000000060000000)
	for index := 0; index < len(objects); index++ {
		object := objects[index]
		if object == nil || object.Image == nil || object.Image.Dynamic == nil {
			continue
		}
		for _, needed := range object.Image.Dynamic.Needed {
			if registry.Has(needed) {
				continue
			}
			reader, size, name, err := open(needed)
			if err != nil {
				return fmt.Errorf("loader64: open DT_NEEDED %q for %s: %w", needed, object.Name, err)
			}
			if reader == nil || size <= 0 {
				return fmt.Errorf("loader64: invalid DT_NEEDED object %q", needed)
			}
			image, err := coreelf.Parse64(reader, size)
			if err != nil {
				return fmt.Errorf("loader64: parse shared object %q: %w", needed, err)
			}
			if image.Interp != "" || image.Header.Type != elf.ET_DYN {
				return fmt.Errorf("loader64: shared object %q must be ET_DYN without PT_INTERP", needed)
			}
			bias, nextBias, err := nextBias64(image, next)
			if err != nil {
				return fmt.Errorf("loader64: choose bias for shared object %q: %w", needed, err)
			}
			space, err := Load64(reader, size, image, memory, bias)
			if err != nil {
				return fmt.Errorf("loader64: load shared object %q: %w", needed, err)
			}
			if name == "" {
				name = needed
			}
			if _, err := registry.AddWithReader(name, space, reader, size); err != nil {
				return fmt.Errorf("loader64: register shared object %q: %w", needed, err)
			}
			next = nextBias
			objects = registry.Objects()
		}
	}
	return nil
}

func nextBias64(image *coreelf.Image64, next uint64) (corecpu.Address64, uint64, error) {
	if image == nil {
		return 0, 0, fmt.Errorf("nil image")
	}
	start, end, err := image.LoadRange()
	if err != nil {
		return 0, 0, err
	}
	if end <= start || start > next {
		return 0, 0, fmt.Errorf("invalid load range %#x-%#x for window %#x", start, end, next)
	}
	bias := (next - start) &^ (uint64(coreelf.PageSize) - 1)
	biasedEnd, err := addAddress64(end, bias)
	if err != nil || !canonicalRange64(bias+start, biasedEnd) {
		return 0, 0, fmt.Errorf("biased load range overflows canonical address space")
	}
	nextEnd, err := alignUp64Checked(biasedEnd)
	if err != nil || nextEnd > math.MaxUint64-uint64(coreelf.PageSize) {
		return 0, 0, fmt.Errorf("shared-object allocation window exhausted")
	}
	return corecpu.Address64(bias), nextEnd + uint64(coreelf.PageSize), nil
}

func loadTLSModules64(memory *corecpu.Memory64, registry *ObjectRegistry64) (*TLSLayout64, error) {
	if memory == nil || registry == nil {
		return nil, nil
	}
	specs := make([]TLSModuleSpec64, 0)
	moduleID := uint64(1)
	for _, object := range registry.Objects() {
		if object == nil || object.Reader == nil || object.Size <= 0 || object.Image == nil || !hasTLS64(object.Image) {
			continue
		}
		specs = append(specs, TLSModuleSpec64{ID: moduleID, Name: object.Name, Reader: object.Reader, Size: object.Size, Image: object.Image, Bias: object.Space.Bias})
		moduleID++
	}
	return LoadTLSModules64(memory, specs, DefaultTLSBase64(), 0)
}

func hasTLS64(image *coreelf.Image64) bool {
	if image == nil {
		return false
	}
	for _, segment := range image.Segments {
		if segment.Type == elf.PT_TLS && segment.MemSize != 0 {
			return true
		}
	}
	return false
}

func ResolveNeededPath64(base, needed string) string {
	if needed == "" {
		return ""
	}
	if path.IsAbs(needed) {
		return path.Clean(needed)
	}
	if base == "" {
		return path.Clean(needed)
	}
	return path.Join(path.Dir(base), needed)
}
