package cpu

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
)

var (
	ErrUnhandledInterrupt = errors.New("cpu: unhandled interrupt")
	ErrStepLimit          = errors.New("cpu: instruction step limit reached")
	ErrDivisionByZero     = errors.New("cpu: division by zero")
	ErrDivisionOverflow   = errors.New("cpu: division quotient overflow")
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
	case OpMovByteImm:
		if err := storeOperand(state, instruction.Dst, uint32(instruction.Imm)); err != nil {
			return instruction, err
		}
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
	case OpAddOperands, OpSubOperands:
		left, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		right, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		result := left
		if instruction.Op == OpAddOperands {
			result = e.add(state, left, right)
		} else {
			result = e.sub(state, left, right)
		}
		if err := storeOperand(state, instruction.Dst, result); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpMulImplicit:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		wide := uint64(state.EAXValue()) * uint64(value)
		state.SetEAX(uint32(wide))
		state.Set(EDX, uint32(wide>>32))
		setMultiplicationFlags(state, wide>>32 != 0)
		state.EIP = next
	case OpIMulImplicit:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		wide := int64(int32(state.EAXValue())) * int64(int32(value))
		low := uint32(wide)
		state.SetEAX(low)
		state.Set(EDX, uint32(uint64(wide)>>32))
		setMultiplicationFlags(state, int64(int32(low)) != wide)
		state.EIP = next
	case OpIMulOperands:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		var wide int64
		if instruction.Group == 1 {
			wide = int64(int32(value)) * int64(instruction.Imm)
		} else {
			left, err := loadOperand(state, instruction.Dst)
			if err != nil {
				return instruction, err
			}
			wide = int64(int32(left)) * int64(int32(value))
		}
		result := uint32(wide)
		if err := storeOperand(state, instruction.Dst, result); err != nil {
			return instruction, err
		}
		setMultiplicationFlags(state, int64(int32(result)) != wide)
		state.EIP = next
	case OpDivImplicit:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		if value == 0 {
			return instruction, ErrDivisionByZero
		}
		dividend := (uint64(state.Get(EDX)) << 32) | uint64(state.EAXValue())
		quotient := dividend / uint64(value)
		if quotient > uint64(^uint32(0)) {
			return instruction, ErrDivisionOverflow
		}
		state.SetEAX(uint32(quotient))
		state.Set(EDX, uint32(dividend%uint64(value)))
		state.EIP = next
	case OpIDivImplicit:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		divisor := int64(int32(value))
		if divisor == 0 {
			return instruction, ErrDivisionByZero
		}
		dividend := (int64(int32(state.Get(EDX))) << 32) | int64(uint64(state.EAXValue()))
		if dividend == -1<<63 && divisor == -1 {
			return instruction, ErrDivisionOverflow
		}
		quotient := dividend / divisor
		if quotient < -1<<31 || quotient > 1<<31-1 {
			return instruction, ErrDivisionOverflow
		}
		remainder := dividend % divisor
		state.SetEAX(uint32(int32(quotient)))
		state.Set(EDX, uint32(int32(remainder)))
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
	case OpRotate:
		value, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		count := uint32(instruction.Imm) & 0x1f
		if instruction.Src.Width == 1 {
			count = state.Get(ECX) & 0x1f
		}
		if count == 0 {
			state.EIP = next
			break
		}
		carryIn := state.Flag(FlagCF)
		result, carry, overflow, overflowDefined := rotateValue(value, count, instruction.Group, carryIn)
		if err := storeOperand(state, instruction.Dst, result); err != nil {
			return instruction, err
		}
		state.CollapseFlags()
		if carry {
			state.EFlags |= FlagCF
		} else {
			state.EFlags &^= FlagCF
		}
		if overflowDefined {
			if overflow {
				state.EFlags |= FlagOF
			} else {
				state.EFlags &^= FlagOF
			}
		}
		state.ExpandFlags()
		state.EIP = next
	case OpUnary:
		value, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		result := value
		if instruction.Group == 0 { // NOT: flags are unaffected.
			result = ^value
		} else { // NEG: 0 - value updates arithmetic flags.
			result = e.sub(state, 0, value)
		}
		if err := storeOperand(state, instruction.Dst, result); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpSetcc:
		value := byte(0)
		if conditionValue(state, instruction.Group) {
			value = 1
		}
		if err := storeOperand(state, instruction.Dst, uint32(value)); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpCMOVcc:
		if conditionValue(state, instruction.Group) {
			value, err := loadOperand(state, instruction.Src)
			if err != nil {
				return instruction, err
			}
			if err := storeOperand(state, instruction.Dst, value); err != nil {
				return instruction, err
			}
		}
		state.EIP = next
	case OpXchg:
		dstValue, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		srcValue, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		if err := storeOperand(state, instruction.Dst, srcValue); err != nil {
			return instruction, err
		}
		if err := storeOperand(state, instruction.Src, dstValue); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpCmpxchg:
		width := instruction.Dst.Width
		accumulator := Operand{Reg: EAX, Width: width}
		accValue, err := loadOperand(state, accumulator)
		if err != nil {
			return instruction, err
		}
		dstValue, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		srcValue, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		comparison := subWidth(accValue, dstValue, width)
		if comparison.equal {
			if err := storeOperand(state, instruction.Dst, srcValue); err != nil {
				return instruction, err
			}
		} else if err := storeOperand(state, accumulator, dstValue); err != nil {
			return instruction, err
		}
		setLazyArithmeticWidth(state, accValue, dstValue, comparison.result, width, true)
		state.EIP = next
	case OpXadd:
		dstValue, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		srcValue, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		sum := addWidth(dstValue, srcValue, instruction.Dst.Width)
		if err := storeOperand(state, instruction.Dst, sum.result); err != nil {
			return instruction, err
		}
		if err := storeOperand(state, instruction.Src, dstValue); err != nil {
			return instruction, err
		}
		setLazyArithmeticWidth(state, dstValue, srcValue, sum.result, instruction.Dst.Width, false)
		state.EIP = next
	case OpAdcOperands, OpSbbOperands, OpAdcImm, OpSbbImm:
		left, err := loadOperand(state, instruction.Dst)
		if err != nil {
			return instruction, err
		}
		right := uint32(instruction.Imm)
		if instruction.Op == OpAdcOperands || instruction.Op == OpSbbOperands {
			right, err = loadOperand(state, instruction.Src)
			if err != nil {
				return instruction, err
			}
		}
		carryIn := uint32(0)
		if state.Flag(FlagCF) {
			carryIn = 1
		}
		var result uint32
		if instruction.Op == OpAdcOperands || instruction.Op == OpAdcImm {
			result = e.adc(state, left, right, instruction.Dst.Width, carryIn)
		} else {
			result = e.sbb(state, left, right, instruction.Dst.Width, carryIn)
		}
		if err := storeOperand(state, instruction.Dst, result); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpLahf:
		state.CollapseFlags()
		var ah uint32
		for _, flag := range []uint32{FlagCF, FlagPF, FlagAF, FlagZF, FlagSF} {
			if state.EFlags&flag != 0 {
				switch flag {
				case FlagCF:
					ah |= 1 << 0
				case FlagPF:
					ah |= 1 << 2
				case FlagAF:
					ah |= 1 << 4
				case FlagZF:
					ah |= 1 << 6
				case FlagSF:
					ah |= 1 << 7
				}
			}
		}
		ah |= 1 << 1 // LAHF always reports the reserved bit 1 as set.
		state.Set(EAX, (state.EAXValue()&^uint32(0xff00))|(ah<<8))
		state.EIP = next
	case OpSahf:
		state.CollapseFlags()
		ah := (state.EAXValue() >> 8) & 0xff
		var flags uint32
		if ah&(1<<0) != 0 {
			flags |= FlagCF
		}
		if ah&(1<<2) != 0 {
			flags |= FlagPF
		}
		if ah&(1<<4) != 0 {
			flags |= FlagAF
		}
		if ah&(1<<6) != 0 {
			flags |= FlagZF
		}
		if ah&(1<<7) != 0 {
			flags |= FlagSF
		}
		state.EFlags = (state.EFlags &^ (FlagCF | FlagPF | FlagAF | FlagZF | FlagSF)) | flags | FlagIF
		state.ExpandFlags()
		state.EIP = next
	case OpBswap:
		value := state.Get(instruction.Dst.Reg)
		state.Set(instruction.Dst.Reg, bits.ReverseBytes32(value))
		state.EIP = next
	case OpBitTest:
		bitIndex, target, err := bitTestTarget(state, instruction)
		if err != nil {
			return instruction, err
		}
		value, err := loadOperand(state, target)
		if err != nil {
			return instruction, err
		}
		bit := uint32(1) << bitIndex
		oldSet := value&bit != 0
		state.CollapseFlags()
		if oldSet {
			state.EFlags |= FlagCF
		} else {
			state.EFlags &^= FlagCF
		}
		state.ExpandFlags()
		switch instruction.Group {
		case 0: // BT only tests CF.
		case 1: // BTS sets the selected bit.
			value |= bit
		case 2: // BTR resets the selected bit.
			value &^= bit
		case 3: // BTC complements the selected bit.
			value ^= bit
		default:
			return instruction, fmt.Errorf("cpu: unsupported bit-test group %d", instruction.Group)
		}
		if instruction.Group != 0 {
			if err := storeOperand(state, target, value); err != nil {
				return instruction, err
			}
		}
		state.EIP = next
	case OpBitScan:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		state.CollapseFlags()
		if value == 0 {
			state.EFlags |= FlagZF
			state.ExpandFlags()
			state.EIP = next
			break
		}
		state.EFlags &^= FlagZF
		state.ExpandFlags()
		var index uint32
		if instruction.Group == 0 {
			index = uint32(bits.TrailingZeros32(value))
		} else if instruction.Group == 1 {
			index = uint32(31 - bits.LeadingZeros32(value))
		} else {
			return instruction, fmt.Errorf("cpu: unsupported bit-scan group %d", instruction.Group)
		}
		state.Set(instruction.Dst.Reg, index)
		state.EIP = next
	case OpPopcnt:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		count := uint32(bits.OnesCount32(value))
		state.Set(instruction.Dst.Reg, count)
		state.CollapseFlags()
		state.EFlags &^= FlagCF | FlagPF | FlagAF | FlagSF | FlagOF | FlagZF
		if count == 0 {
			state.EFlags |= FlagZF
		}
		state.ExpandFlags()
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
	case OpMovsxRegOperand:
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return instruction, err
		}
		switch instruction.Src.Width {
		case 1:
			value = uint32(int32(int8(value)))
		case 2:
			value = uint32(int32(int16(value)))
		default:
			return instruction, fmt.Errorf("cpu: unsupported MOVSX source width %d", instruction.Src.Width)
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
	case OpPopMem:
		value, err := pop(state)
		if err != nil {
			return instruction, err
		}
		if err := storeOperand(state, instruction.Dst, value); err != nil {
			return instruction, err
		}
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
	case OpJcc:
		if conditionValue(state, instruction.Group) {
			state.EIP = uint32(int64(next) + int64(instruction.Rel))
		} else {
			state.EIP = next
		}
	case OpLoop:
		take := false
		switch instruction.Group {
		case 0: // LOOP: decrement ECX and branch when non-zero.
			count := state.Get(ECX) - 1
			state.Set(ECX, count)
			take = count != 0
		case 1: // LOOPE/LOOPZ: decrement ECX and require ZF.
			count := state.Get(ECX) - 1
			state.Set(ECX, count)
			take = count != 0 && state.Flag(FlagZF)
		case 2: // LOOPNE/LOOPNZ: decrement ECX and require !ZF.
			count := state.Get(ECX) - 1
			state.Set(ECX, count)
			take = count != 0 && !state.Flag(FlagZF)
		case 3: // JECXZ: test the full 32-bit ECX without changing it.
			take = state.Get(ECX) == 0
		case 4: // JCXZ: address-size override selects the low 16-bit CX.
			take = uint16(state.Get(ECX)) == 0
		default:
			return instruction, fmt.Errorf("cpu: unsupported loop group %d", instruction.Group)
		}
		if take {
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
	case OpMovs:
		if err := executeString(state, instruction, func() (bool, error) {
			value, err := readStringValue(state, Address(state.Get(ESI)), uint32(instruction.Imm))
			if err != nil {
				return false, err
			}
			if err := writeStringValue(state, Address(state.Get(EDI)), uint32(instruction.Imm), value); err != nil {
				return false, err
			}
			advanceStringIndex(state, ESI, uint32(instruction.Imm))
			advanceStringIndex(state, EDI, uint32(instruction.Imm))
			return true, nil
		}); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpStos:
		if err := executeString(state, instruction, func() (bool, error) {
			width := uint32(instruction.Imm)
			value := state.EAXValue()
			if width == 1 {
				value &= 0xff
			}
			if err := writeStringValue(state, Address(state.Get(EDI)), width, value); err != nil {
				return false, err
			}
			advanceStringIndex(state, EDI, width)
			return true, nil
		}); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpLods:
		if err := executeString(state, instruction, func() (bool, error) {
			width := uint32(instruction.Imm)
			value, err := readStringValue(state, Address(state.Get(ESI)), width)
			if err != nil {
				return false, err
			}
			if width == 1 {
				state.Set(EAX, (state.EAXValue()&^0xff)|(value&0xff))
			} else {
				state.SetEAX(value)
			}
			advanceStringIndex(state, ESI, width)
			return true, nil
		}); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpScas:
		if err := executeString(state, instruction, func() (bool, error) {
			width := uint32(instruction.Imm)
			value, err := readStringValue(state, Address(state.Get(EDI)), width)
			if err != nil {
				return false, err
			}
			accumulator := state.EAXValue()
			if width == 1 {
				accumulator &= 0xff
			}
			e.sub(state, accumulator, value)
			advanceStringIndex(state, EDI, width)
			return stringRepeatContinue(state, instruction.Group), nil
		}); err != nil {
			return instruction, err
		}
		state.EIP = next
	case OpCmps:
		if err := executeString(state, instruction, func() (bool, error) {
			width := uint32(instruction.Imm)
			left, err := readStringValue(state, Address(state.Get(ESI)), width)
			if err != nil {
				return false, err
			}
			right, err := readStringValue(state, Address(state.Get(EDI)), width)
			if err != nil {
				return false, err
			}
			e.sub(state, left, right)
			advanceStringIndex(state, ESI, width)
			advanceStringIndex(state, EDI, width)
			return stringRepeatContinue(state, instruction.Group), nil
		}); err != nil {
			return instruction, err
		}
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

func bitTestTarget(state *MachineState, instruction Instruction) (uint32, Operand, error) {
	var bitIndex uint32
	if instruction.Src.Width == 0 {
		bitIndex = uint32(instruction.Imm) & 31
	} else {
		value, err := loadOperand(state, instruction.Src)
		if err != nil {
			return 0, Operand{}, err
		}
		bitIndex = value
	}
	target := instruction.Dst
	if target.IsMem && instruction.Src.Width != 0 {
		// Register-indexed memory bit operations address the 32-bit word
		// containing the requested bit, then use the index modulo 32.
		target.Memory.Disp += int32((bitIndex >> 5) * 4)
	}
	return bitIndex & 31, target, nil
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
		value := state.Get(operand.Reg)
		switch operand.Width {
		case 1:
			return (value >> (8 * operand.ByteOffset)) & 0xff, nil
		case 2:
			return value & 0xffff, nil
		default:
			return value, nil
		}
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
		switch operand.Width {
		case 1:
			mask := uint32(0xff) << (8 * operand.ByteOffset)
			current := state.Get(operand.Reg)
			state.Set(operand.Reg, (current&^mask)|((value&0xff)<<(8*operand.ByteOffset)))
		case 2:
			current := state.Get(operand.Reg)
			state.Set(operand.Reg, (current&^0xffff)|(value&0xffff))
		default:
			state.Set(operand.Reg, value)
		}
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

type widthArithmetic struct {
	result uint32
	equal  bool
}

func widthMask(width uint8) (mask, sign uint32) {
	switch width {
	case 1:
		return 0xff, 0x80
	case 2:
		return 0xffff, 0x8000
	default:
		return 0xffffffff, 0x80000000
	}
}

func (e *Executor) adc(state *MachineState, left, right uint32, width uint8, carryIn uint32) uint32 {
	mask, sign := widthMask(width)
	left &= mask
	right &= mask
	var result uint32
	var carry bool
	if width == 4 {
		var carryOut uint32
		result, carryOut = bits.Add32(left, right, carryIn)
		carry = carryOut != 0
	} else {
		sum := uint64(left) + uint64(right) + uint64(carryIn)
		result = uint32(sum) & mask
		carry = sum > uint64(mask)
	}
	overflow := ((^(left ^ right)) & (left ^ result) & sign) != 0
	setLazyArithmeticWidthWithCarry(state, left, right, result, width, carry, overflow)
	return result
}

func (e *Executor) sbb(state *MachineState, left, right uint32, width uint8, borrowIn uint32) uint32 {
	mask, sign := widthMask(width)
	left &= mask
	right &= mask
	var result uint32
	var borrow bool
	if width == 4 {
		var borrowOut uint32
		result, borrowOut = bits.Sub32(left, right, borrowIn)
		borrow = borrowOut != 0
	} else {
		subtrahend := uint64(right) + uint64(borrowIn)
		result = uint32(uint64(left)-subtrahend) & mask
		borrow = uint64(left) < subtrahend
	}
	overflow := ((left ^ right) & (left ^ result) & sign) != 0
	setLazyArithmeticWidthWithCarry(state, left, right, result, width, borrow, overflow)
	return result
}

func setLazyArithmeticWidthWithCarry(state *MachineState, left, right, result uint32, width uint8, carry, overflow bool) {
	mask, sign := widthMask(width)
	left &= mask
	right &= mask
	result &= mask
	if result&sign != 0 {
		result |= ^mask
	}
	state.SetLazyArithmetic(left, right, result, carry, overflow, true)
}

func addWidth(left, right uint32, width uint8) widthArithmetic {
	mask, _ := widthMask(width)
	left &= mask
	right &= mask
	return widthArithmetic{result: (left + right) & mask}
}

func subWidth(left, right uint32, width uint8) widthArithmetic {
	mask, _ := widthMask(width)
	left &= mask
	right &= mask
	return widthArithmetic{result: (left - right) & mask, equal: left == right}
}

func setLazyArithmeticWidth(state *MachineState, left, right, result uint32, width uint8, subtraction bool) {
	mask, sign := widthMask(width)
	left &= mask
	right &= mask
	result &= mask
	var carry bool
	if subtraction {
		carry = left < right
	} else {
		carry = uint64(left)+uint64(right) > uint64(mask)
	}
	overflow := false
	if subtraction {
		overflow = ((left ^ right) & (left ^ result) & sign) != 0
	} else {
		overflow = ((^(left ^ right)) & (left ^ result) & sign) != 0
	}
	if result&sign != 0 {
		result |= ^mask
	}
	state.SetLazyArithmetic(left, right, result, carry, overflow, true)
}

func setMultiplicationFlags(state *MachineState, overflow bool) {
	state.CF = boolByte(overflow)
	state.OF = boolByte(overflow)
	state.Lazy = 0
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

func rotateValue(value, count uint32, group uint8, carryIn bool) (result uint32, carry, overflow, overflowDefined bool) {
	count &= 0x1f
	if count == 0 {
		return value, carryIn, false, false
	}
	switch group {
	case 0: // ROL.
		result = bits.RotateLeft32(value, int(count))
		carry = result&1 != 0
		if count == 1 {
			overflow = (result >> 31) != (result & 1)
			overflowDefined = true
		}
	case 1: // ROR.
		result = bits.RotateLeft32(value, -int(count))
		carry = result>>31 != 0
		if count == 1 {
			overflow = (result>>31)&1 != (result>>30)&1
			overflowDefined = true
		}
	case 2: // RCL, rotate through the carry flag.
		carryBit := uint32(0)
		if carryIn {
			carryBit = 1
		}
		for i := uint32(0); i < count; i++ {
			nextCarry := value>>31 != 0
			value = (value << 1) | carryBit
			if nextCarry {
				carryBit = 1
			} else {
				carryBit = 0
			}
		}
		result = value
		carry = carryBit != 0
		if count == 1 {
			overflow = (result >> 31) != (result & 1)
			overflowDefined = true
		}
	case 3: // RCR, rotate through the carry flag.
		carryBit := uint32(0)
		if carryIn {
			carryBit = 1
		}
		for i := uint32(0); i < count; i++ {
			nextCarry := value&1 != 0
			value = (value >> 1) | (carryBit << 31)
			if nextCarry {
				carryBit = 1
			} else {
				carryBit = 0
			}
		}
		result = value
		carry = carryBit != 0
		if count == 1 {
			overflow = (result>>31)&1 != (result>>30)&1
			overflowDefined = true
		}
	default:
		return value, carryIn, false, false
	}
	return result, carry, overflow, overflowDefined
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

func conditionValue(state *MachineState, condition uint8) bool {
	cf := state.Flag(FlagCF)
	of := state.Flag(FlagOF)
	sf := state.Flag(FlagSF)
	zf := state.Flag(FlagZF)
	pf := state.Flag(FlagPF)
	switch condition {
	case 0: // B/NAE/C
		return cf
	case 1: // AE/NB/NC
		return !cf
	case 2: // E/Z
		return zf
	case 3: // NE/NZ
		return !zf
	case 4: // BE/NA
		return cf || zf
	case 5: // A/NBE
		return !cf && !zf
	case 6: // S
		return sf
	case 7: // NS
		return !sf
	case 8: // P/PE
		return pf
	case 9: // NP/PO
		return !pf
	case 10: // L/NGE
		return sf != of
	case 11: // GE/NL
		return sf == of
	case 12: // LE/NG
		return zf || sf != of
	case 13: // G/ NLE
		return !zf && sf == of
	case 14: // O
		return of
	case 15: // NO
		return !of
	default:
		return false
	}
}

func executeString(state *MachineState, instruction Instruction, step func() (bool, error)) error {
	repeat := instruction.Group
	count := uint32(1)
	if repeat != 0 {
		count = state.Get(ECX)
		if count == 0 {
			return nil
		}
	}
	for count != 0 {
		continueRepeat, err := step()
		if err != nil {
			return err
		}
		if repeat == 0 {
			return nil
		}
		count--
		state.Set(ECX, count)
		if count == 0 || !continueRepeat {
			return nil
		}
	}
	return nil
}

func stringRepeatContinue(state *MachineState, repeat uint8) bool {
	switch repeat {
	case 1: // REP/REPE: continue while ZF=1 for SCAS/CMPS.
		return state.Flag(FlagZF)
	case 2: // REPNE: continue while ZF=0 for SCAS/CMPS.
		return !state.Flag(FlagZF)
	default:
		return false
	}
}

func readStringValue(state *MachineState, address Address, width uint32) (uint32, error) {
	switch width {
	case 1:
		var raw [1]byte
		if err := state.Memory.Read(address, raw[:]); err != nil {
			return 0, err
		}
		return uint32(raw[0]), nil
	case 4:
		var raw [4]byte
		if err := state.Memory.Read(address, raw[:]); err != nil {
			return 0, err
		}
		return binary.LittleEndian.Uint32(raw[:]), nil
	default:
		return 0, ErrUnsupportedAddressing
	}
}

func writeStringValue(state *MachineState, address Address, width, value uint32) error {
	switch width {
	case 1:
		return state.Memory.Write(address, []byte{byte(value)})
	case 4:
		return state.Memory.Write(address, uint32Bytes(value))
	default:
		return ErrUnsupportedAddressing
	}
}

func advanceStringIndex(state *MachineState, register Reg32, width uint32) {
	value := state.Get(register)
	if state.Flag(FlagDF) {
		state.Set(register, value-width)
	} else {
		state.Set(register, value+width)
	}
}
