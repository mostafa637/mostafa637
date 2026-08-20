package machinecode

type Op uint8

const (
	OpNOP Op = iota
	OpMOVImm
	OpADDImm
	OpSUBImm
	OpRET
	OpSyscall
	OpMOVReg
	OpADDReg
	OpSUBReg
	OpANDImm
	OpORImm
	OpXORImm
	OpCMPImm
	OpLoad64
	OpStore64
	OpJmp
	OpJcc
	OpCall
	OpPush64
	OpPop64
	OpLEA64
	OpExtend64
)

type Instruction struct {
	Op          Op
	Dst         int16
	Src         int16
	Imm         int64
	Target      uint64
	Fallthrough uint64
	Cond        uint8
	MemBase     int16
	MemIndex    int16
	MemScale    uint8
	MemRIP      bool
	NextPC      uint64
	Width       uint8
	DstWidth    uint8
	Signed      bool
}

type Emitter interface {
	Emit([]Instruction) ([]byte, error)
}
