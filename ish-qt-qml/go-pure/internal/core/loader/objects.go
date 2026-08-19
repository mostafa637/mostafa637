package loader

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"path"

	corecpu "github.com/mostafa637/mostafa637/go-pure/internal/core/cpu"
	coreelf "github.com/mostafa637/mostafa637/go-pure/internal/core/elf"
)

// Object is one mapped ELF object participating in dynamic linking. Name is
// normally the path used to load it; SONAME is registered as an additional
// lookup alias when present.
type Object struct {
	Name   string
	Image  *coreelf.Image
	Space  *AddressSpace
	SONAME string
	Reader io.ReaderAt
	Size   int64
}

// ObjectRegistry stores all mapped objects in load order. The order is also
// the initial global symbol lookup order, matching the main executable first
// rule used by the i386 ELF ABI.
type ObjectRegistry struct {
	objects []*Object
	byName  map[string]*Object
	symbols map[string]uint32
	memory  *corecpu.Memory
}

// LoadedObject is an API-compatible alias for callers using the loader name.
type LoadedObject = Object

// NewObjectRegistry accepts optional guest memory for eager symbol indexing.
func NewObjectRegistry(memory ...*corecpu.Memory) *ObjectRegistry {
	registry := &ObjectRegistry{byName: make(map[string]*Object), symbols: make(map[string]uint32)}
	if len(memory) > 0 {
		registry.memory = memory[0]
	}
	return registry
}

func (r *ObjectRegistry) Register(name string, space *AddressSpace) error {
	_, err := r.Add(name, space)
	return err
}

// AddWithReader registers an object and retains the ReaderAt needed for later
// PT_TLS template allocation. The reader is not used for mapped code bytes.
func (r *ObjectRegistry) AddWithReader(name string, space *AddressSpace, reader io.ReaderAt, size int64) (*Object, error) {
	object, err := r.Add(name, space)
	if err != nil {
		return nil, err
	}
	object.Reader = reader
	object.Size = size
	return object, nil
}

// Resolve returns the biased address of a globally defined symbol.
func (r *ObjectRegistry) Resolve(name string) (uint32, bool) {
	if r == nil {
		return 0, false
	}
	value, ok := r.symbols[name]
	return value, ok
}

func (r *ObjectRegistry) Add(name string, space *AddressSpace) (*Object, error) {
	if r == nil || space == nil || space.Image == nil {
		return nil, fmt.Errorf("loader: invalid ELF object")
	}
	if name == "" {
		name = fmt.Sprintf("<object-%d>", len(r.objects))
	}
	if existing := r.Lookup(name); existing != nil {
		return existing, nil
	}
	object := &Object{Name: name, Image: space.Image, Space: space}
	if space.Image.Dynamic != nil {
		object.SONAME = space.Image.Dynamic.SONAME
	}
	r.objects = append(r.objects, object)
	r.byName[name] = object
	if baseName := path.Base(name); baseName != "." && baseName != "/" {
		if _, exists := r.byName[baseName]; !exists {
			r.byName[baseName] = object
		}
	}
	if object.SONAME != "" {
		r.byName[object.SONAME] = object
	}
	if r.memory != nil {
		if err := r.indexSymbols(object); err != nil {
			return nil, err
		}
	}
	return object, nil
}

func (r *ObjectRegistry) Lookup(name string) *Object {
	if r == nil || name == "" {
		return nil
	}
	if object := r.byName[name]; object != nil {
		return object
	}
	return r.byName[path.Base(name)]
}

func (r *ObjectRegistry) Objects() []*Object {
	if r == nil {
		return nil
	}
	objects := make([]*Object, len(r.objects))
	copy(objects, r.objects)
	return objects
}

func (r *ObjectRegistry) Has(name string) bool { return r.Lookup(name) != nil }

func (r *ObjectRegistry) indexSymbols(object *Object) error {
	if object == nil || object.Image == nil || object.Image.Dynamic == nil {
		return nil
	}
	count := symbolCount(r.memory, object.Space)
	for index := uint32(0); index < count; index++ {
		name, value, section, err := readSymbol(r.memory, object.Space, index)
		if err != nil {
			return err
		}
		if name == "" || section == shnUndef {
			continue
		}
		if section != shnAbs {
			value = add32Modulo(object.Space.Bias, value)
		}
		if _, exists := r.symbols[name]; !exists {
			r.symbols[name] = value
		}
	}
	return nil
}

// ApplyRelocationsWithRegistry applies relocations for one mapped object and
// resolves undefined symbols against the complete global object set.
func ApplyRelocationsWithRegistry(memory *corecpu.Memory, space *AddressSpace, registry *ObjectRegistry) error {
	if memory == nil || space == nil {
		return nil
	}
	object := &Object{Image: space.Image, Space: space}
	if registry != nil {
		for _, candidate := range registry.objects {
			if candidate != nil && candidate.Space == space {
				object = candidate
				break
			}
		}
	}
	return applyRelocationsForObject(memory, registry, object)
}

// ApplyAllRelocations applies relocations for every object in load order.
func ApplyAllRelocations(memory *corecpu.Memory, registry *ObjectRegistry) error {
	if memory == nil || registry == nil {
		return nil
	}
	for _, object := range registry.objects {
		if err := ApplyRelocationsWithRegistry(memory, object.Space, registry); err != nil {
			return fmt.Errorf("loader: relocate %s: %w", object.Name, err)
		}
	}
	return nil
}

func applyRelocationsForObject(memory *corecpu.Memory, registry *ObjectRegistry, object *Object) error {
	if object == nil || object.Space == nil || object.Image == nil || object.Image.Dynamic == nil {
		return nil
	}
	info := object.Image.Dynamic
	if err := applyRelocationTableWithRegistry(memory, object.Space, registry, info.Rel, info.RelSz, info.RelEnt); err != nil {
		return err
	}
	if info.JmpRel == 0 || info.PltRelSz == 0 {
		return nil
	}
	switch info.PltRel {
	case 0, dtRel:
		return applyRelocationTableWithRegistry(memory, object.Space, registry, info.JmpRel, info.PltRelSz, info.RelEnt)
	case dtRela:
		return fmt.Errorf("i386 DT_RELA PLT relocations are unsupported")
	default:
		return fmt.Errorf("unsupported DT_PLTREL value %d", info.PltRel)
	}
}

func applyRelocationTableWithRegistry(memory *corecpu.Memory, space *AddressSpace, registry *ObjectRegistry, table, size, entrySize uint32) error {
	if table == 0 || size == 0 {
		return nil
	}
	if entrySize == 0 {
		entrySize = 8
	}
	if entrySize != 8 || size%entrySize != 0 {
		return fmt.Errorf("unsupported i386 REL table entry size %d", entrySize)
	}
	for index := uint32(0); index < size/entrySize; index++ {
		recordAddress, err := imageAddress(space.Bias, table+index*entrySize)
		if err != nil {
			return fmt.Errorf("REL[%d] address: %w", index, err)
		}
		var record [8]byte
		if err := memory.Read(recordAddress, record[:]); err != nil {
			return fmt.Errorf("read REL[%d] at %#x: %w", index, recordAddress, err)
		}
		offset := binary.LittleEndian.Uint32(record[:4])
		relocationInfo := binary.LittleEndian.Uint32(record[4:])
		target, err := imageAddress(space.Bias, offset)
		if err != nil {
			return fmt.Errorf("REL[%d] target: %w", index, err)
		}
		if err := applyOneRelocationWithRegistry(memory, space, registry, relocationInfo&0xff, relocationInfo>>8, target, index); err != nil {
			return err
		}
	}
	return nil
}

func applyOneRelocationWithRegistry(memory *corecpu.Memory, space *AddressSpace, registry *ObjectRegistry, relocationType, symbolIndex uint32, target corecpu.Address, index uint32) error {
	if relocationType == R386None {
		return nil
	}
	if relocationType == R386Relative {
		if symbolIndex != 0 {
			return fmt.Errorf("R_386_RELATIVE[%d] has symbol index %d", index, symbolIndex)
		}
		var addend [4]byte
		if err := memory.Read(target, addend[:]); err != nil {
			return fmt.Errorf("read RELATIVE addend at %#x: %w", target, err)
		}
		return writeRelocationValue(memory, target, space.Bias+binary.LittleEndian.Uint32(addend[:]))
	}
	symbol, err := resolveSymbolFromRegistry(memory, registry, space, symbolIndex)
	if err != nil {
		return fmt.Errorf("resolve symbol %d for relocation[%d]: %w", symbolIndex, index, err)
	}
	var raw [4]byte
	if err := memory.Read(target, raw[:]); err != nil {
		return fmt.Errorf("read relocation addend at %#x: %w", target, err)
	}
	addend := binary.LittleEndian.Uint32(raw[:])
	var value uint32
	switch relocationType {
	case R38632:
		value = symbol + addend
	case R386PC32:
		value = symbol + addend - uint32(target)
	case R386GlobDat, R386JMPSlot:
		value = symbol
	default:
		return fmt.Errorf("unsupported i386 relocation type %d at %#x", relocationType, target)
	}
	return writeRelocationValue(memory, target, value)
}

func resolveSymbolFromRegistry(memory *corecpu.Memory, registry *ObjectRegistry, requester *AddressSpace, index uint32) (uint32, error) {
	name, value, section, err := readSymbol(memory, requester, index)
	if err != nil {
		return 0, err
	}
	if section != shnUndef {
		if section == shnAbs {
			return value, nil
		}
		return add32Modulo(requester.Bias, value), nil
	}
	if registry == nil {
		return 0, fmt.Errorf("undefined symbol %q", name)
	}
	for _, object := range registry.objects {
		resolved, ok, resolveErr := findDefinedSymbol(memory, object, name)
		if resolveErr != nil {
			return 0, resolveErr
		}
		if ok {
			return resolved, nil
		}
	}
	return 0, fmt.Errorf("undefined symbol %q", name)
}

func readSymbol(memory *corecpu.Memory, space *AddressSpace, index uint32) (name string, value uint32, section uint16, err error) {
	if space == nil || space.Image == nil || space.Image.Dynamic == nil {
		return "", 0, 0, fmt.Errorf("symbol table is missing")
	}
	info := space.Image.Dynamic
	entrySize := info.SymEnt
	if entrySize == 0 {
		entrySize = 16
	}
	if entrySize != 16 {
		return "", 0, 0, fmt.Errorf("unsupported symbol entry size %d", entrySize)
	}
	if info.SymSz != 0 && index >= info.SymSz/entrySize {
		return "", 0, 0, fmt.Errorf("symbol index %d is outside table", index)
	}
	entryOffset := uint64(info.SymTab) + uint64(index)*uint64(entrySize)
	if entryOffset > math.MaxUint32 {
		return "", 0, 0, fmt.Errorf("symbol address overflows 32-bit space")
	}
	address, err := imageAddress(space.Bias, uint32(entryOffset))
	if err != nil {
		return "", 0, 0, err
	}
	var raw [16]byte
	if err := memory.Read(address, raw[:]); err != nil {
		return "", 0, 0, fmt.Errorf("read symbol at %#x: %w", address, err)
	}
	name = symbolName(memory, space, binary.LittleEndian.Uint32(raw[:4]))
	return name, binary.LittleEndian.Uint32(raw[4:8]), binary.LittleEndian.Uint16(raw[14:16]), nil
}

func findDefinedSymbol(memory *corecpu.Memory, object *Object, name string) (uint32, bool, error) {
	if object == nil || object.Space == nil || object.Image == nil || object.Image.Dynamic == nil {
		return 0, false, nil
	}
	count := symbolCount(memory, object.Space)
	if count == 0 {
		return 0, false, nil
	}
	for index := uint32(0); index < count; index++ {
		symbolNameValue, value, section, err := readSymbol(memory, object.Space, index)
		if err != nil {
			return 0, false, err
		}
		if symbolNameValue != name || section == shnUndef {
			continue
		}
		if section == shnAbs {
			return value, true, nil
		}
		return add32Modulo(object.Space.Bias, value), true, nil
	}
	return 0, false, nil
}

func symbolCount(memory *corecpu.Memory, space *AddressSpace) uint32 {
	if space == nil || space.Image == nil || space.Image.Dynamic == nil {
		return 0
	}
	info := space.Image.Dynamic
	if info.SymSz != 0 && info.SymEnt != 0 {
		return info.SymSz / info.SymEnt
	}
	if info.Hash != 0 {
		address, err := imageAddress(space.Bias, info.Hash)
		if err == nil {
			var header [8]byte
			if memory.Read(address, header[:]) == nil {
				return binary.LittleEndian.Uint32(header[4:])
			}
		}
	}
	if info.GNUHash != 0 {
		return gnuHashSymbolCount(memory, space, info.GNUHash)
	}
	return 0
}

func gnuHashSymbolCount(memory *corecpu.Memory, space *AddressSpace, offset uint32) uint32 {
	address, err := imageAddress(space.Bias, offset)
	if err != nil {
		return 0
	}
	var header [16]byte
	if memory.Read(address, header[:]) != nil {
		return 0
	}
	nbuckets := binary.LittleEndian.Uint32(header[0:4])
	symoffset := binary.LittleEndian.Uint32(header[4:8])
	bloomSize := binary.LittleEndian.Uint32(header[8:12])
	if nbuckets == 0 || bloomSize > 1<<20 || nbuckets > 1<<20 {
		return 0
	}
	bucketOffset := uint64(offset) + 16 + uint64(bloomSize)*4
	if bucketOffset+uint64(nbuckets)*4 > math.MaxUint32 {
		return 0
	}
	var maxBucket uint32
	var bucket [4]byte
	for index := uint32(0); index < nbuckets; index++ {
		bucketAddress, addressErr := imageAddress(space.Bias, uint32(bucketOffset)+index*4)
		if addressErr != nil || memory.Read(bucketAddress, bucket[:]) != nil {
			return 0
		}
		value := binary.LittleEndian.Uint32(bucket[:])
		if value > maxBucket {
			maxBucket = value
		}
	}
	if maxBucket < symoffset {
		return symoffset
	}
	chainOffset := bucketOffset + uint64(nbuckets)*4 + uint64(maxBucket-symoffset)*4
	if chainOffset > math.MaxUint32 {
		return 0
	}
	for count := uint32(0); count < 1<<20; count++ {
		chainAddress, addressErr := imageAddress(space.Bias, uint32(chainOffset)+count*4)
		if addressErr != nil || memory.Read(chainAddress, bucket[:]) != nil {
			return 0
		}
		if binary.LittleEndian.Uint32(bucket[:])&1 != 0 {
			return maxBucket + count + 1
		}
	}
	return 0
}

func add32Modulo(left, right uint32) uint32 { return left + right }
