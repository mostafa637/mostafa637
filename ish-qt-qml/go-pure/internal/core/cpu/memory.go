package cpu

import (
	"errors"
	"fmt"
	"sync"
)

const (
	PageBits  = 12
	PageSize  = 1 << PageBits
	MemPages  = 1 << 20
	BadPage   = Page(MemPages)
	FirstHole = Page(0x40000)
	LastHole  = Page(0xf7ffd)
)

type Address uint32
type Page uint32
type Pages uint32

type Access uint8

const (
	Read Access = iota
	Write
	WritePtrace
)

type Flags uint32

const (
	PRead Flags = 1 << iota
	PWrite
	PExec
	PGrowDown
	PCOW
	PAnonymous
	PShared
)

var (
	ErrUnmapped   = errors.New("cpu memory: unmapped address")
	ErrProtection = errors.New("cpu memory: protection fault")
	ErrRange      = errors.New("cpu memory: invalid page range")
)

type backing struct {
	data []byte
	refs int
}

type pageEntry struct {
	data   *backing
	offset int
	flags  Flags
}

// Mapping describes one mapped page without exposing backing ownership.
type Mapping struct {
	Page  Page
	Flags Flags
}

// Memory is a sparse 32-bit page map. It mirrors the observable behavior of
// iSH's mem/pt_entry layer while remaining safe for Go tests and Android.
type Memory struct {
	mu      sync.RWMutex
	pages   map[Page]*pageEntry
	changes uint64
}

func NewMemory() *Memory {
	return &Memory{pages: make(map[Page]*pageEntry)}
}

func (m *Memory) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for page := range m.pages {
		m.removePageLocked(page)
	}
	m.pages = make(map[Page]*pageEntry)
	m.changes++
}

func (m *Memory) Changes() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.changes
}

func (m *Memory) Page(page Page) (Mapping, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.pages[page]
	if !ok || entry == nil || entry.data == nil {
		return Mapping{}, false
	}
	return Mapping{Page: page, Flags: entry.flags}, true
}

func (m *Memory) IsHole(start Page, pages Pages) bool {
	if !validRange(start, pages) {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for page := start; page < start+Page(pages); page++ {
		if _, ok := m.pages[page]; ok {
			return false
		}
	}
	return true
}

func (m *Memory) FindHole(size Pages) Page {
	if size == 0 || Page(size) > LastHole-FirstHole+1 {
		return BadPage
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for end := LastHole + 1; end >= FirstHole+Page(size); end-- {
		start := end - Page(size)
		free := true
		for page := start; page < end; page++ {
			if _, ok := m.pages[page]; ok {
				free = false
				break
			}
		}
		if free {
			return start
		}
	}
	return BadPage
}

func (m *Memory) MapNothing(start Page, pages Pages, flags Flags) error {
	if pages == 0 {
		return nil
	}
	data := make([]byte, int(pages)*PageSize)
	return m.MapBytes(start, pages, data, 0, flags|PAnonymous)
}

func (m *Memory) MapBytes(start Page, pages Pages, data []byte, offset int, flags Flags) error {
	if !validRange(start, pages) || offset < 0 || len(data) < int(pages)*PageSize+offset {
		return ErrRange
	}
	if pages == 0 {
		return nil
	}
	copyData := make([]byte, len(data))
	copy(copyData, data)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unmapRangeLocked(start, pages)
	back := &backing{data: copyData, refs: int(pages)}
	for i := Page(0); i < Page(pages); i++ {
		m.pages[start+i] = &pageEntry{data: back, offset: offset + int(i)*PageSize, flags: flags}
	}
	m.changes++
	return nil
}

func (m *Memory) Map(start Page, pages Pages, flags Flags) error {
	return m.MapNothing(start, pages, flags)
}

func (m *Memory) Unmap(start Page, pages Pages) error {
	if !validRange(start, pages) {
		return ErrRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for page := start; page < start+Page(pages); page++ {
		if _, ok := m.pages[page]; !ok {
			return ErrUnmapped
		}
	}
	m.unmapRangeLocked(start, pages)
	m.changes++
	return nil
}

func (m *Memory) UnmapAlways(start Page, pages Pages) error {
	if !validRange(start, pages) {
		return ErrRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unmapRangeLocked(start, pages)
	m.changes++
	return nil
}

func (m *Memory) SetFlags(start Page, pages Pages, flags Flags) error {
	if !validRange(start, pages) {
		return ErrRange
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for page := start; page < start+Page(pages); page++ {
		entry, ok := m.pages[page]
		if !ok || entry == nil {
			return ErrUnmapped
		}
		entry.flags = flags
	}
	m.changes++
	return nil
}

func (m *Memory) CopyOnWrite(src, dst *Memory, start Page, pages Pages) error {
	if src == nil || dst == nil || !validRange(start, pages) {
		return ErrRange
	}
	if src == dst {
		return nil
	}
	// Lock order is stable to avoid deadlocks when callers copy concurrently.
	first, second := src, dst
	if fmt.Sprintf("%p", first) > fmt.Sprintf("%p", second) {
		first, second = second, first
	}
	first.mu.Lock()
	second.mu.Lock()
	defer first.mu.Unlock()
	defer second.mu.Unlock()
	for page := start; page < start+Page(pages); page++ {
		if old := dst.pages[page]; old != nil {
			dst.removePageLocked(page)
		}
		entry := src.pages[page]
		if entry == nil {
			continue
		}
		if entry.flags&PShared == 0 {
			entry.flags |= PCOW
		}
		entry.data.refs++
		dst.pages[page] = &pageEntry{data: entry.data, offset: entry.offset, flags: entry.flags}
	}
	src.changes++
	dst.changes++
	return nil
}

func (m *Memory) Read(addr Address, dst []byte) error {
	for len(dst) > 0 {
		m.mu.Lock()
		entry, pageOffset, err := m.translateForWriteLocked(addr, Read)
		m.mu.Unlock()
		if err != nil {
			return err
		}
		available := PageSize - pageOffset
		if available > len(dst) {
			available = len(dst)
		}
		copy(dst[:available], entry.data.data[entry.offset+pageOffset:entry.offset+pageOffset+available])
		dst = dst[available:]
		addr += Address(available)
	}
	return nil
}

func (m *Memory) Write(addr Address, src []byte) error {
	for len(src) > 0 {
		m.mu.Lock()
		entry, pageOffset, err := m.translateForWriteLocked(addr, Write)
		if err != nil {
			m.mu.Unlock()
			return err
		}
		available := PageSize - pageOffset
		if available > len(src) {
			available = len(src)
		}
		copy(entry.data.data[entry.offset+pageOffset:entry.offset+pageOffset+available], src[:available])
		m.mu.Unlock()
		src = src[available:]
		addr += Address(available)
	}
	return nil
}

func (m *Memory) Translate(addr Address, access Access) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, offset, err := m.translateForWriteLocked(addr, access)
	if err != nil {
		return nil, err
	}
	return entry.data.data[entry.offset+offset : entry.offset+PageSize], nil
}

func (m *Memory) SegvReason(addr Address) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, ok := m.pages[Page(addr>>PageBits)]; !ok {
		return ErrUnmapped
	}
	return ErrProtection
}

func (m *Memory) translateForWriteLocked(addr Address, access Access) (*pageEntry, int, error) {
	page := Page(addr >> PageBits)
	entry, ok := m.pages[page]
	if !ok || entry == nil {
		if access == WritePtrace {
			return nil, 0, ErrUnmapped
		}
		for next := page + 1; next < MemPages; next++ {
			candidate := m.pages[next]
			if candidate == nil {
				continue
			}
			if candidate.flags&PGrowDown == 0 {
				break
			}
			if err := m.mapNothingLocked(page, 1, PWrite|PGrowDown); err != nil {
				return nil, 0, err
			}
			entry = m.pages[page]
			break
		}
		if entry == nil {
			return nil, 0, ErrUnmapped
		}
	}
	if access == Write && entry.flags&PWrite == 0 {
		return nil, 0, ErrProtection
	}
	if access == WritePtrace {
		entry.flags |= PWrite | PCOW
	}
	if access == Write || access == WritePtrace {
		if entry.flags&PCOW != 0 {
			m.copyPageLocked(page)
			entry = m.pages[page]
		}
	}
	return entry, int(addr & Address(PageSize-1)), nil
}

func (m *Memory) mapNothingLocked(start Page, pages Pages, flags Flags) error {
	data := make([]byte, int(pages)*PageSize)
	m.unmapRangeLocked(start, pages)
	back := &backing{data: data, refs: int(pages)}
	for i := Page(0); i < Page(pages); i++ {
		m.pages[start+i] = &pageEntry{data: back, offset: int(i) * PageSize, flags: flags | PAnonymous}
	}
	m.changes++
	return nil
}

func (m *Memory) copyPageLocked(page Page) {
	entry := m.pages[page]
	if entry == nil || entry.flags&PCOW == 0 {
		return
	}
	data := make([]byte, PageSize)
	copy(data, entry.data.data[entry.offset:entry.offset+PageSize])
	m.removePageLocked(page)
	m.pages[page] = &pageEntry{data: &backing{data: data, refs: 1}, flags: entry.flags &^ PCOW}
	m.changes++
}

func (m *Memory) unmapRangeLocked(start Page, pages Pages) {
	for page := start; page < start+Page(pages); page++ {
		m.removePageLocked(page)
	}
}

func (m *Memory) removePageLocked(page Page) {
	entry := m.pages[page]
	if entry == nil {
		return
	}
	delete(m.pages, page)
	if entry.data != nil {
		entry.data.refs--
	}
}

func validRange(start Page, pages Pages) bool {
	if pages == 0 {
		return true
	}
	return start < MemPages && Page(pages) <= MemPages-start
}
