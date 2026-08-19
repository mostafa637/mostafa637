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
	OpPopReg
	OpPushImm
	OpCallRel
	OpRet
	OpJmpRel
	OpJzRel
	OpJnzRel
	OpInt
	OpHalt
	OpPushFlags
	OpPopFlags
	OpCLD
	OpSTD
)

type Instruction struct {
	Op     Op
	Len    uint32
	Reg    Reg32
	Reg2   Reg32
	Imm    int32
	Rel    int32
	Vector uint8
	Group  uint8
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
		if modrm>>6 != 3 {
			return Instruction{}, fmt.Errorf("%w: modrm=%#x", ErrUnsupportedAddressing, modrm)
		}
		reg := Reg32((modrm >> 3) & 7)
		rm := Reg32(modrm & 7)
		instruction.Reg = rm
		instruction.Reg2 = reg
		switch opcode {
		case 0x89:
			instruction.Op = OpMovRegReg
		case 0x8B:
			instruction.Reg, instruction.Reg2 = reg, rm
			instruction.Op = OpMovRegReg
		case 0x31:
			instruction.Op = OpXorRegReg
		case 0x39:
			instruction.Op = OpCmpRegReg
		case 0x85:
			instruction.Op = OpTestRegReg
		}
	case opcode == 0x83 || opcode == 0x81:
		modrm, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		if modrm>>6 != 3 {
			return Instruction{}, fmt.Errorf("%w: modrm=%#x", ErrUnsupportedAddressing, modrm)
		}
		instruction.Group = (modrm >> 3) & 7
		instruction.Reg = Reg32(modrm & 7)
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
		switch instruction.Group {
		case 0:
			instruction.Op = OpAddRegImm
		case 5:
			instruction.Op = OpSubRegImm
		case 7:
			instruction.Op = OpCmpRegImm
		default:
			return Instruction{}, fmt.Errorf("%w: group=%d", ErrUnsupportedInstruction, instruction.Group)
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
