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
	OpSetcc
	OpCMOVcc
	OpXchg
	OpIncReg
	OpDecReg
	OpPushReg
	OpPushMem
	OpPopReg
	OpPopMem
	OpPushImm
	OpCallRel
	OpCallOperand
	OpRet
	OpJmpRel
	OpJmpOperand
	OpJzRel
	OpJnzRel
	OpJcc
	OpLoop
	OpInt
	OpHalt
	OpPushFlags
	OpPopFlags
	OpCLD
	OpSTD
	OpAddOperandImm
	OpSubOperandImm
	OpAddOperands
	OpSubOperands
	OpMulImplicit
	OpIMulImplicit
	OpIMulOperands
	OpDivImplicit
	OpIDivImplicit
	OpMovs
	OpStos
	OpLods
	OpScas
	OpCmps
	OpCmpOperandImm
	OpIncOperand
	OpDecOperand
	OpImulRegOperand
	OpMovzxRegOperand
	OpMovsxRegOperand
	OpPushAll
	OpPopAll
	OpLeave
	OpRetImm
	OpCWDE
	OpCDQ
	OpCmpxchg
	OpXadd
	OpAdcOperands
	OpAdcImm
	OpSbbOperands
	OpSbbImm
	OpLahf
	OpSahf
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
	Reg        Reg32
	Memory     MemoryOperand
	IsMem      bool
	Width      uint8 // operand width in bytes; the current core supports 1, 2, and 4
	ByteOffset uint8 // byte-register offset within the 32-bit register (0 or 1)
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
	if addSub, handled, err := decodeX86AddSub(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return addSub, nil
	}
	if mulDiv, handled, err := decodeX86MulDiv(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return mulDiv, nil
	}
	if stringInstruction, handled, err := decodeX86String(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return stringInstruction, nil
	}
	if dataMovement, handled, err := decodeX86DataMovement(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return dataMovement, nil
	}
	if atomicInstruction, handled, err := decodeX86Atomic(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return atomicInstruction, nil
	}
	if carryInstruction, handled, err := decodeX86Carry(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return carryInstruction, nil
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
	if setcc, handled, err := decodeX86Setcc(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return setcc, nil
	}
	if cmov, handled, err := decodeX86CMOVcc(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return cmov, nil
	}
	if jcc, handled, err := decodeX86Jcc(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return jcc, nil
	}
	if loop, handled, err := decodeX86Loop(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return loop, nil
	}
	if xchg, handled, err := decodeX86Xchg(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return xchg, nil
	}
	if stack, handled, err := decodeX86Stack(disassembled); handled {
		if err != nil {
			return Instruction{}, err
		}
		return stack, nil
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

func decodeX86DataMovement(inst x86asm.Inst) (Instruction, bool, error) {
	switch inst.Op {
	case x86asm.MOVZX, x86asm.MOVSX:
		if inst.DataSize != 32 || len(inst.Args) < 2 || inst.Args[0] == nil || inst.Args[1] == nil {
			return Instruction{}, true, fmt.Errorf("%w: %v data size %d or operands", ErrUnsupportedAddressing, inst.Op, inst.DataSize)
		}
		dst, ok, err := x86Operand32(inst.Args[0])
		if err != nil || !ok || dst.IsMem {
			if err == nil {
				err = ErrUnsupportedAddressing
			}
			return Instruction{}, true, err
		}
		var src Operand
		switch inst.MemBytes {
		case 1:
			src, ok, err = x86Operand8(inst.Args[1])
		case 2:
			src, ok, err = x86Operand16(inst.Args[1])
		default:
			if reg, isReg := inst.Args[1].(x86asm.Reg); isReg {
				if isX86ByteReg(reg) {
					src, ok, err = x86Operand8(reg)
				} else if isX86WordReg(reg) {
					src, ok, err = x86Operand16(reg)
				} else {
					err = fmt.Errorf("%w: %v source register", ErrUnsupportedAddressing, reg)
				}
			} else {
				err = fmt.Errorf("%w: %v source width", ErrUnsupportedAddressing, inst.Op)
			}
		}
		if err != nil || !ok {
			if err == nil {
				err = ErrUnsupportedAddressing
			}
			return Instruction{}, true, err
		}
		if src.Width != 1 && src.Width != 2 {
			return Instruction{}, true, fmt.Errorf("%w: %v source width %d", ErrUnsupportedAddressing, inst.Op, src.Width)
		}
		op := OpMovzxRegOperand
		if inst.Op == x86asm.MOVSX {
			op = OpMovsxRegOperand
		}
		return Instruction{Op: op, Len: uint32(inst.Len), Dst: dst, Src: src}, true, nil
	case x86asm.LEA:
		if inst.DataSize != 32 || inst.AddrSize != 32 || len(inst.Args) < 2 || inst.Args[0] == nil || inst.Args[1] == nil {
			return Instruction{}, true, fmt.Errorf("%w: LEA data/address size %d/%d or operands", ErrUnsupportedAddressing, inst.DataSize, inst.AddrSize)
		}
		dst, ok, err := x86Operand32(inst.Args[0])
		if err != nil || !ok || dst.IsMem {
			if err == nil {
				err = ErrUnsupportedAddressing
			}
			return Instruction{}, true, err
		}
		src, ok, err := x86Operand32(inst.Args[1])
		if err != nil || !ok || !src.IsMem {
			if err == nil {
				err = ErrUnsupportedAddressing
			}
			return Instruction{}, true, err
		}
		return Instruction{Op: OpLeaRegMem, Len: uint32(inst.Len), Dst: dst, Src: src}, true, nil
	case x86asm.CWDE:
		if inst.DataSize != 32 || inst.Args[0] != nil {
			return Instruction{}, true, fmt.Errorf("%w: CWDE data size %d or operands", ErrUnsupportedAddressing, inst.DataSize)
		}
		return Instruction{Op: OpCWDE, Len: uint32(inst.Len)}, true, nil
	case x86asm.CDQ:
		if inst.DataSize != 32 || inst.Args[0] != nil {
			return Instruction{}, true, fmt.Errorf("%w: CDQ data size %d or operands", ErrUnsupportedAddressing, inst.DataSize)
		}
		return Instruction{Op: OpCDQ, Len: uint32(inst.Len)}, true, nil
	default:
		return Instruction{}, false, nil
	}
}

func isX86ByteReg(reg x86asm.Reg) bool {
	switch reg {
	case x86asm.AL, x86asm.CL, x86asm.DL, x86asm.BL, x86asm.AH, x86asm.CH, x86asm.DH, x86asm.BH:
		return true
	default:
		return false
	}
}

func isX86WordReg(reg x86asm.Reg) bool {
	switch reg {
	case x86asm.AX, x86asm.CX, x86asm.DX, x86asm.BX, x86asm.SP, x86asm.BP, x86asm.SI, x86asm.DI:
		return true
	default:
		return false
	}
}

func x86Operand16(arg x86asm.Arg) (Operand, bool, error) {
	if memory, ok := arg.(x86asm.Mem); ok {
		operand, ok, err := x86Operand32(memory)
		if err != nil || !ok {
			return Operand{}, false, err
		}
		operand.Width = 2
		return operand, true, nil
	}
	reg, ok := arg.(x86asm.Reg)
	if !ok || !isX86WordReg(reg) {
		return Operand{}, false, fmt.Errorf("%w: word register %v", ErrUnsupportedAddressing, reg)
	}
	var base Reg32
	switch reg {
	case x86asm.AX:
		base = EAX
	case x86asm.CX:
		base = ECX
	case x86asm.DX:
		base = EDX
	case x86asm.BX:
		base = EBX
	case x86asm.SP:
		base = ESP
	case x86asm.BP:
		base = EBP
	case x86asm.SI:
		base = ESI
	case x86asm.DI:
		base = EDI
	}
	return Operand{Reg: base, Width: 2}, true, nil
}

func decodeX86Carry(inst x86asm.Inst) (Instruction, bool, error) {
	switch inst.Op {
	case x86asm.LAHF:
		if inst.DataSize != 32 || inst.Args[0] != nil {
			return Instruction{}, true, fmt.Errorf("%w: LAHF data size %d or operands", ErrUnsupportedAddressing, inst.DataSize)
		}
		return Instruction{Op: OpLahf, Len: uint32(inst.Len)}, true, nil
	case x86asm.SAHF:
		if inst.DataSize != 32 || inst.Args[0] != nil {
			return Instruction{}, true, fmt.Errorf("%w: SAHF data size %d or operands", ErrUnsupportedAddressing, inst.DataSize)
		}
		return Instruction{Op: OpSahf, Len: uint32(inst.Len)}, true, nil
	case x86asm.ADC, x86asm.SBB:
		if inst.DataSize != 32 || len(inst.Args) < 2 || inst.Args[0] == nil || inst.Args[1] == nil {
			return Instruction{}, true, fmt.Errorf("%w: %v data size %d or operands", ErrUnsupportedAddressing, inst.Op, inst.DataSize)
		}
		width := uint8(4)
		if inst.MemBytes == 1 {
			width = 1
		} else if inst.MemBytes != 0 && inst.MemBytes != 4 {
			return Instruction{}, true, fmt.Errorf("%w: %v memory width %d", ErrUnsupportedAddressing, inst.Op, inst.MemBytes)
		}
		if reg, ok := inst.Args[0].(x86asm.Reg); ok && isX86ByteReg(reg) {
			width = 1
		}
		var dst Operand
		var ok bool
		var err error
		if width == 1 {
			dst, ok, err = x86Operand8(inst.Args[0])
		} else {
			dst, ok, err = x86Operand32(inst.Args[0])
		}
		if err != nil || !ok || dst.Width != width {
			if err == nil {
				err = fmt.Errorf("%w: %v destination width", ErrUnsupportedAddressing, inst.Op)
			}
			return Instruction{}, true, err
		}
		switch source := inst.Args[1].(type) {
		case x86asm.Imm:
			op := OpAdcImm
			if inst.Op == x86asm.SBB {
				op = OpSbbImm
			}
			return Instruction{Op: op, Len: uint32(inst.Len), Dst: dst, Imm: int32(source)}, true, nil
		default:
			var src Operand
			if width == 1 {
				src, ok, err = x86Operand8(inst.Args[1])
			} else {
				src, ok, err = x86Operand32(inst.Args[1])
			}
			if err != nil || !ok || src.Width != width {
				if err == nil {
					err = fmt.Errorf("%w: %v source width", ErrUnsupportedAddressing, inst.Op)
				}
				return Instruction{}, true, err
			}
			op := OpAdcOperands
			if inst.Op == x86asm.SBB {
				op = OpSbbOperands
			}
			return Instruction{Op: op, Len: uint32(inst.Len), Dst: dst, Src: src}, true, nil
		}
	default:
		return Instruction{}, false, nil
	}
}

func decodeX86Atomic(inst x86asm.Inst) (Instruction, bool, error) {
	if inst.Op != x86asm.CMPXCHG && inst.Op != x86asm.XADD {
		return Instruction{}, false, nil
	}
	if inst.DataSize != 32 || len(inst.Args) < 2 || inst.Args[0] == nil || inst.Args[1] == nil {
		return Instruction{}, true, fmt.Errorf("%w: %v data size %d or operands", ErrUnsupportedAddressing, inst.Op, inst.DataSize)
	}
	width := uint8(4)
	if inst.MemBytes == 1 {
		width = 1
	} else if inst.MemBytes != 0 && inst.MemBytes != 4 {
		return Instruction{}, true, fmt.Errorf("%w: %v memory width %d", ErrUnsupportedAddressing, inst.Op, inst.MemBytes)
	}
	if reg, ok := inst.Args[0].(x86asm.Reg); ok && isX86ByteReg(reg) {
		width = 1
	}
	if reg, ok := inst.Args[1].(x86asm.Reg); ok && isX86ByteReg(reg) {
		if width == 4 {
			width = 1
		}
	}
	var dst, src Operand
	var ok bool
	var err error
	if width == 1 {
		dst, ok, err = x86Operand8(inst.Args[0])
		if err == nil && ok {
			src, ok, err = x86Operand8(inst.Args[1])
		}
	} else {
		dst, ok, err = x86Operand32(inst.Args[0])
		if err == nil && ok {
			src, ok, err = x86Operand32(inst.Args[1])
		}
	}
	if err != nil || !ok || dst.Width != width || src.Width != width {
		if err == nil {
			err = fmt.Errorf("%w: %v operand width", ErrUnsupportedAddressing, inst.Op)
		}
		return Instruction{}, true, err
	}
	op := OpCmpxchg
	if inst.Op == x86asm.XADD {
		op = OpXadd
	}
	return Instruction{Op: op, Len: uint32(inst.Len), Dst: dst, Src: src}, true, nil
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

func x86Operand8(arg x86asm.Arg) (Operand, bool, error) {
	if memory, ok := arg.(x86asm.Mem); ok {
		operand, ok, err := x86Operand32(memory)
		if err != nil || !ok {
			return Operand{}, false, err
		}
		operand.Width = 1
		return operand, true, nil
	}
	reg, ok := arg.(x86asm.Reg)
	if !ok {
		return Operand{}, false, nil
	}
	var base Reg32
	var offset uint8
	switch reg {
	case x86asm.AL:
		base = EAX
	case x86asm.CL:
		base = ECX
	case x86asm.DL:
		base = EDX
	case x86asm.BL:
		base = EBX
	case x86asm.AH:
		base, offset = EAX, 1
	case x86asm.CH:
		base, offset = ECX, 1
	case x86asm.DH:
		base, offset = EDX, 1
	case x86asm.BH:
		base, offset = EBX, 1
	default:
		return Operand{}, false, fmt.Errorf("%w: byte register %v", ErrUnsupportedAddressing, reg)
	}
	return Operand{Reg: base, Width: 1, ByteOffset: offset}, true, nil
}

func decodeX86Setcc(inst x86asm.Inst) (Instruction, bool, error) {
	if len(inst.Args) < 1 || inst.Args[0] == nil {
		return Instruction{}, false, nil
	}
	var condition uint8
	switch inst.Op {
	case x86asm.SETB:
		condition = 0
	case x86asm.SETAE:
		condition = 1
	case x86asm.SETE:
		condition = 2
	case x86asm.SETNE:
		condition = 3
	case x86asm.SETBE:
		condition = 4
	case x86asm.SETA:
		condition = 5
	case x86asm.SETS:
		condition = 6
	case x86asm.SETNS:
		condition = 7
	case x86asm.SETP:
		condition = 8
	case x86asm.SETNP:
		condition = 9
	case x86asm.SETL:
		condition = 10
	case x86asm.SETGE:
		condition = 11
	case x86asm.SETLE:
		condition = 12
	case x86asm.SETG:
		condition = 13
	case x86asm.SETO:
		condition = 14
	case x86asm.SETNO:
		condition = 15
	default:
		return Instruction{}, false, nil
	}
	dst, ok, err := x86Operand8(inst.Args[0])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	return Instruction{Op: OpSetcc, Len: uint32(inst.Len), Dst: dst, Group: condition}, true, nil
}

func decodeX86CMOVcc(inst x86asm.Inst) (Instruction, bool, error) {
	if inst.DataSize != 32 || len(inst.Args) < 2 || inst.Args[0] == nil || inst.Args[1] == nil {
		return Instruction{}, false, nil
	}
	var condition uint8
	switch inst.Op {
	case x86asm.CMOVA:
		condition = 5
	case x86asm.CMOVAE:
		condition = 1
	case x86asm.CMOVB:
		condition = 0
	case x86asm.CMOVBE:
		condition = 4
	case x86asm.CMOVE:
		condition = 2
	case x86asm.CMOVG:
		condition = 13
	case x86asm.CMOVGE:
		condition = 11
	case x86asm.CMOVL:
		condition = 10
	case x86asm.CMOVLE:
		condition = 12
	case x86asm.CMOVNE:
		condition = 3
	case x86asm.CMOVNO:
		condition = 15
	case x86asm.CMOVNP:
		condition = 9
	case x86asm.CMOVNS:
		condition = 7
	case x86asm.CMOVO:
		condition = 14
	case x86asm.CMOVP:
		condition = 8
	case x86asm.CMOVS:
		condition = 6
	default:
		return Instruction{}, false, nil
	}
	dst, ok, err := x86Operand32(inst.Args[0])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	if dst.IsMem {
		return Instruction{}, true, fmt.Errorf("%w: CMOV destination %v", ErrUnsupportedAddressing, inst.Args[0])
	}
	src, ok, err := x86Operand32(inst.Args[1])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	return Instruction{Op: OpCMOVcc, Len: uint32(inst.Len), Dst: dst, Src: src, Group: condition}, true, nil
}

func decodeX86Jcc(inst x86asm.Inst) (Instruction, bool, error) {
	if inst.DataSize != 32 || len(inst.Args) < 1 || inst.Args[0] == nil {
		return Instruction{}, false, nil
	}
	var condition uint8
	switch inst.Op {
	case x86asm.JB:
		condition = 0
	case x86asm.JAE:
		condition = 1
	case x86asm.JE:
		condition = 2
	case x86asm.JNE:
		condition = 3
	case x86asm.JBE:
		condition = 4
	case x86asm.JA:
		condition = 5
	case x86asm.JS:
		condition = 6
	case x86asm.JNS:
		condition = 7
	case x86asm.JP:
		condition = 8
	case x86asm.JNP:
		condition = 9
	case x86asm.JL:
		condition = 10
	case x86asm.JGE:
		condition = 11
	case x86asm.JLE:
		condition = 12
	case x86asm.JG:
		condition = 13
	case x86asm.JO:
		condition = 14
	case x86asm.JNO:
		condition = 15
	default:
		return Instruction{}, false, nil
	}
	relative, ok := inst.Args[0].(x86asm.Rel)
	if !ok {
		return Instruction{}, true, fmt.Errorf("%w: %v operand %T", ErrUnsupportedAddressing, inst.Op, inst.Args[0])
	}
	return Instruction{Op: OpJcc, Len: uint32(inst.Len), Rel: int32(relative), Group: condition}, true, nil
}

func decodeX86Loop(inst x86asm.Inst) (Instruction, bool, error) {
	if len(inst.Args) < 1 || inst.Args[0] == nil {
		return Instruction{}, false, nil
	}
	var group uint8
	switch inst.Op {
	case x86asm.LOOP:
		if inst.DataSize != 32 || inst.AddrSize != 32 {
			return Instruction{}, true, fmt.Errorf("%w: LOOP data/address size %d/%d", ErrUnsupportedAddressing, inst.DataSize, inst.AddrSize)
		}
		group = 0
	case x86asm.LOOPE:
		if inst.DataSize != 32 || inst.AddrSize != 32 {
			return Instruction{}, true, fmt.Errorf("%w: LOOPE data/address size %d/%d", ErrUnsupportedAddressing, inst.DataSize, inst.AddrSize)
		}
		group = 1
	case x86asm.LOOPNE:
		if inst.DataSize != 32 || inst.AddrSize != 32 {
			return Instruction{}, true, fmt.Errorf("%w: LOOPNE data/address size %d/%d", ErrUnsupportedAddressing, inst.DataSize, inst.AddrSize)
		}
		group = 2
	case x86asm.JECXZ:
		if inst.DataSize != 32 || inst.AddrSize != 32 {
			return Instruction{}, true, fmt.Errorf("%w: JECXZ data/address size %d/%d", ErrUnsupportedAddressing, inst.DataSize, inst.AddrSize)
		}
		group = 3
	case x86asm.JCXZ:
		if inst.DataSize != 32 || inst.AddrSize != 16 {
			return Instruction{}, true, fmt.Errorf("%w: JCXZ data/address size %d/%d", ErrUnsupportedAddressing, inst.DataSize, inst.AddrSize)
		}
		group = 4
	default:
		return Instruction{}, false, nil
	}
	relative, ok := inst.Args[0].(x86asm.Rel)
	if !ok {
		return Instruction{}, true, fmt.Errorf("%w: %v operand %T", ErrUnsupportedAddressing, inst.Op, inst.Args[0])
	}
	return Instruction{Op: OpLoop, Len: uint32(inst.Len), Rel: int32(relative), Group: group}, true, nil
}

func decodeX86Xchg(inst x86asm.Inst) (Instruction, bool, error) {
	if inst.Op != x86asm.XCHG || inst.DataSize != 32 || len(inst.Args) < 2 || inst.Args[0] == nil || inst.Args[1] == nil {
		return Instruction{}, false, nil
	}

	width := inst.MemBytes
	if width == 0 {
		width = 4
		if reg, ok := inst.Args[0].(x86asm.Reg); ok {
			switch reg {
			case x86asm.AL, x86asm.CL, x86asm.DL, x86asm.BL, x86asm.AH, x86asm.CH, x86asm.DH, x86asm.BH:
				width = 1
			}
		}
	}
	if width != 1 && width != 4 {
		return Instruction{}, true, fmt.Errorf("%w: XCHG width %d", ErrUnsupportedAddressing, width)
	}

	decodeOperand := x86Operand32
	if width == 1 {
		decodeOperand = x86Operand8
	}
	dst, ok, err := decodeOperand(inst.Args[0])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	src, ok, err := decodeOperand(inst.Args[1])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	if dst.Width != src.Width {
		return Instruction{}, true, fmt.Errorf("%w: XCHG operand widths %d and %d", ErrUnsupportedAddressing, dst.Width, src.Width)
	}
	if dst.IsMem && src.IsMem {
		return Instruction{}, true, fmt.Errorf("%w: XCHG memory to memory", ErrUnsupportedAddressing)
	}
	return Instruction{Op: OpXchg, Len: uint32(inst.Len), Dst: dst, Src: src}, true, nil
}

func decodeX86AddSub(inst x86asm.Inst) (Instruction, bool, error) {
	if inst.DataSize != 32 || len(inst.Args) < 2 || inst.Args[0] == nil || inst.Args[1] == nil {
		return Instruction{}, false, nil
	}
	if inst.Op != x86asm.ADD && inst.Op != x86asm.SUB {
		return Instruction{}, false, nil
	}
	if inst.MemBytes != 0 && inst.MemBytes != 4 {
		return Instruction{}, true, fmt.Errorf("%w: %v memory width %d", ErrUnsupportedAddressing, inst.Op, inst.MemBytes)
	}

	dst, ok, err := x86Operand32(inst.Args[0])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	if dst.Width != 4 {
		return Instruction{}, true, fmt.Errorf("%w: %v destination width %d", ErrUnsupportedAddressing, inst.Op, dst.Width)
	}

	if imm, ok := inst.Args[1].(x86asm.Imm); ok {
		if inst.Op == x86asm.ADD {
			return Instruction{Op: OpAddOperandImm, Len: uint32(inst.Len), Dst: dst, Imm: int32(imm)}, true, nil
		}
		return Instruction{Op: OpSubOperandImm, Len: uint32(inst.Len), Dst: dst, Imm: int32(imm)}, true, nil
	}

	src, ok, err := x86Operand32(inst.Args[1])
	if err != nil || !ok {
		return Instruction{}, true, err
	}
	if src.Width != 4 {
		return Instruction{}, true, fmt.Errorf("%w: %v source width %d", ErrUnsupportedAddressing, inst.Op, src.Width)
	}
	if inst.Op == x86asm.ADD {
		return Instruction{Op: OpAddOperands, Len: uint32(inst.Len), Dst: dst, Src: src}, true, nil
	}
	return Instruction{Op: OpSubOperands, Len: uint32(inst.Len), Dst: dst, Src: src}, true, nil
}

func decodeX86MulDiv(inst x86asm.Inst) (Instruction, bool, error) {
	if inst.DataSize != 32 || len(inst.Args) < 1 || inst.Args[0] == nil {
		return Instruction{}, false, nil
	}
	switch inst.Op {
	case x86asm.MUL, x86asm.IMUL, x86asm.DIV, x86asm.IDIV:
	default:
		return Instruction{}, false, nil
	}
	if inst.MemBytes != 0 && inst.MemBytes != 4 {
		return Instruction{}, true, fmt.Errorf("%w: %v memory width %d", ErrUnsupportedAddressing, inst.Op, inst.MemBytes)
	}

	decodeOneOperand := func(arg x86asm.Arg) (Operand, error) {
		operand, ok, err := x86Operand32(arg)
		if err != nil {
			return Operand{}, err
		}
		if !ok || operand.Width != 4 {
			return Operand{}, fmt.Errorf("%w: %v source width", ErrUnsupportedAddressing, inst.Op)
		}
		return operand, nil
	}

	switch inst.Op {
	case x86asm.MUL:
		src, err := decodeOneOperand(inst.Args[0])
		if err != nil {
			return Instruction{}, true, err
		}
		return Instruction{Op: OpMulImplicit, Len: uint32(inst.Len), Src: src}, true, nil
	case x86asm.DIV:
		src, err := decodeOneOperand(inst.Args[0])
		if err != nil {
			return Instruction{}, true, err
		}
		return Instruction{Op: OpDivImplicit, Len: uint32(inst.Len), Src: src}, true, nil
	case x86asm.IDIV:
		src, err := decodeOneOperand(inst.Args[0])
		if err != nil {
			return Instruction{}, true, err
		}
		return Instruction{Op: OpIDivImplicit, Len: uint32(inst.Len), Src: src}, true, nil
	case x86asm.IMUL:
		if len(inst.Args) < 2 || inst.Args[1] == nil {
			src, err := decodeOneOperand(inst.Args[0])
			if err != nil {
				return Instruction{}, true, err
			}
			return Instruction{Op: OpIMulImplicit, Len: uint32(inst.Len), Src: src}, true, nil
		}
		dst, ok, err := x86Operand32(inst.Args[0])
		if err != nil || !ok || dst.Width != 4 {
			if err == nil {
				err = fmt.Errorf("%w: IMUL destination width", ErrUnsupportedAddressing)
			}
			return Instruction{}, true, err
		}
		src, err := decodeOneOperand(inst.Args[1])
		if err != nil {
			return Instruction{}, true, err
		}
		if len(inst.Args) < 3 || inst.Args[2] == nil {
			return Instruction{Op: OpIMulOperands, Len: uint32(inst.Len), Dst: dst, Src: src}, true, nil
		}
		imm, ok := inst.Args[2].(x86asm.Imm)
		if !ok {
			return Instruction{}, true, fmt.Errorf("%w: IMUL immediate %T", ErrUnsupportedAddressing, inst.Args[2])
		}
		return Instruction{Op: OpIMulOperands, Len: uint32(inst.Len), Dst: dst, Src: src, Imm: int32(imm), Group: 1}, true, nil
	}
	return Instruction{}, false, nil
}

func decodeX86String(inst x86asm.Inst) (Instruction, bool, error) {
	var op Op
	var width uint8
	switch inst.Op {
	case x86asm.MOVSB:
		op, width = OpMovs, 1
	case x86asm.MOVSD:
		op, width = OpMovs, 4
	case x86asm.STOSB:
		op, width = OpStos, 1
	case x86asm.STOSD:
		op, width = OpStos, 4
	case x86asm.LODSB:
		op, width = OpLods, 1
	case x86asm.LODSD:
		op, width = OpLods, 4
	case x86asm.SCASB:
		op, width = OpScas, 1
	case x86asm.SCASD:
		op, width = OpScas, 4
	case x86asm.CMPSB:
		op, width = OpCmps, 1
	case x86asm.CMPSD:
		op, width = OpCmps, 4
	default:
		return Instruction{}, false, nil
	}
	if inst.DataSize != 32 || inst.AddrSize != 32 {
		return Instruction{}, true, fmt.Errorf("%w: %v data/address size %d/%d", ErrUnsupportedAddressing, inst.Op, inst.DataSize, inst.AddrSize)
	}

	repeat := uint8(0)
	for _, prefix := range inst.Prefix {
		if prefix == 0 {
			break
		}
		switch prefix & 0xff {
		case x86asm.PrefixREP & 0xff:
			if repeat != 0 {
				return Instruction{}, true, fmt.Errorf("%w: %v has multiple repeat prefixes", ErrUnsupportedInstruction, inst.Op)
			}
			repeat = 1
		case x86asm.PrefixREPN & 0xff:
			if repeat != 0 {
				return Instruction{}, true, fmt.Errorf("%w: %v has multiple repeat prefixes", ErrUnsupportedInstruction, inst.Op)
			}
			repeat = 2
		default:
			return Instruction{}, true, fmt.Errorf("%w: %v prefix %v", ErrUnsupportedInstruction, inst.Op, prefix)
		}
	}
	if repeat == 2 && op != OpScas && op != OpCmps {
		return Instruction{}, true, fmt.Errorf("%w: REPNE %v", ErrUnsupportedInstruction, inst.Op)
	}
	return Instruction{Op: op, Len: uint32(inst.Len), Imm: int32(width), Group: repeat}, true, nil
}

func decodeX86Stack(inst x86asm.Inst) (Instruction, bool, error) {
	if inst.DataSize != 32 {
		return Instruction{}, false, nil
	}
	firstArg := func() (x86asm.Arg, bool) {
		if len(inst.Args) == 0 || inst.Args[0] == nil {
			return nil, false
		}
		return inst.Args[0], true
	}
	switch inst.Op {
	case x86asm.PUSH:
		arg, ok := firstArg()
		if !ok {
			return Instruction{}, true, fmt.Errorf("%w: PUSH without operand", ErrInvalidInstruction)
		}
		if immediate, ok := arg.(x86asm.Imm); ok {
			return Instruction{Op: OpPushImm, Len: uint32(inst.Len), Imm: int32(immediate)}, true, nil
		}
		operand, ok, err := x86Operand32(arg)
		if err != nil || !ok {
			return Instruction{}, true, err
		}
		if operand.IsMem {
			return Instruction{Op: OpPushMem, Len: uint32(inst.Len), Src: operand}, true, nil
		}
		return Instruction{Op: OpPushReg, Len: uint32(inst.Len), Reg: operand.Reg}, true, nil
	case x86asm.POP:
		arg, ok := firstArg()
		if !ok {
			return Instruction{}, true, fmt.Errorf("%w: POP without operand", ErrInvalidInstruction)
		}
		operand, ok, err := x86Operand32(arg)
		if err != nil || !ok {
			return Instruction{}, true, err
		}
		if operand.IsMem {
			return Instruction{Op: OpPopMem, Len: uint32(inst.Len), Dst: operand}, true, nil
		}
		return Instruction{Op: OpPopReg, Len: uint32(inst.Len), Reg: operand.Reg}, true, nil
	case x86asm.CALL:
		arg, ok := firstArg()
		if !ok {
			return Instruction{}, true, fmt.Errorf("%w: CALL without operand", ErrInvalidInstruction)
		}
		if relative, ok := arg.(x86asm.Rel); ok {
			return Instruction{Op: OpCallRel, Len: uint32(inst.Len), Rel: int32(relative)}, true, nil
		}
		operand, ok, err := x86Operand32(arg)
		if err != nil || !ok {
			return Instruction{}, true, err
		}
		return Instruction{Op: OpCallOperand, Len: uint32(inst.Len), Src: operand}, true, nil
	case x86asm.RET:
		if len(inst.Args) == 0 || inst.Args[0] == nil {
			return Instruction{Op: OpRet, Len: uint32(inst.Len)}, true, nil
		}
		immediate, ok := inst.Args[0].(x86asm.Imm)
		if !ok || immediate < 0 || immediate > 0xffff {
			return Instruction{}, true, fmt.Errorf("%w: RET operand %v", ErrUnsupportedInstruction, inst.Args[0])
		}
		return Instruction{Op: OpRetImm, Len: uint32(inst.Len), Imm: int32(immediate)}, true, nil
	case x86asm.PUSHFD:
		return Instruction{Op: OpPushFlags, Len: uint32(inst.Len)}, true, nil
	case x86asm.POPFD:
		return Instruction{Op: OpPopFlags, Len: uint32(inst.Len)}, true, nil
	case x86asm.PUSHAD:
		return Instruction{Op: OpPushAll, Len: uint32(inst.Len)}, true, nil
	case x86asm.POPAD:
		return Instruction{Op: OpPopAll, Len: uint32(inst.Len)}, true, nil
	case x86asm.LEAVE:
		return Instruction{Op: OpLeave, Len: uint32(inst.Len)}, true, nil
	default:
		return Instruction{}, false, nil
	}
}
