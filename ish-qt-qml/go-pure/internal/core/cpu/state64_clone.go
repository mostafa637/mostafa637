package cpu

// Clone copies the architectural state and attaches it to the supplied
// independent address space. The caller may then adjust syscall return
// registers or stack state for the child process.
func (s *MachineState64) Clone(memory *Memory64) *MachineState64 {
	if s == nil || memory == nil {
		return nil
	}
	clone := *s
	clone.Memory = memory
	return &clone
}
