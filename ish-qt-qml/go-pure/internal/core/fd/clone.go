package fd

// Clone duplicates the descriptor table while retaining each shared open-file
// description. This matches fork semantics: descriptor-local cloexec state is
// copied, while the underlying File and its current offset remain shared.
func (t *Table) Clone() *Table {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	clone := &Table{
		entries: make(map[int32]*File, len(t.entries)),
		cloexec: make(map[int32]bool, len(t.cloexec)),
		next:    t.next,
	}
	for fd, file := range t.entries {
		if file == nil {
			continue
		}
		file.retain()
		clone.entries[fd] = file
	}
	for fd, cloexec := range t.cloexec {
		clone.cloexec[fd] = cloexec
	}
	return clone
}
