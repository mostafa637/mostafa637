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
)

type Instruction struct {
	Op  Op
	Dst int16
	Src int16
	Imm int64
}

type Emitter interface {
	Emit([]Instruction) ([]byte, error)
}
