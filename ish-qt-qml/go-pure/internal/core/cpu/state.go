package cpu

import (
	"math/bits"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/emu/fpu"
)

type Reg32 uint8

const (
	EAX Reg32 = iota
	ECX
	EDX
	EBX
	ESP
	EBP
	ESI
	EDI
	RegCount
	RegNone = RegCount
)

func (r Reg32) String() string {
	switch r {
	case EAX:
		return "eax"
	case ECX:
		return "ecx"
	case EDX:
		return "edx"
	case EBX:
		return "ebx"
	case ESP:
		return "esp"
	case EBP:
		return "ebp"
	case ESI:
		return "esi"
	case EDI:
		return "edi"
	default:
		return "?"
	}
}

const (
	FlagCF uint32 = 1 << 0
	FlagPF uint32 = 1 << 2
	FlagAF uint32 = 1 << 4
	FlagZF uint32 = 1 << 6
	FlagSF uint32 = 1 << 7
	FlagIF uint32 = 1 << 9
	FlagDF uint32 = 1 << 10
	FlagOF uint32 = 1 << 11
)

const (
	lazyPF uint8 = 1 << iota
	lazyZF
	lazySF
	lazyAF
)

// MachineState is the emulator-facing register set. The memory pointer is
// intentionally typed as the Pure Go Memory implementation, not an OS pointer.
type MachineState struct {
	Memory *Memory
	Cycle  int64
	Regs   [RegCount]uint32
	EIP    uint32
	EFlags uint32

	DFOffset uint32
	CF       uint8
	OF       uint8
	Res      uint32
	Op1      uint32
	Op2      uint32
	Lazy     uint8

	MM  [8]uint64
	XMM [8][16]byte
	FP  [8]fpu.Value
	FSW uint16
	FCW uint16

	GS         uint16
	TLS        uint32
	FaultAt    Address
	FaultWrite bool
	TrapNo     uint32
	Poked      bool
}

func NewMachineState(memory *Memory) *MachineState {
	state := &MachineState{Memory: memory, FCW: 0x037f}
	state.EFlags = FlagIF
	return state
}

func (s *MachineState) Get(reg Reg32) uint32 {
	if reg >= RegCount {
		return 0
	}
	return s.Regs[reg]
}

func (s *MachineState) Set(reg Reg32, value uint32) {
	if reg < RegCount {
		s.Regs[reg] = value
	}
}

func (s *MachineState) EAXValue() uint32 { return s.Regs[EAX] }
func (s *MachineState) SetEAX(v uint32)  { s.Regs[EAX] = v }

func (s *MachineState) Flag(flag uint32) bool {
	switch flag {
	case FlagCF:
		return s.CF != 0
	case FlagOF:
		return s.OF != 0
	case FlagZF:
		if s.Lazy&lazyZF != 0 {
			return s.Res == 0
		}
	case FlagSF:
		if s.Lazy&lazySF != 0 {
			return int32(s.Res) < 0
		}
	case FlagPF:
		if s.Lazy&lazyPF != 0 {
			return bits.OnesCount8(uint8(s.Res))%2 == 0
		}
	case FlagAF:
		if s.Lazy&lazyAF != 0 {
			return ((s.Op1 ^ s.Op2 ^ s.Res) >> 4 & 1) != 0
		}
	}
	return s.EFlags&flag != 0
}

func (s *MachineState) SetLazyArithmetic(op1, op2, result uint32, carry, overflow bool, computeAF bool) {
	s.Op1 = op1
	s.Op2 = op2
	s.Res = result
	s.CF = boolByte(carry)
	s.OF = boolByte(overflow)
	s.Lazy = lazyPF | lazyZF | lazySF
	if computeAF {
		s.Lazy |= lazyAF
	}
}

func (s *MachineState) CollapseFlags() {
	var flags uint32
	for _, flag := range []uint32{FlagCF, FlagPF, FlagAF, FlagZF, FlagSF, FlagOF} {
		if s.Flag(flag) {
			flags |= flag
		}
	}
	// IF and unrelated reserved/control bits remain stable.
	s.EFlags = (s.EFlags &^ (FlagCF | FlagPF | FlagAF | FlagZF | FlagSF | FlagOF)) | flags | FlagIF
	s.CF = boolByte(s.EFlags&FlagCF != 0)
	s.OF = boolByte(s.EFlags&FlagOF != 0)
	s.Lazy = 0
}

func (s *MachineState) ExpandFlags() {
	s.CF = boolByte(s.EFlags&FlagCF != 0)
	s.OF = boolByte(s.EFlags&FlagOF != 0)
	s.Lazy = 0
}

func (s *MachineState) SetEFlags(value uint32) {
	s.EFlags = value | FlagIF
	s.ExpandFlags()
}

func boolByte(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}
