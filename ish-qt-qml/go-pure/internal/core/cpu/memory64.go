package cpu

import (
	"encoding/binary"
	"sync"
)

// Address64 and Page64 are guest addresses, never host pointers.
type Address64 uint64
type Page64 uint64

const (
	Page64Bits Page64 = 12
	Page64Size        = uint64(1) << Page64Bits
)

type page64Entry struct {
	data  []byte
	flags Flags
	gen   uint64
}

// Memory64 is a sparse long-mode guest address space. Each mapped page has a
// generation counter so a block cache can invalidate translated code whenever
// a guest page is unmapped, remapped, or written.
type Memory64 struct {
	mu      sync.RWMutex
	pages   map[Page64]*page64Entry
	gens    map[Page64]uint64
	changes uint64
}

func NewMemory64() *Memory64 {
	return &Memory64{pages: make(map[Page64]*page64Entry), gens: make(map[Page64]uint64)}
}

func (m *Memory64) Changes() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.changes
}

func canonicalAddress64(addr Address64) bool {
	// In 48-bit long mode, bits 63..48 must sign-extend bit 47. The 17-bit
	// prefix is therefore either all zeroes or all ones.
	prefix := uint64(addr) >> 47
	return prefix == 0 || prefix == 0x1ffff
}

func range64Valid(start Address64, length uint64) bool {
	if length == 0 {
		return canonicalAddress64(start)
	}
	end := uint64(start) + length - 1
	if end < uint64(start) {
		return false
	}
	return canonicalAddress64(start) && canonicalAddress64(Address64(end))
}

func page64Range(start Address64, length uint64) (Page64, Page64, bool) {
	if !range64Valid(start, length) {
		return 0, 0, false
	}
	first := Page64(uint64(start) >> Page64Bits)
	if length == 0 {
		return first, first, true
	}
	last := Page64((uint64(start) + length - 1) >> Page64Bits)
	return first, last + 1, true
}

func (m *Memory64) Map(start Address64, length uint64, flags Flags) error {
	if length == 0 {
		return nil
	}
	first, end, ok := page64Range(start, length)
	if !ok {
		return ErrRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for page := first; page < end; page++ {
		m.gens[page]++
		m.pages[page] = &page64Entry{data: make([]byte, int(Page64Size)), flags: flags}
	}
	m.changes++
	return nil
}

func (m *Memory64) MapBytes(start Address64, data []byte, flags Flags) error {
	if len(data) == 0 {
		return nil
	}
	first, end, ok := page64Range(start, uint64(len(data)))
	if !ok {
		return ErrRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for page := first; page < end; page++ {
		pageStart := uint64(page) * Page64Size
		from := uint64(start)
		if pageStart > from {
			from = pageStart
		}
		to := pageStart + Page64Size
		if to > uint64(start)+uint64(len(data)) {
			to = uint64(start) + uint64(len(data))
		}
		entry := &page64Entry{data: make([]byte, int(Page64Size)), flags: flags}
		copy(entry.data[int(from-pageStart):int(to-pageStart)], data[int(from-uint64(start)):int(to-uint64(start))])
		m.gens[page]++
		entry.gen = m.gens[page]
		m.pages[page] = entry
	}
	m.changes++
	return nil
}

func (m *Memory64) Unmap(start Address64, length uint64) error {
	if length == 0 {
		return nil
	}
	first, end, ok := page64Range(start, length)
	if !ok {
		return ErrRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for page := first; page < end; page++ {
		if m.pages[page] == nil {
			return ErrUnmapped
		}
	}
	for page := first; page < end; page++ {
		delete(m.pages, page)
		m.gens[page]++
	}
	m.changes++
	return nil
}

func (m *Memory64) UnmapAlways(start Address64, length uint64) error {
	if length == 0 {
		return nil
	}
	first, end, ok := page64Range(start, length)
	if !ok {
		return ErrRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for page := first; page < end; page++ {
		delete(m.pages, page)
		m.gens[page]++
	}
	m.changes++
	return nil
}

func (m *Memory64) SetFlags(start Address64, length uint64, flags Flags) error {
	first, end, ok := page64Range(start, length)
	if !ok {
		return ErrRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for page := first; page < end; page++ {
		entry := m.pages[page]
		if entry == nil {
			return ErrUnmapped
		}
		entry.flags = flags
		m.gens[page]++
		entry.gen = m.gens[page]
	}
	m.changes++
	return nil
}

func (m *Memory64) PageGeneration(page Page64) (uint64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.pages[page]
	if !ok || entry == nil {
		return m.gens[page], false
	}
	return entry.gen, true
}

func (m *Memory64) MappingFlags(page Page64) (Flags, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.pages[page]
	if !ok || entry == nil {
		return 0, false
	}
	return entry.flags, true
}

func (m *Memory64) readWrite(addr Address64, buf []byte, access Access) error {
	if !range64Valid(addr, uint64(len(buf))) {
		return ErrRange
	}
	for len(buf) > 0 {
		page := Page64(uint64(addr) >> Page64Bits)
		offset := uint64(addr) & (Page64Size - 1)
		chunk := int(Page64Size - offset)
		if chunk > len(buf) {
			chunk = len(buf)
		}
		m.mu.Lock()
		entry := m.pages[page]
		if entry == nil {
			m.mu.Unlock()
			return ErrUnmapped
		}
		if access == Read {
			if entry.flags&PRead == 0 {
				m.mu.Unlock()
				return ErrProtection
			}
			copy(buf[:chunk], entry.data[int(offset):int(offset)+chunk])
		} else {
			if entry.flags&PWrite == 0 {
				m.mu.Unlock()
				return ErrProtection
			}
			copy(entry.data[int(offset):int(offset)+chunk], buf[:chunk])
			m.gens[page]++
			entry.gen = m.gens[page]
			m.changes++
		}
		m.mu.Unlock()
		addr += Address64(chunk)
		buf = buf[chunk:]
	}
	return nil
}

func (m *Memory64) Read(addr Address64, dst []byte) error {
	return m.readWrite(addr, dst, Read)
}

func (m *Memory64) Write(addr Address64, src []byte) error {
	return m.readWrite(addr, src, Write)
}

func (m *Memory64) ReadUint64(addr Address64) (uint64, error) {
	var raw [8]byte
	if err := m.Read(addr, raw[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(raw[:]), nil
}

func (m *Memory64) WriteUint64(addr Address64, value uint64) error {
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], value)
	return m.Write(addr, raw[:])
}

// AtomicRMW executes one guest-width read-modify-write while holding the
// address-space lock. It is the Pure-Go equivalent of the locked memory part
// needed by XADD/XCHG; it never exposes a host pointer to guest code.
func (m *Memory64) AtomicRMW(addr Address64, width uint8, update func(uint64) uint64) (uint64, error) {
	if m == nil || update == nil || (width != 1 && width != 2 && width != 4 && width != 8) {
		return 0, ErrRange
	}
	if !range64Valid(addr, uint64(width)) {
		return 0, ErrRange
	}
	first, end, ok := page64Range(addr, uint64(width))
	if !ok {
		return 0, ErrRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for page := first; page < end; page++ {
		entry := m.pages[page]
		if entry == nil {
			return 0, ErrUnmapped
		}
		if entry.flags&(PRead|PWrite) != (PRead | PWrite) {
			return 0, ErrProtection
		}
	}
	var raw [8]byte
	for i := uint8(0); i < width; i++ {
		guest := uint64(addr) + uint64(i)
		page := Page64(guest >> Page64Bits)
		offset := guest & (Page64Size - 1)
		raw[i] = m.pages[page].data[offset]
	}
	var old uint64
	switch width {
	case 1:
		old = uint64(raw[0])
	case 2:
		old = uint64(raw[0]) | uint64(raw[1])<<8
	case 4:
		old = uint64(raw[0]) | uint64(raw[1])<<8 | uint64(raw[2])<<16 | uint64(raw[3])<<24
	case 8:
		old = binary.LittleEndian.Uint64(raw[:])
	}
	mask := mask64Width(width)
	value := update(old) & mask
	for i := uint8(0); i < width; i++ {
		raw[i] = byte(value >> (8 * i))
	}
	for i := uint8(0); i < width; i++ {
		guest := uint64(addr) + uint64(i)
		page := Page64(guest >> Page64Bits)
		offset := guest & (Page64Size - 1)
		m.pages[page].data[offset] = raw[i]
	}
	for page := first; page < end; page++ {
		m.gens[page]++
		m.pages[page].gen = m.gens[page]
	}
	m.changes++
	return old, nil
}

// AtomicCompareExchange performs a guest-width compare-and-exchange and
// returns the value observed before the operation.
func (m *Memory64) AtomicCompareExchange(addr Address64, width uint8, expected, replacement uint64) (uint64, error) {
	return m.AtomicRMW(addr, width, func(old uint64) uint64 {
		if old == (expected & mask64Width(width)) {
			return replacement
		}
		return old
	})
}

// AtomicCompareExchange128 performs the guest CMPXCHG16B operation on a
// naturally aligned 16-byte memory value. It returns the observed low/high
// halves and whether the compare matched.
func (m *Memory64) AtomicCompareExchange128(addr Address64, expectedLo, expectedHi, replacementLo, replacementHi uint64) (oldLo, oldHi uint64, exchanged bool, err error) {
	if m == nil || uint64(addr)&15 != 0 || !range64Valid(addr, 16) {
		return 0, 0, false, ErrRange
	}
	first, end, ok := page64Range(addr, 16)
	if !ok {
		return 0, 0, false, ErrRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for page := first; page < end; page++ {
		entry := m.pages[page]
		if entry == nil {
			return 0, 0, false, ErrUnmapped
		}
		if entry.flags&(PRead|PWrite) != (PRead | PWrite) {
			return 0, 0, false, ErrProtection
		}
	}
	var raw [16]byte
	for i := uint64(0); i < 16; i++ {
		guest := uint64(addr) + i
		page := Page64(guest >> Page64Bits)
		offset := guest & (Page64Size - 1)
		raw[i] = m.pages[page].data[offset]
	}
	oldLo = binary.LittleEndian.Uint64(raw[0:8])
	oldHi = binary.LittleEndian.Uint64(raw[8:16])
	if oldLo != expectedLo || oldHi != expectedHi {
		return oldLo, oldHi, false, nil
	}
	binary.LittleEndian.PutUint64(raw[0:8], replacementLo)
	binary.LittleEndian.PutUint64(raw[8:16], replacementHi)
	for i := uint64(0); i < 16; i++ {
		guest := uint64(addr) + i
		page := Page64(guest >> Page64Bits)
		offset := guest & (Page64Size - 1)
		m.pages[page].data[offset] = raw[i]
	}
	for page := first; page < end; page++ {
		m.gens[page]++
		m.pages[page].gen = m.gens[page]
	}
	m.changes++
	return oldLo, oldHi, true, nil
}
