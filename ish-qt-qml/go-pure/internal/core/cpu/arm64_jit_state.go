package cpu

type ARM64State struct {
	Regs   [31]uint64
	SP     uint64
	PC     uint64
	N, Z   bool
	C, V   bool
	Memory *Memory64
	Halted bool
}

func NewARM64State(memory *Memory64) *ARM64State {
	return &ARM64State{Memory: memory}
}

func (s *ARM64State) read(ref arm64RegRef) uint64 {
	if ref.zero {
		return 0
	}
	if ref.sp {
		return s.SP
	}
	return s.Regs[ref.index]
}

func (s *ARM64State) write(ref arm64RegRef, value uint64) {
	if ref.zero {
		return
	}
	if ref.sp {
		s.SP = value
		return
	}
	s.Regs[ref.index] = value
}

func (s *ARM64State) setNZ(value uint64) {
	s.Z = value == 0
	s.N = value>>63 != 0
}
