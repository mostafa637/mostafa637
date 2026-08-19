package cpu

import (
	"encoding/binary"
	"errors"
	"math/bits"
)

var (
	ErrUnhandledInterrupt = errors.New("cpu: unhandled interrupt")
	ErrStepLimit          = errors.New("cpu: instruction step limit reached")
)

type SyscallHandler func(*MachineState) int32

type Executor struct {
	Syscall SyscallHandler
	Halted  bool
}

func NewExecutor(syscall SyscallHandler) *Executor {
	return &Executor{Syscall: syscall}
}

func (e *Executor) Step(state *MachineState) (Instruction, error) {
	if state == nil || state.Memory == nil {
		return Instruction{}, ErrUnmapped
	}
	if e.Halted {
		return Instruction{}, nil
	}
	instruction, err := Decode(state.Memory, Address(state.EIP))
	if err != nil {
		return Instruction{}, err
	}
	next := state.EIP + instruction.Len
	state.Cycle++

	switch instruction.Op {
	case OpNop:
		state.EIP = next
	case OpMovRegImm:
		state.Set(instruction.Reg, uint32(instruction.Imm))
		state.EIP = next
	case OpMovRegReg:
		state.Set(instruction.Reg, state.Get(instruction.Reg2))
		state.EIP = next
	case OpMovRegMem:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		state.Set(instruction.Dst.Reg, value)
		state.EIP = next
	case OpMovMemReg:
		if err := storeOperand(state, instruction.Dst, state.Get(instruction.Src.Reg)); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpMovMemImm:
		if err := storeOperand(state, instruction.Dst, uint32(instruction.Imm)); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpLeaRegMem:
		address, err := effectiveAddress(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		state.Set(instruction.Dst.Reg, uint32(address))
		state.EIP = next
	case OpAddEAXImm:
		state.SetEAX(e.add(state, state.EAXValue(), uint32(instruction.Imm)))
		state.EIP = next
	case OpSubEAXImm:
		state.SetEAX(e.sub(state, state.EAXValue(), uint32(instruction.Imm)))
		state.EIP = next
	case OpAddRegImm:
		reg := state.Get(instruction.Reg)
		state.Set(instruction.Reg, e.add(state, reg, uint32(instruction.Imm)))
		state.EIP = next
	case OpAddOperandImm:
		left, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		if err := storeOperand(state, instruction.Dst, e.add(state, left, uint32(instruction.Imm))); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpSubRegImm:
		reg := state.Get(instruction.Reg)
		state.Set(instruction.Reg, e.sub(state, reg, uint32(instruction.Imm)))
		state.EIP = next
	case OpSubOperandImm:
		left, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		if err := storeOperand(state, instruction.Dst, e.sub(state, left, uint32(instruction.Imm))); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpCmpOperandImm:
		left, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		e.sub(state, left, uint32(instruction.Imm))
		state.EIP = next
	case OpXorRegReg:
		left, right := state.Get(instruction.Reg), state.Get(instruction.Reg2)
		result := left ^ right
		state.Set(instruction.Reg, result)
		state.SetLazyArithmetic(left, right, result, false, false, false)
		state.EIP = next
	case OpCmpRegReg:
		left, right := state.Get(instruction.Reg), state.Get(instruction.Reg2)
		e.sub(state, left, right)
		state.EIP = next
	case OpCmpRegImm:
		left := state.Get(instruction.Reg)
		e.sub(state, left, uint32(instruction.Imm))
		state.EIP = next
	case OpCmpEAXImm:
		e.sub(state, state.EAXValue(), uint32(instruction.Imm))
		state.EIP = next
	case OpTestRegReg:
		left, right := state.Get(instruction.Reg), state.Get(instruction.Reg2)
		result := left & right
		state.SetLazyArithmetic(left, right, result, false, false, false)
		state.EIP = next
	case OpTestOperands:
		left, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		right, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		state.SetLazyArithmetic(left, right, left&right, false, false, false)
		state.EIP = next
	case OpTestImm:
		left, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		right := uint32(instruction.Imm)
		state.SetLazyArithmetic(left, right, left&right, false, false, false)
		state.EIP = next
	case OpLogical:
		left, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		right, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		result := logicalValue(left, right, instruction.Group)
		if err := storeOperand(state, instruction.Dst, result); err != nil {
			return instruction, err
		}
		state.SetLazyArithmetic(left, right, result, false, false, false)
		state.EIP = next
	case OpLogicalImm:
		left, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		result := logicalValue(left, uint32(instruction.Imm), instruction.Group)
		if err := storeOperand(state, instruction.Dst, result); err != nil {
			return instruction, err
		}
		state.SetLazyArithmetic(left, uint32(instruction.Imm), result, false, false, false)
		state.EIP = next
	case OpShift:
		value, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		count := uint32(instruction.Imm) & 0x1f
		if instruction.Src.Width == 1 {
			count = state.Get(ECX) & 0x1f
		}
		if count != 0 {
			result, carry, overflow := shiftValue(value, count, instruction.Group)
			if err := storeOperand(state, instruction.Dst, result); err != nil {
				return instruction, err
			}
			state.SetLazyArithmetic(value, count, result, carry, overflow, false)
		}
		state.EIP = next
	case OpIncReg:
		carry := state.Flag(FlagCF)
		reg := state.Get(instruction.Reg)
		state.Set(instruction.Reg, e.add(state, reg, 1))
		state.CF = boolByte(carry)
		state.EIP = next
	case OpDecReg:
		carry := state.Flag(FlagCF)
		reg := state.Get(instruction.Reg)
		state.Set(instruction.Reg, e.sub(state, reg, 1))
		state.CF = boolByte(carry)
		state.EIP = next
	case OpIncOperand:
		carry := state.Flag(FlagCF)
		value, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		if err := storeOperand(state, instruction.Dst, e.add(state, value, 1)); err != nil {
			return instruction, err
		}
		state.CF = boolByte(carry)
		state.EIP = next
	case OpDecOperand:
		carry := state.Flag(FlagCF)
		value, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		if err := storeOperand(state, instruction.Dst, e.sub(state, value, 1)); err != nil {
			return instruction, err
		}
		state.CF = boolByte(carry)
		state.EIP = next
	case OpImulRegOperand:
		left := state.Get(instruction.Dst.Reg)
		right, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		wide := int64(int32(left)) * int64(int32(right))
		result := uint32(wide)
		overflow := wide != int64(int32(result))
		state.Set(instruction.Dst.Reg, result)
		state.SetLazyArithmetic(left, right, result, overflow, overflow, false)
		state.EIP = next
	case OpMovzxRegOperand:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		if instruction.Src.Width == 1 {
			value &= 0xff
		} else if instruction.Src.Width == 2 {
			value &= 0xffff
		}
		state.Set(instruction.Dst.Reg, value)
		state.EIP = next
	case OpPushReg:
		if err := push(state, state.Get(instruction.Reg)); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpPushMem:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		if err := push(state, value); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpPopReg:
		value, err := pop(state)
		if err != nil {
			return instruction, err
		}
		state.Set(instruction.Reg, value)
		state.EIP = next
	case OpPushImm:
		if err := push(state, uint32(instruction.Imm)); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpCallRel:
		if err := push(state, next); err != nil {
			return instruction, err
		}
		state.EIP = uint32(int64(next) + int64(instruction.Rel))
	case OpCallOperand:
		target, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		if err := push(state, next); err != nil {
			return instruction, err
		}
		state.EIP = target
	case OpRet:
		value, err := pop(state)
		if err != nil {
			return instruction, err
		}
		state.EIP = value
	case OpJmpRel:
		state.EIP = uint32(int64(next) + int64(instruction.Rel))
	case OpJmpOperand:
		target, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		state.EIP = target
	case OpJzRel:
		if state.Flag(FlagZF) {
			state.EIP = uint32(int64(next) + int64(instruction.Rel))
		} else {
			state.EIP = next
		}
	case OpJnzRel:
		if !state.Flag(FlagZF) {
			state.EIP = uint32(int64(next) + int64(instruction.Rel))
		} else {
			state.EIP = next
		}
	case OpInt:
		state.EIP = next
		state.TrapNo = uint32(instruction.Vector)
		if instruction.Vector != 0x80 || e.Syscall == nil {
			return instruction, ErrUnhandledInterrupt
		}
		e.Syscall(state)
	case OpHalt:
		state.EIP = next
		e.Halted = true
	case OpPushFlags:
		state.CollapseFlags()
		if err := push(state, state.EFlags); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpPopFlags:
		value, err := pop(state)
		if err != nil {
			return instruction, err
		}
		state.SetEFlags(value)
		state.EIP = next
	case OpPushAll:
		oldESP := state.Get(ESP)
		for _, reg := range []Reg32{EAX, ECX, EDX, EBX} {
			if err := push(state, state.Get(reg)); err != nil {
				return instruction, err
			}
		}
		if err := push(state, oldESP); err != nil {
			return instruction, err
		}
		for _, reg := range []Reg32{EBP, ESI, EDI} {
			if err := push(state, state.Get(reg)); err != nil {
				return instruction, err
			}
		}
		state.EIP = next
	case OpPopAll:
		for _, reg := range []Reg32{EDI, ESI, EBP} {
			value, err := pop(state)
			if err != nil {
				return instruction, err
			}
			state.Set(reg, value)
		}
		if _, err := pop(state); err != nil { // saved ESP is discarded
			return instruction, err
		}
		for _, reg := range []Reg32{EBX, EDX, ECX, EAX} {
			value, err := pop(state)
			if err != nil {
				return instruction, err
			}
			state.Set(reg, value)
		}
		state.EIP = next
	case OpLeave:
		state.Set(ESP, state.Get(EBP))
		value, err := pop(state)
		if err != nil {
			return instruction, err
		}
		state.Set(EBP, value)
		state.EIP = next
	case OpRetImm:
		value, err := pop(state)
		if err != nil {
			return instruction, err
		}
		state.Set(ESP, state.Get(ESP)+uint32(uint16(instruction.Imm)))
		state.EIP = value
	case OpCWDE:
		state.SetEAX(uint32(int32(int16(state.EAXValue() & 0xffff))))
		state.EIP = next
	case OpCDQ:
		if int32(state.EAXValue()) < 0 {
			state.Set(EDX, ^uint32(0))
		} else {
			state.Set(EDX, 0)
		}
		state.EIP = next
	case OpCLD:
		state.EFlags &^= FlagDF
		state.DFOffset = 0
		state.EIP = next
	case OpSTD:
		state.EFlags |= FlagDF
		state.DFOffset = ^uint32(0)
		state.EIP = next
	default:
		return instruction, ErrUnsupportedInstruction
	}
	return instruction, nil
}

func (e *Executor) Run(state *MachineState, maxSteps int) error {
	if maxSteps <= 0 {
		return ErrStepLimit
	}
	for i := 0; i < maxSteps; i++ {
		if e.Halted {
			return nil
		}
		if _, err := e.Step(state); err != nil {
			return err
		}
	}
	if e.Halted {
		return nil
	}
	return ErrStepLimit
}

func effectiveAddress(state *MachineState, operand Operand) (Address, error) {
	if !operand.IsMem {
		return 0, ErrUnsupportedAddressing
	}
	memory := operand.Memory
	var address uint32
	if memory.HasBase {
		address += state.Get(memory.Base)
	}
	if memory.HasIndex {
		address += state.Get(memory.Index) * uint32(memory.Scale)
	}
	address += uint32(memory.Disp)
	switch memory.Segment {
	case SegmentFS:
		address += state.FSBase
	case SegmentGS:
		base := state.GSBase
		if base == 0 {
			base = state.TLS
		}
		address += base
	}
	return Address(address), nil
}

func loadOperand(state *MachineState, operand Operand) (uint32, error) {
	if !operand.IsMem {
		return state.Get(operand.Reg), nil
	}
	address, err := effectiveAddress(state, operand)
	if err != nil {
		return 0, err
	}
	width := operand.Width
	if width == 0 {
		width = 4
	}
	var raw [4]byte
	if err := state.Memory.Read(address, raw[:width]); err != nil {
		return 0, err
	}
	switch width {
	case 1:
		return uint32(raw[0]), nil
	case 2:
		return uint32(binary.LittleEndian.Uint16(raw[:2])), nil
	case 4:
		return binary.LittleEndian.Uint32(raw[:]), nil
	default:
		return 0, ErrUnsupportedAddressing
	}
}

func storeOperand(state *MachineState, operand Operand, value uint32) error {
	if !operand.IsMem {
		state.Set(operand.Reg, value)
		return nil
	}
	address, err := effectiveAddress(state, operand)
	if err != nil {
		return err
	}
	width := operand.Width
	if width == 0 {
		width = 4
	}
	switch width {
	case 1:
		return state.Memory.Write(address, []byte{byte(value)})
	case 2:
		var raw [2]byte
		binary.LittleEndian.PutUint16(raw[:], uint16(value))
		return state.Memory.Write(address, raw[:])
	case 4:
		return state.Memory.Write(address, uint32Bytes(value))
	default:
		return ErrUnsupportedAddressing
	}
}

func (e *Executor) add(state *MachineState, left, right uint32) uint32 {
	result, carry := bits.Add32(left, right, 0)
	overflow := ((^(left ^ right)) & (left ^ result) & 0x80000000) != 0
	state.SetLazyArithmetic(left, right, result, carry != 0, overflow, true)
	return result
}

func (e *Executor) sub(state *MachineState, left, right uint32) uint32 {
	result, borrow := bits.Sub32(left, right, 0)
	overflow := ((left ^ right) & (left ^ result) & 0x80000000) != 0
	state.SetLazyArithmetic(left, right, result, borrow != 0, overflow, true)
	return result
}

func push(state *MachineState, value uint32) error {
	sp := state.Get(ESP) - 4
	if err := state.Memory.Write(Address(sp), uint32Bytes(value)); err != nil {
		return err
	}
	state.Set(ESP, sp)
	return nil
}

func pop(state *MachineState) (uint32, error) {
	sp := state.Get(ESP)
	var raw [4]byte
	if err := state.Memory.Read(Address(sp), raw[:]); err != nil {
		return 0, err
	}
	state.Set(ESP, sp+4)
	return binary.LittleEndian.Uint32(raw[:]), nil
}

func uint32Bytes(value uint32) []byte {
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], value)
	return raw[:]
}

func logicalValue(left, right uint32, group uint8) uint32 {
	switch group {
	case 0: // OR
		return left | right
	case 1: // AND
		return left & right
	case 2: // XOR
		return left ^ right
	default:
		return 0
	}
}

func shiftValue(value, count uint32, group uint8) (result uint32, carry, overflow bool) {
	if count == 0 {
		return value, false, false
	}
	switch group {
	case 0: // SHL/SAL
		result = value << count
		carry = (value>>(32-count))&1 != 0
		if count == 1 {
			overflow = ((result ^ value) & 0x80000000) != 0
		}
	case 1: // SHR
		result = value >> count
		carry = (value>>(count-1))&1 != 0
		if count == 1 {
			overflow = value&0x80000000 != 0
		}
	case 2: // SAR
		result = uint32(int32(value) >> count)
		carry = (value>>(count-1))&1 != 0
	default:
		return value, false, false
	}
	return result, carry, overflow
}
