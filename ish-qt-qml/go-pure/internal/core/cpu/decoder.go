package cpu

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrInvalidInstruction     = errors.New("cpu: invalid instruction")
	ErrUnsupportedInstruction = errors.New("cpu: unsupported instruction")
	ErrUnsupportedAddressing  = errors.New("cpu: unsupported addressing mode")
)

type Op uint8

const (
	OpNop Op = iota
	OpMovRegImm
	OpMovRegReg
	OpMovRegMem
	OpMovMemReg
	OpMovMemImm
	OpLeaRegMem
	OpAddRegImm
	OpSubRegImm
	OpAddEAXImm
	OpSubEAXImm
	OpXorRegReg
	OpCmpRegReg
	OpCmpRegImm
	OpCmpEAXImm
	OpTestRegReg
	OpIncReg
	OpDecReg
	OpPushReg
	OpPushMem
	OpPopReg
	OpPushImm
	OpCallRel
	OpCallOperand
	OpRet
	OpJmpRel
	OpJmpOperand
	OpJzRel
	OpJnzRel
	OpInt
	OpHalt
	OpPushFlags
	OpPopFlags
	OpCLD
	OpSTD
)

type MemoryOperand struct {
	Base     Reg32
	Index    Reg32
	Scale    uint8
	Disp     int32
	HasBase  bool
	HasIndex bool
}

type Operand struct {
	Reg    Reg32
	Memory MemoryOperand
	IsMem  bool
}

type Instruction struct {
	Op     Op
	Len    uint32
	Reg    Reg32
	Reg2   Reg32
	Imm    int32
	Rel    int32
	Vector uint8
	Group  uint8
	Dst    Operand
	Src    Operand
}

type codeReader struct {
	memory *Memory
	at     Address
	start  Address
}

func newCodeReader(memory *Memory, at Address) *codeReader {
	return &codeReader{memory: memory, at: at, start: at}
}

func (r *codeReader) byte() (byte, error) {
	var value [1]byte
	if r.memory == nil {
		return 0, ErrUnmapped
	}
	if err := r.memory.Read(r.at, value[:]); err != nil {
		return 0, err
	}
	r.at++
	return value[0], nil
}

func (r *codeReader) u32() (uint32, error) {
	var value [4]byte
	for i := range value {
		b, err := r.byte()
		if err != nil {
			return 0, err
		}
		value[i] = b
	}
	return binary.LittleEndian.Uint32(value[:]), nil
}

func (r *codeReader) length() uint32 { return uint32(r.at - r.start) }

func regOperand(reg Reg32) Operand { return Operand{Reg: reg} }

func (o Operand) isRegister() bool { return !o.IsMem }

func (o Operand) String() string {
	if !o.IsMem {
		return o.Reg.String()
	}
	return fmt.Sprintf("[%s+%s*%d%+d]", o.Memory.Base, o.Memory.Index, o.Memory.Scale, o.Memory.Disp)
}

func parseModRM(reader *codeReader, modrm byte) (Reg32, Operand, error) {
	mod := modrm >> 6
	reg := Reg32((modrm >> 3) & 7)
	rm := modrm & 7
	if mod == 3 {
		return reg, regOperand(Reg32(rm)), nil
	}

	memory := MemoryOperand{Scale: 1}
	if rm == 4 {
		sib, err := reader.byte()
		if err != nil {
			return 0, Operand{}, err
		}
		memory.Scale = 1 << (sib >> 6)
		index := Reg32((sib >> 3) & 7)
		base := Reg32(sib & 7)
		if index != ESP {
			memory.Index = index
			memory.HasIndex = true
		}
		if mod == 0 && base == EBP {
			value, err := reader.u32()
			if err != nil {
				return 0, Operand{}, err
			}
			memory.Disp = int32(value)
		} else {
			memory.Base = base
			memory.HasBase = true
		}
	} else if mod == 0 && rm == 5 {
		value, err := reader.u32()
		if err != nil {
			return 0, Operand{}, err
		}
		memory.Disp = int32(value)
	} else {
		memory.Base = Reg32(rm)
		memory.HasBase = true
	}

	switch mod {
	case 1:
		value, err := reader.byte()
		if err != nil {
			return 0, Operand{}, err
		}
		memory.Disp += int32(int8(value))
	case 2:
		value, err := reader.u32()
		if err != nil {
			return 0, Operand{}, err
		}
		memory.Disp += int32(value)
	}
	return reg, Operand{Memory: memory, IsMem: true}, nil
}

func Decode(memory *Memory, eip Address) (Instruction, error) {
	reader := newCodeReader(memory, eip)
	opcode, err := reader.byte()
	if err != nil {
		return Instruction{}, err
	}
	instruction := Instruction{Len: 1}

	switch {
	case opcode == 0x90:
		instruction.Op = OpNop
	case opcode >= 0xB8 && opcode <= 0xBF:
		imm, err := reader.u32()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpMovRegImm
		instruction.Reg = Reg32(opcode - 0xB8)
		instruction.Imm = int32(imm)
	case opcode == 0x05:
		imm, err := reader.u32()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpAddEAXImm
		instruction.Imm = int32(imm)
	case opcode == 0x2D:
		imm, err := reader.u32()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpSubEAXImm
		instruction.Imm = int32(imm)
	case opcode == 0x3D:
		imm, err := reader.u32()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpCmpEAXImm
		instruction.Imm = int32(imm)
	case opcode >= 0x40 && opcode <= 0x47:
		instruction.Op = OpIncReg
		instruction.Reg = Reg32(opcode - 0x40)
	case opcode >= 0x48 && opcode <= 0x4F:
		instruction.Op = OpDecReg
		instruction.Reg = Reg32(opcode - 0x48)
	case opcode >= 0x50 && opcode <= 0x57:
		instruction.Op = OpPushReg
		instruction.Reg = Reg32(opcode - 0x50)
	case opcode >= 0x58 && opcode <= 0x5F:
		instruction.Op = OpPopReg
		instruction.Reg = Reg32(opcode - 0x58)
	case opcode == 0x68:
		imm, err := reader.u32()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpPushImm
		instruction.Imm = int32(imm)
	case opcode == 0x6A:
		imm, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpPushImm
		instruction.Imm = int32(int8(imm))
	case opcode == 0x89 || opcode == 0x8B || opcode == 0x31 || opcode == 0x39 || opcode == 0x85:
		modrm, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		reg, operand, err := parseModRM(reader, modrm)
		if err != nil {
			return Instruction{}, err
		}
		if opcode == 0x89 {
			if operand.isRegister() {
				instruction.Op = OpMovRegReg
				instruction.Reg, instruction.Reg2 = operand.Reg, reg
			} else {
				instruction.Op = OpMovMemReg
				instruction.Dst, instruction.Src = operand, regOperand(reg)
			}
		} else if opcode == 0x8B {
			if operand.isRegister() {
				instruction.Op = OpMovRegReg
				instruction.Reg, instruction.Reg2 = reg, operand.Reg
			} else {
				instruction.Op = OpMovRegMem
				instruction.Dst, instruction.Src = regOperand(reg), operand
			}
		} else {
			if operand.IsMem {
				return Instruction{}, fmt.Errorf("%w: opcode=%#x", ErrUnsupportedAddressing, opcode)
			}
			instruction.Reg, instruction.Reg2 = operand.Reg, reg
			switch opcode {
			case 0x31:
				instruction.Op = OpXorRegReg
			case 0x39:
				instruction.Op = OpCmpRegReg
			case 0x85:
				instruction.Op = OpTestRegReg
			}
		}
	case opcode == 0x8D:
		modrm, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		reg, operand, err := parseModRM(reader, modrm)
		if err != nil {
			return Instruction{}, err
		}
		if !operand.IsMem {
			return Instruction{}, ErrUnsupportedAddressing
		}
		instruction.Op = OpLeaRegMem
		instruction.Dst, instruction.Src = regOperand(reg), operand
	case opcode == 0xC7:
		modrm, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		reg, operand, err := parseModRM(reader, modrm)
		if err != nil {
			return Instruction{}, err
		}
		if reg != EAX {
			return Instruction{}, fmt.Errorf("%w: c7 group=%d", ErrUnsupportedInstruction, reg)
		}
		imm, err := reader.u32()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Imm = int32(imm)
		if operand.IsMem {
			instruction.Op = OpMovMemImm
			instruction.Dst = operand
		} else {
			instruction.Op = OpMovRegImm
			instruction.Reg = operand.Reg
		}
	case opcode == 0xA1 || opcode == 0xA3:
		address, err := reader.u32()
		if err != nil {
			return Instruction{}, err
		}
		operand := Operand{Memory: MemoryOperand{Disp: int32(address), Scale: 1}, IsMem: true}
		if opcode == 0xA1 {
			instruction.Op = OpMovRegMem
			instruction.Dst, instruction.Src = regOperand(EAX), operand
		} else {
			instruction.Op = OpMovMemReg
			instruction.Dst, instruction.Src = operand, regOperand(EAX)
		}
	case opcode == 0x83 || opcode == 0x81:
		modrm, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		reg, operand, err := parseModRM(reader, modrm)
		if err != nil {
			return Instruction{}, err
		}
		instruction.Group = uint8(reg)
		if opcode == 0x83 {
			imm, err := reader.byte()
			if err != nil {
				return Instruction{}, err
			}
			instruction.Imm = int32(int8(imm))
		} else {
			imm, err := reader.u32()
			if err != nil {
				return Instruction{}, err
			}
			instruction.Imm = int32(imm)
		}
		if operand.IsMem {
			return Instruction{}, fmt.Errorf("%w: group=%d", ErrUnsupportedAddressing, reg)
		}
		instruction.Reg = operand.Reg
		switch reg {
		case 0:
			instruction.Op = OpAddRegImm
		case 5:
			instruction.Op = OpSubRegImm
		case 7:
			instruction.Op = OpCmpRegImm
		default:
			return Instruction{}, fmt.Errorf("%w: group=%d", ErrUnsupportedInstruction, reg)
		}
	case opcode == 0xFF:
		modrm, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		reg, operand, err := parseModRM(reader, modrm)
		if err != nil {
			return Instruction{}, err
		}
		instruction.Group = uint8(reg)
		switch reg {
		case 2:
			instruction.Op = OpCallOperand
			instruction.Src = operand
		case 4:
			instruction.Op = OpJmpOperand
			instruction.Src = operand
		case 6:
			instruction.Op = OpPushMem
			instruction.Src = operand
		default:
			return Instruction{}, fmt.Errorf("%w: ff group=%d", ErrUnsupportedInstruction, reg)
		}
	case opcode == 0xE8:
		rel, err := reader.u32()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpCallRel
		instruction.Rel = int32(rel)
	case opcode == 0xC3:
		instruction.Op = OpRet
	case opcode == 0xEB:
		rel, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpJmpRel
		instruction.Rel = int32(int8(rel))
	case opcode == 0xE9:
		rel, err := reader.u32()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpJmpRel
		instruction.Rel = int32(rel)
	case opcode == 0x74 || opcode == 0x75:
		rel, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Rel = int32(int8(rel))
		if opcode == 0x74 {
			instruction.Op = OpJzRel
		} else {
			instruction.Op = OpJnzRel
		}
	case opcode == 0x0F:
		extension, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		if extension != 0x84 && extension != 0x85 {
			return Instruction{}, fmt.Errorf("%w: 0f %#x", ErrUnsupportedInstruction, extension)
		}
		rel, err := reader.u32()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Rel = int32(rel)
		if extension == 0x84 {
			instruction.Op = OpJzRel
		} else {
			instruction.Op = OpJnzRel
		}
	case opcode == 0xCD:
		vector, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpInt
		instruction.Vector = vector
	case opcode == 0xF4:
		instruction.Op = OpHalt
	case opcode == 0x9C:
		instruction.Op = OpPushFlags
	case opcode == 0x9D:
		instruction.Op = OpPopFlags
	case opcode == 0xFC:
		instruction.Op = OpCLD
	case opcode == 0xFD:
		instruction.Op = OpSTD
	default:
		return Instruction{}, fmt.Errorf("%w: opcode=%#x", ErrUnsupportedInstruction, opcode)
	}
	instruction.Len = reader.length()
	return instruction, nil
}
