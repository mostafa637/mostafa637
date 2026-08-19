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
	case OpSubRegImm:
		reg := state.Get(instruction.Reg)
		state.Set(instruction.Reg, e.sub(state, reg, uint32(instruction.Imm)))
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
	case OpPushReg:
		if err := push(state, state.Get(instruction.Reg)); err != nil {
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
	case OpRet:
		value, err := pop(state)
		if err != nil {
			return instruction, err
		}
		state.EIP = value
	case OpJmpRel:
		state.EIP = uint32(int64(next) + int64(instruction.Rel))
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
