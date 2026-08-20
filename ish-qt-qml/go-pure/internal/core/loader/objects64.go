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

const (
	shnUndef64 = 0
	shnAbs64   = 0xfff1

	rX8664None         = 0
	rX8664_64          = 1
	rX8664PC32         = 2
	rX8664GOT32        = 3
	rX8664PLT32        = 4
	rX8664Copy         = 5
	rX8664GlobDat      = 6
	rX8664JumpSlot     = 7
	rX8664Relative     = 8
	rX8664GOTPCRel     = 9
	rX8664_32          = 10
	rX8664_32S         = 11
	rX8664DTPMod64     = 16
	rX8664DTPOff64     = 17
	rX8664TPOff64      = 18
	rX8664TLSDESC      = 36
	rX8664GOTPCRelX    = 41
	rX8664REXGOTPCRelX = 42
)

// Object64 is one mapped ELF64 object participating in dynamic linking.
type Object64 struct {
	Name   string
	Image  *coreelf.Image64
	Space  *AddressSpace64
	SONAME string
	Reader io.ReaderAt
	Size   int64
}

type LoadedObject64 = Object64

// ObjectRegistry64 stores mapped objects in load order. The order is the
// global lookup order, with the main executable first.
type ObjectRegistry64 struct {
	objects []*Object64
	byName  map[string]*Object64
}

func NewObjectRegistry64() *ObjectRegistry64 {
	return &ObjectRegistry64{byName: make(map[string]*Object64)}
}

func (r *ObjectRegistry64) Add(name string, space *AddressSpace64) (*Object64, error) {
	return r.AddWithReader(name, space, nil, 0)
}

func (r *ObjectRegistry64) AddWithReader(name string, space *AddressSpace64, reader io.ReaderAt, size int64) (*Object64, error) {
	if r == nil || space == nil || space.Image == nil {
		return nil, fmt.Errorf("loader64: invalid ELF object")
	}
	if r.byName == nil {
		r.byName = make(map[string]*Object64)
	}
	if name == "" {
		name = fmt.Sprintf("<object-%d>", len(r.objects))
	}
	if existing := r.Lookup(name); existing != nil {
		return existing, nil
	}
	object := &Object64{Name: name, Image: space.Image, Space: space, Reader: reader, Size: size}
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
	return object, nil
}

func (r *ObjectRegistry64) Lookup(name string) *Object64 {
	if r == nil || name == "" || r.byName == nil {
		return nil
	}
	if object := r.byName[name]; object != nil {
		return object
	}
	return r.byName[path.Base(name)]
}

func (r *ObjectRegistry64) Has(name string) bool { return r.Lookup(name) != nil }

func (r *ObjectRegistry64) Objects() []*Object64 {
	if r == nil {
		return nil
	}
	objects := make([]*Object64, len(r.objects))
	copy(objects, r.objects)
	return objects
}

func (r *ObjectRegistry64) Resolve(name string) (corecpu.Address64, bool) {
	if r == nil || name == "" {
		return 0, false
	}
	for _, object := range r.objects {
		if value, ok := definedSymbol64(object, name); ok {
			return value, true
		}
	}
	return 0, false
}

func ApplyRelocations64(memory *corecpu.Memory64, object *Object64, registry *ObjectRegistry64) error {
	return applyRelocations64(memory, object, registry, nil)
}

func applyRelocations64(memory *corecpu.Memory64, object *Object64, registry *ObjectRegistry64, tls *TLSLayout64) error {
	if memory == nil || object == nil || object.Space == nil || object.Image == nil || object.Image.Dynamic == nil {
		return nil
	}
	info := object.Image.Dynamic
	if info.RelSz != 0 {
		if err := applyRel64(memory, object, registry, info.Rel, info.RelSz, info.RelEnt, false, tls); err != nil {
			return err
		}
	}
	if info.RelaSz != 0 {
		if err := applyRel64(memory, object, registry, info.Rela, info.RelaSz, info.RelaEnt, true, tls); err != nil {
			return err
		}
	}
	if info.JmpRel != 0 && info.PltRelSz != 0 {
		entrySize := info.RelEnt
		rela := false
		switch info.PltRel {
		case 7:
			rela = true
			entrySize = info.RelaEnt
		case 17:
			rela = false
		default:
			return fmt.Errorf("loader64: unsupported DT_PLTREL value %d", info.PltRel)
		}
		if err := applyRel64(memory, object, registry, info.JmpRel, info.PltRelSz, entrySize, rela, tls); err != nil {
			return fmt.Errorf("loader64: apply PLT relocations: %w", err)
		}
	}
	return nil
}

func ApplyAllRelocations64(memory *corecpu.Memory64, registry *ObjectRegistry64) error {
	return ApplyAllRelocations64WithTLS(memory, registry, nil)
}

func ApplyAllRelocations64WithTLS(memory *corecpu.Memory64, registry *ObjectRegistry64, tls *TLSLayout64) error {
	if memory == nil || registry == nil {
		return nil
	}
	for _, object := range registry.objects {
		if err := applyRelocations64(memory, object, registry, tls); err != nil {
			return fmt.Errorf("loader64: relocate %s: %w", object.Name, err)
		}
	}
	return nil
}

func applyRel64(memory *corecpu.Memory64, object *Object64, registry *ObjectRegistry64, table, size, entrySize uint64, rela bool, tls *TLSLayout64) error {
	if table == 0 || size == 0 {
		return nil
	}
	if entrySize == 0 {
		if rela {
			entrySize = 24
		} else {
			entrySize = 16
		}
	}
	wanted := uint64(16)
	if rela {
		wanted = 24
	}
	if entrySize != wanted || size%entrySize != 0 {
		return fmt.Errorf("loader64: unsupported relocation table entry size %d", entrySize)
	}
	count := size / entrySize
	for index := uint64(0); index < count; index++ {
		recordOffset, ok := addNoOverflow64(table, index*entrySize)
		if !ok {
			return fmt.Errorf("loader64: relocation table address overflows")
		}
		recordAddress, err := addAddress64(uint64(object.Space.Bias), recordOffset)
		if err != nil {
			return fmt.Errorf("loader64: relocation[%d] address: %w", index, err)
		}
		if rela {
			var raw [24]byte
			if err := memory.Read(corecpu.Address64(recordAddress), raw[:]); err != nil {
				return fmt.Errorf("loader64: read RELA[%d] at %#x: %w", index, recordAddress, err)
			}
			offset := binary.LittleEndian.Uint64(raw[:8])
			info := binary.LittleEndian.Uint64(raw[8:16])
			addend := int64(binary.LittleEndian.Uint64(raw[16:]))
			if err := applyOneRel64(memory, object, registry, offset, info, addend, true, index, tls); err != nil {
				return err
			}
			continue
		}
		var raw [16]byte
		if err := memory.Read(corecpu.Address64(recordAddress), raw[:]); err != nil {
			return fmt.Errorf("loader64: read REL[%d] at %#x: %w", index, recordAddress, err)
		}
		offset := binary.LittleEndian.Uint64(raw[:8])
		info := binary.LittleEndian.Uint64(raw[8:16])
		var addendBytes [8]byte
		target, err := addAddress64(uint64(object.Space.Bias), offset)
		if err != nil {
			return fmt.Errorf("loader64: REL[%d] target: %w", index, err)
		}
		if err := memory.Read(corecpu.Address64(target), addendBytes[:]); err != nil {
			return fmt.Errorf("loader64: read REL addend[%d] at %#x: %w", index, target, err)
		}
		if err := applyOneRel64(memory, object, registry, offset, info, int64(binary.LittleEndian.Uint64(addendBytes[:])), false, index, tls); err != nil {
			return err
		}
	}
	return nil
}

func applyOneRel64(memory *corecpu.Memory64, object *Object64, registry *ObjectRegistry64, offset, info uint64, addend int64, explicitAddend bool, index uint64, tls *TLSLayout64) error {
	typeID := uint32(info & 0xffffffff)
	symbolIndex := info >> 32
	targetValue, err := addAddress64(uint64(object.Space.Bias), offset)
	if err != nil {
		return fmt.Errorf("loader64: relocation[%d] target: %w", index, err)
	}
	target := corecpu.Address64(targetValue)
	if typeID == rX8664None {
		return nil
	}
	if typeID == rX8664Relative {
		if symbolIndex != 0 {
			return fmt.Errorf("loader64: R_X86_64_RELATIVE[%d] has symbol index %d", index, symbolIndex)
		}
		value, ok := signedAdd64(uint64(object.Space.Bias), addend)
		if !ok {
			return fmt.Errorf("loader64: R_X86_64_RELATIVE[%d] overflows", index)
		}
		return write64(memory, target, value)
	}
	if typeID == rX8664Copy {
		return fmt.Errorf("loader64: R_X86_64_COPY[%d] is unsupported", index)
	}
	if typeID == rX8664DTPMod64 || typeID == rX8664DTPOff64 || typeID == rX8664TPOff64 || typeID == rX8664TLSDESC {
		if tls == nil {
			return fmt.Errorf("loader64: TLS relocation type %d[%d] requires TLS layout", typeID, index)
		}
		if typeID == rX8664TLSDESC {
			return fmt.Errorf("loader64: TLSDESC relocation[%d] is unsupported", index)
		}
		definingObject, symbol, ok := resolveSymbolObject64(object, registry, symbolIndex)
		if !ok {
			return fmt.Errorf("loader64: undefined TLS symbol index %d for relocation[%d]", symbolIndex, index)
		}
		module, ok := tlsModuleForObject64(tls, definingObject)
		if !ok {
			return fmt.Errorf("loader64: TLS module for %s is unavailable", definingObject.Name)
		}
		switch typeID {
		case rX8664DTPMod64:
			return write64(memory, target, module.ID)
		case rX8664DTPOff64:
			value, valueOK := signedAddUnsigned64(symbol.Value, addend)
			if !valueOK {
				return fmt.Errorf("loader64: DTPOFF64 relocation[%d] overflows", index)
			}
			return write64(memory, target, value)
		case rX8664TPOff64:
			if module.Block.Start < tls.ThreadPointer {
				return fmt.Errorf("loader64: TPOFF64 module precedes thread pointer")
			}
			baseOffset := uint64(module.Block.Start - tls.ThreadPointer)
			value, valueOK := addNoOverflow64(baseOffset, symbol.Value)
			if !valueOK {
				return fmt.Errorf("loader64: TPOFF64 symbol offset[%d] overflows", index)
			}
			value, valueOK = signedAddUnsigned64(value, addend)
			if !valueOK {
				return fmt.Errorf("loader64: TPOFF64 relocation[%d] overflows", index)
			}
			return write64(memory, target, value)
		}
	}
	if !explicitAddend {
		// REL is accepted for compatibility, although x86-64 normally uses RELA.
		if typeID == rX8664Relative && symbolIndex != 0 {
			return fmt.Errorf("loader64: invalid RELATIVE symbol index")
		}
	}
	symbol, ok := resolveRelocationSymbol64(object, registry, symbolIndex)
	if !ok {
		return fmt.Errorf("loader64: undefined symbol index %d for relocation[%d]", symbolIndex, index)
	}
	P := uint64(target)
	S := uint64(symbol)
	A := addend
	var value uint64
	switch typeID {
	case rX8664_64, rX8664GlobDat, rX8664JumpSlot:
		value, ok = signedAddUnsigned64(S, A)
		if !ok {
			return fmt.Errorf("loader64: relocation[%d] arithmetic overflows", index)
		}
		return write64(memory, target, value)
	case rX8664PC32, rX8664PLT32:
		value, ok = signedSubAdd64(S, A, P)
		if !ok || value > math.MaxUint32 {
			return fmt.Errorf("loader64: relocation[%d] does not fit PC32", index)
		}
		return write32(memory, target, uint32(value))
	case rX8664_32:
		value, ok = signedAddUnsigned64(S, A)
		if !ok || value > math.MaxUint32 {
			return fmt.Errorf("loader64: relocation[%d] does not fit 32-bit unsigned", index)
		}
		return write32(memory, target, uint32(value))
	case rX8664_32S:
		value, ok = signedAddUnsigned64(S, A)
		if !ok || value > math.MaxInt32 {
			return fmt.Errorf("loader64: relocation[%d] does not fit 32-bit signed", index)
		}
		return write32(memory, target, uint32(value))
	case rX8664GOT32, rX8664GOTPCRel, rX8664GOTPCRelX, rX8664REXGOTPCRelX:
		if object.Image.Dynamic == nil || object.Image.Dynamic.PltGot == 0 {
			return fmt.Errorf("loader64: relocation[%d] requires DT_PLTGOT", index)
		}
		got, gotErr := addAddress64(uint64(object.Space.Bias), object.Image.Dynamic.PltGot)
		if gotErr != nil {
			return fmt.Errorf("loader64: relocation[%d] GOT overflows: %w", index, gotErr)
		}
		if typeID == rX8664GOT32 {
			value, ok = signedAddUnsigned64(got, A)
			if !ok || value > math.MaxUint32 {
				return fmt.Errorf("loader64: relocation[%d] GOT32 overflow", index)
			}
		} else {
			value, ok = signedSubAdd64(got, A, P)
			if !ok || value > math.MaxUint32 {
				return fmt.Errorf("loader64: relocation[%d] GOTPCREL overflow", index)
			}
		}
		return write32(memory, target, uint32(value))
	default:
		return fmt.Errorf("loader64: unsupported relocation type %d at %#x", typeID, target)
	}
}

func resolveRelocationSymbol64(object *Object64, registry *ObjectRegistry64, index uint64) (corecpu.Address64, bool) {
	if object == nil || object.Image == nil || index >= uint64(len(object.Image.DynamicSymbols)) {
		return 0, false
	}
	symbol := object.Image.DynamicSymbols[index]
	if symbol.Section != shnUndef64 {
		if symbol.Section == shnAbs64 {
			return corecpu.Address64(symbol.Value), true
		}
		value, valueErr := addAddress64(uint64(object.Space.Bias), symbol.Value)
		return corecpu.Address64(value), valueErr == nil
	}
	if symbol.Name == "" || registry == nil {
		return 0, false
	}
	return registry.Resolve(symbol.Name)
}

func definedSymbol64(object *Object64, name string) (corecpu.Address64, bool) {
	if object == nil || object.Image == nil || object.Space == nil {
		return 0, false
	}
	for _, symbol := range object.Image.DynamicSymbols {
		if symbol.Name != name || symbol.Section == shnUndef64 {
			continue
		}
		if symbol.Section == shnAbs64 {
			return corecpu.Address64(symbol.Value), true
		}
		value, valueErr := addAddress64(uint64(object.Space.Bias), symbol.Value)
		if valueErr == nil {
			return corecpu.Address64(value), true
		}
	}
	return 0, false
}

func signedAddUnsigned64(base uint64, addend int64) (uint64, bool) {
	if addend >= 0 {
		return addNoOverflow64(base, uint64(addend))
	}
	amount := uint64(-(addend + 1)) + 1
	if amount > base {
		return 0, false
	}
	return base - amount, true
}

func signedAdd64(base uint64, addend int64) (uint64, bool) {
	return signedAddUnsigned64(base, addend)
}

func signedSubAdd64(left uint64, addend int64, right uint64) (uint64, bool) {
	value, ok := signedAddUnsigned64(left, addend)
	if !ok || value < right {
		return 0, false
	}
	return value - right, true
}

func write64(memory *corecpu.Memory64, address corecpu.Address64, value uint64) error {
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], value)
	if err := memory.Write(address, raw[:]); err != nil {
		return fmt.Errorf("loader64: write relocation at %#x: %w", address, err)
	}
	return nil
}

func write32(memory *corecpu.Memory64, address corecpu.Address64, value uint32) error {
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], value)
	if err := memory.Write(address, raw[:]); err != nil {
		return fmt.Errorf("loader64: write relocation at %#x: %w", address, err)
	}
	return nil
}

func resolveSymbolObject64(object *Object64, registry *ObjectRegistry64, index uint64) (*Object64, coreelf.Symbol64, bool) {
	if object == nil || object.Image == nil || index >= uint64(len(object.Image.DynamicSymbols)) {
		return nil, coreelf.Symbol64{}, false
	}
	symbol := object.Image.DynamicSymbols[index]
	if symbol.Section != shnUndef64 {
		return object, symbol, true
	}
	if symbol.Name == "" || registry == nil {
		return nil, coreelf.Symbol64{}, false
	}
	for _, candidate := range registry.objects {
		if candidate == nil || candidate.Image == nil {
			continue
		}
		for _, candidateSymbol := range candidate.Image.DynamicSymbols {
			if candidateSymbol.Name == symbol.Name && candidateSymbol.Section != shnUndef64 {
				return candidate, candidateSymbol, true
			}
		}
	}
	return nil, coreelf.Symbol64{}, false
}

func tlsModuleForObject64(layout *TLSLayout64, object *Object64) (TLSModule64, bool) {
	if layout == nil || object == nil {
		return TLSModule64{}, false
	}
	for _, module := range layout.Modules {
		if module.Name == object.Name {
			return module, true
		}
	}
	objectBase := path.Base(object.Name)
	for _, module := range layout.Modules {
		if path.Base(module.Name) == objectBase {
			return module, true
		}
	}
	return TLSModule64{}, false
}
