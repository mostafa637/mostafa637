package cpu

// Clone creates an independent sparse guest address space. It deliberately
// copies page contents rather than sharing host backing arrays, so a child
// created by fork cannot mutate the parent's memory through a shared slice.
func (m *Memory64) Clone() *Memory64 {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	clone := &Memory64{
		pages:   make(map[Page64]*page64Entry, len(m.pages)),
		gens:    make(map[Page64]uint64, len(m.gens)),
		changes: m.changes,
	}
	for page, entry := range m.pages {
		if entry == nil {
			continue
		}
		data := make([]byte, len(entry.data))
		copy(data, entry.data)
		clone.pages[page] = &page64Entry{data: data, flags: entry.flags, gen: entry.gen}
	}
	for page, generation := range m.gens {
		clone.gens[page] = generation
	}
	return clone
}
