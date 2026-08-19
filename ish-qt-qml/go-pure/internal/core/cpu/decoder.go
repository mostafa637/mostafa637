package cpu

import (
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/arch/x86/x86asm"
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
	OpLogical
	OpLogicalImm
	OpTestOperands
	OpTestImm
	OpShift
	OpUnary
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
	OpAddOperandImm
	OpSubOperandImm
	OpCmpOperandImm
	OpIncOperand
	OpDecOperand
	OpImulRegOperand
	OpMovzxRegOperand
	OpPushAll
	OpPopAll
	OpLeave
	OpRetImm
	OpCWDE
	OpCDQ
)

type Segment uint8

const (
	SegmentNone Segment = iota
	SegmentFS
	SegmentGS
)

type MemoryOperand struct {
	Base     Reg32
	Index    Reg32
	Scale    uint8
	Disp     int32
	HasBase  bool
	HasIndex bool
	Segment  Segment
}

type Operand struct {
	Reg    Reg32
	Memory MemoryOperand
	IsMem  bool
	Width  uint8 // operand width in bytes; the current core supports 1, 2, and 4
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

func regOperand(reg Reg32) Operand { return Operand{Reg: reg, Width: 4} }

func (o Operand) isRegister() bool { return !o.IsMem }

func (o Operand) String() string {
	if !o.IsMem {
		return o.Reg.String()
	}
	return fmt.Sprintf("[%s+%s*%d%+d]", o.Memory.Base, o.Memory.Index, o.Memory.Scale, o.Memory.Disp)
}

func parseModRM(reader *codeReader, modrm byte, segment Segment) (Reg32, Operand, error) {
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
	memory.Segment = segment
	return reg, Operand{Memory: memory, IsMem: true, Width: 4}, nil
}

// Disassemble32 decodes one guest instruction with the canonical x86asm
// decoder in 32-bit mode. The executor still consumes the project's compact
// Instruction IR so unsupported semantics remain explicit at the execution
// boundary.
func Disassemble32(memory *Memory, eip Address) (x86asm.Inst, error) {
	if memory == nil {
		return x86asm.Inst{}, ErrUnmapped
	}
	var code [15]byte
	read := 0
	for read < len(code) {
		if err := memory.Read(eip+Address(read), code[read:read+1]); err != nil {
			if read == 0 {
				return x86asm.Inst{}, err
			}
			break
		}
		read++
	}
	inst, err := x86asm.Decode(code[:read], 32)
	if err != nil {
		return x86asm.Inst{}, fmt.Errorf("cpu: x86asm decode at %#x: %w", eip, err)
	}
	return inst, nil
}

func Decode(memory *Memory, eip Address) (Instruction, error) {
	disassembled, err := Disassemble32(memory, eip)
	if err != nil {
		return Instruction{}, err
	}
	if logical, handled, err := decodeX86Logical(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return logical, nil
	}
	if shift, handled, err := decodeX86Shift(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return shift, nil
	}
	if unary, handled, err := decodeX86Unary(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return unary, nil
	}
	reader := newCodeReader(memory, eip)
	opcode, err := reader.byte()
	if err != nil {
		return Instruction{}, err
	}
	var segment Segment
	for opcode == 0x64 || opcode == 0x65 {
		if opcode == 0x64 {
			segment = SegmentFS
		} else {
			segment = SegmentGS
		}
		opcode, err = reader.byte()
		if err != nil {
			return Instruction{}, err
		}
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
		reg, operand, err := parseModRM(reader, modrm, segment)
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
		reg, operand, err := parseModRM(reader, modrm, segment)
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
		reg, operand, err := parseModRM(reader, modrm, segment)
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
		reg, operand, err := parseModRM(reader, modrm, segment)
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
		instruction.Dst = operand
		instruction.Reg = operand.Reg
		switch reg {
		case 0:
			if operand.IsMem {
				instruction.Op = OpAddOperandImm
			} else {
				instruction.Op = OpAddRegImm
			}
		case 5:
			if operand.IsMem {
				instruction.Op = OpSubOperandImm
			} else {
				instruction.Op = OpSubRegImm
			}
		case 7:
			if operand.IsMem {
				instruction.Op = OpCmpOperandImm
			} else {
				instruction.Op = OpCmpRegImm
			}
		default:
			return Instruction{}, fmt.Errorf("%w: group=%d", ErrUnsupportedInstruction, reg)
		}
	case opcode == 0xFF:
		modrm, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		reg, operand, err := parseModRM(reader, modrm, segment)
		if err != nil {
			return Instruction{}, err
		}
		instruction.Group = uint8(reg)
		switch reg {
		case 0:
			if operand.IsMem {
				instruction.Op = OpIncOperand
				instruction.Dst = operand
			} else {
				return Instruction{}, fmt.Errorf("%w: ff /0 register", ErrUnsupportedInstruction)
			}
		case 1:
			if operand.IsMem {
				instruction.Op = OpDecOperand
				instruction.Dst = operand
			} else {
				return Instruction{}, fmt.Errorf("%w: ff /1 register", ErrUnsupportedInstruction)
			}
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
		switch extension {
		case 0x84, 0x85:
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
		case 0xAF:
			modrm, err := reader.byte()
			if err != nil {
				return Instruction{}, err
			}
			reg, operand, err := parseModRM(reader, modrm, segment)
			if err != nil {
				return Instruction{}, err
			}
			instruction.Op = OpImulRegOperand
			instruction.Dst = regOperand(reg)
			instruction.Src = operand
		case 0xB6, 0xB7:
			modrm, err := reader.byte()
			if err != nil {
				return Instruction{}, err
			}
			reg, operand, err := parseModRM(reader, modrm, segment)
			if err != nil {
				return Instruction{}, err
			}
			if !operand.IsMem {
				return Instruction{}, fmt.Errorf("%w: movzx register source", ErrUnsupportedAddressing)
			}
			if extension == 0xB6 {
				operand.Width = 1
			} else {
				operand.Width = 2
			}
			instruction.Op = OpMovzxRegOperand
			instruction.Dst = regOperand(reg)
			instruction.Src = operand
		default:
			return Instruction{}, fmt.Errorf("%w: 0f %#x", ErrUnsupportedInstruction, extension)
		}
	case opcode == 0xCD:
		vector, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpInt
		instruction.Vector = vector
	case opcode == 0x60:
		instruction.Op = OpPushAll
	case opcode == 0x61:
		instruction.Op = OpPopAll
	case opcode == 0xC9:
		instruction.Op = OpLeave
	case opcode == 0xC2:
		lo, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		hi, err := reader.byte()
		if err != nil {
			return Instruction{}, err
		}
		instruction.Op = OpRetImm
		instruction.Imm = int32(uint16(lo) | uint16(hi)<<8)
	case opcode == 0x98:
		instruction.Op = OpCWDE
	case opcode == 0x99:
		instruction.Op = OpCDQ
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
	if instruction.Len != uint32(disassembled.Len) {
		return Instruction{}, fmt.Errorf("cpu: decoder length mismatch at %#x: IR=%d x86asm=%d", eip, instruction.Len, disassembled.Len)
	}
	return instruction, nil
}

func decodeX86Logical(inst x86asm.Inst) (Instruction, bool, error) {
	if inst.DataSize != 32 || len(inst.Args) < 2 || inst.Args[0] == nil || inst.Args[1] == nil {
		return Instruction{}, false, nil
	}
	var group uint8
	switch inst.Op {
	case x86asm.OR:
		group = 0
	case x86asm.AND:
		group = 1
	case x86asm.XOR:
		group = 2
	case x86asm.TEST:
		group = 3
	default:
		return Instruction{}, false, nil
	}
	dst, ok, err := x86Operand32(inst.Args[0])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	if imm, ok := inst.Args[1].(x86asm.Imm); ok {
		if inst.Op == x86asm.TEST {
			return Instruction{Op: OpTestImm, Len: uint32(inst.Len), Dst: dst, Imm: int32(imm)}, true, nil
		}
		return Instruction{Op: OpLogicalImm, Len: uint32(inst.Len), Dst: dst, Group: group, Imm: int32(imm)}, true, nil
	}
	src, ok, err := x86Operand32(inst.Args[1])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	if inst.Op == x86asm.TEST {
		if !dst.IsMem && !src.IsMem {
			return Instruction{Op: OpTestRegReg, Len: uint32(inst.Len), Reg: dst.Reg, Reg2: src.Reg}, true, nil
		}
		return Instruction{Op: OpTestOperands, Len: uint32(inst.Len), Dst: dst, Src: src}, true, nil
	}
	return Instruction{Op: OpLogical, Len: uint32(inst.Len), Dst: dst, Src: src, Group: group}, true, nil
}

func x86Operand32(arg x86asm.Arg) (Operand, bool, error) {
	switch value := arg.(type) {
	case x86asm.Reg:
		reg, ok := x86Reg32(value)
		if !ok {
			return Operand{}, false, fmt.Errorf("%w: register %v", ErrUnsupportedAddressing, value)
		}
		return regOperand(reg), true, nil
	case x86asm.Mem:
		if value.Disp < -1<<31 || value.Disp > 1<<31-1 || value.Scale > 8 {
			return Operand{}, false, fmt.Errorf("%w: memory %v", ErrUnsupportedAddressing, value)
		}
		memory := MemoryOperand{Disp: int32(value.Disp), Scale: value.Scale}
		if memory.Scale == 0 {
			memory.Scale = 1
		}
		if value.Base != 0 {
			base, ok := x86Reg32(value.Base)
			if !ok {
				return Operand{}, false, fmt.Errorf("%w: base %v", ErrUnsupportedAddressing, value.Base)
			}
			memory.Base, memory.HasBase = base, true
		}
		if value.Index != 0 {
			index, ok := x86Reg32(value.Index)
			if !ok {
				return Operand{}, false, fmt.Errorf("%w: index %v", ErrUnsupportedAddressing, value.Index)
			}
			memory.Index, memory.HasIndex = index, true
		}
		switch value.Segment {
		case x86asm.FS:
			memory.Segment = SegmentFS
		case x86asm.GS:
			memory.Segment = SegmentGS
		}
		return Operand{Memory: memory, IsMem: true, Width: 4}, true, nil
	default:
		return Operand{}, false, nil
	}
}

func x86Reg32(reg x86asm.Reg) (Reg32, bool) {
	switch reg {
	case x86asm.EAX:
		return EAX, true
	case x86asm.ECX:
		return ECX, true
	case x86asm.EDX:
		return EDX, true
	case x86asm.EBX:
		return EBX, true
	case x86asm.ESP:
		return ESP, true
	case x86asm.EBP:
		return EBP, true
	case x86asm.ESI:
		return ESI, true
	case x86asm.EDI:
		return EDI, true
	default:
		return RegNone, false
	}
}

func decodeX86Shift(inst x86asm.Inst) (Instruction, bool, error) {
	if inst.DataSize != 32 || len(inst.Args) < 2 || inst.Args[0] == nil || inst.Args[1] == nil {
		return Instruction{}, false, nil
	}
	var group uint8
	switch inst.Op {
	case x86asm.SHL:
		group = 0
	case x86asm.SHR:
		group = 1
	case x86asm.SAR:
		group = 2
	default:
		return Instruction{}, false, nil
	}
	dst, ok, err := x86Operand32(inst.Args[0])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	instruction := Instruction{Op: OpShift, Len: uint32(inst.Len), Dst: dst, Group: group}
	switch value := inst.Args[1].(type) {
	case x86asm.Imm:
		instruction.Imm = int32(value) & 0x1f
	case x86asm.Reg:
		if value != x86asm.CL {
			return Instruction{}, true, fmt.Errorf("%w: shift count register %v", ErrUnsupportedAddressing, value)
		}
		// Width 1 marks the implicit CL count without adding a new IR type.
		instruction.Src = Operand{Reg: ECX, Width: 1}
	default:
		return Instruction{}, true, fmt.Errorf("%w: shift count %T", ErrUnsupportedAddressing, inst.Args[1])
	}
	return instruction, true, nil
}

func decodeX86Unary(inst x86asm.Inst) (Instruction, bool, error) {
	if inst.DataSize != 32 || len(inst.Args) < 1 || inst.Args[0] == nil {
		return Instruction{}, false, nil
	}
	var group uint8
	switch inst.Op {
	case x86asm.NOT:
		group = 0
	case x86asm.NEG:
		group = 1
	default:
		return Instruction{}, false, nil
	}
	dst, ok, err := x86Operand32(inst.Args[0])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	return Instruction{Op: OpUnary, Len: uint32(inst.Len), Dst: dst, Group: group}, true, nil
}
