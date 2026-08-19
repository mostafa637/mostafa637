package cpu

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"math/bits"

	"github.com/mostafa637/mostafa637/go-pure/internal/core/emu/fpu"
)

// Reg64 identifies one of the sixteen general-purpose registers available in
// long mode. The numbering deliberately matches the architectural register
// encoding used by x86asm after REX extension.
type Reg64 uint8

const (
	RAX Reg64 = iota
	RCX
	RDX
	RBX
	RSP
	RBP
	RSI
	RDI
	R8
	R9
	R10
	R11
	R12
	R13
	R14
	R15
	Reg64Count
	Reg64None = Reg64Count
)

func (r Reg64) String() string {
	switch r {
	case RAX:
		return "rax"
	case RCX:
		return "rcx"
	case RDX:
		return "rdx"
	case RBX:
		return "rbx"
	case RSP:
		return "rsp"
	case RBP:
		return "rbp"
	case RSI:
		return "rsi"
	case RDI:
		return "rdi"
	case R8:
		return "r8"
	case R9:
		return "r9"
	case R10:
		return "r10"
	case R11:
		return "r11"
	case R12:
		return "r12"
	case R13:
		return "r13"
	case R14:
		return "r14"
	case R15:
		return "r15"
	default:
		return "?"
	}
}

const (
	Flag64CF uint64 = 1 << 0
	Flag64PF uint64 = 1 << 2
	Flag64AF uint64 = 1 << 4
	Flag64ZF uint64 = 1 << 6
	Flag64SF uint64 = 1 << 7
	Flag64TF uint64 = 1 << 8
	Flag64IF uint64 = 1 << 9
	Flag64DF uint64 = 1 << 10
	Flag64OF uint64 = 1 << 11
)

const (
	lazy64PF uint8 = 1 << iota
	lazy64ZF
	lazy64SF
	lazy64AF
)

// RDRAND64Provider supplies an architectural random value of the requested
// width. The boolean reports whether a value was available, matching RDRAND's
// CF result. It is injectable so deterministic tests can cover both outcomes.
type RDRAND64Provider func(width uint8) (value uint64, ok bool)

func defaultRDRAND64(width uint8) (uint64, bool) {
	if width != 2 && width != 4 && width != 8 {
		return 0, false
	}
	var bytes [8]byte
	if _, err := cryptorand.Read(bytes[:width]); err != nil {
		return 0, false
	}
	return binary.LittleEndian.Uint64(bytes[:]), true
}

// MachineState64 is the guest-visible long-mode state. It is intentionally
// separate from MachineState: widening the existing i386 fields would silently
// change stack, flag, and syscall semantics for the already-supported ABI.
type MachineState64 struct {
	Memory *Memory64
	Cycle  uint64
	Regs   [Reg64Count]uint64
	RIP    uint64
	RFLAGS uint64

	CF        uint8
	OF        uint8
	Res       uint64
	Op1       uint64
	Op2       uint64
	Lazy      uint8
	LazyWidth uint8

	XMM       [16][16]byte
	MMX       [8]uint64
	FP        [8]fpu.Value
	FSW       uint16
	FCW       uint16
	FTW       uint8
	FOP       uint16
	FIP       uint64
	FDP       uint64
	MXCSR     uint32
	MXCSRMask uint32
	FSBase    uint64
	GSBase    uint64
	TLS       uint64
	XCR0      uint64

	RDRAND RDRAND64Provider

	TrapNo           uint64
	FaultAt          Address64
	FaultWrite       bool
	Halted           bool
	Poked            uint32
	InstructionCount uint64
}

func NewMachineState64(memory *Memory64) *MachineState64 {
	return &MachineState64{Memory: memory, RFLAGS: Flag64IF, FCW: 0x037f, MXCSR: 0x1f80, MXCSRMask: 0xffbf, XCR0: 0x3, RDRAND: defaultRDRAND64}
}

func (s *MachineState64) Get(reg Reg64) uint64 {
	if reg >= Reg64Count {
		return 0
	}
	return s.Regs[reg]
}

func (s *MachineState64) Set(reg Reg64, value uint64) {
	if reg < Reg64Count {
		s.Regs[reg] = value
	}
}

func (s *MachineState64) Flag(flag uint64) bool {
	switch flag {
	case Flag64CF:
		return s.CF != 0
	case Flag64OF:
		return s.OF != 0
	case Flag64ZF:
		if s.Lazy&lazy64ZF != 0 {
			return s.Res == 0
		}
	case Flag64SF:
		if s.Lazy&lazy64SF != 0 {
			width := s.LazyWidth
			if width != 1 && width != 2 && width != 4 && width != 8 {
				width = 8
			}
			return s.Res&(uint64(1)<<(uint(width)*8-1)) != 0
		}
	case Flag64PF:
		if s.Lazy&lazy64PF != 0 {
			return bits.OnesCount8(uint8(s.Res))%2 == 0
		}
	case Flag64AF:
		if s.Lazy&lazy64AF != 0 {
			return ((s.Op1 ^ s.Op2 ^ s.Res) >> 4 & 1) != 0
		}
	}
	return s.RFLAGS&flag != 0
}

func (s *MachineState64) SetLazyArithmetic(op1, op2, result uint64, carry, overflow bool, computeAF bool) {
	s.SetLazyArithmeticWidth(op1, op2, result, carry, overflow, computeAF, 8)
}

func (s *MachineState64) SetLazyArithmeticWidth(op1, op2, result uint64, carry, overflow bool, computeAF bool, width uint8) {
	s.Op1 = op1
	s.Op2 = op2
	s.Res = result
	s.CF = boolByte64(carry)
	s.OF = boolByte64(overflow)
	s.LazyWidth = width
	s.Lazy = lazy64PF | lazy64ZF | lazy64SF
	if computeAF {
		s.Lazy |= lazy64AF
	}
}

func (s *MachineState64) CollapseFlags() {
	const arithmetic = Flag64CF | Flag64PF | Flag64AF | Flag64ZF | Flag64SF | Flag64OF
	var flags uint64
	for _, flag := range []uint64{Flag64CF, Flag64PF, Flag64AF, Flag64ZF, Flag64SF, Flag64OF} {
		if s.Flag(flag) {
			flags |= flag
		}
	}
	s.RFLAGS = (s.RFLAGS &^ arithmetic) | flags | Flag64IF
	s.CF = boolByte64(s.RFLAGS&Flag64CF != 0)
	s.OF = boolByte64(s.RFLAGS&Flag64OF != 0)
	s.Lazy = 0
	s.LazyWidth = 0
}

func (s *MachineState64) ExpandFlags() {
	s.CF = boolByte64(s.RFLAGS&Flag64CF != 0)
	s.OF = boolByte64(s.RFLAGS&Flag64OF != 0)
	s.Lazy = 0
	s.LazyWidth = 0
}

func (s *MachineState64) SetRFLAGS(value uint64) {
	s.RFLAGS = value | Flag64IF
	s.ExpandFlags()
}

const (
	fpu64StatusC0 uint16 = 1 << 8
	fpu64StatusC1 uint16 = 1 << 9
	fpu64StatusC2 uint16 = 1 << 10
	fpu64StatusC3 uint16 = 1 << 14
)

// EnterMMX marks the architectural transition made by MMX/SSE bridge
// instructions. The current state model has no x87 tag word, but it does
// preserve the architectural TOP reset required by the transition.
func (s *MachineState64) EnterMMX() {
	s.SetFPUTop(0)
	s.FTW = 0xff
}

func (s *MachineState64) FPUTop() uint8 {
	return uint8((s.FSW >> 11) & 7)
}

func (s *MachineState64) SetFPUTop(top uint8) {
	s.FSW = (s.FSW &^ (7 << 11)) | uint16(top&7)<<11
}

func (s *MachineState64) MoveFPUTop(delta int8) {
	top := int16(s.FPUTop()) + int16(delta)
	top %= 8
	if top < 0 {
		top += 8
	}
	s.SetFPUTop(uint8(top))
}

func (s *MachineState64) FPAt(index uint8) fpu.Value {
	return s.FP[(s.FPUTop()+index)&7]
}

func (s *MachineState64) SetFPAt(index uint8, value fpu.Value) {
	physical := (s.FPUTop() + index) & 7
	s.FP[physical] = value
	s.FTW |= 1 << physical
}

func (s *MachineState64) PushFP(value fpu.Value) {
	s.MoveFPUTop(-1)
	physical := s.FPUTop()
	s.FP[physical] = value
	s.FTW |= 1 << physical
}

func (s *MachineState64) PopFP() fpu.Value {
	physical := s.FPUTop()
	value := s.FP[physical]
	s.FTW &^= 1 << physical
	s.MoveFPUTop(1)
	return value
}

func (s *MachineState64) SetFPUCondition(c0, c1, c2, c3 bool) {
	s.FSW &^= fpu64StatusC0 | fpu64StatusC1 | fpu64StatusC2 | fpu64StatusC3
	if c0 {
		s.FSW |= fpu64StatusC0
	}
	if c1 {
		s.FSW |= fpu64StatusC1
	}
	if c2 {
		s.FSW |= fpu64StatusC2
	}
	if c3 {
		s.FSW |= fpu64StatusC3
	}
}

func (s *MachineState64) FPUStatusWord() uint16 {
	return s.FSW
}

func boolByte64(value bool) uint8 {
	if value {
		return 1
	}
	return 0
}
